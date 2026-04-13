package govis

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/packages"
)

// extractRoutes scans for .GET(), .POST(), .PUT(), .DELETE(), .PATCH() calls
// and tags handler nodes with the matched API route strings.
// A single handler struct may have multiple methods registered to different routes,
// so we store all routes in Meta["routes"] (newline-separated) and Meta["route"] holds the first.
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

				httpMethod := sel.Sel.Name
				if httpMethod != "GET" && httpMethod != "POST" && httpMethod != "PUT" &&
					httpMethod != "DELETE" && httpMethod != "PATCH" {
					return true
				}

				if len(call.Args) >= 2 {
					var pathStr string
					if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
						pathStr = strings.Trim(lit.Value, "\"")
					}

					if pathStr != "" {
						handlerArg := call.Args[len(call.Args)-1]
						tagHandlerWithRoute(handlerArg, pkg, graph, httpMethod, pathStr)
					}
				}
				return true
			})
		}
	}
}

// tagHandlerWithRoute identifies the handler node for a route registration and tags it.
// It handles patterns like:
//   - h.MethodName  (method on a struct — most common in GoFr/Echo/Gin)
//   - handlerFunc   (standalone function)
func tagHandlerWithRoute(expr ast.Expr, pkg *packages.Package, graph *Graph, httpMethod, pathStr string) {
	route := fmt.Sprintf("%s %s", httpMethod, pathStr)

	switch e := expr.(type) {
	case *ast.SelectorExpr:
		// Pattern: h.GetPricing — the receiver type is the handler struct
		if xIdent, ok := e.X.(*ast.Ident); ok {
			if typObj := pkg.TypesInfo.TypeOf(xIdent); typObj != nil {
				cleanType := strings.TrimLeft(typObj.String(), "*")
				if node, exists := graph.Nodes[cleanType]; exists {
					addRoute(node, route)
				}
			}
		}
	case *ast.Ident:
		// Pattern: standalone function — try to match by name
		name := e.Name
		for _, node := range graph.Nodes {
			if node.Kind == KindHandler && node.Name == name {
				addRoute(node, route)
			}
		}
	}
}

// addRoute appends a route to a node's Meta, storing all routes newline-separated.
// Meta["route"] always holds the first (primary) route for backwards compatibility.
func addRoute(node *Node, route string) {
	existing := node.Meta["routes"]
	if existing == "" {
		node.Meta["routes"] = route
		node.Meta["route"] = route // first route = primary
	} else {
		// Avoid duplicates
		for _, r := range strings.Split(existing, "\n") {
			if r == route {
				return
			}
		}
		node.Meta["routes"] = existing + "\n" + route
	}
}
