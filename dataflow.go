package reqflow

// DataFlow represents a single request lifecycle path through the architecture.
type DataFlow struct {
	Entry string   // Handler node ID
	Path  []string // Ordered node IDs: handler → service → store
	Route string   // API route if available
}

// ExtractDataFlows traces request lifecycles by walking from KindHandler nodes
// through EdgeDepends/EdgeCalls edges, building linear chains through service
// and store layers. It creates EdgeFlows edges for the direct flow connections.
func ExtractDataFlows(graph *Graph) []DataFlow {
	// Build adjacency list from EdgeDepends and EdgeCalls
	adj := make(map[string][]string)
	for _, edge := range graph.Edges {
		if edge.Kind == EdgeDepends || edge.Kind == EdgeCalls {
			adj[edge.From] = append(adj[edge.From], edge.To)
		}
	}

	var flows []DataFlow

	// Start BFS from each handler node
	for _, node := range graph.Nodes {
		if node.Kind != KindHandler {
			continue
		}

		route := node.Meta["route"]
		if route == "" {
			route = node.Name
		}

		// BFS to find all reachable nodes following the layer order
		visited := map[string]bool{node.ID: true}
		path := []string{node.ID}
		queue := []string{node.ID}

		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]

			for _, next := range adj[current] {
				if visited[next] {
					continue
				}
				nextNode := graph.Nodes[next]
				if nextNode == nil {
					continue
				}
				// Only follow meaningful architectural nodes
				if nextNode.Kind == KindService || nextNode.Kind == KindStore ||
					nextNode.Kind == KindModel || nextNode.Kind == KindEvent ||
					nextNode.Kind == KindGRPC {
					visited[next] = true
					path = append(path, next)
					queue = append(queue, next)
				}
			}
		}

		if len(path) > 1 {
			flows = append(flows, DataFlow{
				Entry: node.ID,
				Path:  path,
				Route: route,
			})

			// Add EdgeFlows between consecutive nodes in the path
			for i := 0; i < len(path)-1; i++ {
				graph.AddEdge(path[i], path[i+1], EdgeFlows)
			}

			// Tag nodes with their position in the flow
			for i, id := range path {
				if n, exists := graph.Nodes[id]; exists {
					n.Meta["dataflow_position"] = string(rune('0' + i))
					n.Meta["dataflow_route"] = route
				}
			}
		}
	}

	return flows
}
