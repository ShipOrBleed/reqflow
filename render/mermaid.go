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

	// Apply styling colors!
	fmt.Fprintln(w, "\n  %% Color Coding Layers")
	fmt.Fprintln(w, "  classDef handler fill:#d4edda,stroke:#28a745,color:#155724")
	fmt.Fprintln(w, "  classDef service fill:#cce5ff,stroke:#007bff,color:#004085")
	fmt.Fprintln(w, "  classDef store fill:#ffeeba,stroke:#ffc107,color:#856404")
	fmt.Fprintln(w, "  classDef model fill:#f8d7da,stroke:#dc3545,color:#721c24")
	
	for _, node := range g.Nodes {
		switch node.Kind {
		case structmap.KindHandler:
			fmt.Fprintf(w, "  class %s handler\n", sanitizeID(node.ID))
		case structmap.KindService:
			fmt.Fprintf(w, "  class %s service\n", sanitizeID(node.ID))
		case structmap.KindStore:
			fmt.Fprintf(w, "  class %s store\n", sanitizeID(node.ID))
		case structmap.KindModel:
			fmt.Fprintf(w, "  class %s model\n", sanitizeID(node.ID))
		}
	}

	return nil
}

func (m *MermaidRenderer) renderNode(w io.Writer, n *structmap.Node) {
	nodeID := sanitizeID(n.ID)
	
	fmt.Fprintf(w, "    class %s {\n", nodeID)

	switch n.Kind {
	case structmap.KindInterface:
		fmt.Fprintln(w, "      <<interface>>")
	case structmap.KindHandler:
		fmt.Fprintln(w, "      <<handler>>")
	case structmap.KindService:
		fmt.Fprintln(w, "      <<service>>")
	case structmap.KindStore:
		fmt.Fprintln(w, "      <<store>>")
	case structmap.KindModel:
		fmt.Fprintln(w, "      <<model>>")
	case structmap.KindFunc:
		fmt.Fprintln(w, "      <<function>>")
	}

	if route, ok := n.Meta["route"]; ok {
		fmt.Fprintf(w, "      +Route() %s\n", route)
	}

	for _, f := range n.Fields {
		fmt.Fprintf(w, "      +%s %s\n", f.Name, sanitizeTypeName(f.Type))
	}
	for _, m := range n.Methods {
		fmt.Fprintf(w, "      +%s()\n", m)
	}
	fmt.Fprintln(w, "    }")
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
