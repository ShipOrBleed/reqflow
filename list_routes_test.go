package reqflow

import (
	"os"
	"strings"
	"testing"
)

func TestListRoutes_Basic(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"handler/handler.go": `package handler

import "net/http"

type OrderHandler struct{}

func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {}
func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {}
`,
		"router/router.go": `package router

type App struct{}
func (a *App) GET(path string, h interface{}) {}
func (a *App) POST(path string, h interface{}) {}
`,
		"main.go": `package main

import (
	"testmod/handler"
	"testmod/router"
)

func main() {
	app := &router.App{}
	h := &handler.OrderHandler{}
	app.GET("/orders/{id}", h.GetOrder)
	app.POST("/orders", h.CreateOrder)
}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{})
	routes := ListRoutes(graph)

	if len(routes) != 2 {
		t.Fatalf("Expected 2 routes, got %d", len(routes))
	}

	// Verify route data
	found := make(map[string]bool)
	for _, r := range routes {
		found[r.Method+" "+r.Path] = true
		if r.HandlerName != "OrderHandler" {
			t.Errorf("Expected handler name OrderHandler, got %q", r.HandlerName)
		}
	}

	if !found["GET /orders/{id}"] {
		t.Error("Expected GET /orders/{id} route")
	}
	if !found["POST /orders"] {
		t.Error("Expected POST /orders route")
	}
}

func TestListRoutes_InlineHandlers(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"router/router.go": `package router

type App struct{}
func (a *App) GET(path string, handler interface{}) {}
`,
		"main.go": `package main

import "testmod/router"

func main() {
	app := &router.App{}
	app.GET("/health", func() interface{} { return "ok" })
}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{})
	routes := ListRoutes(graph)

	if len(routes) != 1 {
		t.Fatalf("Expected 1 route, got %d", len(routes))
	}
	if routes[0].Method != "GET" {
		t.Errorf("Expected method GET, got %q", routes[0].Method)
	}
	if routes[0].Path != "/health" {
		t.Errorf("Expected path /health, got %q", routes[0].Path)
	}
}

func TestListRoutes_SortedOutput(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"handler/handler.go": `package handler

import "net/http"

type Handler struct{}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {}
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request)    {}
func (h *Handler) GetOrders(w http.ResponseWriter, r *http.Request)  {}
func (h *Handler) PostOrders(w http.ResponseWriter, r *http.Request) {}
`,
		"router/router.go": `package router

type App struct{}
func (a *App) GET(path string, h interface{})    {}
func (a *App) POST(path string, h interface{})   {}
func (a *App) DELETE(path string, h interface{}) {}
`,
		"main.go": `package main

import (
	"testmod/handler"
	"testmod/router"
)

func main() {
	app := &router.App{}
	h := &handler.Handler{}
	app.DELETE("/users/{id}", h.DeleteUser)
	app.GET("/users/{id}", h.GetUser)
	app.GET("/orders", h.GetOrders)
	app.POST("/orders", h.PostOrders)
}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{})
	routes := ListRoutes(graph)

	if len(routes) != 4 {
		t.Fatalf("Expected 4 routes, got %d", len(routes))
	}

	// Sorted by path then method: /orders GET, /orders POST, /users/{id} DELETE, /users/{id} GET
	expected := []string{
		"GET /orders",
		"POST /orders",
		"DELETE /users/{id}",
		"GET /users/{id}",
	}

	for i, r := range routes {
		got := r.Method + " " + r.Path
		if got != expected[i] {
			t.Errorf("Route[%d] = %q, want %q", i, got, expected[i])
		}
	}
}

func TestListRoutes_EmptyGraph(t *testing.T) {
	g := NewGraph()
	routes := ListRoutes(g)

	if len(routes) != 0 {
		t.Errorf("Expected 0 routes for empty graph, got %d", len(routes))
	}

	// Also test FormatRoutesText with empty routes
	output := FormatRoutesText(routes)
	if !strings.Contains(output, "No routes found") {
		t.Errorf("Expected 'No routes found' message, got %q", output)
	}
}

func TestListRoutes_NilGraph(t *testing.T) {
	routes := ListRoutes(nil)
	if routes != nil {
		t.Errorf("Expected nil routes for nil graph, got %v", routes)
	}
}

func TestFormatRoutesText_WithRoutes(t *testing.T) {
	routes := []RouteInfo{
		{Method: "GET", Path: "/users", HandlerName: "Handler", MethodName: "GetUsers", File: "/app/handler.go", Line: 10},
		{Method: "POST", Path: "/users", HandlerName: "Handler", MethodName: "CreateUser", File: "/app/handler.go", Line: 30},
	}
	output := FormatRoutesText(routes)

	if !strings.Contains(output, "GET") || !strings.Contains(output, "/users") {
		t.Error("Expected GET /users in output")
	}
	if !strings.Contains(output, "Handler.GetUsers()") {
		t.Error("Expected Handler.GetUsers() in output")
	}
	if !strings.Contains(output, "Handler.CreateUser()") {
		t.Error("Expected Handler.CreateUser() in output")
	}
	if !strings.Contains(output, "2 routes across 1 handlers") {
		t.Error("Expected summary line")
	}
}

func TestFormatRoutesText_MultipleHandlers(t *testing.T) {
	routes := []RouteInfo{
		{Method: "GET", Path: "/a", HandlerName: "HandlerA", MethodName: "Get"},
		{Method: "GET", Path: "/b", HandlerName: "HandlerB", MethodName: "Get"},
	}
	output := FormatRoutesText(routes)

	if !strings.Contains(output, "2 routes across 2 handlers") {
		t.Errorf("Expected '2 routes across 2 handlers', got: %s", output)
	}
}

func TestFormatRoutesJSON(t *testing.T) {
	routes := []RouteInfo{
		{Method: "GET", Path: "/users", HandlerName: "Handler", MethodName: "GetUsers", File: "/app/handler.go", Line: 10},
	}
	output := FormatRoutesJSON(routes)

	if !strings.Contains(output, `"method":"GET"`) {
		t.Error("Expected JSON with method field")
	}
	if !strings.Contains(output, `"path":"/users"`) {
		t.Error("Expected JSON with path field")
	}
	if !strings.Contains(output, `"handler":"Handler"`) {
		t.Error("Expected JSON with handler field")
	}
	if !strings.HasPrefix(output, "[") || !strings.Contains(output, "]") {
		t.Error("Expected JSON array")
	}
}

func TestFormatRoutesJSON_Empty(t *testing.T) {
	output := FormatRoutesJSON(nil)
	if !strings.HasPrefix(output, "[") {
		t.Error("Expected JSON array even for empty")
	}
}

func TestHandlerDisplay_WithMethod(t *testing.T) {
	r := RouteInfo{HandlerName: "UserHandler", MethodName: "Get"}
	if got := handlerDisplay(r); got != "UserHandler.Get()" {
		t.Errorf("handlerDisplay = %q, want UserHandler.Get()", got)
	}
}

func TestHandlerDisplay_WithoutMethod(t *testing.T) {
	r := RouteInfo{HandlerName: "InlineHandler"}
	if got := handlerDisplay(r); got != "InlineHandler()" {
		t.Errorf("handlerDisplay = %q, want InlineHandler()", got)
	}
}

func TestShortLocation(t *testing.T) {
	cases := []struct {
		file string
		line int
		want string
	}{
		{"/a/b/c/handler.go", 42, "c/handler.go:42"},
		{"/handler.go", 10, "/handler.go:10"},
		{"", 0, ""},
		{"/a/b/c/d.go", 0, "c/d.go"},
	}
	for _, tc := range cases {
		got := shortLocation(tc.file, tc.line)
		if got != tc.want {
			t.Errorf("shortLocation(%q, %d) = %q, want %q", tc.file, tc.line, got, tc.want)
		}
	}
}

func TestParseClientKind(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"client.go": `package testmod

type DiscovererClient struct {
	baseURL string
}

func (c *DiscovererClient) Discover() error { return nil }
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{})

	node, ok := graph.Nodes["testmod.DiscovererClient"]
	if !ok {
		t.Fatal("Expected DiscovererClient node")
	}
	if node.Kind != KindClient {
		t.Errorf("DiscovererClient should be KindClient, got %s", node.Kind)
	}
}

func TestParseStoreNotClient(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"store.go": `package testmod

import "database/sql"

type UserStore struct {
	db *sql.DB
}

func (s *UserStore) GetByID(id int) error { return nil }
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{})

	node, ok := graph.Nodes["testmod.UserStore"]
	if !ok {
		t.Fatal("Expected UserStore node")
	}
	if node.Kind != KindStore {
		t.Errorf("UserStore with *sql.DB field should be KindStore, got %s", node.Kind)
	}
}
