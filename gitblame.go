package govis

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// ExtractContributors runs git log per node file to identify contributors,
// tagging nodes with primary author, contributor count, and bus-factor risk.
func ExtractContributors(dir string, graph *Graph) {
	workDir := dir
	if workDir == "./..." || workDir == "" {
		workDir = "."
	}

	if _, err := exec.Command("git", "-C", workDir, "rev-parse", "--git-dir").Output(); err != nil {
		return
	}

	fileNodes := make(map[string][]string)
	for _, node := range graph.Nodes {
		if node.File != "" {
			fileNodes[node.File] = append(fileNodes[node.File], node.ID)
		}
	}

	for file, nodeIDs := range fileNodes {
		authors := gitAuthors(workDir, file)
		if len(authors) == 0 {
			continue
		}

		// Sort by commit count descending
		type authorCount struct {
			Name  string
			Count int
		}
		var sorted []authorCount
		for name, count := range authors {
			sorted = append(sorted, authorCount{name, count})
		}
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Count > sorted[j].Count
		})

		primary := sorted[0].Name
		count := len(authors)
		busFactor := "safe"
		if count == 1 {
			busFactor = "risk"
		} else if count == 2 {
			busFactor = "low"
		}

		for _, id := range nodeIDs {
			if node, exists := graph.Nodes[id]; exists {
				node.Meta["contributor_primary"] = primary
				node.Meta["contributor_count"] = fmt.Sprintf("%d", count)
				node.Meta["bus_factor"] = busFactor
			}
		}
	}
}

// gitAuthors returns a map of author name → commit count for a file.
func gitAuthors(workDir, file string) map[string]int {
	out, err := exec.Command("git", "-C", workDir, "log", "--format=%aN", "--follow", "--", file).Output()
	if err != nil {
		return nil
	}

	authors := make(map[string]int)
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		name := strings.TrimSpace(line)
		if name != "" {
			authors[name]++
		}
	}
	return authors
}
