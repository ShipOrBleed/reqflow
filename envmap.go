package reqflow

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/packages"
)

// ExtractEnvMap scans AST for environment variable reads via os.Getenv,
// os.LookupEnv, and viper.Get* calls. Creates KindEnvVar nodes and
// EdgeReads edges linking the consuming function/struct to the env var.
func ExtractEnvMap(pkgs []*packages.Package, graph *Graph) {
	seen := make(map[string]string) // env var name → node ID

	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				varName, method := extractEnvCall(call)
				if varName == "" {
					return true
				}

				// Find the enclosing struct/function node
				ownerID := findEnclosingNode(call.Pos(), pkg, graph)

				// Create or reuse env var node
				envID, exists := seen[varName]
				if !exists {
					envID = fmt.Sprintf("env.%s", varName)
					envNode := &Node{
						ID:      envID,
						Kind:    KindEnvVar,
						Name:    varName,
						Package: "environment",
						File:    pkg.Fset.Position(call.Pos()).Filename,
						Line:    pkg.Fset.Position(call.Pos()).Line,
						Meta: map[string]string{
							"env_var_name": varName,
							"access_via":  method,
						},
					}

					// Try to detect default value from common patterns:
					// os.Getenv("X") or fallback with if/or
					if def := extractDefault(call); def != "" {
						envNode.Meta["default_value"] = def
					}

					graph.AddNode(envNode)
					seen[varName] = envID
				}

				// Link the consumer to the env var
				if ownerID != "" {
					graph.AddEdge(ownerID, envID, EdgeReads)
				}

				return true
			})
		}
	}
}

// extractEnvCall checks if a call expression is an env var read and returns
// the variable name and access method.
func extractEnvCall(call *ast.CallExpr) (varName, method string) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", ""
	}

	funcName := sel.Sel.Name

	// Identify the receiver package
	var pkgName string
	if ident, ok := sel.X.(*ast.Ident); ok {
		pkgName = ident.Name
	}

	switch {
	case pkgName == "os" && (funcName == "Getenv" || funcName == "LookupEnv"):
		if len(call.Args) >= 1 {
			if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
				return strings.Trim(lit.Value, `"`), "os." + funcName
			}
		}

	case pkgName == "viper" && strings.HasPrefix(funcName, "Get"):
		// viper.GetString, viper.GetInt, viper.GetBool, viper.Get, etc.
		if len(call.Args) >= 1 {
			if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
				return strings.Trim(lit.Value, `"`), "viper." + funcName
			}
		}
	}

	return "", ""
}

// extractDefault tries to detect a default value pattern, e.g.:
// if val := os.Getenv("X"); val == "" { val = "default" }
// This is a best-effort heuristic.
func extractDefault(call *ast.CallExpr) string {
	// Check if the call is inside an if-init statement — too complex for AST alone.
	// Instead check for a common pattern: the second argument to a helper function
	// like getEnvOrDefault("KEY", "default")
	// For now, return empty — can be enhanced later.
	return ""
}

// findEnclosingNode finds the graph node ID that encloses the given position.
func findEnclosingNode(pos token.Pos, pkg *packages.Package, graph *Graph) string {
	filename := pkg.Fset.Position(pos).Filename
	line := pkg.Fset.Position(pos).Line

	// Find the closest node by file and line proximity
	var bestID string
	bestDist := int(^uint(0) >> 1) // max int

	for _, node := range graph.Nodes {
		if node.Package != pkg.PkgPath {
			continue
		}
		if node.File == filename && node.Line <= line {
			dist := line - node.Line
			if dist < bestDist {
				bestDist = dist
				bestID = node.ID
			}
		}
	}

	return bestID
}
