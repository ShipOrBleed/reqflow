// Package govis provides architecture visualization and static analysis for Go
// codebases. It parses Go ASTs to build dependency graphs, detect architectural
// patterns (handlers, services, stores, models), and render interactive
// visualizations in multiple formats.
//
// Install the CLI:
//
//	go install github.com/thzgajendra/reqflow/cmd/govis@latest
//
// Use as a library:
//
//	graph, err := reqflow.Parse(reqflow.ParseOptions{Dir: "."})
//	renderer := &render.InteractiveRenderer{}
//	renderer.Render(graph, os.Stdout)
package reqflow

// NodeKind classifies a graph node into an architectural layer.
type NodeKind string

const (
	KindStruct    NodeKind = "struct"
	KindInterface NodeKind = "interface"
	KindFunc      NodeKind = "func"
	KindHandler   NodeKind = "handler"    // HTTP handler
	KindStore     NodeKind = "store"      // DB layer / Repository
	KindModel     NodeKind = "model"      // DB entity / Model
	KindService   NodeKind = "service"    // Business logic
	KindEvent     NodeKind = "event"      // Event Bus (Kafka/Rabbit)
	KindMiddleware NodeKind = "middleware" // HTTP Middleware
	KindGRPC      NodeKind = "grpc"       // gRPC service
	KindInfra     NodeKind = "infra"      // External infrastructure
	KindRoute     NodeKind = "route"      // API endpoint
	KindEnvVar    NodeKind = "envvar"     // Environment variable
	KindTable     NodeKind = "table"      // Database table
	KindDep       NodeKind = "dependency" // go.mod transitive dependency
	KindContainer NodeKind = "container"  // Docker/K8s container
	KindProtoRPC  NodeKind = "proto_rpc"  // Proto RPC method
	KindProtoMsg  NodeKind = "proto_msg"  // Proto message type
)

// Node represents a single component in the architecture graph (struct, interface,
// function, handler, service, store, model, etc.). Nodes are uniquely identified
// by their ID (typically "package.TypeName") and carry metadata in the Meta map.
type Node struct {
	ID      string            // "pkg/path.TypeName"
	Kind    NodeKind
	Name    string
	Package string
	Fields  []Field
	Methods []string
	File    string
	Line    int
	Meta    map[string]string // e.g. "route": "GET /users"
}

// Field represents a struct field with its name, type, and struct tag.
type Field struct{ Name, Type, Tag string }

// EdgeKind classifies the relationship between two nodes.
type EdgeKind string

const (
	EdgeEmbeds     EdgeKind = "embeds"
	EdgeImplements EdgeKind = "implements"
	EdgeDepends    EdgeKind = "depends"
	EdgeCalls      EdgeKind = "calls"      // function-to-function call
	EdgeFlows      EdgeKind = "flows"      // request data flow (handler→service→store)
	EdgeReads      EdgeKind = "reads"      // reads env var
	EdgeMapsTo     EdgeKind = "maps_to"    // model→table mapping
	EdgeTransitive EdgeKind = "transitive" // transitive dependency
	EdgePublishes  EdgeKind = "publishes"  // event publish
	EdgeSubscribes EdgeKind = "subscribes" // event subscribe
	EdgeRPC        EdgeKind = "rpc"        // cross-service RPC call
)

// Edge represents a directed relationship between two nodes in the graph.
type Edge struct {
	From, To string
	Kind     EdgeKind
}

// Graph is the core data structure representing the architecture of a Go codebase.
// It contains nodes (components), edges (relationships), and clusters (package groupings).
type Graph struct {
	Nodes    map[string]*Node
	Edges    []Edge
	Clusters map[string][]string // pkg path → []node IDs
	Meta     map[string]string   // graph-level metadata (repo name, commit SHA, etc.)
}

// NewGraph creates an empty, initialized Graph ready for use.
func NewGraph() *Graph {
	return &Graph{
		Nodes:    make(map[string]*Node),
		Clusters: make(map[string][]string),
		Meta:     make(map[string]string),
	}
}

// AddNode adds a node to the graph, initializing its Meta map if nil,
// and registering it in the appropriate package cluster.
func (g *Graph) AddNode(n *Node) {
	if n.Meta == nil {
		n.Meta = make(map[string]string)
	}
	g.Nodes[n.ID] = n
	g.Clusters[n.Package] = append(g.Clusters[n.Package], n.ID)
}

// AddEdge adds a directed edge between two nodes.
func (g *Graph) AddEdge(from, to string, kind EdgeKind) {
	g.Edges = append(g.Edges, Edge{From: from, To: to, Kind: kind})
}
