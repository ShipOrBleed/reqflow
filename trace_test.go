package reqflow

import (
	"os"
	"testing"
)

// ─── Helpers ─────────────────────────────────────────────────────────────────

// traceFixture creates a minimal graph with handler → service → store → model chain.
func traceFixture() *Graph {
	g := NewGraph()
	g.AddNode(&Node{
		ID: "pkg.UserHandler", Kind: KindHandler, Name: "UserHandler", Package: "pkg",
		Meta:    map[string]string{"route": "GET /users/{id}", "routes": "GET /users/{id}\nPOST /users"},
		Methods: []string{"GetUser", "CreateUser"},
	})
	g.AddNode(&Node{
		ID: "pkg.UserService", Kind: KindService, Name: "UserService", Package: "pkg",
		Methods: []string{"FindByID", "Create"},
	})
	g.AddNode(&Node{
		ID: "pkg.UserStore", Kind: KindStore, Name: "UserStore", Package: "pkg",
		Methods: []string{"SelectByID", "Insert"},
	})
	g.AddNode(&Node{
		ID: "pkg.User", Kind: KindModel, Name: "User", Package: "models",
		Fields: []Field{{Name: "ID", Type: "int"}, {Name: "Name", Type: "string"}},
	})
	g.AddEdge("pkg.UserHandler", "pkg.UserService", EdgeDepends)
	g.AddEdge("pkg.UserService", "pkg.UserStore", EdgeDepends)
	g.AddEdge("pkg.UserStore", "pkg.User", EdgeDepends)
	return g
}

// ─── Trace unit tests (graph-level) ──────────────────────────────────────────

func TestTraceExactMatch(t *testing.T) {
	g := traceFixture()
	r := Trace("GET /users/{id}", g)

	if r.NotFound {
		t.Fatal("Expected route to be found")
	}
	if r.Route != "GET /users/{id}" {
		t.Errorf("Route = %q, want %q", r.Route, "GET /users/{id}")
	}
	if len(r.Chain) != 4 {
		t.Fatalf("Chain length = %d, want 4 (handler+service+store+model), got: %v", len(r.Chain), chainNames(r.Chain))
	}
	if r.Chain[0].Kind != KindHandler {
		t.Errorf("Chain[0].Kind = %s, want handler", r.Chain[0].Kind)
	}
	if r.Chain[1].Kind != KindService {
		t.Errorf("Chain[1].Kind = %s, want service", r.Chain[1].Kind)
	}
	if r.Chain[2].Kind != KindStore {
		t.Errorf("Chain[2].Kind = %s, want store", r.Chain[2].Kind)
	}
	if r.Chain[3].Kind != KindModel {
		t.Errorf("Chain[3].Kind = %s, want model", r.Chain[3].Kind)
	}
}

func TestTraceSecondRouteOnSameHandler(t *testing.T) {
	g := traceFixture()
	r := Trace("POST /users", g)

	if r.NotFound {
		t.Fatal("POST /users should match the same handler via multi-route meta")
	}
	if r.Chain[0].Name != "UserHandler" {
		t.Errorf("Expected UserHandler, got %s", r.Chain[0].Name)
	}
}

func TestTracePathOnlyMatch(t *testing.T) {
	g := traceFixture()
	r := Trace("/users/{id}", g) // no HTTP method

	if r.NotFound {
		t.Fatal("Path-only query should match")
	}
}

func TestTracePartialSubstringMatch(t *testing.T) {
	g := traceFixture()
	r := Trace("users", g)

	if r.NotFound {
		t.Fatal("Partial substring 'users' should match /users/{id}")
	}
}

func TestTraceNotFound(t *testing.T) {
	g := traceFixture()
	r := Trace("DELETE /nonexistent", g)

	if !r.NotFound {
		t.Error("Expected NotFound=true for unmatched route")
	}
	if r.Chain != nil {
		t.Error("Chain should be nil when not found")
	}
}

func TestTraceEmptyGraph(t *testing.T) {
	r := Trace("GET /anything", NewGraph())
	if !r.NotFound {
		t.Error("Expected NotFound on empty graph")
	}
}

func TestTraceWithTable(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "h", Kind: KindHandler, Name: "Handler", Package: "p",
		Meta: map[string]string{"route": "GET /orders", "routes": "GET /orders"}})
	g.AddNode(&Node{ID: "s", Kind: KindStore, Name: "Store", Package: "p"})
	g.AddNode(&Node{ID: "table.orders", Kind: KindTable, Name: "orders", Package: "db"})
	g.AddEdge("h", "s", EdgeDepends)
	g.AddEdge("s", "table.orders", EdgeMapsTo)

	r := Trace("GET /orders", g)
	if r.NotFound {
		t.Fatal("Route should be found")
	}
	if len(r.Tables) != 1 || r.Tables[0] != "orders" {
		t.Errorf("Expected tables=[orders], got %v", r.Tables)
	}
}

func TestTraceWithEnvVar(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "h", Kind: KindHandler, Name: "Handler", Package: "p",
		Meta: map[string]string{"route": "GET /health", "routes": "GET /health"}})
	g.AddNode(&Node{ID: "env.DATABASE_URL", Kind: KindEnvVar, Name: "DATABASE_URL", Package: "env"})
	g.AddEdge("h", "env.DATABASE_URL", EdgeReads)

	r := Trace("GET /health", g)
	if len(r.EnvVars) != 1 || r.EnvVars[0] != "DATABASE_URL" {
		t.Errorf("Expected env vars=[DATABASE_URL], got %v", r.EnvVars)
	}
}

func TestTraceInterfaceFollowsToImpl(t *testing.T) {
	// Handler → ServiceInterface ← service (concrete implements)
	// service → Store
	// Trace should reach Store even though it goes through an interface.
	g := NewGraph()
	g.AddNode(&Node{ID: "h", Kind: KindHandler, Name: "Handler", Package: "p",
		Meta: map[string]string{"route": "POST /items", "routes": "POST /items"}})
	g.AddNode(&Node{ID: "iface.Service", Kind: KindInterface, Name: "Service", Package: "p"})
	g.AddNode(&Node{ID: "svc.service", Kind: KindService, Name: "service", Package: "svc"})
	g.AddNode(&Node{ID: "store.Store", Kind: KindStore, Name: "Store", Package: "store"})

	g.AddEdge("h", "iface.Service", EdgeDepends)
	g.AddEdge("svc.service", "iface.Service", EdgeImplements) // concrete → interface
	g.AddEdge("svc.service", "store.Store", EdgeDepends)

	r := Trace("POST /items", g)
	if r.NotFound {
		t.Fatal("Route not found")
	}

	kindSet := make(map[NodeKind]bool)
	for _, n := range r.Chain {
		kindSet[n.Kind] = true
	}
	if !kindSet[KindStore] {
		t.Errorf("Expected Store in chain; got: %v", chainNames(r.Chain))
	}
}

// ─── Integration tests (real AST parsing) ────────────────────────────────────

func TestTraceEndToEnd_NetHTTP(t *testing.T) {
	// Use a struct method-based router to exercise route extraction.
	// reqflow extracts routes from receiver.METHOD("/path", handler) calls.
	dir := helperWriteModule(t, map[string]string{
		"handler/handler.go": `package handler

import "net/http"

type UserHandler struct{ svc *UserService }

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {}
`,
		"handler/service.go": `package handler

type UserService struct{ store *UserStore }

func (s *UserService) FindByID(id string) {}
`,
		"handler/store.go": `package handler

type UserStore struct{}

func (s *UserStore) Select(id string) {}
`,
		// Fake router that matches the receiver.GET pattern reqflow detects
		"router/router.go": `package router

type App struct{}
func (a *App) GET(path string, handler interface{}) {}
func (a *App) POST(path string, handler interface{}) {}
`,
		"main.go": `package main

import (
	"testmod/handler"
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

	// Verify handler is detected
	var handlerNode *Node
	for _, n := range graph.Nodes {
		if n.Kind == KindHandler && n.Name == "UserHandler" {
			handlerNode = n
			break
		}
	}
	if handlerNode == nil {
		t.Fatal("Expected UserHandler to be detected as KindHandler")
	}

	r := Trace("/users", graph)
	if r.NotFound {
		t.Fatal("Expected to find /users handler")
	}
	if r.Chain[0].Kind != KindHandler {
		t.Errorf("Chain[0].Kind = %s, want handler", r.Chain[0].Kind)
	}
}

func TestTraceEndToEnd_HandlerServiceStore(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"internal/handler/handler.go": `package handler

import "net/http"

type OrderService interface {
	CreateOrder(id string) error
}

type OrderHandler struct {
	svc OrderService
}

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {}
`,
		"internal/service/service.go": `package service

type Store interface {
	Insert(id string) error
}

type OrderService struct {
	store Store
}

func (s *OrderService) CreateOrder(id string) error { return nil }
`,
		"internal/store/store.go": `package store

type OrderStore struct{}

func (s *OrderStore) Insert(id string) error { return nil }
`,
		"router/router.go": `package router

type App struct{}
func (a *App) POST(path string, handler interface{}) {}
`,
		"main.go": `package main

import (
	hpkg "testmod/internal/handler"
	spkg "testmod/internal/service"
	storepkg "testmod/internal/store"
	"testmod/router"
)

func main() {
	_ = &storepkg.OrderStore{}
	_ = &spkg.OrderService{}
	h := &hpkg.OrderHandler{}
	app := &router.App{}
	app.POST("/orders", h.CreateOrder)
}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{})

	r := Trace("/orders", graph)
	if r.NotFound {
		t.Fatal("Expected handler to be found for /orders")
	}
	if r.Chain[0].Kind != KindHandler {
		t.Errorf("Chain[0].Kind = %s, want handler", r.Chain[0].Kind)
	}

	// Chain should reach Service layer (handler.svc OrderService field → interface → impl)
	kindSet := make(map[NodeKind]bool)
	for _, n := range r.Chain {
		kindSet[n.Kind] = true
	}
	if !kindSet[KindService] && !kindSet[KindInterface] {
		t.Errorf("Expected Service or Interface in chain, got: %v", chainNames(r.Chain))
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func chainNames(nodes []*Node) []string {
	names := make([]string, len(nodes))
	for i, n := range nodes {
		names[i] = string(n.Kind) + ":" + n.Name
	}
	return names
}
