// Package reqflow statically traces HTTP request paths through Go codebases.
//
// Given a route like "POST /orders", reqflow finds the handler, follows the
// call chain through services and stores, and shows exactly which methods
// are called at each layer — with file names and line numbers.
//
// # Install the CLI
//
//	go install github.com/ShipOrBleed/reqflow/cmd/reqflow@latest
//
// # Trace a request
//
//	graph, err := reqflow.Parse(reqflow.ParseOptions{Dir: "."})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	result := reqflow.Trace("POST /orders", graph)
//	for _, node := range result.Chain {
//	    fmt.Printf("[%s] %s\n", node.Kind, node.Name)
//	}
//
// # List all routes
//
//	routes := reqflow.ListRoutes(graph)
//	for _, r := range routes {
//	    fmt.Printf("%s %s → %s.%s()\n", r.Method, r.Path, r.HandlerName, r.MethodName)
//	}
//
// # Supported frameworks
//
// GoFr, Gin, Echo, Fiber, and net/http handlers are automatically detected.
// Stores are detected by struct field types (*sql.DB, *gorm.DB, *mongo.Client, etc.),
// not by naming conventions.
package reqflow

import (
	"fmt"
	"os"
)

// Version is set at build time via ldflags.
var Version = "dev"

// Warn prints a warning message to stderr. Used by analysis functions
// that encounter non-fatal errors (e.g., git not available, file not found).
func Warn(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "  ⚠ "+format+"\n", args...)
}

// NodeKind classifies a graph node into an architectural layer.
type NodeKind string

const (
	KindStruct    NodeKind = "struct"
	KindInterface NodeKind = "interface"
	KindFunc      NodeKind = "func"
	KindHandler   NodeKind = "handler"    // HTTP handler
	KindStore     NodeKind = "store"      // DB layer / Repository
	KindClient    NodeKind = "client"    // External HTTP/gRPC client
	KindModel     NodeKind = "model"      // DB entity / Model
	KindService   NodeKind = "service"    // Business logic
	KindEvent     NodeKind = "event"      // Event Bus (Kafka/Rabbit)
	KindMiddleware NodeKind = "middleware" // HTTP Middleware
	KindGRPC      NodeKind = "grpc"       // gRPC service
	KindInfra     NodeKind = "infra"      // External infrastructure
	KindRoute     NodeKind = "route"      // API endpoint
	KindEnvVar    NodeKind = "envvar"     // Environment variable
	KindTable     NodeKind = "table"      // Database table
	KindDep       NodeKind = "dependency" // go.mod transitive dependency
	KindContainer NodeKind = "container"  // Docker/K8s container
	KindProtoRPC  NodeKind = "proto_rpc"  // Proto RPC method
	KindProtoMsg  NodeKind = "proto_msg"  // Proto message type
)

// Node represents a single component in the architecture graph (struct, interface,
// function, handler, service, store, model, etc.). Nodes are uniquely identified
// by their ID (typically "package.TypeName") and carry metadata in the Meta map.
type Node struct {
	ID      string            // "pkg/path.TypeName"
	Kind    NodeKind
	Name    string
	Package string
	Fields  []Field
	Methods []string
	File    string
	Line    int
	Meta    map[string]string // e.g. "route": "GET /users"
}

// Field represents a struct field with its name, type, and struct tag.
type Field struct{ Name, Type, Tag string }

// EdgeKind classifies the relationship between two nodes.
type EdgeKind string

const (
	EdgeEmbeds     EdgeKind = "embeds"
	EdgeImplements EdgeKind = "implements"
	EdgeDepends    EdgeKind = "depends"
	EdgeCalls      EdgeKind = "calls"      // function-to-function call
	EdgeFlows      EdgeKind = "flows"      // request data flow (handler→service→store)
	EdgeReads      EdgeKind = "reads"      // reads env var
	EdgeMapsTo     EdgeKind = "maps_to"    // model→table mapping
	EdgeTransitive EdgeKind = "transitive" // transitive dependency
	EdgePublishes  EdgeKind = "publishes"  // event publish
	EdgeSubscribes EdgeKind = "subscribes" // event subscribe
	EdgeRPC        EdgeKind = "rpc"        // cross-service RPC call
)

// Edge represents a directed relationship between two nodes in the graph.
type Edge struct {
	From, To string
	Kind     EdgeKind
}

// Graph is the core data structure representing the architecture of a Go codebase.
// It contains nodes (components), edges (relationships), and clusters (package groupings).
type Graph struct {
	Nodes       map[string]*Node
	Edges       []Edge
	Clusters    map[string][]string // pkg path → []node IDs
	Meta        map[string]string   // graph-level metadata (repo name, commit SHA, etc.)
	MethodCalls MethodCallIndex     // method-level call index for trace precision
}

// NewGraph creates an empty, initialized Graph ready for use.
func NewGraph() *Graph {
	return &Graph{
		Nodes:    make(map[string]*Node),
		Clusters: make(map[string][]string),
		Meta:     make(map[string]string),
	}
}

// AddNode adds a node to the graph, initializing its Meta map if nil,
// and registering it in the appropriate package cluster.
func (g *Graph) AddNode(n *Node) {
	if n.Meta == nil {
		n.Meta = make(map[string]string)
	}
	g.Nodes[n.ID] = n
	g.Clusters[n.Package] = append(g.Clusters[n.Package], n.ID)
}

// AddEdge adds a directed edge between two nodes.
func (g *Graph) AddEdge(from, to string, kind EdgeKind) {
	g.Edges = append(g.Edges, Edge{From: from, To: to, Kind: kind})
}
