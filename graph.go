package structmap

type NodeKind string

const (
	KindStruct    NodeKind = "struct"
	KindInterface NodeKind = "interface"
	KindFunc      NodeKind = "func"
	KindHandler   NodeKind = "handler" // HTTP handler
	KindStore     NodeKind = "store"   // DB layer / Repository
	KindModel     NodeKind = "model"   // DB entity / Model
	KindService   NodeKind = "service" // Business logic
)

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

type Field struct{ Name, Type, Tag string }

type EdgeKind string

const (
	EdgeEmbeds     EdgeKind = "embeds"
	EdgeImplements EdgeKind = "implements"
	EdgeDepends    EdgeKind = "depends"
)

type Edge struct {
	From, To string
	Kind     EdgeKind
}

type Graph struct {
	Nodes    map[string]*Node
	Edges    []Edge
	Clusters map[string][]string // pkg path → []node IDs
}

func NewGraph() *Graph {
	return &Graph{
		Nodes:    make(map[string]*Node),
		Clusters: make(map[string][]string),
	}
}

func (g *Graph) AddNode(n *Node) {
	if n.Meta == nil {
		n.Meta = make(map[string]string)
	}
	g.Nodes[n.ID] = n
	g.Clusters[n.Package] = append(g.Clusters[n.Package], n.ID)
}

func (g *Graph) AddEdge(from, to string, kind EdgeKind) {
	g.Edges = append(g.Edges, Edge{From: from, To: to, Kind: kind})
}
