package structmap

import (
	"os/exec"
	"strings"
)

// ExtractPRImpact identifies which graph nodes are affected by changes
// between the current HEAD and a base ref. Tags nodes as direct (file changed)
// or indirect (depends on a changed node).
func ExtractPRImpact(dir, baseRef string, graph *Graph) {
	workDir := dir
	if workDir == "./..." || workDir == "" {
		workDir = "."
	}

	// Get changed files between base and HEAD
	out, err := exec.Command("git", "-C", workDir, "diff", "--name-only", baseRef+"...HEAD").Output()
	if err != nil {
		// Fallback: try without range (uncommitted changes)
		out, err = exec.Command("git", "-C", workDir, "diff", "--name-only", baseRef).Output()
		if err != nil {
			return
		}
	}

	changedFiles := make(map[string]bool)
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		file := strings.TrimSpace(line)
		if file != "" {
			changedFiles[file] = true
		}
	}

	if len(changedFiles) == 0 {
		return
	}

	// Map changed files to direct-impact nodes
	directNodes := make(map[string]bool)
	for _, node := range graph.Nodes {
		if node.File == "" {
			continue
		}
		// Check if node's file matches any changed file (by suffix match)
		for changedFile := range changedFiles {
			if strings.HasSuffix(node.File, changedFile) || strings.HasSuffix(changedFile, node.File) {
				directNodes[node.ID] = true
				node.Meta["pr_impact"] = "direct"
				break
			}
		}
	}

	// Build reverse adjacency for BFS (who depends on this node?)
	reverseAdj := make(map[string][]string)
	for _, edge := range graph.Edges {
		reverseAdj[edge.To] = append(reverseAdj[edge.To], edge.From)
	}

	// BFS outward from direct nodes to find indirect impact (1 hop)
	indirectNodes := make(map[string]bool)
	for directID := range directNodes {
		for _, dependentID := range reverseAdj[directID] {
			if !directNodes[dependentID] && !indirectNodes[dependentID] {
				indirectNodes[dependentID] = true
				if node, exists := graph.Nodes[dependentID]; exists {
					node.Meta["pr_impact"] = "indirect"
				}
			}
		}
	}

	// Also check forward dependencies (what this node depends on that changed)
	forwardAdj := make(map[string][]string)
	for _, edge := range graph.Edges {
		forwardAdj[edge.From] = append(forwardAdj[edge.From], edge.To)
	}

	for directID := range directNodes {
		for _, depID := range forwardAdj[directID] {
			if !directNodes[depID] && !indirectNodes[depID] {
				indirectNodes[depID] = true
				if node, exists := graph.Nodes[depID]; exists {
					if node.Meta["pr_impact"] == "" {
						node.Meta["pr_impact"] = "indirect"
					}
				}
			}
		}
	}
}
