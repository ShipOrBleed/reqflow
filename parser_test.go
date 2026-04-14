package reqflow

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
	_, err := LoadConfig("/nonexistent/.reqflow.yml")
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
	os.WriteFile(filepath.Join(dir, ".reqflow.yml"), []byte(configContent), 0644)

	cfg, err := LoadConfig(filepath.Join(dir, ".reqflow.yml"))
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

// ==================== shouldIgnorePackage ====================

func TestShouldIgnorePackage(t *testing.T) {
	cases := []struct {
		pkgPath  string
		patterns []string
		want     bool
	}{
		{"github.com/acme/app/vendor/lib", []string{"vendor"}, true},
		{"github.com/acme/app/internal/service", []string{"vendor"}, false},
		{"github.com/acme/app/mock_store", []string{"mock", "vendor"}, true},
		{"github.com/acme/app/store", []string{"mock", "vendor"}, false},
		{"anything", []string{}, false},
	}
	for _, tc := range cases {
		got := shouldIgnorePackage(tc.pkgPath, tc.patterns)
		if got != tc.want {
			t.Errorf("shouldIgnorePackage(%q, %v) = %v, want %v", tc.pkgPath, tc.patterns, got, tc.want)
		}
	}
}

func TestShouldIgnorePackage_EmptyPatterns(t *testing.T) {
	if shouldIgnorePackage("github.com/acme/app", nil) {
		t.Error("Expected false for nil patterns")
	}
}

// ==================== DB Client Detection (structural store) ====================

func TestParseDBFieldPromotesToStore(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"store.go": `package testmod

import "database/sql"

type UserRepo struct {
	db *sql.DB
}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{})

	node, ok := graph.Nodes["testmod.UserRepo"]
	if !ok {
		t.Fatal("Expected UserRepo node")
	}
	if node.Kind != KindStore {
		t.Errorf("UserRepo with *sql.DB field should be KindStore, got %s", node.Kind)
	}
}

func TestParseGormDBPromotesToStore(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"go.mod": "module testmod\n\ngo 1.22\n\nrequire gorm.io/gorm v1.23.0\n",
		"store.go": `package testmod

type ProductRepository struct {
	db interface{ Find(dest interface{}, conds ...interface{}) interface{} }
}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{})

	// ProductRepository matches store keyword "Repository"
	node, ok := graph.Nodes["testmod.ProductRepository"]
	if !ok {
		t.Fatal("Expected ProductRepository node")
	}
	if node.Kind != KindStore {
		t.Errorf("ProductRepository should be KindStore, got %s", node.Kind)
	}
}

// ==================== GoFr framework handler detection ====================

func TestParseGoFrHandlerDetected(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"internal/handler/handler.go": `package handler

// Simulate gofr.Context as a local type for testing
type Context struct{}

type UserHandler struct{}

func (h *UserHandler) GetUser(ctx *Context) (interface{}, error) {
	return nil, nil
}
`,
		"router/router.go": `package router

type App struct{}
func (a *App) GET(path string, handler interface{}) {}
`,
		"main.go": `package main

import (
	"testmod/internal/handler"
	"testmod/router"
)

func main() {
	app := &router.App{}
	h := &handler.UserHandler{}
	app.GET("/users", h.GetUser)
}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{})

	r := Trace("/users", graph)
	if r.NotFound {
		t.Fatal("Expected /users to be found")
	}
	if r.Chain[0].Kind != KindHandler {
		t.Errorf("Expected KindHandler, got %s", r.Chain[0].Kind)
	}
}


