package govis

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/packages"
)

// ExtractAPIMap scans handler function bodies for request/response type bindings
// (c.Bind, c.ShouldBind, json.Decode for requests; c.JSON, json.Encode for responses)
// and creates KindRoute nodes linking handlers to their request/response models.
func ExtractAPIMap(pkgs []*packages.Package, graph *Graph) {
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				fn, ok := n.(*ast.FuncDecl)
				if !ok {
					return true
				}

				// Only inspect handler functions (methods on handler types or standalone handlers)
				fnID := ""
				if fn.Recv != nil && len(fn.Recv.List) > 0 {
					recvType := fn.Recv.List[0].Type
					if star, ok := recvType.(*ast.StarExpr); ok {
						recvType = star.X
					}
					if ident, ok := recvType.(*ast.Ident); ok {
						parentID := fmt.Sprintf("%s.%s", pkg.PkgPath, ident.Name)
						if node, exists := graph.Nodes[parentID]; exists && node.Kind == KindHandler {
							fnID = parentID
						}
					}
				} else {
					fnID = fmt.Sprintf("%s.%s", pkg.PkgPath, fn.Name.Name)
					if node, exists := graph.Nodes[fnID]; exists && node.Kind != KindHandler {
						fnID = ""
					}
				}

				if fnID == "" || fn.Body == nil {
					return true
				}

				handlerNode := graph.Nodes[fnID]
				route := handlerNode.Meta["route"]
				if route == "" {
					route = fn.Name.Name
				}

				var reqTypes, respTypes []string

				ast.Inspect(fn.Body, func(inner ast.Node) bool {
					call, ok := inner.(*ast.CallExpr)
					if !ok {
						return true
					}
					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok {
						return true
					}

					method := sel.Sel.Name

					// Request binding: c.Bind, c.ShouldBind, c.ShouldBindJSON, c.BindJSON, json.NewDecoder...Decode
					if isBindMethod(method) && len(call.Args) >= 1 {
						if typeName := extractTypeName(call.Args[0], pkg); typeName != "" {
							reqTypes = appendUnique(reqTypes, typeName)
						}
					}

					// Response: c.JSON, c.XML, json.NewEncoder...Encode
					if isResponseMethod(method) {
						for _, arg := range call.Args {
							if typeName := extractTypeName(arg, pkg); typeName != "" {
								respTypes = appendUnique(respTypes, typeName)
							}
						}
					}

					return true
				})

				// Create a KindRoute node for this endpoint
				routeID := fmt.Sprintf("%s#%s", fnID, fn.Name.Name)
				routeNode := &Node{
					ID:      routeID,
					Kind:    KindRoute,
					Name:    route,
					Package: pkg.PkgPath,
					File:    pkg.Fset.Position(fn.Pos()).Filename,
					Line:    pkg.Fset.Position(fn.Pos()).Line,
					Meta:    make(map[string]string),
				}

				if len(reqTypes) > 0 {
					routeNode.Meta["request_types"] = strings.Join(reqTypes, ",")
				}
				if len(respTypes) > 0 {
					routeNode.Meta["response_types"] = strings.Join(respTypes, ",")
				}
				if handlerNode.Meta["route"] != "" {
					routeNode.Meta["route"] = handlerNode.Meta["route"]
				}

				// Only add if we found something meaningful
				if len(reqTypes) > 0 || len(respTypes) > 0 || handlerNode.Meta["route"] != "" {
					graph.AddNode(routeNode)
					graph.AddEdge(routeID, fnID, EdgeDepends)

					// Link to request/response model nodes if they exist in the graph
					for _, rt := range reqTypes {
						if _, exists := graph.Nodes[rt]; exists {
							graph.AddEdge(routeID, rt, EdgeDepends)
						}
					}
					for _, rt := range respTypes {
						if _, exists := graph.Nodes[rt]; exists {
							graph.AddEdge(routeID, rt, EdgeDepends)
						}
					}
				}

				return true
			})
		}
	}
}

func isBindMethod(method string) bool {
	binds := []string{
		"Bind", "BindJSON", "BindXML", "BindQuery", "BindUri",
		"ShouldBind", "ShouldBindJSON", "ShouldBindXML", "ShouldBindQuery", "ShouldBindUri",
		"Decode", // json.Decoder.Decode
	}
	for _, b := range binds {
		if method == b {
			return true
		}
	}
	return false
}

func isResponseMethod(method string) bool {
	resps := []string{
		"JSON", "XML", "String", "Data", "HTML",
		"Encode", // json.Encoder.Encode
	}
	for _, r := range resps {
		if method == r {
			return true
		}
	}
	return false
}

func extractTypeName(expr ast.Expr, pkg *packages.Package) string {
	// Handle &SomeType{} — unary expression
	if unary, ok := expr.(*ast.UnaryExpr); ok && unary.Op == token.AND {
		return extractTypeName(unary.X, pkg)
	}
	// Handle SomeType{} — composite literal
	if comp, ok := expr.(*ast.CompositeLit); ok {
		if typObj := pkg.TypesInfo.TypeOf(comp.Type); typObj != nil {
			return strings.TrimLeft(typObj.String(), "*")
		}
		if ident, ok := comp.Type.(*ast.Ident); ok {
			return fmt.Sprintf("%s.%s", pkg.PkgPath, ident.Name)
		}
		if sel, ok := comp.Type.(*ast.SelectorExpr); ok {
			return sel.Sel.Name
		}
	}
	// Handle bare identifier
	if ident, ok := expr.(*ast.Ident); ok {
		if typObj := pkg.TypesInfo.TypeOf(ident); typObj != nil {
			return strings.TrimLeft(typObj.String(), "*")
		}
	}
	return ""
}

func appendUnique(slice []string, val string) []string {
	for _, s := range slice {
		if s == val {
			return slice
		}
	}
	return append(slice, val)
}
