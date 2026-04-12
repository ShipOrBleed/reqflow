package structmap

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/packages"
)

// extractEvents scans for .Publish(), .Subscribe(), .Produce(), .Consume(), .Emit()
// calls and creates KindEvent nodes with edges to the calling struct.
func extractEvents(pkgs []*packages.Package, graph *Graph) {
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			var currentCallerID string
			ast.Inspect(file, func(n ast.Node) bool {
				if fn, ok := n.(*ast.FuncDecl); ok {
					if fn.Recv != nil && len(fn.Recv.List) > 0 {
						if star, ok := fn.Recv.List[0].Type.(*ast.StarExpr); ok {
							if ident, ok := star.X.(*ast.Ident); ok {
								currentCallerID = fmt.Sprintf("%s.%s", pkg.PkgPath, ident.Name)
							}
						} else if ident, ok := fn.Recv.List[0].Type.(*ast.Ident); ok {
							currentCallerID = fmt.Sprintf("%s.%s", pkg.PkgPath, ident.Name)
						}
					} else {
						currentCallerID = fmt.Sprintf("%s.%s", pkg.PkgPath, fn.Name.Name)
					}
				}

				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}

				method := sel.Sel.Name
				isEvent := method == "Publish" || method == "Produce" || method == "Emit" || method == "Subscribe" || method == "Consume"

				if isEvent && len(call.Args) > 0 {
					var topicStr string
					if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
						topicStr = strings.Trim(lit.Value, "\"")
					}

					if topicStr != "" {
						busID := "eventbus." + topicStr
						if _, exists := graph.Nodes[busID]; !exists {
							graph.AddNode(&Node{
								ID:      busID,
								Kind:    KindEvent,
								Name:    "📢 Topic: " + topicStr,
								Package: "event",
							})
						}

						if currentCallerID != "" {
							if _, exists := graph.Nodes[currentCallerID]; exists {
								if method == "Subscribe" || method == "Consume" {
									graph.AddEdge(busID, currentCallerID, EdgeDepends)
								} else {
									graph.AddEdge(currentCallerID, busID, EdgeDepends)
								}
							}
						}
					}
				}
				return true
			})
		}
	}
}
