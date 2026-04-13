package govis

import (
	"os"
	"path/filepath"
	"testing"
)

// helperWriteModule creates a temp Go module with the given source files.
// Returns the temp dir path. Caller must defer os.RemoveAll(dir).
func helperWriteModule(t *testing.T, files map[string]string) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "govistest")
	if err != nil {
		t.Fatal(err)
	}

	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module testmod\n\ngo 1.22\n"), 0644)

	for name, content := range files {
		path := filepath.Join(dir, name)
		os.MkdirAll(filepath.Dir(path), 0755)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}

	return dir
}

func helperParse(t *testing.T, dir string, opts ParseOptions) *Graph {
	t.Helper()
	opts.Dir = dir
	graph, err := Parse(opts)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if graph == nil {
		t.Fatal("Expected non-nil graph")
	}
	return graph
}

// ==================== Type Harvesting ====================

func TestParseStructsAndInterfaces(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"models.go": `package testmod

type User struct {
	ID   int
	Name string
}

type UserRepository interface {
	GetByID(id int) User
	Save(u User) error
}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{})

	if _, ok := graph.Nodes["testmod.User"]; !ok {
		t.Error("Expected testmod.User struct node")
	}
	if _, ok := graph.Nodes["testmod.UserRepository"]; !ok {
		t.Error("Expected testmod.UserRepository interface node")
	}

	// UserRepository should be detected as a store (contains "Repository")
	if node := graph.Nodes["testmod.UserRepository"]; node != nil {
		if node.Kind != KindStore {
			t.Errorf("Expected UserRepository kind=store, got %s", node.Kind)
		}
	}
}

func TestParseLayerDetection(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"app.go": `package testmod

type UserHandler struct {}
type UserService struct {}
type UserStore struct {}
type UserModel struct {}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{})

	tests := map[string]NodeKind{
		"testmod.UserHandler": KindHandler,
		"testmod.UserService": KindService,
		"testmod.UserStore":   KindStore,
		"testmod.UserModel":   KindModel,
	}

	for id, expectedKind := range tests {
		node, ok := graph.Nodes[id]
		if !ok {
			t.Errorf("Expected node %s to exist", id)
			continue
		}
		if node.Kind != expectedKind {
			t.Errorf("Node %s: expected kind=%s, got %s", id, expectedKind, node.Kind)
		}
	}
}

func TestParseMockFilesExcluded(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"real.go": `package testmod
type RealService struct {}
`,
		"mock_service.go": `package testmod
type MockService struct {}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{})

	if _, ok := graph.Nodes["testmod.RealService"]; !ok {
		t.Error("Expected RealService to exist")
	}
	if _, ok := graph.Nodes["testmod.MockService"]; ok {
		t.Error("MockService should be excluded (mock file)")
	}
}

// ==================== Interface Resolution ====================

func TestParseInterfaceImplementation(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"types.go": `package testmod

type Greeter interface {
	Greet() string
}

type EnglishGreeter struct {}

func (e *EnglishGreeter) Greet() string {
	return "Hello"
}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{})

	foundImplements := false
	for _, edge := range graph.Edges {
		if edge.From == "testmod.EnglishGreeter" && edge.To == "testmod.Greeter" && edge.Kind == EdgeImplements {
			foundImplements = true
		}
	}
	if !foundImplements {
		t.Error("Expected EnglishGreeter implements Greeter edge")
	}
}

// ==================== Constructor Dependencies ====================

func TestParseConstructorDependencies(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"service.go": `package testmod

type DB struct {}
type Logger struct {}

type AppService struct {
	db  *DB
	log *Logger
}

func NewAppService(db *DB, log *Logger) *AppService {
	return &AppService{db: db, log: log}
}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{})

	constructor, ok := graph.Nodes["testmod.NewAppService"]
	if !ok {
		t.Fatal("Expected NewAppService node")
	}
	if constructor.Meta["is_constructor"] != "true" {
		t.Error("Expected NewAppService to be tagged as constructor")
	}
	if constructor.Meta["deps"] == "" {
		t.Error("Expected NewAppService to have deps metadata")
	}
}

// ==================== Route Extraction ====================

func TestParseEventExtraction(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"events.go": `package testmod

import "fmt"

type OrderService struct {}

func (o *OrderService) Process() {
	o.Publish("order_created")
}

func (o *OrderService) Publish(topic string) {
	fmt.Println(topic)
}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{})

	if _, ok := graph.Nodes["eventbus.order_created"]; !ok {
		t.Error("Expected eventbus.order_created event node")
	}
}

// ==================== Dead Code Detection ====================

func TestDeadCodeDetection(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"app.go": `package testmod

type UsedService struct {}
type OrphanService struct {}
type Consumer struct { svc *UsedService }

func NewConsumer(svc *UsedService) *Consumer {
	return &Consumer{svc: svc}
}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{})

	hasIncoming := make(map[string]bool)
	for _, e := range graph.Edges {
		hasIncoming[e.To] = true
	}

	if hasIncoming["testmod.OrphanService"] {
		t.Error("OrphanService should have zero incoming edges (dead code)")
	}
}

// ==================== Graph Operations ====================

func TestGraphAddNodeAndEdge(t *testing.T) {
	g := NewGraph()

	g.AddNode(&Node{ID: "a", Kind: KindStruct, Name: "A", Package: "pkg"})
	g.AddNode(&Node{ID: "b", Kind: KindStruct, Name: "B", Package: "pkg"})
	g.AddEdge("a", "b", EdgeDepends)

	if len(g.Nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(g.Nodes))
	}
	if len(g.Edges) != 1 {
		t.Errorf("Expected 1 edge, got %d", len(g.Edges))
	}
	if g.Edges[0].Kind != EdgeDepends {
		t.Errorf("Expected EdgeDepends, got %s", g.Edges[0].Kind)
	}
	if len(g.Clusters["pkg"]) != 2 {
		t.Errorf("Expected 2 nodes in cluster 'pkg', got %d", len(g.Clusters["pkg"]))
	}
}

func TestGraphMetaInitialized(t *testing.T) {
	g := NewGraph()
	if g.Meta == nil {
		t.Error("Expected Meta to be initialized")
	}
	g.AddNode(&Node{ID: "x", Name: "X", Package: "p"})
	if g.Nodes["x"].Meta == nil {
		t.Error("Expected node Meta to be initialized by AddNode")
	}
}

// ==================== Cycles Detection ====================

func TestDetectCyclesFound(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "a", Kind: KindService, Name: "A", Package: "p"})
	g.AddNode(&Node{ID: "b", Kind: KindService, Name: "B", Package: "p"})
	g.AddNode(&Node{ID: "c", Kind: KindService, Name: "C", Package: "p"})
	g.AddEdge("a", "b", EdgeDepends)
	g.AddEdge("b", "c", EdgeDepends)
	g.AddEdge("c", "a", EdgeDepends)

	cycles := DetectCycles(g)
	if len(cycles) == 0 {
		t.Error("Expected at least one cycle (a->b->c->a)")
	}
}

func TestDetectCyclesNone(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "a", Kind: KindService, Name: "A", Package: "p"})
	g.AddNode(&Node{ID: "b", Kind: KindService, Name: "B", Package: "p"})
	g.AddEdge("a", "b", EdgeDepends)

	cycles := DetectCycles(g)
	if len(cycles) != 0 {
		t.Errorf("Expected zero cycles, got %d", len(cycles))
	}
}

// ==================== Metrics ====================

func TestComputeMetrics(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "a", Kind: KindHandler, Name: "A", Package: "p"})
	g.AddNode(&Node{ID: "b", Kind: KindService, Name: "B", Package: "p"})
	g.AddNode(&Node{ID: "c", Kind: KindStore, Name: "C", Package: "p"})
	g.AddEdge("a", "b", EdgeDepends)
	g.AddEdge("a", "c", EdgeDepends)
	g.AddEdge("b", "c", EdgeDepends)

	metrics := ComputeMetrics(g)
	if len(metrics) == 0 {
		t.Fatal("Expected metrics")
	}

	// Node "a" should have fan-out=2, fan-in=0
	for _, m := range metrics {
		if m.ID == "a" {
			if m.FanOut != 2 {
				t.Errorf("Node a: expected fan-out=2, got %d", m.FanOut)
			}
			if m.FanIn != 0 {
				t.Errorf("Node a: expected fan-in=0, got %d", m.FanIn)
			}
		}
		// Node "c" should have fan-in=2, fan-out=0
		if m.ID == "c" {
			if m.FanIn != 2 {
				t.Errorf("Node c: expected fan-in=2, got %d", m.FanIn)
			}
			if m.FanOut != 0 {
				t.Errorf("Node c: expected fan-out=0, got %d", m.FanOut)
			}
		}
	}
}

// ==================== Stitch ====================

func TestStitchMergesGraphs(t *testing.T) {
	g1 := NewGraph()
	g1.AddNode(&Node{ID: "svc1.Handler", Kind: KindHandler, Name: "Handler", Package: "svc1"})

	g2 := NewGraph()
	g2.AddNode(&Node{ID: "svc2.Store", Kind: KindStore, Name: "Store", Package: "svc2"})

	merged := Stitch([]*Graph{g1, g2})

	if len(merged.Nodes) != 2 {
		t.Errorf("Expected 2 nodes after stitch, got %d", len(merged.Nodes))
	}
	if _, ok := merged.Nodes["svc1.Handler"]; !ok {
		t.Error("Expected svc1.Handler in merged graph")
	}
	if _, ok := merged.Nodes["svc2.Store"]; !ok {
		t.Error("Expected svc2.Store in merged graph")
	}
}

func TestPrefixNodes(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "Handler", Kind: KindHandler, Name: "Handler", Package: "pkg"})
	g.AddEdge("Handler", "Handler", EdgeDepends)

	g.PrefixNodes("svc1")

	if _, ok := g.Nodes["svc1:Handler"]; !ok {
		t.Error("Expected prefixed node svc1:Handler")
	}
	if g.Edges[0].From != "svc1:Handler" {
		t.Errorf("Expected prefixed edge from, got %s", g.Edges[0].From)
	}
}

// ==================== Data Flow ====================

func TestExtractDataFlows(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "h", Kind: KindHandler, Name: "Handler", Package: "p", Meta: map[string]string{"route": "GET /users"}})
	g.AddNode(&Node{ID: "s", Kind: KindService, Name: "Service", Package: "p"})
	g.AddNode(&Node{ID: "r", Kind: KindStore, Name: "Repo", Package: "p"})
	g.AddEdge("h", "s", EdgeDepends)
	g.AddEdge("s", "r", EdgeDepends)

	flows := ExtractDataFlows(g)
	if len(flows) == 0 {
		t.Fatal("Expected at least one data flow")
	}
	if flows[0].Route != "GET /users" {
		t.Errorf("Expected route 'GET /users', got '%s'", flows[0].Route)
	}
	if len(flows[0].Path) != 3 {
		t.Errorf("Expected path length 3 (handler->service->store), got %d", len(flows[0].Path))
	}
}

func TestExtractDataFlowsNoHandlers(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "s", Kind: KindService, Name: "Service", Package: "p"})

	flows := ExtractDataFlows(g)
	if len(flows) != 0 {
		t.Errorf("Expected zero flows when no handlers, got %d", len(flows))
	}
}

// ==================== Table Map ====================

func TestToSnakeCase(t *testing.T) {
	tests := map[string]string{
		"UserProfile":  "user_profile",
		"ID":           "i_d",
		"HTTPHandler":  "h_t_t_p_handler",
		"simpleCase":   "simple_case",
	}
	for input, expected := range tests {
		got := toSnakeCase(input)
		if got != expected {
			t.Errorf("toSnakeCase(%q) = %q, want %q", input, got, expected)
		}
	}
}

func TestExtractTagValue(t *testing.T) {
	tag := "`gorm:\"column:user_name;type:varchar(100)\" json:\"name,omitempty\" db:\"uname\"`"

	if v := extractTagValue(tag, "gorm", "column"); v != "user_name" {
		t.Errorf("Expected gorm column=user_name, got %q", v)
	}
	if v := extractTagValue(tag, "json", ""); v != "name" {
		t.Errorf("Expected json=name, got %q", v)
	}
	if v := extractTagValue(tag, "db", ""); v != "uname" {
		t.Errorf("Expected db=uname, got %q", v)
	}
	if v := extractTagValue(tag, "missing", ""); v != "" {
		t.Errorf("Expected empty for missing tag, got %q", v)
	}
}

// ==================== Config ====================

func TestLoadConfigMissing(t *testing.T) {
	_, err := LoadConfig("/nonexistent/.govis.yml")
	if err == nil {
		t.Error("Expected error for missing config file")
	}
}

func TestLoadConfigValid(t *testing.T) {
	dir, _ := os.MkdirTemp("", "govisconfig")
	defer os.RemoveAll(dir)

	configContent := `
linter:
  vet_rules:
    - "handler!store"
parser:
  ignore_packages:
    - "vendor"
  domain_naming:
    service_match: ".*Service$"
thresholds:
  max_cycles: 3
`
	os.WriteFile(filepath.Join(dir, ".govis.yml"), []byte(configContent), 0644)

	cfg, err := LoadConfig(filepath.Join(dir, ".govis.yml"))
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if len(cfg.Linter.VetRules) != 1 || cfg.Linter.VetRules[0] != "handler!store" {
		t.Errorf("Expected vet rule 'handler!store', got %v", cfg.Linter.VetRules)
	}
	if cfg.ServiceRegex == nil {
		t.Error("Expected ServiceRegex to be compiled")
	}
}

// ==================== Focus ====================

func TestApplyFocus(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "target", Kind: KindService, Name: "TargetService", Package: "p"})
	g.AddNode(&Node{ID: "neighbor", Kind: KindStore, Name: "Store", Package: "p"})
	g.AddNode(&Node{ID: "distant", Kind: KindHandler, Name: "Handler", Package: "p"})
	g.AddEdge("target", "neighbor", EdgeDepends)

	applyFocus(g, "TargetService")

	if _, ok := g.Nodes["target"]; !ok {
		t.Error("Expected target node to remain")
	}
	if _, ok := g.Nodes["neighbor"]; !ok {
		t.Error("Expected neighbor node to remain (1-degree connection)")
	}
	if _, ok := g.Nodes["distant"]; ok {
		t.Error("Expected distant node to be pruned")
	}
}

// ==================== Trace ====================

func TestTrace(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "h", Kind: KindHandler, Name: "Handler", Package: "pkg",
		Meta: map[string]string{"route": "GET /users", "routes": "GET /users\nPOST /users"}})
	g.AddNode(&Node{ID: "s", Kind: KindService, Name: "UserService", Package: "pkg"})
	g.AddNode(&Node{ID: "r", Kind: KindStore, Name: "UserStore", Package: "pkg"})
	g.AddNode(&Node{ID: "m", Kind: KindModel, Name: "User", Package: "pkg"})
	g.AddEdge("h", "s", EdgeDepends)
	g.AddEdge("s", "r", EdgeDepends)
	g.AddEdge("r", "m", EdgeDepends)

	result := Trace("GET /users", g)
	if result.NotFound {
		t.Fatal("Expected route to be found")
	}
	if result.Route != "GET /users" {
		t.Errorf("Expected route 'GET /users', got %q", result.Route)
	}
	if len(result.Chain) != 4 {
		t.Errorf("Expected chain of 4 nodes, got %d: %v", len(result.Chain), nodeNames(result.Chain))
	}
	if result.Chain[0].Kind != KindHandler {
		t.Errorf("Expected first node to be handler, got %s", result.Chain[0].Kind)
	}
}

func TestTraceNotFound(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "h", Kind: KindHandler, Name: "Handler", Package: "pkg",
		Meta: map[string]string{"route": "GET /users"}})

	result := Trace("DELETE /nonexistent", g)
	if !result.NotFound {
		t.Error("Expected NotFound=true for unmatched route")
	}
}

func TestTracePartialMatch(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "h", Kind: KindHandler, Name: "Handler", Package: "pkg",
		Meta: map[string]string{"route": "POST /api/orders", "routes": "POST /api/orders"}})

	result := Trace("orders", g)
	if result.NotFound {
		t.Error("Expected partial route match to succeed")
	}
}

func nodeNames(nodes []*Node) []string {
	names := make([]string, len(nodes))
	for i, n := range nodes {
		names[i] = n.Name
	}
	return names
}
