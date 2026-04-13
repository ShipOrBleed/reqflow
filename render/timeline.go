package render

import (
	"fmt"
	"io"
	"strings"

	reqflow "github.com/thzgajendra/reqflow"
)

// TimelineRenderer generates a Markdown timeline showing architecture evolution
// across git tags/releases.
type TimelineRenderer struct {
	Snapshots []reqflow.EvolutionSnapshot
}

func (t *TimelineRenderer) Render(g *reqflow.Graph, w io.Writer) error {
	if len(t.Snapshots) == 0 {
		fmt.Fprintln(w, "No evolution snapshots available.")
		return nil
	}

	fmt.Fprintln(w, "# Architecture Evolution Timeline")
	fmt.Fprintln(w, "")

	// Summary table
	fmt.Fprintln(w, "| Version | Nodes | Edges | Packages | Added | Removed |")
	fmt.Fprintln(w, "|---------|-------|-------|----------|-------|---------|")
	for _, s := range t.Snapshots {
		fmt.Fprintf(w, "| `%s` | %d | %d | %d | +%d | -%d |\n",
			s.Ref, s.NodeCount, s.EdgeCount, s.Packages, len(s.Added), len(s.Removed))
	}
	fmt.Fprintln(w, "")

	// Kind breakdown per version
	fmt.Fprintln(w, "## Component Breakdown")
	fmt.Fprintln(w, "")

	// Collect all kinds
	allKinds := make(map[reqflow.NodeKind]bool)
	for _, s := range t.Snapshots {
		for k := range s.KindCount {
			allKinds[k] = true
		}
	}

	// Header
	header := "| Kind |"
	sep := "|------|"
	for _, s := range t.Snapshots {
		header += fmt.Sprintf(" `%s` |", s.Ref)
		sep += "------|"
	}
	fmt.Fprintln(w, header)
	fmt.Fprintln(w, sep)

	for kind := range allKinds {
		row := fmt.Sprintf("| %s |", kind)
		for _, s := range t.Snapshots {
			row += fmt.Sprintf(" %d |", s.KindCount[kind])
		}
		fmt.Fprintln(w, row)
	}
	fmt.Fprintln(w, "")

	// Detailed changes per version
	fmt.Fprintln(w, "## Changes Per Version")
	fmt.Fprintln(w, "")

	for i, s := range t.Snapshots {
		if i == 0 {
			continue // Skip first — it's the baseline
		}

		fmt.Fprintf(w, "### %s\n\n", s.Ref)

		if len(s.Added) > 0 {
			fmt.Fprintln(w, "**Added:**")
			for _, id := range s.Added {
				name := id
				if parts := strings.Split(id, "."); len(parts) > 1 {
					name = parts[len(parts)-1]
				}
				fmt.Fprintf(w, "- `%s`\n", name)
			}
			fmt.Fprintln(w, "")
		}

		if len(s.Removed) > 0 {
			fmt.Fprintln(w, "**Removed:**")
			for _, id := range s.Removed {
				name := id
				if parts := strings.Split(id, "."); len(parts) > 1 {
					name = parts[len(parts)-1]
				}
				fmt.Fprintf(w, "- `%s`\n", name)
			}
			fmt.Fprintln(w, "")
		}

		if len(s.Added) == 0 && len(s.Removed) == 0 {
			fmt.Fprintln(w, "No structural changes.")
			fmt.Fprintln(w, "")
		}
	}

	return nil
}
