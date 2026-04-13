package govis

// ============================================================
// CIRCULAR DEPENDENCY DETECTION (DFS Cycle Finder)
// ============================================================

type CyclePath []string

// DetectCycles performs DFS-based cycle detection on the graph.
func DetectCycles(g *Graph) []CyclePath {
	adjacency := make(map[string][]string)
	for _, e := range g.Edges {
		adjacency[e.From] = append(adjacency[e.From], e.To)
	}

	visited := make(map[string]bool)
	inStack := make(map[string]bool)
	var cycles []CyclePath

	var dfs func(node string, path []string)
	dfs = func(node string, path []string) {
		visited[node] = true
		inStack[node] = true
		path = append(path, node)

		for _, neighbor := range adjacency[node] {
			if !visited[neighbor] {
				dfs(neighbor, path)
			} else if inStack[neighbor] {
				cycleStart := -1
				for i, p := range path {
					if p == neighbor {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					cycle := make(CyclePath, len(path[cycleStart:]))
					copy(cycle, path[cycleStart:])
					cycle = append(cycle, neighbor)
					cycles = append(cycles, cycle)
				}
			}
		}
		inStack[node] = false
	}

	for id := range g.Nodes {
		if !visited[id] {
			dfs(id, nil)
		}
	}

	return cycles
}
