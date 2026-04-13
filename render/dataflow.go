package render

import (
	"fmt"
	"io"

	structmap "github.com/zopdev/govis"
)

// DataFlowRenderer generates Mermaid sequence diagrams showing request
// data flow from handlers through services to stores.
type DataFlowRenderer struct{}

func (d *DataFlowRenderer) Render(g *structmap.Graph, w io.Writer) error {
	flows := structmap.ExtractDataFlows(g)

	if len(flows) == 0 {
		fmt.Fprintln(w, "No data flows detected (no handler→service→store chains found).")
		return nil
	}

	fmt.Fprintln(w, "sequenceDiagram")
	fmt.Fprintln(w, "  autonumber")
	fmt.Fprintln(w, "")

	// Declare participants in order
	participants := make(map[string]bool)
	for _, flow := range flows {
		for _, id := range flow.Path {
			if !participants[id] {
				participants[id] = true
				node := g.Nodes[id]
				if node != nil {
					label := fmt.Sprintf("%s [%s]", node.Name, node.Kind)
					fmt.Fprintf(w, "  participant %s as %s\n", sanitizeID(id), label)
				}
			}
		}
	}
	fmt.Fprintln(w, "")

	// Draw flow sequences
	for _, flow := range flows {
		fmt.Fprintf(w, "  Note over %s: %s\n", sanitizeID(flow.Entry), flow.Route)

		for i := 0; i < len(flow.Path)-1; i++ {
			from := flow.Path[i]
			to := flow.Path[i+1]

			fromNode := g.Nodes[from]
			toNode := g.Nodes[to]

			label := "→"
			if toNode != nil {
				switch toNode.Kind {
				case structmap.KindService:
					label = "process"
				case structmap.KindStore:
					label = "query"
				case structmap.KindModel:
					label = "map"
				case structmap.KindEvent:
					label = "publish"
				case structmap.KindGRPC:
					label = "call"
				}
			}

			_ = fromNode
			fmt.Fprintf(w, "  %s->>%s: %s\n", sanitizeID(from), sanitizeID(to), label)
		}

		// Add return arrows for stores back through the chain
		if len(flow.Path) >= 2 {
			last := flow.Path[len(flow.Path)-1]
			first := flow.Path[0]
			fmt.Fprintf(w, "  %s-->>%s: response\n", sanitizeID(last), sanitizeID(first))
		}

		fmt.Fprintln(w, "")
	}

	return nil
}
