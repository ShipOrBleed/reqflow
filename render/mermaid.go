package render

import (
	"fmt"
	"io"
	"strings"

	reqflow "github.com/thzgajendra/reqflow"
)

// Renderer defines standard graph output capability
type Renderer interface {
	Render(g *reqflow.Graph, w io.Writer) error
}

type MermaidRenderer struct{}

func (m *MermaidRenderer) Render(g *reqflow.Graph, w io.Writer) error {
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
		case reqflow.EdgeImplements:
			fmt.Fprintf(w, "  %s ..|> %s : implements\n", fromID, toID)
		case reqflow.EdgeDepends:
			fmt.Fprintf(w, "  %s --> %s : depends\n", fromID, toID)
		case reqflow.EdgeEmbeds:
			fmt.Fprintf(w, "  %s --|> %s : embeds\n", fromID, toID)
		case reqflow.EdgeCalls:
			fmt.Fprintf(w, "  %s -..-> %s : calls\n", fromID, toID)
		case reqflow.EdgeFlows:
			fmt.Fprintf(w, "  %s ===> %s : flows\n", fromID, toID)
		case reqflow.EdgeReads:
			fmt.Fprintf(w, "  %s -.-> %s : reads\n", fromID, toID)
		case reqflow.EdgeMapsTo:
			fmt.Fprintf(w, "  %s --o %s : maps_to\n", fromID, toID)
		case reqflow.EdgePublishes:
			fmt.Fprintf(w, "  %s -..-> %s : publishes\n", fromID, toID)
		case reqflow.EdgeSubscribes:
			fmt.Fprintf(w, "  %s <-..- %s : subscribes\n", fromID, toID)
		case reqflow.EdgeRPC:
			fmt.Fprintf(w, "  %s <==> %s : rpc\n", fromID, toID)
		case reqflow.EdgeTransitive:
			fmt.Fprintf(w, "  %s -.-> %s : transitive\n", fromID, toID)
		default:
			fmt.Fprintf(w, "  %s --> %s\n", fromID, toID)
		}
	}

	// Apply styling colors!
	fmt.Fprintln(w, "\n  %% Color Coding Layers")
	fmt.Fprintln(w, "  classDef handler fill:#d4edda,stroke:#28a745,color:#155724")
	fmt.Fprintln(w, "  classDef service fill:#cce5ff,stroke:#007bff,color:#004085")
	fmt.Fprintln(w, "  classDef store fill:#ffeeba,stroke:#ffc107,color:#856404")
	fmt.Fprintln(w, "  classDef model fill:#f8d7da,stroke:#dc3545,color:#721c24")
	fmt.Fprintln(w, "  classDef event fill:#e2e3e5,stroke:#343a40,stroke-dasharray: 5 5,color:#343a40")
	fmt.Fprintln(w, "  classDef middleware fill:#fff3cd,stroke:#856404,stroke-dasharray: 3 3,color:#856404")
	fmt.Fprintln(w, "  classDef grpc fill:#d1ecf1,stroke:#0c5460,color:#0c5460")
	fmt.Fprintln(w, "  classDef infra fill:#e8daef,stroke:#6c3483,color:#6c3483")
	fmt.Fprintln(w, "  classDef diffnew fill:#d4edda,stroke:#28a745,color:#155724,stroke-width:4px,stroke-dasharray: 5 5")
	fmt.Fprintln(w, "  classDef diffdel fill:#f8d7da,stroke:#dc3545,color:#721c24,stroke-width:4px,stroke-dasharray: 5 5")
	fmt.Fprintln(w, "  classDef coverCritical fill:#f8d7da,stroke:#dc3545,color:#721c24,stroke-width:3px")
	fmt.Fprintln(w, "  classDef coverLow fill:#fff3cd,stroke:#ffc107,color:#856404,stroke-width:3px")
	fmt.Fprintln(w, "  classDef coverHealthy fill:#d4edda,stroke:#28a745,color:#155724,stroke-width:3px")
	fmt.Fprintln(w, "  classDef churnHot fill:#f8d7da,stroke:#dc3545,color:#721c24,stroke-width:4px")
	fmt.Fprintln(w, "  classDef churnWarm fill:#fff3cd,stroke:#ffc107,color:#856404,stroke-width:3px")
	fmt.Fprintln(w, "  classDef churnCold fill:#cce5ff,stroke:#007bff,color:#004085,stroke-width:2px")
	fmt.Fprintln(w, "  classDef impactDirect fill:#f8d7da,stroke:#dc3545,color:#721c24,stroke-width:4px,stroke-dasharray: 8 4")
	fmt.Fprintln(w, "  classDef impactIndirect fill:#fff3cd,stroke:#ffc107,color:#856404,stroke-width:3px,stroke-dasharray: 4 4")
	
	for _, node := range g.Nodes {
		if node.Meta["diff"] == "new" {
			fmt.Fprintf(w, "  class %s diffnew\n", sanitizeID(node.ID))
			continue
		} else if node.Meta["diff"] == "deleted" {
			fmt.Fprintf(w, "  class %s diffdel\n", sanitizeID(node.ID))
			continue
		}

		switch node.Kind {
		case reqflow.KindHandler:
			fmt.Fprintf(w, "  class %s handler\n", sanitizeID(node.ID))
		case reqflow.KindService:
			fmt.Fprintf(w, "  class %s service\n", sanitizeID(node.ID))
		case reqflow.KindStore:
			fmt.Fprintf(w, "  class %s store\n", sanitizeID(node.ID))
		case reqflow.KindModel:
			fmt.Fprintf(w, "  class %s model\n", sanitizeID(node.ID))
		case reqflow.KindEvent:
			fmt.Fprintf(w, "  class %s event\n", sanitizeID(node.ID))
		case reqflow.KindMiddleware:
			fmt.Fprintf(w, "  class %s middleware\n", sanitizeID(node.ID))
		case reqflow.KindGRPC:
			fmt.Fprintf(w, "  class %s grpc\n", sanitizeID(node.ID))
		case reqflow.KindInfra:
			fmt.Fprintf(w, "  class %s infra\n", sanitizeID(node.ID))
		}
		
		// Churn heatmap overlay
		if risk, ok := node.Meta["churn_risk"]; ok {
			switch risk {
			case "hot":
				fmt.Fprintf(w, "  class %s churnHot\n", sanitizeID(node.ID))
			case "warm":
				fmt.Fprintf(w, "  class %s churnWarm\n", sanitizeID(node.ID))
			case "cold":
				fmt.Fprintf(w, "  class %s churnCold\n", sanitizeID(node.ID))
			}
		}

		// PR impact overlay
		if impact, ok := node.Meta["pr_impact"]; ok {
			switch impact {
			case "direct":
				fmt.Fprintf(w, "  class %s impactDirect\n", sanitizeID(node.ID))
			case "indirect":
				fmt.Fprintf(w, "  class %s impactIndirect\n", sanitizeID(node.ID))
			}
		}

		// Coverage heatmap overlay (overrides kind color when coverage data exists)
		if risk, ok := node.Meta["coverage_risk"]; ok {
			switch risk {
			case "critical":
				fmt.Fprintf(w, "  class %s coverCritical\n", sanitizeID(node.ID))
			case "low":
				fmt.Fprintf(w, "  class %s coverLow\n", sanitizeID(node.ID))
			case "healthy":
				fmt.Fprintf(w, "  class %s coverHealthy\n", sanitizeID(node.ID))
			}
		}

		// 🔗 IDE Deep Links (Click-To-Code)
		if node.File != "" && node.Line > 0 {
			// Generate direct vscode file link
			link := fmt.Sprintf("vscode://file%s:%d", node.File, node.Line)
			fmt.Fprintf(w, "  click %s href \"%s\" \"Open in VSCode\"\n", sanitizeID(node.ID), link)
		}
	}

	return nil
}

func (m *MermaidRenderer) renderNode(w io.Writer, n *reqflow.Node) {
	nodeID := sanitizeID(n.ID)
	
	fmt.Fprintf(w, "    class %s {\n", nodeID)

	switch n.Kind {
	case reqflow.KindInterface:
		fmt.Fprintln(w, "      <<interface>>")
	case reqflow.KindHandler:
		fmt.Fprintln(w, "      <<handler>>")
	case reqflow.KindService:
		fmt.Fprintln(w, "      <<service>>")
	case reqflow.KindStore:
		fmt.Fprintln(w, "      <<store>>")
	case reqflow.KindModel:
		fmt.Fprintln(w, "      <<model>>")
	case reqflow.KindFunc:
		fmt.Fprintln(w, "      <<function>>")
	}

	if route, ok := n.Meta["route"]; ok {
		fmt.Fprintf(w, "      +Route() %s\n", route)
	}
	
	if ks, ok := n.Meta["vitess_keyspace"]; ok {
		shardStatus := "Unsharded"
		if n.Meta["vitess_sharded"] == "true" {
			shardStatus = "Sharded"
		}
		fmt.Fprintf(w, "      +Vitess() Keyspace: %s (%s)\n", ks, shardStatus)
		if vidx, vok := n.Meta["vitess_vindex"]; vok {
			fmt.Fprintf(w, "      +Vindex() %s\n", vidx)
		}
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
