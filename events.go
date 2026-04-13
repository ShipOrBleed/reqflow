package structmap

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/packages"
)

// Publish/produce/emit methods indicate event publishing.
var publishMethods = map[string]bool{
	"Publish": true, "Produce": true, "Emit": true,
	"WriteMessages": true, // kafka-go Writer
	"PublishMsg":    true, // NATS
}

// Subscribe/consume methods indicate event consumption.
var subscribeMethods = map[string]bool{
	"Subscribe": true, "Consume": true,
	"ReadMessage":  true, "FetchMessage": true, // kafka-go Reader
	"QueueSubscribe": true, // NATS
	"ConsumeMessage": true, // generic
}

// extractEvents scans for event bus calls (Kafka, RabbitMQ, NATS, generic pub/sub)
// and creates KindEvent nodes with EdgePublishes/EdgeSubscribes edges.
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
				isPub := publishMethods[method]
				isSub := subscribeMethods[method]

				if (isPub || isSub) && len(call.Args) > 0 {
					topicStr := extractTopicName(call, method)

					if topicStr != "" {
						busID := "eventbus." + topicStr
						if _, exists := graph.Nodes[busID]; !exists {
							graph.AddNode(&Node{
								ID:      busID,
								Kind:    KindEvent,
								Name:    topicStr,
								Package: "event",
								Meta: map[string]string{
									"topic": topicStr,
								},
							})
						}

						if currentCallerID != "" {
							if _, exists := graph.Nodes[currentCallerID]; exists {
								if isSub {
									graph.AddEdge(busID, currentCallerID, EdgeSubscribes)
								} else {
									graph.AddEdge(currentCallerID, busID, EdgePublishes)
								}
							}
						}
					}
				}

				// Detect kafka.NewReader/NewWriter with Topic config
				if method == "NewReader" || method == "NewWriter" {
					if topic := extractKafkaConfigTopic(call); topic != "" {
						busID := "eventbus." + topic
						if _, exists := graph.Nodes[busID]; !exists {
							graph.AddNode(&Node{
								ID:      busID,
								Kind:    KindEvent,
								Name:    topic,
								Package: "event",
								Meta: map[string]string{
									"topic":  topic,
									"broker": "kafka",
								},
							})
						}
						if currentCallerID != "" {
							if _, exists := graph.Nodes[currentCallerID]; exists {
								if method == "NewReader" {
									graph.AddEdge(busID, currentCallerID, EdgeSubscribes)
								} else {
									graph.AddEdge(currentCallerID, busID, EdgePublishes)
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

// extractTopicName tries to extract the topic string from a call's arguments.
func extractTopicName(call *ast.CallExpr, method string) string {
	for _, arg := range call.Args {
		if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			return strings.Trim(lit.Value, "\"")
		}
	}
	return ""
}

// extractKafkaConfigTopic extracts the Topic field from kafka.ReaderConfig{} or kafka.WriterConfig{}
func extractKafkaConfigTopic(call *ast.CallExpr) string {
	if len(call.Args) == 0 {
		return ""
	}
	comp, ok := call.Args[0].(*ast.CompositeLit)
	if !ok {
		return ""
	}
	for _, elt := range comp.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if ident, ok := kv.Key.(*ast.Ident); ok && ident.Name == "Topic" {
			if lit, ok := kv.Value.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				return strings.Trim(lit.Value, "\"")
			}
		}
	}
	return ""
}
