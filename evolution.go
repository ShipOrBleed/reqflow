package reqflow

import (
	"fmt"
	"os/exec"
	"strings"
)

// EvolutionSnapshot represents the graph state at a specific git ref.
type EvolutionSnapshot struct {
	Ref       string
	NodeCount int
	EdgeCount int
	Packages  int
	KindCount map[NodeKind]int
	Added     []string // node IDs added since previous snapshot
	Removed   []string // node IDs removed since previous snapshot
}

// ExtractEvolution parses the codebase at each git tag/ref and compares
// the graph structure across versions to build an evolution timeline.
func ExtractEvolution(dir string, refs []string, baseOpts ParseOptions) []EvolutionSnapshot {
	workDir := dir
	if workDir == "./..." || workDir == "" {
		workDir = "."
	}

	if _, err := exec.Command("git", "-C", workDir, "rev-parse", "--git-dir").Output(); err != nil {
		Warn("evolution: git not available in %s (skipping)", workDir)
		return nil
	}

	var snapshots []EvolutionSnapshot
	var prevNodeIDs map[string]bool

	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}

		snapshot := buildSnapshot(workDir, ref, baseOpts, prevNodeIDs)
		if snapshot != nil {
			snapshots = append(snapshots, *snapshot)
			// Update prevNodeIDs for next iteration
			prevNodeIDs = make(map[string]bool)
			for _, id := range snapshot.Added {
				prevNodeIDs[id] = true
			}
			// Carry forward non-removed nodes
			if len(snapshots) > 1 {
				prev := snapshots[len(snapshots)-2]
				for id := range getNodeIDSet(prev) {
					isRemoved := false
					for _, r := range snapshot.Removed {
						if r == id {
							isRemoved = true
							break
						}
					}
					if !isRemoved {
						prevNodeIDs[id] = true
					}
				}
			}
			for _, id := range snapshot.Added {
				prevNodeIDs[id] = true
			}
		}
	}

	return snapshots
}

func buildSnapshot(workDir, ref string, baseOpts ParseOptions, prevNodeIDs map[string]bool) *EvolutionSnapshot {
	// Create a temporary worktree for the ref
	worktreePath := fmt.Sprintf("/tmp/govis-evolution-%s", sanitizeRef(ref))

	// Clean up any previous worktree
	exec.Command("git", "-C", workDir, "worktree", "remove", "--force", worktreePath).Run()

	// Add worktree at the ref
	if out, err := exec.Command("git", "-C", workDir, "worktree", "add", "--detach", worktreePath, ref).CombinedOutput(); err != nil {
		fmt.Printf("Warning: could not checkout %s: %s\n", ref, string(out))
		return nil
	}
	defer exec.Command("git", "-C", workDir, "worktree", "remove", "--force", worktreePath).Run()

	// Parse at that ref with minimal options (no git features to avoid recursion)
	opts := ParseOptions{
		Dir:    worktreePath,
		Config: baseOpts.Config,
	}
	graph, err := Parse(opts)
	if err != nil {
		return nil
	}

	snapshot := &EvolutionSnapshot{
		Ref:       ref,
		NodeCount: len(graph.Nodes),
		EdgeCount: len(graph.Edges),
		Packages:  len(graph.Clusters),
		KindCount: make(map[NodeKind]int),
	}

	currentIDs := make(map[string]bool)
	for _, node := range graph.Nodes {
		snapshot.KindCount[node.Kind]++
		currentIDs[node.ID] = true
	}

	// Compare with previous snapshot
	if prevNodeIDs != nil {
		for id := range currentIDs {
			if !prevNodeIDs[id] {
				snapshot.Added = append(snapshot.Added, id)
			}
		}
		for id := range prevNodeIDs {
			if !currentIDs[id] {
				snapshot.Removed = append(snapshot.Removed, id)
			}
		}
	} else {
		// First snapshot — all nodes are "added"
		for id := range currentIDs {
			snapshot.Added = append(snapshot.Added, id)
		}
	}

	return snapshot
}

func getNodeIDSet(s EvolutionSnapshot) map[string]bool {
	set := make(map[string]bool)
	for _, id := range s.Added {
		set[id] = true
	}
	return set
}

func sanitizeRef(ref string) string {
	ref = strings.ReplaceAll(ref, "/", "_")
	ref = strings.ReplaceAll(ref, ".", "_")
	return ref
}
