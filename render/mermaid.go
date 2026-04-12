package render

import (
	"fmt"
	"io"
	"strings"

	"github.com/zopdev/govis"
)

// Renderer defines standard graph output capability
type Renderer interface {
	Render(g *structmap.Graph, w io.Writer) error
}

type MermaidRenderer struct{}

func (m *MermaidRenderer) Render(g *structmap.Graph, w io.Writer) error {
	fmt.Fprintln(w, "classDiagram")

	// Render nodes inside clusters (packages)
	for pkg, nodeIDs := range g.Clusters {
		pkgName := strings.ReplaceAll(pkg, "/", "_")
		pkgName = strings.ReplaceAll(pkgName, ".", "_")
		pkgName = strings.ReplaceAll(pkgName, "-", "_")

		fmt.Fprintf(w, "  namespace %s {\n", pkgName)
		for _, id := range nodeIDs {
			node := g.Nodes[id]
			m.renderNode(w, node)
		}
		fmt.Fprintln(w, "  }")
	}

	// Render edges
	for _, edge := range g.Edges {
		fromID := sanitizeID(edge.From)
		toID := sanitizeID(edge.To)
		
		switch edge.Kind {
		case structmap.EdgeImplements:
			fmt.Fprintf(w, "  %s ..|> %s : implements\n", fromID, toID)
		case structmap.EdgeDepends:
			fmt.Fprintf(w, "  %s --> %s : depends\n", fromID, toID)
		case structmap.EdgeEmbeds:
			fmt.Fprintf(w, "  %s --|> %s : embeds\n", fromID, toID)
		}
	}

	return nil
}

func (m *MermaidRenderer) renderNode(w io.Writer, n *structmap.Node) {
	nodeID := sanitizeID(n.ID)
	
	switch n.Kind {
	case structmap.KindStruct:
		fmt.Fprintf(w, "    class %s {\n", nodeID)
		for _, f := range n.Fields {
			fmt.Fprintf(w, "      +%s %s\n", f.Name, sanitizeTypeName(f.Type))
		}
		for _, m := range n.Methods {
			fmt.Fprintf(w, "      +%s()\n", m)
		}
		fmt.Fprintln(w, "    }")
	case structmap.KindInterface:
		fmt.Fprintf(w, "    class %s {\n", nodeID)
		fmt.Fprintln(w, "      <<interface>>")
		for _, m := range n.Methods {
			fmt.Fprintf(w, "      +%s()\n", m)
		}
		fmt.Fprintln(w, "    }")
	case structmap.KindHandler:
		fmt.Fprintf(w, "    class %s {\n", nodeID)
		fmt.Fprintln(w, "      <<handler>>")
		for _, m := range n.Methods {
			fmt.Fprintf(w, "      +%s()\n", m)
		}
		fmt.Fprintln(w, "    }")
	case structmap.KindFunc:
		fmt.Fprintf(w, "    class %s {\n", nodeID)
		fmt.Fprintln(w, "      <<function>>")
		fmt.Fprintln(w, "    }")
	}
}

func sanitizeID(id string) string {
	s := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, id)
	return s
}

func sanitizeTypeName(t string) string {
	return strings.ReplaceAll(strings.ReplaceAll(t, "*", ""), "[]", "")
}
