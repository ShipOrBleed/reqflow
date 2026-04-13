package structmap

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// ExtractChurn runs git log per node file to count total and recent commits,
// then tags nodes with churn metadata and risk level.
func ExtractChurn(dir string, graph *Graph) {
	workDir := dir
	if workDir == "./..." || workDir == "" {
		workDir = "."
	}

	// Verify git is available
	if _, err := exec.Command("git", "-C", workDir, "rev-parse", "--git-dir").Output(); err != nil {
		return
	}

	// Collect unique files from graph nodes
	fileNodes := make(map[string][]string) // file → []node IDs
	for _, node := range graph.Nodes {
		if node.File != "" {
			fileNodes[node.File] = append(fileNodes[node.File], node.ID)
		}
	}

	for file, nodeIDs := range fileNodes {
		// Total commit count
		total := gitCommitCount(workDir, file, "")
		// Recent commit count (last 6 months)
		recent := gitCommitCount(workDir, file, "6 months ago")

		// Determine risk
		risk := "cold"
		if total > 50 || recent > 20 {
			risk = "hot"
		} else if total > 10 || recent > 5 {
			risk = "warm"
		}

		// Tag all nodes in this file
		for _, id := range nodeIDs {
			if node, exists := graph.Nodes[id]; exists {
				node.Meta["churn_total"] = fmt.Sprintf("%d", total)
				node.Meta["churn_recent"] = fmt.Sprintf("%d", recent)
				node.Meta["churn_risk"] = risk
			}
		}
	}
}

// gitCommitCount returns the number of commits that touched a file.
// If since is non-empty, only counts commits after that date.
func gitCommitCount(workDir, file, since string) int {
	args := []string{"-C", workDir, "log", "--oneline", "--follow", "--", file}
	if since != "" {
		args = []string{"-C", workDir, "log", "--oneline", "--follow", "--since=" + since, "--", file}
	}

	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return 0
	}

	lines := strings.TrimSpace(string(out))
	if lines == "" {
		return 0
	}

	count := 0
	for _, line := range strings.Split(lines, "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}

	return count
}

// GetChurnSummary returns a summary string of churn statistics.
func GetChurnSummary(graph *Graph) string {
	hot, warm, cold := 0, 0, 0
	for _, node := range graph.Nodes {
		switch node.Meta["churn_risk"] {
		case "hot":
			hot++
		case "warm":
			warm++
		case "cold":
			cold++
		}
	}

	total := hot + warm + cold
	if total == 0 {
		return ""
	}

	return fmt.Sprintf("Churn: %d hot, %d warm, %d cold (of %d nodes with files)",
		hot, warm, cold, total)
}

// ParseChurnTotal is a helper to extract churn_total as int from node meta.
func ParseChurnTotal(node *Node) int {
	if v, ok := node.Meta["churn_total"]; ok {
		n, _ := strconv.Atoi(v)
		return n
	}
	return 0
}
