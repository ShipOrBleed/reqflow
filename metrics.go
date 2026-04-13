package govis

import "sort"

// ============================================================
// COMPLEXITY & COUPLING METRICS (Fan-In / Fan-Out)
// ============================================================

type NodeMetrics struct {
	ID      string
	Name    string
	Kind    NodeKind
	FanIn   int
	FanOut  int
	Methods int
	Package string
}

// ComputeMetrics calculates fan-in, fan-out, and method counts per node.
func ComputeMetrics(g *Graph) []NodeMetrics {
	fanIn := make(map[string]int)
	fanOut := make(map[string]int)

	for _, e := range g.Edges {
		fanOut[e.From]++
		fanIn[e.To]++
	}

	var metrics []NodeMetrics
	for id, n := range g.Nodes {
		metrics = append(metrics, NodeMetrics{
			ID:      id,
			Name:    n.Name,
			Kind:    n.Kind,
			FanIn:   fanIn[id],
			FanOut:  fanOut[id],
			Methods: len(n.Methods),
			Package: n.Package,
		})
	}

	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].FanOut+metrics[i].FanIn > metrics[j].FanOut+metrics[j].FanIn
	})

	return metrics
}
