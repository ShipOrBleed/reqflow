package render

import (
	"fmt"
	"io"
	"strings"

	reqflow "github.com/thzgajendra/reqflow"
)

type DOTRenderer struct{}

func (d *DOTRenderer) Render(g *reqflow.Graph, w io.Writer) error {
	fmt.Fprintln(w, "digraph G {")
	fmt.Fprintln(w, "  rankdir=LR")
	fmt.Fprintln(w, "  node [shape=record]")

	// Print clusters
	clusterIdx := 0
	for pkg, nodeIDs := range g.Clusters {
		fmt.Fprintf(w, "  subgraph cluster_%d {\n", clusterIdx)
		fmt.Fprintf(w, "    label=\"%s\"\n", pkg)

		for _, id := range nodeIDs {
			node := g.Nodes[id]
			d.renderNode(w, node)
		}
		fmt.Fprintln(w, "  }")
		clusterIdx++
	}

	// Print edges
	for _, edge := range g.Edges {
		fromID := sanitizeID(edge.From)
		toID := sanitizeID(edge.To)

		label := ""
		style := ""
		switch edge.Kind {
		case reqflow.EdgeImplements:
			label = "implements"
			style = "dashed"
		case reqflow.EdgeDepends:
			label = "depends"
			style = "solid"
		case reqflow.EdgeEmbeds:
			label = "embeds"
			style = "solid"
		}

		fmt.Fprintf(w, "  %s -> %s [label=\"%s\", style=\"%s\"]\n", fromID, toID, label, style)
	}

	fmt.Fprintln(w, "}")
	return nil
}

func (d *DOTRenderer) renderNode(w io.Writer, n *reqflow.Node) {
	nodeID := sanitizeID(n.ID)
	
	// Create a record-shaped label
	var methods []string
	for _, m := range n.Methods {
		methods = append(methods, "+"+m)
	}
	
	var fields []string
	for _, f := range n.Fields {
		fields = append(fields, fmt.Sprintf("+%s %s", f.Name, sanitizeTypeName(f.Type)))
	}

	lines := []string{}
	title := n.Name
	switch n.Kind {
	case reqflow.KindInterface:
		title = fmt.Sprintf("«interface»\\n%s", n.Name)
	case reqflow.KindHandler:
		title = fmt.Sprintf("«handler»\\n%s", n.Name)
	case reqflow.KindService:
		title = fmt.Sprintf("«service»\\n%s", n.Name)
	case reqflow.KindStore:
		title = fmt.Sprintf("«store»\\n%s", n.Name)
	case reqflow.KindModel:
		title = fmt.Sprintf("«model»\\n%s", n.Name)
	}

	lines = append(lines, title)

	if len(fields) > 0 {
		lines = append(lines, strings.Join(fields, "\\l")+"\\l")
	} else {
		lines = append(lines, "")
	}

	if len(methods) > 0 {
		lines = append(lines, strings.Join(methods, "\\l")+"\\l")
	} else {
		lines = append(lines, "")
	}

	label := fmt.Sprintf("{%s|%s|%s}", lines[0], lines[1], lines[2])
	
	// Add color filling for node rendering
	colorAttr := `fillcolor="white", style="filled"`
	switch n.Kind {
	case reqflow.KindHandler:
		colorAttr = `fillcolor="#d4edda", style="filled", color="#155724"`
	case reqflow.KindService:
		colorAttr = `fillcolor="#cce5ff", style="filled", color="#004085"`
	case reqflow.KindStore:
		colorAttr = `fillcolor="#ffeeba", style="filled", color="#856404"`
	case reqflow.KindModel:
		colorAttr = `fillcolor="#f8d7da", style="filled", color="#721c24"`
	}

	fmt.Fprintf(w, "    %s [label=\"%s\", %s]\n", nodeID, label, colorAttr)
}
