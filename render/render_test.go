package render

import (
	"bytes"
	"strings"
	"testing"

	reqflow "github.com/thzgajendra/reqflow"
)

// ─── TraceRenderer tests ──────────────────────────────────────────────────────

func traceGraph() (*reqflow.Graph, *reqflow.TraceResult) {
	g := reqflow.NewGraph()
	g.AddNode(&reqflow.Node{
		ID: "h", Kind: reqflow.KindHandler, Name: "UserHandler", Package: "pkg",
		Meta:    map[string]string{"route": "GET /users/{id}", "routes": "GET /users/{id}\nPOST /users"},
		Methods: []string{"GetUser", "CreateUser"},
	})
	g.AddNode(&reqflow.Node{
		ID: "s", Kind: reqflow.KindService, Name: "UserService", Package: "pkg",
		Methods: []string{"FindByID", "Create"},
	})
	g.AddNode(&reqflow.Node{
		ID: "r", Kind: reqflow.KindStore, Name: "UserStore", Package: "pkg",
		Methods: []string{"Select", "Insert"},
	})
	g.AddNode(&reqflow.Node{
		ID: "m", Kind: reqflow.KindModel, Name: "User", Package: "models",
		Fields: []reqflow.Field{{Name: "ID", Type: "int"}, {Name: "Name", Type: "string"}},
	})
	g.AddEdge("h", "s", reqflow.EdgeDepends)
	g.AddEdge("s", "r", reqflow.EdgeDepends)
	g.AddEdge("r", "m", reqflow.EdgeDepends)
	result := reqflow.Trace("GET /users/{id}", g)
	return g, result
}

func TestTraceRendererText(t *testing.T) {
	_, result := traceGraph()
	var buf bytes.Buffer
	tr := &TraceRenderer{Format: "text"}
	if err := tr.RenderTrace(result, &buf); err != nil {
		t.Fatalf("RenderTrace text failed: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "UserHandler") {
		t.Error("Expected UserHandler in text output")
	}
	if !strings.Contains(out, "delegates to") {
		t.Error("Expected 'delegates to' in text output")
	}
	if !strings.Contains(out, "queries via") {
		t.Error("Expected 'queries via' in text output")
	}
	if !strings.Contains(out, "maps to model") {
		t.Error("Expected 'maps to model' in text output")
	}
}

func TestTraceRendererHTML(t *testing.T) {
	_, result := traceGraph()
	var buf bytes.Buffer
	tr := &TraceRenderer{Format: "html"}
	if err := tr.RenderTrace(result, &buf); err != nil {
		t.Fatalf("RenderTrace html failed: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "<!DOCTYPE html>") {
		t.Error("Expected HTML doctype")
	}
	if !strings.Contains(out, "UserHandler") {
		t.Error("Expected UserHandler in HTML output")
	}
}

func TestTraceRendererNotFound(t *testing.T) {
	result := &reqflow.TraceResult{Route: "GET /missing", NotFound: true}
	var buf bytes.Buffer
	tr := &TraceRenderer{Format: "text"}
	if err := tr.RenderTrace(result, &buf); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "No handler found") {
		t.Error("Expected 'No handler found' message")
	}
}

func TestTraceRendererHTMLNotFound(t *testing.T) {
	result := &reqflow.TraceResult{Route: "DELETE /gone", NotFound: true}
	var buf bytes.Buffer
	tr := &TraceRenderer{Format: "html"}
	if err := tr.RenderTrace(result, &buf); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "<!DOCTYPE html>") {
		t.Error("Expected HTML even for not-found")
	}
}

func TestTraceRendererText_WithCalledMethods(t *testing.T) {
	result := &reqflow.TraceResult{
		Route: "GET /items",
		Handler: &reqflow.Node{ID: "h", Kind: reqflow.KindHandler, Name: "Handler", Package: "pkg",
			Meta: map[string]string{"route_method:GET /items": "ListItems",
				"method_file:ListItems": "/app/handler.go", "method_line:ListItems": "42"}},
		Chain: []*reqflow.Node{
			{ID: "h", Kind: reqflow.KindHandler, Name: "Handler", Package: "pkg",
				Meta: map[string]string{"route_method:GET /items": "ListItems",
					"method_file:ListItems": "/app/handler.go", "method_line:ListItems": "42"}},
			{ID: "s", Kind: reqflow.KindService, Name: "Service", Package: "svc",
				Methods: []string{"ListItems", "CreateItem"},
				Meta:    map[string]string{}},
		},
		CalledMethods: map[string][]string{
			"s": {"ListItems"},
		},
		MethodCalls: reqflow.MethodCallIndex{
			"h.ListItems": {{FieldName: "svc", TargetMethod: "ListItems"}},
		},
	}

	var buf bytes.Buffer
	tr := &TraceRenderer{Format: "text"}
	if err := tr.RenderTrace(result, &buf); err != nil {
		t.Fatalf("Error: %v", err)
	}
	out := buf.String()

	// Handler should show specific method, not all methods
	if !strings.Contains(out, "ListItems()") {
		t.Error("Expected ListItems() in output")
	}
	// Sub-call should be shown
	if !strings.Contains(out, "svc.ListItems()") {
		t.Error("Expected sub-call → svc.ListItems()")
	}
	// Method line should be used (42), not struct line
	if !strings.Contains(out, ":42") {
		t.Error("Expected method line number :42")
	}
}

func TestTraceRendererText_WithClientKind(t *testing.T) {
	result := &reqflow.TraceResult{
		Route: "GET /data",
		Handler: &reqflow.Node{ID: "h", Kind: reqflow.KindHandler, Name: "H", Package: "p",
			Meta: map[string]string{}},
		Chain: []*reqflow.Node{
			{ID: "h", Kind: reqflow.KindHandler, Name: "H", Package: "p", Meta: map[string]string{}},
			{ID: "c", Kind: reqflow.KindClient, Name: "APIClient", Package: "client", Meta: map[string]string{}},
		},
	}

	var buf bytes.Buffer
	tr := &TraceRenderer{Format: "text"}
	tr.RenderTrace(result, &buf)
	out := buf.String()

	if !strings.Contains(out, "[C]") {
		t.Error("Expected [C] badge for KindClient")
	}
	if !strings.Contains(out, "External Client") {
		t.Error("Expected 'External Client' label for KindClient")
	}
}

func TestTraceRendererText_MultiMatch(t *testing.T) {
	result := &reqflow.TraceResult{
		Route:      "/orders",
		MultiMatch: []string{"GET /orders", "POST /orders", "DELETE /orders/{id}"},
	}

	var buf bytes.Buffer
	tr := &TraceRenderer{Format: "text"}
	tr.RenderTrace(result, &buf)
	out := buf.String()

	if !strings.Contains(out, "Multiple routes") {
		t.Error("Expected 'Multiple routes' in multi-match output")
	}
	if !strings.Contains(out, "1.") || !strings.Contains(out, "2.") || !strings.Contains(out, "3.") {
		t.Error("Expected numbered list")
	}
}

func TestTraceRendererHTML_WithTables(t *testing.T) {
	result := &reqflow.TraceResult{
		Route: "GET /orders",
		Handler: &reqflow.Node{ID: "h", Kind: reqflow.KindHandler, Name: "H", Package: "p",
			Meta: map[string]string{}},
		Chain: []*reqflow.Node{
			{ID: "h", Kind: reqflow.KindHandler, Name: "H", Package: "p", Meta: map[string]string{}},
		},
		Tables:  []string{"orders", "order_items"},
		EnvVars: []string{"DB_HOST"},
	}

	var buf bytes.Buffer
	tr := &TraceRenderer{Format: "html"}
	tr.RenderTrace(result, &buf)
	out := buf.String()

	if !strings.Contains(out, "orders") {
		t.Error("Expected table name in HTML")
	}
	if !strings.Contains(out, "DB_HOST") {
		t.Error("Expected env var in HTML")
	}
}

func TestTraceRendererText_TablesAndEnvVars(t *testing.T) {
	result := &reqflow.TraceResult{
		Route: "GET /orders",
		Handler: &reqflow.Node{ID: "h", Kind: reqflow.KindHandler, Name: "H", Package: "p",
			Meta: map[string]string{}},
		Chain: []*reqflow.Node{
			{ID: "h", Kind: reqflow.KindHandler, Name: "H", Package: "p", Meta: map[string]string{}},
		},
		Tables:  []string{"orders"},
		EnvVars: []string{"DB_URL", "REDIS_HOST"},
	}

	var buf bytes.Buffer
	tr := &TraceRenderer{Format: "text"}
	tr.RenderTrace(result, &buf)
	out := buf.String()

	if !strings.Contains(out, "Database tables") {
		t.Error("Expected 'Database tables' section")
	}
	if !strings.Contains(out, "orders") {
		t.Error("Expected 'orders' table")
	}
	if !strings.Contains(out, "Environment variables") {
		t.Error("Expected 'Environment variables' section")
	}
	if !strings.Contains(out, "DB_URL") || !strings.Contains(out, "REDIS_HOST") {
		t.Error("Expected env vars in output")
	}
}
