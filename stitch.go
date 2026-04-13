package govis

import (
	"fmt"
	"strings"
)

// Stitch merges multiple graphs into one. Useful for microservice architectures
// where each service has its own govis export.
func Stitch(graphs []*Graph) *Graph {
	base := NewGraph()

	for _, g := range graphs {
		// Merge nodes
		for id, node := range g.Nodes {
			if _, exists := base.Nodes[id]; !exists {
				base.AddNode(node)
			} else {
				// Optionally merge metadata if node exists (e.g. from multiple versions)
				for k, v := range node.Meta {
					base.Nodes[id].Meta[k] = v
				}
			}
		}

		// Merge edges
		for _, edge := range g.Edges {
			base.AddEdge(edge.From, edge.To, edge.Kind)
		}

		// Merge clusters
		for pkg, ids := range g.Clusters {
			base.Clusters[pkg] = append(base.Clusters[pkg], ids...)
		}
	}

	return base
}

// PrefixNodes adds a prefix to every node ID in the graph. 
// Useful when stitching services with clashing names.
func (g *Graph) PrefixNodes(prefix string) {
	newNodes := make(map[string]*Node)
	newClusters := make(map[string][]string)
	
	for id, n := range g.Nodes {
		newID := fmt.Sprintf("%s:%s", prefix, id)
		n.ID = newID
		newNodes[newID] = n
	}
	
	for i := range g.Edges {
		g.Edges[i].From = fmt.Sprintf("%s:%s", prefix, g.Edges[i].From)
		g.Edges[i].To = fmt.Sprintf("%s:%s", prefix, g.Edges[i].To)
	}
	
	for pkg, ids := range g.Clusters {
		var newIDs []string
		for _, id := range ids {
			newIDs = append(newIDs, fmt.Sprintf("%s:%s", prefix, id))
		}
		newClusters[pkg] = newIDs
	}
	
	g.Nodes = newNodes
	g.Clusters = newClusters
}

// StitchWithServiceMap merges graphs and detects cross-service edges by matching
// HTTP client URLs to handler routes and gRPC client dials to server registrations.
func StitchWithServiceMap(graphs []*Graph) *Graph {
	base := Stitch(graphs)
	detectCrossServiceEdges(base)
	return base
}

// detectCrossServiceEdges finds cross-service communication patterns:
// 1. HTTP routes in one service matching URL patterns in another
// 2. gRPC server registrations matching client dial targets
// 3. Event topics matching across services
func detectCrossServiceEdges(graph *Graph) {
	// Collect all routes by path
	routeIndex := make(map[string]string) // path → handler node ID
	for id, node := range graph.Nodes {
		if route, ok := node.Meta["route"]; ok {
			parts := strings.SplitN(route, " ", 2)
			if len(parts) == 2 {
				routeIndex[parts[1]] = id
			}
		}
	}

	// Collect gRPC services by name
	grpcIndex := make(map[string]string) // service name → node ID
	for id, node := range graph.Nodes {
		if node.Kind == KindGRPC {
			name := strings.TrimPrefix(node.Name, "⚡ gRPC: ")
			grpcIndex[name] = id
		}
	}

	// Scan all nodes for HTTP client calls or gRPC dials
	for id, node := range graph.Nodes {
		// Check meta for outbound HTTP calls
		if url, ok := node.Meta["http_client_url"]; ok {
			for path, handlerID := range routeIndex {
				if strings.Contains(url, path) {
					graph.AddEdge(id, handlerID, EdgeRPC)
				}
			}
		}

		// Check meta for gRPC client connections
		if target, ok := node.Meta["grpc_dial_target"]; ok {
			for svcName, grpcID := range grpcIndex {
				if strings.Contains(target, strings.ToLower(svcName)) {
					graph.AddEdge(id, grpcID, EdgeRPC)
				}
			}
		}
	}

	// Match event topics across services (publishers → subscribers via same topic)
	topicPublishers := make(map[string][]string)  // topic → publisher node IDs
	topicSubscribers := make(map[string][]string)  // topic → subscriber node IDs

	for _, edge := range graph.Edges {
		if edge.Kind == EdgePublishes {
			toNode := graph.Nodes[edge.To]
			if toNode != nil && toNode.Kind == KindEvent {
				topic := toNode.Meta["topic"]
				if topic == "" {
					topic = toNode.Name
				}
				topicPublishers[topic] = append(topicPublishers[topic], edge.From)
			}
		}
		if edge.Kind == EdgeSubscribes {
			fromNode := graph.Nodes[edge.From]
			if fromNode != nil && fromNode.Kind == KindEvent {
				topic := fromNode.Meta["topic"]
				if topic == "" {
					topic = fromNode.Name
				}
				topicSubscribers[topic] = append(topicSubscribers[topic], edge.To)
			}
		}
	}

	// Create RPC edges between publishers and subscribers of the same topic
	for topic, publishers := range topicPublishers {
		if subscribers, ok := topicSubscribers[topic]; ok {
			for _, pub := range publishers {
				for _, sub := range subscribers {
					if pub != sub {
						graph.AddEdge(pub, sub, EdgeRPC)
					}
				}
			}
		}
	}
}
