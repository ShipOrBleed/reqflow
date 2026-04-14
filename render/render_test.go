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
