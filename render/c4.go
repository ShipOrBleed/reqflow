package render

import (
	"fmt"
	"io"
	"strings"

	govis "github.com/thzgajendra/govis"
)

// C4Renderer generates C4 Model PlantUML notation.
type C4Renderer struct{}

func (c *C4Renderer) Render(g *govis.Graph, w io.Writer) error {
	fmt.Fprintln(w, "@startuml")
	fmt.Fprintln(w, "!include https://raw.githubusercontent.com/plantuml-stdlib/C4-PlantUML/master/C4_Component.puml")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "top to bottom direction")
	fmt.Fprintln(w, "")

	// Track external infrastructure to draw as System/External
	var infraNodes []string

	// Group nodes by package/cluster
	fmt.Fprintln(w, "System_Boundary(app, \"Go Backend Application\") {")
	for pkg, nodeIDs := range g.Clusters {
		// Clean up package name for display
		pkgName := pkg
		if parts := strings.Split(pkgName, "/"); len(parts) > 0 {
			pkgName = parts[len(parts)-1]
		}
		
		fmt.Fprintf(w, "  Container_Boundary(%s_pkg, \"Package: %s\") {\n", safePUMLID(pkgName), pkg)
		for _, id := range nodeIDs {
			node := g.Nodes[id]
			if node.Kind == govis.KindInfra {
				infraNodes = append(infraNodes, id)
				continue // Draw infra outside boundary
			}
			
			tech := "Go Struct"
			if node.Kind == govis.KindInterface {
				tech = "Go Interface"
			}
			
			format := "Component"
			if node.Kind == govis.KindStore {
				format = "ComponentDb"
			}
			
			desc := string(node.Kind)
			if route, ok := node.Meta["route"]; ok {
				desc = route
			}
			
			fmt.Fprintf(w, "    %s(%s, \"%s\", \"%s\", \"%s\")\n", format, safePUMLID(id), node.Name, tech, desc)
		}
		fmt.Fprintln(w, "  }")
	}
	fmt.Fprintln(w, "}")
	fmt.Fprintln(w, "")
	
	// Draw Infrastructure outside the boundary
	if len(infraNodes) > 0 {
		fmt.Fprintln(w, "System_Ext(external, \"External Infrastructure\") {")
		for _, id := range infraNodes {
			if node, ok := g.Nodes[id]; ok {
				fmt.Fprintf(w, "  SystemDb(%s, \"%s\", \"External component/SDK\")\n", safePUMLID(id), node.Name)
			}
		}
		fmt.Fprintln(w, "}")
		fmt.Fprintln(w, "")
	}

	// Edges
	for _, edge := range g.Edges {
		relTxt := "Uses"
		if edge.Kind == govis.EdgeImplements {
			relTxt = "Implements"
		} else if edge.Kind == govis.EdgeEmbeds {
			relTxt = "Embeds"
		}
		fmt.Fprintf(w, "Rel(%s, %s, \"%s\")\n", safePUMLID(edge.From), safePUMLID(edge.To), relTxt)
	}

	fmt.Fprintln(w, "@enduml")
	return nil
}

func safePUMLID(id string) string {
	id = strings.ReplaceAll(id, ".", "_")
	id = strings.ReplaceAll(id, "/", "_")
	id = strings.ReplaceAll(id, "-", "_")
	return id
}
