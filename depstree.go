package structmap

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// goModule represents a Go module from `go list -m -json all`.
type goModule struct {
	Path     string    `json:"Path"`
	Version  string    `json:"Version"`
	Indirect bool      `json:"Indirect"`
	Main     bool      `json:"Main"`
	Dir      string    `json:"Dir"`
	GoMod    string    `json:"GoMod"`
	Replace  *goModule `json:"Replace"`
}

// ExtractDepTree runs `go list -m -json all` to discover the full transitive
// dependency tree and creates KindDep nodes with EdgeTransitive edges.
func ExtractDepTree(dir string, graph *Graph) {
	workDir := dir
	if workDir == "./..." || workDir == "" {
		workDir = "."
	}

	cmd := exec.Command("go", "list", "-m", "-json", "all")
	cmd.Dir = workDir
	out, err := cmd.Output()
	if err != nil {
		return
	}

	// go list -m -json outputs concatenated JSON objects (no array wrapper)
	var modules []goModule
	decoder := json.NewDecoder(strings.NewReader(string(out)))
	for decoder.More() {
		var m goModule
		if err := decoder.Decode(&m); err != nil {
			break
		}
		modules = append(modules, m)
	}

	if len(modules) == 0 {
		return
	}

	// Find the main module
	var mainModule string
	for _, m := range modules {
		if m.Main {
			mainModule = m.Path
			break
		}
	}

	// Create nodes for all dependencies
	for _, m := range modules {
		if m.Main {
			continue
		}

		path := m.Path
		if m.Replace != nil {
			path = m.Replace.Path
		}

		depID := fmt.Sprintf("dep.%s", strings.ReplaceAll(path, "/", "_"))
		version := m.Version
		if m.Replace != nil && m.Replace.Version != "" {
			version = m.Replace.Version
		}

		// Short name: last path segment
		parts := strings.Split(path, "/")
		shortName := parts[len(parts)-1]
		if version != "" {
			shortName = fmt.Sprintf("%s@%s", shortName, version)
		}

		depNode := &Node{
			ID:      depID,
			Kind:    KindDep,
			Name:    shortName,
			Package: "dependencies",
			Meta: map[string]string{
				"module_path": path,
				"version":     version,
			},
		}

		if m.Indirect {
			depNode.Meta["indirect"] = "true"
		} else {
			depNode.Meta["indirect"] = "false"
		}

		if m.Replace != nil {
			depNode.Meta["replaced_by"] = m.Replace.Path
		}

		graph.AddNode(depNode)

		// Direct deps get an edge from the main module (represented as graph meta)
		if !m.Indirect && mainModule != "" {
			// Connect to existing infra nodes if matching
			for _, existing := range graph.Nodes {
				if existing.Kind == KindInfra && strings.Contains(existing.ID, strings.ReplaceAll(path, "/", "_")) {
					graph.AddEdge(existing.ID, depID, EdgeTransitive)
				}
			}
		}
	}

	// Build edges between dependencies based on their import relationships
	// Direct deps → main module, Indirect deps → some direct dep
	directDeps := make(map[string]string) // module path → node ID
	for _, m := range modules {
		if m.Main || m.Indirect {
			continue
		}
		path := m.Path
		depID := fmt.Sprintf("dep.%s", strings.ReplaceAll(path, "/", "_"))
		directDeps[path] = depID
	}

	// Link indirect deps to their most likely direct parent
	for _, m := range modules {
		if m.Main || !m.Indirect {
			continue
		}
		depID := fmt.Sprintf("dep.%s", strings.ReplaceAll(m.Path, "/", "_"))

		// Heuristic: link to the direct dep with the longest common prefix
		bestMatch := ""
		bestLen := 0
		for directPath, directID := range directDeps {
			prefix := commonPrefix(m.Path, directPath)
			if len(prefix) > bestLen {
				bestLen = len(prefix)
				bestMatch = directID
			}
		}
		if bestMatch != "" && bestLen > 10 {
			graph.AddEdge(bestMatch, depID, EdgeTransitive)
		}
	}

	// Store dep count in graph meta
	graph.Meta["dep_total"] = fmt.Sprintf("%d", len(modules)-1)
	directCount := 0
	for _, m := range modules {
		if !m.Main && !m.Indirect {
			directCount++
		}
	}
	graph.Meta["dep_direct"] = fmt.Sprintf("%d", directCount)
	graph.Meta["dep_indirect"] = fmt.Sprintf("%d", len(modules)-1-directCount)
}

func commonPrefix(a, b string) string {
	i := 0
	for i < len(a) && i < len(b) && a[i] == b[i] {
		i++
	}
	return a[:i]
}
