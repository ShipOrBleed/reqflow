package structmap

import (
	"fmt"
	"go/ast"
	"go/types"
	"os"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

// ============================================================
// 1. CIRCULAR DEPENDENCY DETECTION (DFS Cycle Finder)
// ============================================================

type CyclePath []string

// DetectCycles performs DFS-based cycle detection on the graph.
func DetectCycles(g *Graph) []CyclePath {
	adjacency := make(map[string][]string)
	for _, e := range g.Edges {
		adjacency[e.From] = append(adjacency[e.From], e.To)
	}

	visited := make(map[string]bool)
	inStack := make(map[string]bool)
	var cycles []CyclePath

	var dfs func(node string, path []string)
	dfs = func(node string, path []string) {
		visited[node] = true
		inStack[node] = true
		path = append(path, node)

		for _, neighbor := range adjacency[node] {
			if !visited[neighbor] {
				dfs(neighbor, path)
			} else if inStack[neighbor] {
				cycleStart := -1
				for i, p := range path {
					if p == neighbor {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					cycle := make(CyclePath, len(path[cycleStart:]))
					copy(cycle, path[cycleStart:])
					cycle = append(cycle, neighbor)
					cycles = append(cycles, cycle)
				}
			}
		}

		inStack[node] = false
	}

	for id := range g.Nodes {
		if !visited[id] {
			dfs(id, nil)
		}
	}

	return cycles
}

// ============================================================
// 2. COMPLEXITY & COUPLING METRICS (Fan-In / Fan-Out)
// ============================================================

type NodeMetrics struct {
	ID      string
	Name    string
	Kind    NodeKind
	FanIn   int
	FanOut  int
	Methods int
	Package string
}

func ComputeMetrics(g *Graph) []NodeMetrics {
	fanIn := make(map[string]int)
	fanOut := make(map[string]int)

	for _, e := range g.Edges {
		fanOut[e.From]++
		fanIn[e.To]++
	}

	var metrics []NodeMetrics
	for id, n := range g.Nodes {
		metrics = append(metrics, NodeMetrics{
			ID:      id,
			Name:    n.Name,
			Kind:    n.Kind,
			FanIn:   fanIn[id],
			FanOut:  fanOut[id],
			Methods: len(n.Methods),
			Package: n.Package,
		})
	}

	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].FanOut+metrics[i].FanIn > metrics[j].FanOut+metrics[j].FanIn
	})

	return metrics
}

// ============================================================
// 3. MIDDLEWARE CHAIN DETECTION
// ============================================================

func ExtractMiddleware(pkgs []*packages.Package, graph *Graph) {
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

				if sel.Sel.Name != "Use" {
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
// 4. gRPC / PROTOBUF SERVICE DETECTION
// ============================================================

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
								typStr := types.ExprString(field.Type)
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

// ============================================================
// 5. SWALLOWED ERROR DETECTION
// ============================================================

type SwallowedError struct {
	File     string
	Line     int
	FuncName string
	CallExpr string
}

func DetectSwallowedErrors(pkgs []*packages.Package) []SwallowedError {
	var results []SwallowedError

	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			var currentFunc string

			ast.Inspect(file, func(n ast.Node) bool {
				if fn, ok := n.(*ast.FuncDecl); ok {
					currentFunc = fn.Name.Name
				}

				assign, ok := n.(*ast.AssignStmt)
				if !ok {
					return true
				}

				for i, lhs := range assign.Lhs {
					ident, ok := lhs.(*ast.Ident)
					if !ok || ident.Name != "_" {
						continue
					}

					if len(assign.Rhs) > 0 {
						if call, ok := assign.Rhs[0].(*ast.CallExpr); ok {
							callType := pkg.TypesInfo.TypeOf(call)
							if callType != nil {
								typeStr := callType.String()
								if strings.Contains(typeStr, "error") || isErrorTuple(callType, i) {
									callStr := types.ExprString(call.Fun)
									results = append(results, SwallowedError{
										File:     pkg.Fset.Position(assign.Pos()).Filename,
										Line:     pkg.Fset.Position(assign.Pos()).Line,
										FuncName: currentFunc,
										CallExpr: callStr,
									})
								}
							}
						}
					}
				}

				return true
			})
		}
	}

	return results
}

func isErrorTuple(t types.Type, idx int) bool {
	if tuple, ok := t.(*types.Tuple); ok {
		if idx < tuple.Len() {
			return tuple.At(idx).Type().String() == "error"
		}
	}
	return false
}

// ============================================================
// 6. go.mod EXTERNAL INFRASTRUCTURE MAPPING
// ============================================================

func ExtractGoModDeps(dir string, graph *Graph) {
	modPath := dir
	if modPath == "./..." || modPath == "" {
		modPath = "."
	}

	data, err := os.ReadFile(modPath + "/go.mod")
	if err != nil {
		return
	}
	content := string(data)

	knownInfra := map[string]string{
		"github.com/aws/aws-sdk-go":                  "☁️ AWS SDK",
		"cloud.google.com/go":                        "☁️ GCP SDK",
		"github.com/Azure/azure-sdk":                 "☁️ Azure SDK",
		"github.com/go-redis/redis":                  "🗄️ Redis",
		"github.com/redis/go-redis":                  "🗄️ Redis",
		"github.com/segmentio/kafka-go":              "📨 Kafka",
		"github.com/confluentinc/confluent-kafka-go": "📨 Kafka",
		"github.com/streadway/amqp":                  "📨 RabbitMQ",
		"github.com/rabbitmq/amqp091-go":             "📨 RabbitMQ",
		"github.com/stripe/stripe-go":                "💳 Stripe",
		"github.com/elastic/go-elasticsearch":        "🔍 Elasticsearch",
		"go.mongodb.org/mongo-driver":                "🍃 MongoDB",
		"gorm.io/gorm":                               "🗃️ GORM",
		"github.com/jmoiron/sqlx":                    "🗃️ sqlx",
		"google.golang.org/grpc":                     "⚡ gRPC",
		"google.golang.org/protobuf":                 "⚡ Protobuf",
		"github.com/nats-io/nats.go":                 "📨 NATS",
	}

	for dep, label := range knownInfra {
		if strings.Contains(content, dep) {
			infraID := "infra." + strings.ReplaceAll(dep, "/", "_")
			if _, exists := graph.Nodes[infraID]; !exists {
				graph.AddNode(&Node{
					ID:      infraID,
					Kind:    KindInfra,
					Name:    label,
					Package: "infrastructure",
				})
			}
		}
	}
}
