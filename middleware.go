package structmap

import (
	"fmt"
	"go/ast"
	"strings"

	"golang.org/x/tools/go/packages"
)

// ============================================================
// MIDDLEWARE CHAIN DETECTION
// ============================================================

// ExtractMiddleware scans for .Use() calls to detect middleware registrations.
func ExtractMiddleware(pkgs []*packages.Package, graph *Graph) {
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "Use" {
					return true
				}

				for _, arg := range call.Args {
					var mwName string
					switch a := arg.(type) {
					case *ast.Ident:
						mwName = a.Name
					case *ast.SelectorExpr:
						if ident, ok := a.X.(*ast.Ident); ok {
							mwName = ident.Name + "." + a.Sel.Name
						}
					case *ast.CallExpr:
						if innerSel, ok := a.Fun.(*ast.SelectorExpr); ok {
							if ident, ok := innerSel.X.(*ast.Ident); ok {
								mwName = ident.Name + "." + innerSel.Sel.Name
							}
						} else if ident, ok := a.Fun.(*ast.Ident); ok {
							mwName = ident.Name
						}
					}

					if mwName != "" {
						mwID := fmt.Sprintf("%s.middleware.%s", pkg.PkgPath, mwName)
						if _, exists := graph.Nodes[mwID]; !exists {
							graph.AddNode(&Node{
								ID:      mwID,
								Kind:    KindMiddleware,
								Name:    "🛡 " + mwName,
								Package: pkg.PkgPath,
								File:    pkg.Fset.Position(call.Pos()).Filename,
								Line:    pkg.Fset.Position(call.Pos()).Line,
							})
						}
						for id, node := range graph.Nodes {
							if node.Package == pkg.PkgPath && node.Kind == KindHandler {
								graph.AddEdge(mwID, id, EdgeDepends)
							}
						}
					}
				}
				return true
			})
		}
	}
}

// ============================================================
// gRPC / PROTOBUF SERVICE DETECTION
// ============================================================

// ExtractGRPC detects gRPC service registrations and unimplemented server embeds.
func ExtractGRPC(pkgs []*packages.Package, graph *Graph) {
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				switch node := n.(type) {
				case *ast.CallExpr:
					if sel, ok := node.Fun.(*ast.SelectorExpr); ok {
						if strings.HasPrefix(sel.Sel.Name, "Register") && strings.HasSuffix(sel.Sel.Name, "Server") {
							svcName := strings.TrimPrefix(sel.Sel.Name, "Register")
							svcName = strings.TrimSuffix(svcName, "Server")
							grpcID := fmt.Sprintf("%s.grpc.%s", pkg.PkgPath, svcName)
							if _, exists := graph.Nodes[grpcID]; !exists {
								graph.AddNode(&Node{
									ID:      grpcID,
									Kind:    KindGRPC,
									Name:    "⚡ gRPC: " + svcName,
									Package: pkg.PkgPath,
									File:    pkg.Fset.Position(node.Pos()).Filename,
									Line:    pkg.Fset.Position(node.Pos()).Line,
								})
							}
						}
					}
				case *ast.TypeSpec:
					if st, ok := node.Type.(*ast.StructType); ok {
						for _, field := range st.Fields.List {
							if len(field.Names) == 0 {
								typStr := fmt.Sprintf("%s", field.Type)
								if sel, ok := field.Type.(*ast.SelectorExpr); ok {
									if ident, ok := sel.X.(*ast.Ident); ok {
										typStr = ident.Name + "." + sel.Sel.Name
									}
								}
								if strings.Contains(typStr, "Unimplemented") && strings.Contains(typStr, "Server") {
									parentID := fmt.Sprintf("%s.%s", pkg.PkgPath, node.Name.Name)
									if parentNode, exists := graph.Nodes[parentID]; exists {
										parentNode.Kind = KindGRPC
										parentNode.Meta["grpc_embed"] = typStr
									}
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
