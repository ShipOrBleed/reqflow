package structmap

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	protoServiceRe = regexp.MustCompile(`^\s*service\s+(\w+)\s*\{`)
	protoRPCRe     = regexp.MustCompile(`^\s*rpc\s+(\w+)\s*\(\s*(\w+)\s*\)\s*returns\s*\(\s*(stream\s+)?(\w+)\s*\)`)
	protoMessageRe = regexp.MustCompile(`^\s*message\s+(\w+)\s*\{`)
	protoPackageRe = regexp.MustCompile(`^\s*package\s+([\w.]+)\s*;`)
)

// ExtractProto scans for .proto files and parses service, rpc, and message
// declarations to build KindGRPC, KindProtoRPC, and KindProtoMsg nodes.
func ExtractProto(dir string, graph *Graph) {
	workDir := dir
	if workDir == "./..." || workDir == "" {
		workDir = "."
	}

	filepath.Walk(workDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(info.Name(), ".proto") {
			parseProtoFile(path, graph)
		}
		return nil
	})

	// Cross-reference proto RPC nodes with existing Go gRPC registrations
	crossReferenceGRPC(graph)
}

func parseProtoFile(path string, graph *Graph) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	var (
		pkgName        string
		currentService string
		serviceID      string
	)

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Package declaration
		if m := protoPackageRe.FindStringSubmatch(line); m != nil {
			pkgName = m[1]
			continue
		}

		// Service declaration
		if m := protoServiceRe.FindStringSubmatch(line); m != nil {
			currentService = m[1]
			serviceID = fmt.Sprintf("proto.svc.%s", currentService)
			if pkgName != "" {
				serviceID = fmt.Sprintf("proto.svc.%s.%s", pkgName, currentService)
			}

			if _, exists := graph.Nodes[serviceID]; !exists {
				graph.AddNode(&Node{
					ID:      serviceID,
					Kind:    KindGRPC,
					Name:    currentService,
					Package: "proto",
					File:    path,
					Line:    lineNum,
					Meta: map[string]string{
						"source":        "proto",
						"proto_package": pkgName,
					},
				})
			}
			continue
		}

		// RPC declaration
		if m := protoRPCRe.FindStringSubmatch(line); m != nil && currentService != "" {
			rpcName := m[1]
			requestType := m[2]
			responseType := m[4]
			isStream := m[3] != ""

			rpcID := fmt.Sprintf("%s.%s", serviceID, rpcName)
			rpcNode := &Node{
				ID:      rpcID,
				Kind:    KindProtoRPC,
				Name:    rpcName,
				Package: "proto",
				File:    path,
				Line:    lineNum,
				Meta: map[string]string{
					"request_type":  requestType,
					"response_type": responseType,
					"service":       currentService,
				},
			}
			if isStream {
				rpcNode.Meta["streaming"] = "true"
			}

			graph.AddNode(rpcNode)
			graph.AddEdge(serviceID, rpcID, EdgeDepends)

			// Link to request/response message nodes if they exist
			reqMsgID := resolveProtoMsgID(pkgName, requestType)
			respMsgID := resolveProtoMsgID(pkgName, responseType)

			graph.AddEdge(rpcID, reqMsgID, EdgeDepends)
			graph.AddEdge(rpcID, respMsgID, EdgeDepends)
			continue
		}

		// Message declaration
		if m := protoMessageRe.FindStringSubmatch(line); m != nil {
			msgName := m[1]
			msgID := resolveProtoMsgID(pkgName, msgName)

			if _, exists := graph.Nodes[msgID]; !exists {
				graph.AddNode(&Node{
					ID:      msgID,
					Kind:    KindProtoMsg,
					Name:    msgName,
					Package: "proto",
					File:    path,
					Line:    lineNum,
					Meta: map[string]string{
						"source":        "proto",
						"proto_package": pkgName,
					},
				})
			}

			// Reset current service when we hit a top-level message
			// (messages inside services are handled differently in proto3 but rare)
			continue
		}

		// Detect closing brace for service
		if strings.TrimSpace(line) == "}" && currentService != "" {
			currentService = ""
			serviceID = ""
		}
	}
}

func resolveProtoMsgID(pkgName, msgName string) string {
	if pkgName != "" {
		return fmt.Sprintf("proto.msg.%s.%s", pkgName, msgName)
	}
	return fmt.Sprintf("proto.msg.%s", msgName)
}

// crossReferenceGRPC links proto service nodes to existing Go gRPC
// registration nodes found by ExtractGRPC.
func crossReferenceGRPC(graph *Graph) {
	// Build lookup of Go gRPC nodes by service name
	goGRPC := make(map[string]string) // service name → node ID
	for id, node := range graph.Nodes {
		if node.Kind == KindGRPC && node.Meta["source"] != "proto" {
			// Extract service name from the node name (e.g. "⚡ gRPC: UserService")
			name := strings.TrimPrefix(node.Name, "⚡ gRPC: ")
			goGRPC[name] = id
		}
	}

	// Link proto services to their Go implementations
	for id, node := range graph.Nodes {
		if node.Kind == KindGRPC && node.Meta["source"] == "proto" {
			if goID, exists := goGRPC[node.Name]; exists {
				graph.AddEdge(goID, id, EdgeImplements)
			}
		}
	}
}
