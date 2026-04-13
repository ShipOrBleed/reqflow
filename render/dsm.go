package render

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/zopdev/govis"
)

// DSMRenderer generates a Dependency Structure Matrix (DSM) in text format.
type DSMRenderer struct{}

func (d *DSMRenderer) Render(g *structmap.Graph, w io.Writer) error {
	nodes := make([]string, 0, len(g.Nodes))
	for id := range g.Nodes {
		nodes = append(nodes, id)
	}
	// Sort nodes by package to group them in the matrix
	sort.Strings(nodes)

	// Build dependency map
	deps := make(map[string]map[string]bool)
	for _, edge := range g.Edges {
		if deps[edge.From] == nil {
			deps[edge.From] = make(map[string]bool)
		}
		deps[edge.From][edge.To] = true
	}

	fmt.Fprintln(w, "Dependency Structure Matrix (DSM)")
	fmt.Fprintln(w, "Rows depend on Columns. 'X' indicates dependency, 'I' indicates implementation.")
	fmt.Fprintln(w, "")

	// Print header indices
	fmt.Print("      ")
	for i := range nodes {
		fmt.Printf("%3d", i+1)
	}
	fmt.Fprintln(w, "")

	// Print rows
	for i, rowID := range nodes {
		// Print index and short name
		name := rowID
		if parts := strings.Split(name, "."); len(parts) > 0 {
			name = parts[len(parts)-1]
		}
		if len(name) > 15 {
			name = name[:12] + "..."
		}
		fmt.Printf("%3d %-15s", i+1, name)

		for _, colID := range nodes {
			if rowID == colID {
				fmt.Print("  -")
				continue
			}

			found := false
			// Check for dependency
			if deps[rowID] != nil && deps[rowID][colID] {
				// Find edge type
				for _, e := range g.Edges {
					if e.From == rowID && e.To == colID {
						if e.Type == structmap.EdgeImplements {
							fmt.Print("  I")
						} else {
							fmt.Print("  X")
						}
						found = true
						break
					}
				}
			}
			
			if !found {
				fmt.Print("  .")
			}
		}
		fmt.Fprintln(w, "")
	}

	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Legend:")
	for i, id := range nodes {
		fmt.Printf("[%3d] %s\n", i+1, id)
	}

	return nil
}
