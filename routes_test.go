package reqflow

import (
	"os"
	"strings"
	"testing"
)

// ─── addRoute ─────────────────────────────────────────────────────────────────

func TestAddRoute_FirstRoute(t *testing.T) {
	n := &Node{Meta: map[string]string{}}
	addRoute(n, "GET /users")

	if n.Meta["routes"] != "GET /users" {
		t.Errorf("routes = %q, want %q", n.Meta["routes"], "GET /users")
	}
	if n.Meta["route"] != "GET /users" {
		t.Errorf("route (primary) = %q, want %q", n.Meta["route"], "GET /users")
	}
}

func TestAddRoute_SecondRoute(t *testing.T) {
	n := &Node{Meta: map[string]string{}}
	addRoute(n, "GET /users")
	addRoute(n, "POST /users")

	routes := strings.Split(n.Meta["routes"], "\n")
	if len(routes) != 2 {
		t.Fatalf("Expected 2 routes, got %d: %v", len(routes), routes)
	}
	// Primary route must remain the first one
	if n.Meta["route"] != "GET /users" {
		t.Errorf("Primary route = %q, want %q", n.Meta["route"], "GET /users")
	}
}

func TestAddRoute_Deduplication(t *testing.T) {
	n := &Node{Meta: map[string]string{}}
	addRoute(n, "GET /users")
	addRoute(n, "GET /users") // duplicate
	addRoute(n, "GET /users") // duplicate again

	routes := strings.Split(n.Meta["routes"], "\n")
	if len(routes) != 1 {
		t.Errorf("Expected 1 route after dedup, got %d: %v", len(routes), routes)
	}
}

func TestAddRoute_MultipleDistinct(t *testing.T) {
	n := &Node{Meta: map[string]string{}}
	addRoute(n, "GET /a")
	addRoute(n, "POST /b")
	addRoute(n, "DELETE /c")

	routes := strings.Split(n.Meta["routes"], "\n")
	if len(routes) != 3 {
		t.Fatalf("Expected 3 routes, got %d: %v", len(routes), routes)
	}
}

// ─── Anonymous function handler detection ─────────────────────────────────────

func TestTraceEndToEnd_InlineHandler(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"router/router.go": `package router

type App struct{}
func (a *App) GET(path string, handler interface{}) {}
func (a *App) POST(path string, handler interface{}) {}
`,
		"main.go": `package main

import "testmod/router"

func main() {
	app := &router.App{}
	app.GET("/health", func() interface{} { return "ok" })
	app.POST("/webhook", func() interface{} { return nil })
}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{})

	// Both inline handlers should be detected
	r1 := Trace("/health", graph)
	if r1.NotFound {
		t.Error("Expected inline GET /health handler to be found")
	}
	if r1.Chain[0].Kind != KindHandler {
		t.Errorf("Expected KindHandler, got %s", r1.Chain[0].Kind)
	}

	r2 := Trace("/webhook", graph)
	if r2.NotFound {
		t.Error("Expected inline POST /webhook handler to be found")
	}
}

func TestTraceEndToEnd_InlineHandlerName(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"router/router.go": `package router

type App struct{}
func (a *App) GET(path string, handler interface{}) {}
`,
		"main.go": `package main

import "testmod/router"

func main() {
	app := &router.App{}
	app.GET("/api/status", func() interface{} { return "ok" })
}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{})

	r := Trace("/api/status", graph)
	if r.NotFound {
		t.Fatal("Expected /api/status to be found")
	}
	// Node name should be derived from method+path
	name := r.Chain[0].Name
	if !strings.Contains(name, "GET") || !strings.Contains(name, "api") {
		t.Errorf("Expected name to contain method+path, got %q", name)
	}
}

func TestTraceEndToEnd_InlineAndStructHandlers(t *testing.T) {
	// Mix of inline and struct method handlers in the same repo
	dir := helperWriteModule(t, map[string]string{
		"handler/handler.go": `package handler

import "net/http"

type OrderHandler struct{}

func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {}
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
	app.GET("/health", func() interface{} { return "ok" })
}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{})

	// Struct method handler
	rOrder := Trace("/orders/{id}", graph)
	if rOrder.NotFound {
		t.Error("Expected /orders/{id} to be found via struct handler")
	}

	// Inline handler
	rHealth := Trace("/health", graph)
	if rHealth.NotFound {
		t.Error("Expected /health to be found via inline handler")
	}
}
