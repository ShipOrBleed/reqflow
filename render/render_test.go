package render

import (
	"bytes"
	"strings"
	"testing"

	govis "github.com/thzgajendra/govis"
)

// testGraph builds a small representative graph for renderer testing.
func testGraph() *govis.Graph {
	g := govis.NewGraph()

	g.AddNode(&govis.Node{
		ID: "app.UserHandler", Kind: govis.KindHandler, Name: "UserHandler",
		Package: "app", File: "/app/handler.go", Line: 10,
		Methods: []string{"GetUser", "CreateUser"},
		Meta:    map[string]string{"route": "GET /users/{id}"},
	})
	g.AddNode(&govis.Node{
		ID: "app.UserService", Kind: govis.KindService, Name: "UserService",
		Package: "app", File: "/app/service.go", Line: 20,
		Methods: []string{"FindByID", "Create"},
	})
	g.AddNode(&govis.Node{
		ID: "app.UserStore", Kind: govis.KindStore, Name: "UserStore",
		Package: "app", File: "/app/store.go", Line: 30,
		Fields: []govis.Field{{Name: "db", Type: "*sql.DB"}},
	})
	g.AddNode(&govis.Node{
		ID: "app.User", Kind: govis.KindModel, Name: "User",
		Package: "models", File: "/models/user.go", Line: 5,
		Fields: []govis.Field{{Name: "ID", Type: "int"}, {Name: "Name", Type: "string"}},
	})

	g.AddEdge("app.UserHandler", "app.UserService", govis.EdgeDepends)
	g.AddEdge("app.UserService", "app.UserStore", govis.EdgeDepends)
	g.AddEdge("app.UserStore", "app.User", govis.EdgeDepends)

	return g
}

func TestMermaidRenderer(t *testing.T) {
	var buf bytes.Buffer
	r := &MermaidRenderer{}
	if err := r.Render(testGraph(), &buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "classDiagram") {
		t.Error("Expected 'classDiagram' in mermaid output")
	}
	if !strings.Contains(out, "UserHandler") {
		t.Error("Expected 'UserHandler' in mermaid output")
	}
	if !strings.Contains(out, "depends") {
		t.Error("Expected 'depends' edge label")
	}
	if !strings.Contains(out, "classDef handler") {
		t.Error("Expected handler class definition")
	}
}

func TestHTMLRenderer(t *testing.T) {
	var buf bytes.Buffer
	r := &HTMLRenderer{}
	if err := r.Render(testGraph(), &buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "<!DOCTYPE html>") {
		t.Error("Expected HTML doctype")
	}
	if !strings.Contains(out, "GOVIS") {
		t.Error("Expected GOVIS branding")
	}
	if !strings.Contains(out, "mermaid") {
		t.Error("Expected mermaid script reference")
	}
}

func TestInteractiveRenderer(t *testing.T) {
	var buf bytes.Buffer
	r := &InteractiveRenderer{}
	if err := r.Render(testGraph(), &buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "cytoscape") {
		t.Error("Expected cytoscape.js reference")
	}
	if !strings.Contains(out, "UserHandler") {
		t.Error("Expected UserHandler in graph data")
	}
	if !strings.Contains(out, "handler") {
		t.Error("Expected 'handler' kind in data")
	}
}

func TestJSONRenderer(t *testing.T) {
	var buf bytes.Buffer
	r := &JSONRenderer{}
	if err := r.Render(testGraph(), &buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, `"UserHandler"`) {
		t.Error("Expected UserHandler in JSON")
	}
	if !strings.Contains(out, `"handler"`) {
		t.Error("Expected handler kind in JSON")
	}
}

func TestMarkdownRenderer(t *testing.T) {
	var buf bytes.Buffer
	r := &MarkdownRenderer{}
	if err := r.Render(testGraph(), &buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "UserHandler") {
		t.Error("Expected UserHandler in markdown")
	}
	if !strings.Contains(out, "|") {
		t.Error("Expected markdown table")
	}
}

func TestC4Renderer(t *testing.T) {
	var buf bytes.Buffer
	r := &C4Renderer{}
	if err := r.Render(testGraph(), &buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "@startuml") {
		t.Error("Expected @startuml")
	}
	if !strings.Contains(out, "@enduml") {
		t.Error("Expected @enduml")
	}
	if !strings.Contains(out, "Component") {
		t.Error("Expected C4 Component")
	}
}

func TestDOTRenderer(t *testing.T) {
	var buf bytes.Buffer
	r := &DOTRenderer{}
	if err := r.Render(testGraph(), &buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "digraph") {
		t.Error("Expected 'digraph' in DOT output")
	}
	if !strings.Contains(out, "->") {
		t.Error("Expected '->' edges in DOT output")
	}
}

func TestDSMRenderer(t *testing.T) {
	var buf bytes.Buffer
	r := &DSMRenderer{}
	if err := r.Render(testGraph(), &buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "Dependency Structure Matrix") {
		t.Error("Expected DSM header")
	}
	if !strings.Contains(out, "Legend") {
		t.Error("Expected Legend section")
	}
}

func TestExcalidrawRenderer(t *testing.T) {
	var buf bytes.Buffer
	r := &ExcalidrawRenderer{}
	if err := r.Render(testGraph(), &buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, `"type": "excalidraw"`) {
		t.Error("Expected excalidraw type field")
	}
	if !strings.Contains(out, `"rectangle"`) {
		t.Error("Expected rectangle elements")
	}
}

func TestEmbedRenderer(t *testing.T) {
	var buf bytes.Buffer
	r := &EmbedRenderer{}
	if err := r.Render(testGraph(), &buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "GOVIS") {
		t.Error("Expected GOVIS branding")
	}
	if !strings.Contains(out, "classDiagram") {
		t.Error("Expected embedded mermaid diagram")
	}
}

func TestThreeRenderer(t *testing.T) {
	var buf bytes.Buffer
	r := &ThreeRenderer{}
	if err := r.Render(testGraph(), &buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "three") {
		t.Error("Expected three.js reference")
	}
	if !strings.Contains(out, "3d-force-graph") {
		t.Error("Expected 3d-force-graph reference")
	}
	if !strings.Contains(out, "UserHandler") {
		t.Error("Expected UserHandler in graph data")
	}
}

func TestPDFRendererFallback(t *testing.T) {
	// PDF renderer should fall back to DOT if graphviz is not installed
	var buf bytes.Buffer
	r := &PDFRenderer{}
	if err := r.Render(testGraph(), &buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	out := buf.String()
	// Should contain either PDF binary or DOT fallback
	if len(out) == 0 {
		t.Error("Expected non-empty output from PDF renderer")
	}
}

func TestAPIMapRenderer(t *testing.T) {
	var buf bytes.Buffer
	r := &APIMapRenderer{}
	g := testGraph()
	if err := r.Render(g, &buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	out := buf.String()

	// Should find the route from handler metadata
	if !strings.Contains(out, "GET") || !strings.Contains(out, "/users") {
		t.Error("Expected GET /users route in API map output")
	}
}

func TestDataFlowRenderer(t *testing.T) {
	var buf bytes.Buffer
	r := &DataFlowRenderer{}
	g := testGraph()
	if err := r.Render(g, &buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "sequenceDiagram") {
		t.Error("Expected sequenceDiagram in dataflow output")
	}
}

func TestTimelineRendererEmpty(t *testing.T) {
	var buf bytes.Buffer
	r := &TimelineRenderer{}
	if err := r.Render(testGraph(), &buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "No evolution snapshots") {
		t.Error("Expected empty timeline message")
	}
}

func TestTimelineRendererWithData(t *testing.T) {
	var buf bytes.Buffer
	r := &TimelineRenderer{
		Snapshots: []govis.EvolutionSnapshot{
			{Ref: "v1.0", NodeCount: 10, EdgeCount: 5, Packages: 3, KindCount: map[govis.NodeKind]int{govis.KindHandler: 2}},
			{Ref: "v2.0", NodeCount: 15, EdgeCount: 8, Packages: 4, KindCount: map[govis.NodeKind]int{govis.KindHandler: 3}, Added: []string{"new.Node"}},
		},
	}
	if err := r.Render(testGraph(), &buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "Architecture Evolution Timeline") {
		t.Error("Expected timeline header")
	}
	if !strings.Contains(out, "v1.0") || !strings.Contains(out, "v2.0") {
		t.Error("Expected version refs in timeline")
	}
}

// ==================== Renderer on empty graph ====================

func TestRenderersHandleEmptyGraph(t *testing.T) {
	empty := govis.NewGraph()
	renderers := map[string]Renderer{
		"mermaid":     &MermaidRenderer{},
		"html":        &HTMLRenderer{},
		"interactive": &InteractiveRenderer{},
		"json":        &JSONRenderer{},
		"markdown":    &MarkdownRenderer{},
		"c4":          &C4Renderer{},
		"dot":         &DOTRenderer{},
		"dsm":         &DSMRenderer{},
		"excalidraw":  &ExcalidrawRenderer{},
		"embed":       &EmbedRenderer{},
		"three":       &ThreeRenderer{},
		"apimap":      &APIMapRenderer{},
		"dataflow":    &DataFlowRenderer{},
	}

	for name, r := range renderers {
		var buf bytes.Buffer
		if err := r.Render(empty, &buf); err != nil {
			t.Errorf("%s renderer failed on empty graph: %v", name, err)
		}
		if buf.Len() == 0 {
			t.Errorf("%s renderer produced empty output on empty graph", name)
		}
	}
}
