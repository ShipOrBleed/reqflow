package reqflow

import (
	"os"
	"strings"
	"testing"
)

// ─── Trace: route matching edge cases ─────────────────────────────────────────

func TestTrace_PathWithoutMethod_SingleMatch(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "h", Kind: KindHandler, Name: "H", Package: "p",
		Meta: map[string]string{"routes": "GET /users", "route": "GET /users"}})
	g.AddEdge("h", "h", EdgeDepends)

	r := Trace("/users", g)
	if r.NotFound {
		t.Fatal("Expected /users to match GET /users")
	}
	if r.Route != "GET /users" {
		t.Errorf("Route = %q, want %q", r.Route, "GET /users")
	}
}

func TestTrace_PathWithoutMethod_MultiMatch(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "h", Kind: KindHandler, Name: "H", Package: "p",
		Meta: map[string]string{"routes": "GET /orders\nPOST /orders\nDELETE /orders/{id}"}})

	r := Trace("/orders", g)
	// Should return multi-match for GET and POST (exact path), not DELETE (different path)
	if len(r.MultiMatch) < 2 {
		t.Fatalf("Expected at least 2 matches for /orders, got %d: %v", len(r.MultiMatch), r.MultiMatch)
	}
	// All exact path matches should be GET and POST, not DELETE /orders/{id}
	for _, m := range r.MultiMatch {
		parts := strings.SplitN(m, " ", 2)
		if len(parts) == 2 && parts[1] == "/orders/{id}" {
			t.Errorf("DELETE /orders/{id} should not be in exact matches for /orders")
		}
	}
}

func TestTrace_SubstringPrefersShortest(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "h1", Kind: KindHandler, Name: "H1", Package: "p",
		Meta: map[string]string{"routes": "GET /orgs/{orgID}/budgets/summary", "route": "GET /orgs/{orgID}/budgets/summary"}})
	g.AddNode(&Node{ID: "h2", Kind: KindHandler, Name: "H2", Package: "p",
		Meta: map[string]string{"routes": "GET /orgs/{orgID}/budgets", "route": "GET /orgs/{orgID}/budgets"}})

	r := Trace("GET /orgs/{orgID}/budgets", g)
	if r.NotFound {
		t.Fatal("Route should be found")
	}
	if r.Route != "GET /orgs/{orgID}/budgets" {
		t.Errorf("Should match exact route, got %q", r.Route)
	}
}

func TestTrace_CaseInsensitiveMatch(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "h", Kind: KindHandler, Name: "H", Package: "p",
		Meta: map[string]string{"routes": "GET /Users/{id}", "route": "GET /Users/{id}"}})
	g.AddEdge("h", "h", EdgeDepends)

	r := Trace("get /users/{id}", g)
	if r.NotFound {
		t.Error("Case-insensitive match should work")
	}
}

func TestTrace_WhitespaceInQuery(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "h", Kind: KindHandler, Name: "H", Package: "p",
		Meta: map[string]string{"routes": "GET /health", "route": "GET /health"}})
	g.AddEdge("h", "h", EdgeDepends)

	r := Trace("  GET /health  ", g)
	if r.NotFound {
		t.Error("Whitespace-padded query should still match")
	}
}

func TestTrace_EmptyQuery(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "h", Kind: KindHandler, Name: "H", Package: "p",
		Meta: map[string]string{"routes": "GET /health"}})

	r := Trace("", g)
	if !r.NotFound {
		t.Error("Empty query should return NotFound")
	}
}

func TestTrace_NilGraph(t *testing.T) {
	g := NewGraph()
	r := Trace("GET /anything", g)
	if !r.NotFound {
		t.Error("Empty graph should return NotFound")
	}
}

// ─── Trace: precise chain edge cases ──────────────────────────────────────────

func TestTrace_PreciseChain_HandlerOnly(t *testing.T) {
	// Handler with route_method but no outgoing method calls → should still show handler
	g := NewGraph()
	g.AddNode(&Node{ID: "h", Kind: KindHandler, Name: "H", Package: "p",
		Meta: map[string]string{"routes": "GET /ping", "route": "GET /ping",
			"route_method:GET /ping": "Ping"}})
	g.MethodCalls = make(MethodCallIndex) // empty

	r := Trace("GET /ping", g)
	if r.NotFound {
		t.Fatal("Should find handler")
	}
	// Should fallback to struct-level chain since precise chain has only 1 node
	if len(r.Chain) < 1 {
		t.Error("Chain should have at least the handler")
	}
}

func TestTrace_CalledMethods_PopulatedCorrectly(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "h", Kind: KindHandler, Name: "Handler", Package: "p",
		Fields: []Field{{Name: "svc", Type: "p.Service"}},
		Meta: map[string]string{
			"routes": "GET /items", "route": "GET /items",
			"route_method:GET /items": "ListItems",
		}})
	g.AddNode(&Node{ID: "p.Service", Kind: KindService, Name: "Service", Package: "p",
		Methods: []string{"ListItems", "CreateItem", "DeleteItem"}})
	g.AddEdge("h", "p.Service", EdgeDepends)

	g.MethodCalls = MethodCallIndex{
		"h.ListItems": {
			{FieldName: "svc", TargetMethod: "ListItems"},
		},
	}

	r := Trace("GET /items", g)
	if r.NotFound {
		t.Fatal("Not found")
	}
	// CalledMethods should show only ListItems on Service, not CreateItem/DeleteItem
	called := r.CalledMethods["p.Service"]
	if len(called) != 1 || called[0] != "ListItems" {
		t.Errorf("Expected CalledMethods[Service] = [ListItems], got %v", called)
	}
}

// ─── Trace: interface collapsing ──────────────────────────────────────────────

func TestTrace_InterfaceCollapsed_WhenImplExists(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "h", Kind: KindHandler, Name: "H", Package: "p",
		Fields: []Field{{Name: "svc", Type: "p.Service"}},
		Meta: map[string]string{
			"routes": "GET /x", "route": "GET /x",
			"route_method:GET /x": "Do",
		}})
	g.AddNode(&Node{ID: "p.Service", Kind: KindService, Name: "Service", Package: "p",
		Methods: []string{"Do"}})
	g.AddNode(&Node{ID: "p.svc", Kind: KindService, Name: "svc", Package: "p",
		Methods: []string{"Do"}})
	g.AddEdge("h", "p.Service", EdgeDepends)
	g.AddEdge("p.svc", "p.Service", EdgeImplements)

	g.MethodCalls = MethodCallIndex{
		"h.Do": {{FieldName: "svc", TargetMethod: "Do"}},
	}

	r := Trace("GET /x", g)
	if r.NotFound {
		t.Fatal("Not found")
	}
	// The interface "Service" should be collapsed — only concrete "svc" should appear
	for _, n := range r.Chain {
		if n.ID == "p.Service" {
			t.Error("Interface node should be collapsed when concrete impl is in chain")
		}
	}
}

// ─── Noise filtering ─────────────────────────────────────────────────────────

func TestIsNoiseCall(t *testing.T) {
	cases := []struct {
		field, method string
		want          bool
	}{
		{"Logger", "Errorf", true},
		{"logger", "Info", true},
		{"svc", "GetUser", false},
		{"store", "Insert", false},
		{"mu", "Lock", true},
		{"wg", "Add", true},
		{"Tracer", "Start", true},
		{"svc", "Error", true},    // method name is noise even if field isn't
		{"Logger", "GetUser", true}, // field name is noise even if method isn't
		{"repo", "Save", false},
	}
	for _, tc := range cases {
		got := isNoiseCall(tc.field, tc.method)
		if got != tc.want {
			t.Errorf("isNoiseCall(%q, %q) = %v, want %v", tc.field, tc.method, got, tc.want)
		}
	}
}

// ─── Method call index ────────────────────────────────────────────────────────

func TestBuildMethodCallIndex_EndToEnd(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"handler/handler.go": `package handler

type Service interface {
	GetUser(id string) string
}

type Handler struct {
	svc Service
}

func (h *Handler) GetUser(id string) string {
	return h.svc.GetUser(id)
}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{})

	if graph.MethodCalls == nil {
		t.Fatal("MethodCalls should be populated")
	}

	key := "testmod/handler.Handler.GetUser"
	calls := graph.MethodCalls[key]
	if len(calls) == 0 {
		t.Fatalf("Expected method calls for %s, got none", key)
	}
	if calls[0].FieldName != "svc" || calls[0].TargetMethod != "GetUser" {
		t.Errorf("Expected svc.GetUser call, got %s.%s", calls[0].FieldName, calls[0].TargetMethod)
	}
}

func TestBuildMethodCallIndex_FiltersNoise(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"handler/handler.go": `package handler

type Logger struct{}
func (l *Logger) Errorf(msg string) {}

type Handler struct {
	log *Logger
}

func (h *Handler) Do() {
	h.log.Errorf("something")
}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{})

	key := "testmod/handler.Handler.Do"
	calls := graph.MethodCalls[key]
	// Logger.Errorf should be filtered as noise
	for _, call := range calls {
		if call.FieldName == "log" && call.TargetMethod == "Errorf" {
			t.Error("Logger.Errorf should be filtered as noise")
		}
	}
}

// ─── Parser edge cases ────────────────────────────────────────────────────────

func TestParse_EmptyDirectory(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"empty.go": `package testmod
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{})
	if graph == nil {
		t.Fatal("Graph should not be nil for empty module")
	}
}

func TestParse_NoRoutes_TraceReturnsNotFound(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"service.go": `package testmod

type UserService struct{}
func (s *UserService) Get() {}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{})
	r := Trace("GET /users", graph)
	if !r.NotFound {
		t.Error("Should return NotFound when no routes registered")
	}
}

func TestParse_StructFieldDependency(t *testing.T) {
	// Handler has a field of type Service → should create EdgeDepends
	dir := helperWriteModule(t, map[string]string{
		"handler/handler.go": `package handler

import "net/http"

type Service interface {
	Get() string
}

type Handler struct {
	svc Service
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{})

	handlerNode := graph.Nodes["testmod/handler.Handler"]
	if handlerNode == nil {
		t.Fatal("Expected Handler node")
	}
	if handlerNode.Kind != KindHandler {
		t.Errorf("Expected KindHandler, got %s", handlerNode.Kind)
	}

	// Should have field "svc"
	hasSvcField := false
	for _, f := range handlerNode.Fields {
		if f.Name == "svc" {
			hasSvcField = true
		}
	}
	if !hasSvcField {
		t.Error("Expected 'svc' field on Handler")
	}
}

func TestParse_MultiplePackages(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"handler/handler.go": `package handler

import "net/http"

type Handler struct{}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {}
`,
		"service/service.go": `package service

type UserService struct{}
func (s *UserService) Get() {}
`,
		"store/store.go": `package store

type UserStore struct{}
func (s *UserStore) Select() {}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{})

	if _, ok := graph.Nodes["testmod/handler.Handler"]; !ok {
		t.Error("Expected Handler in handler package")
	}
	if _, ok := graph.Nodes["testmod/service.UserService"]; !ok {
		t.Error("Expected UserService in service package")
	}
	if _, ok := graph.Nodes["testmod/store.UserStore"]; !ok {
		t.Error("Expected UserStore in store package")
	}
}

func TestParse_MethodLineNumbersStored(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"handler.go": `package testmod

import "net/http"

type Handler struct{}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{})

	node := graph.Nodes["testmod.Handler"]
	if node == nil {
		t.Fatal("Expected Handler node")
	}
	// method_line:GetUser should be set
	if node.Meta["method_line:GetUser"] == "" {
		t.Error("Expected method_line:GetUser to be set")
	}
	if node.Meta["method_file:GetUser"] == "" {
		t.Error("Expected method_file:GetUser to be set")
	}
	// Two different methods should have different line numbers
	if node.Meta["method_line:GetUser"] == node.Meta["method_line:CreateUser"] {
		t.Error("GetUser and CreateUser should have different line numbers")
	}
}

// ─── Config edge cases ────────────────────────────────────────────────────────

func TestLoadConfig_InvalidYAML(t *testing.T) {
	dir, _ := os.MkdirTemp("", "reqflowtest")
	defer os.RemoveAll(dir)

	os.WriteFile(dir+"/.reqflow.yml", []byte("invalid: [yaml: broken"), 0644)
	_, err := LoadConfig(dir + "/.reqflow.yml")
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestLoadConfig_EmptyFile(t *testing.T) {
	dir, _ := os.MkdirTemp("", "reqflowtest")
	defer os.RemoveAll(dir)

	os.WriteFile(dir+"/.reqflow.yml", []byte(""), 0644)
	cfg, err := LoadConfig(dir + "/.reqflow.yml")
	if err != nil {
		t.Fatalf("Empty config should not error: %v", err)
	}
	if cfg == nil {
		t.Error("Config should not be nil")
	}
}

// ─── Renderer edge cases ─────────────────────────────────────────────────────

func TestPkgFile_EmptyFile(t *testing.T) {
	// Imported from render package — test the logic
	// pkgFile is in render/trace.go, can't test directly from here
	// but we can test through the full trace rendering
}

func TestTrace_CalledMethods_NilMethodCalls(t *testing.T) {
	// Graph with no MethodCalls should not panic
	g := NewGraph()
	g.AddNode(&Node{ID: "h", Kind: KindHandler, Name: "H", Package: "p",
		Meta: map[string]string{"routes": "GET /test", "route": "GET /test"}})
	g.MethodCalls = nil

	r := Trace("GET /test", g)
	if r.NotFound {
		t.Fatal("Should find handler even without MethodCalls")
	}
}

// ─── findAllMatchingRoutes ─────────────────────────────────────────────────────

func TestFindAllMatchingRoutes_ExactPathOnly(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "h", Kind: KindHandler, Name: "H", Package: "p",
		Meta: map[string]string{"routes": "GET /api/users\nPOST /api/users\nGET /api/users/{id}"}})

	matches := findAllMatchingRoutes("/api/users", g)
	// Should return GET and POST (exact path), not GET /api/users/{id}
	for _, m := range matches {
		if strings.Contains(m, "{id}") {
			t.Errorf("Exact path match should not include /api/users/{id}, got %v", matches)
		}
	}
	if len(matches) != 2 {
		t.Errorf("Expected 2 exact matches, got %d: %v", len(matches), matches)
	}
}

func TestFindAllMatchingRoutes_SubstringFallback(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "h", Kind: KindHandler, Name: "H", Package: "p",
		Meta: map[string]string{"routes": "GET /api/users/{id}\nDELETE /api/users/{id}"}})

	// "users" has no exact path match, falls back to substring
	matches := findAllMatchingRoutes("users", g)
	if len(matches) != 2 {
		t.Errorf("Expected 2 substring matches, got %d: %v", len(matches), matches)
	}
}

func TestFindAllMatchingRoutes_NoMatch(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "h", Kind: KindHandler, Name: "H", Package: "p",
		Meta: map[string]string{"routes": "GET /health"}})

	matches := findAllMatchingRoutes("nonexistent", g)
	if len(matches) != 0 {
		t.Errorf("Expected 0 matches, got %v", matches)
	}
}

// ─── Full end-to-end: handler → service → store with method calls ─────────────

func TestEndToEnd_FullTrace_WithMethodCalls(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"handler/handler.go": `package handler

import "net/http"

type Service interface {
	ListUsers() []string
}

type UserHandler struct {
	svc Service
}

func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	h.svc.ListUsers()
}
`,
		"service/service.go": `package service

type Store interface {
	SelectAll() []string
}

type UserService struct {
	store Store
}

func (s *UserService) ListUsers() []string {
	return s.store.SelectAll()
}
`,
		"store/store.go": `package store

type UserStore struct{}

func (s *UserStore) SelectAll() []string { return nil }
`,
		"router/router.go": `package router

type App struct{}
func (a *App) GET(path string, h interface{}) {}
`,
		"main.go": `package main

import (
	"testmod/handler"
	"testmod/router"
)

func main() {
	app := &router.App{}
	h := &handler.UserHandler{}
	app.GET("/users", h.ListUsers)
}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{})
	r := Trace("GET /users", graph)

	if r.NotFound {
		t.Fatal("Expected /users to be found")
	}
	if r.Route != "GET /users" {
		t.Errorf("Route = %q, want %q", r.Route, "GET /users")
	}
	if r.Chain[0].Kind != KindHandler {
		t.Errorf("Chain[0] should be handler, got %s", r.Chain[0].Kind)
	}

	// Handler should have route_method set
	if r.Handler.Meta["route_method:GET /users"] != "ListUsers" {
		t.Errorf("Expected route_method = ListUsers, got %q", r.Handler.Meta["route_method:GET /users"])
	}

	// Method call index should track h.svc.ListUsers()
	key := r.Handler.ID + ".ListUsers"
	calls := graph.MethodCalls[key]
	if len(calls) == 0 {
		t.Error("Expected method call from Handler.ListUsers to svc.ListUsers")
	} else if calls[0].FieldName != "svc" || calls[0].TargetMethod != "ListUsers" {
		t.Errorf("Expected svc.ListUsers, got %s.%s", calls[0].FieldName, calls[0].TargetMethod)
	}
}

func TestEndToEnd_InlineHandler_NoRouteMethod(t *testing.T) {
	// Inline handlers don't have route_method since there's no h.MethodName
	dir := helperWriteModule(t, map[string]string{
		"router/router.go": `package router

type App struct{}
func (a *App) GET(path string, h interface{}) {}
`,
		"main.go": `package main

import "testmod/router"

func main() {
	app := &router.App{}
	app.GET("/ping", func() string { return "pong" })
}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{})
	r := Trace("GET /ping", graph)

	if r.NotFound {
		t.Fatal("Inline handler should be found")
	}
	// Should not panic even without route_method
	if r.Chain[0].Kind != KindHandler {
		t.Errorf("Expected handler, got %s", r.Chain[0].Kind)
	}
}
