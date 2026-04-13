package govis

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/packages"
)

// extractRoutes scans for .GET(), .POST(), .PUT(), .DELETE(), .PATCH() calls
// and tags handler nodes with the matched API route string.
func extractRoutes(pkgs []*packages.Package, graph *Graph) {
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}

				method := sel.Sel.Name
				if method != "GET" && method != "POST" && method != "PUT" && method != "DELETE" && method != "PATCH" {
					return true
				}

				if len(call.Args) >= 2 {
					var pathStr string
					if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
						pathStr = strings.Trim(lit.Value, "\"")
					}

					if pathStr != "" {
						handlerArg := call.Args[len(call.Args)-1]
						findAndTagHandler(handlerArg, pkg, graph, method, pathStr)
					}
				}
				return true
			})
		}
	}
}

func findAndTagHandler(expr ast.Expr, pkg *packages.Package, graph *Graph, method, pathStr string) {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		if xIdent, ok := e.X.(*ast.Ident); ok {
			if typObj := pkg.TypesInfo.TypeOf(xIdent); typObj != nil {
				cleanType := strings.TrimLeft(typObj.String(), "*")
				if node, exists := graph.Nodes[cleanType]; exists {
					node.Meta["route"] = fmt.Sprintf("%s %s", method, pathStr)
				}
			}
		}
	}
}
