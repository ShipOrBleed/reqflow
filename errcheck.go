package govis

import (
	"go/ast"
	"go/types"
	"os"
	"strings"

	"golang.org/x/tools/go/packages"
)

// ============================================================
// SWALLOWED ERROR DETECTION
// ============================================================

type SwallowedError struct {
	File     string
	Line     int
	FuncName string
	CallExpr string
}

// DetectSwallowedErrors scans AST for `result, _ := someFunc()` patterns
// where `_` discards an error return value.
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
// go.mod EXTERNAL INFRASTRUCTURE MAPPING
// ============================================================

// ExtractGoModDeps parses go.mod to identify cloud/infra dependencies.
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
