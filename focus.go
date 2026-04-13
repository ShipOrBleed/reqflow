package govis

import "strings"

// applyFocus prunes the graph to only include nodes matching the focus
// string and their immediate 1-degree connections.
func applyFocus(g *Graph, focus string) {
	keepNodes := make(map[string]bool)
	lowerFocus := strings.ToLower(focus)

	// Direct matches
	for id, n := range g.Nodes {
		if strings.Contains(strings.ToLower(n.Name), lowerFocus) || strings.Contains(strings.ToLower(id), lowerFocus) {
			keepNodes[id] = true
		}
	}

	// 1 degree of connection
	for _, e := range g.Edges {
		if keepNodes[e.From] {
			keepNodes[e.To] = true
		}
		if keepNodes[e.To] {
			keepNodes[e.From] = true
		}
	}

	// Filter nodes
	for id := range g.Nodes {
		if !keepNodes[id] {
			delete(g.Nodes, id)
			for pkg, ids := range g.Clusters {
				var newIds []string
				for _, cid := range ids {
					if cid != id {
						newIds = append(newIds, cid)
					}
				}
				if len(newIds) == 0 {
					delete(g.Clusters, pkg)
				} else {
					g.Clusters[pkg] = newIds
				}
			}
		}
	}

	// Filter edges
	var newEdges []Edge
	for _, e := range g.Edges {
		if keepNodes[e.From] && keepNodes[e.To] {
			newEdges = append(newEdges, e)
		}
	}
	g.Edges = newEdges
}
