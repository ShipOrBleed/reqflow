package structmap

import (
	"fmt"
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
