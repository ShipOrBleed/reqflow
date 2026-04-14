package reqflow

import "testing"

// ─── EdgeLabel ───────────────────────────────────────────────────────────────

func TestEdgeLabel_HandlerToService(t *testing.T) {
	from := &Node{Kind: KindHandler}
	to := &Node{Kind: KindService}
	if got := EdgeLabel(from, to); got != "delegates to" {
		t.Errorf("EdgeLabel(handler→service) = %q, want %q", got, "delegates to")
	}
}

func TestEdgeLabel_HandlerToStore(t *testing.T) {
	from := &Node{Kind: KindHandler}
	to := &Node{Kind: KindStore}
	if got := EdgeLabel(from, to); got != "queries via" {
		t.Errorf("EdgeLabel(handler→store) = %q, want %q", got, "queries via")
	}
}

func TestEdgeLabel_ServiceToStore(t *testing.T) {
	from := &Node{Kind: KindService}
	to := &Node{Kind: KindStore}
	if got := EdgeLabel(from, to); got != "queries via" {
		t.Errorf("EdgeLabel(service→store) = %q, want %q", got, "queries via")
	}
}

func TestEdgeLabel_ServiceToInterface(t *testing.T) {
	from := &Node{Kind: KindService}
	to := &Node{Kind: KindInterface}
	if got := EdgeLabel(from, to); got != "uses interface" {
		t.Errorf("EdgeLabel(service→interface) = %q, want %q", got, "uses interface")
	}
}

func TestEdgeLabel_StoreToModel(t *testing.T) {
	from := &Node{Kind: KindStore}
	to := &Node{Kind: KindModel}
	if got := EdgeLabel(from, to); got != "maps to model" {
		t.Errorf("EdgeLabel(store→model) = %q, want %q", got, "maps to model")
	}
}

func TestEdgeLabel_StoreToTable(t *testing.T) {
	from := &Node{Kind: KindStore}
	to := &Node{Kind: KindTable}
	if got := EdgeLabel(from, to); got != "writes to" {
		t.Errorf("EdgeLabel(store→table) = %q, want %q", got, "writes to")
	}
}

func TestEdgeLabel_ToEvent(t *testing.T) {
	from := &Node{Kind: KindService}
	to := &Node{Kind: KindEvent}
	if got := EdgeLabel(from, to); got != "publishes event" {
		t.Errorf("EdgeLabel(→event) = %q, want %q", got, "publishes event")
	}
}

func TestEdgeLabel_ToGRPC(t *testing.T) {
	from := &Node{Kind: KindService}
	to := &Node{Kind: KindGRPC}
	if got := EdgeLabel(from, to); got != "calls gRPC" {
		t.Errorf("EdgeLabel(→grpc) = %q, want %q", got, "calls gRPC")
	}
}

func TestEdgeLabel_Default(t *testing.T) {
	from := &Node{Kind: KindStruct}
	to := &Node{Kind: KindStruct}
	if got := EdgeLabel(from, to); got != "→" {
		t.Errorf("EdgeLabel(default) = %q, want %q", got, "→")
	}
}

// ─── rankOf ──────────────────────────────────────────────────────────────────

func TestRankOf_KnownKinds(t *testing.T) {
	cases := []struct {
		kind NodeKind
		want int
	}{
		{KindHandler, 0},
		{KindMiddleware, 1},
		{KindGRPC, 2},
		{KindService, 3},
		{KindInterface, 4},
		{KindEvent, 5},
		{KindStore, 6},
		{KindInfra, 7},
		{KindModel, 8},
		{KindTable, 9},
	}
	for _, tc := range cases {
		n := &Node{Kind: tc.kind}
		if got := rankOf(n); got != tc.want {
			t.Errorf("rankOf(%s) = %d, want %d", tc.kind, got, tc.want)
		}
	}
}

func TestRankOf_UnknownKind(t *testing.T) {
	n := &Node{Kind: KindStruct}
	if got := rankOf(n); got != 99 {
		t.Errorf("rankOf(unknown) = %d, want 99", got)
	}
}

// ─── sortByLayer ─────────────────────────────────────────────────────────────

func TestSortByLayer(t *testing.T) {
	nodes := []*Node{
		{Kind: KindStore},
		{Kind: KindHandler},
		{Kind: KindModel},
		{Kind: KindService},
	}
	sortByLayer(nodes)

	expected := []NodeKind{KindHandler, KindService, KindStore, KindModel}
	for i, n := range nodes {
		if n.Kind != expected[i] {
			t.Errorf("nodes[%d].Kind = %s, want %s", i, n.Kind, expected[i])
		}
	}
}

// ─── nodeRoutes ──────────────────────────────────────────────────────────────

func TestNodeRoutes_Empty(t *testing.T) {
	n := &Node{Meta: map[string]string{}}
	if routes := nodeRoutes(n); routes != nil {
		t.Errorf("Expected nil routes on empty node, got %v", routes)
	}
}

func TestNodeRoutes_FallbackToRoute(t *testing.T) {
	n := &Node{Meta: map[string]string{"route": "GET /users"}}
	routes := nodeRoutes(n)
	if len(routes) != 1 || routes[0] != "GET /users" {
		t.Errorf("Expected [GET /users], got %v", routes)
	}
}

func TestNodeRoutes_MultipleFromRoutes(t *testing.T) {
	n := &Node{Meta: map[string]string{"routes": "GET /a\nPOST /b\nDELETE /c"}}
	routes := nodeRoutes(n)
	if len(routes) != 3 {
		t.Fatalf("Expected 3 routes, got %d: %v", len(routes), routes)
	}
}

func TestNodeRoutes_SkipsBlankLines(t *testing.T) {
	n := &Node{Meta: map[string]string{"routes": "GET /a\n\n\nPOST /b"}}
	routes := nodeRoutes(n)
	if len(routes) != 2 {
		t.Errorf("Expected 2 routes (blank lines skipped), got %d: %v", len(routes), routes)
	}
}

// ─── buildChain deduplication ─────────────────────────────────────────────────

func TestBuildChain_Deduplication(t *testing.T) {
	// If BFS could visit a node twice (via multiple paths), result must be deduped
	g := NewGraph()
	g.AddNode(&Node{ID: "h", Kind: KindHandler, Name: "H", Package: "p",
		Meta: map[string]string{"route": "GET /x", "routes": "GET /x"}})
	g.AddNode(&Node{ID: "s1", Kind: KindService, Name: "S1", Package: "p"})
	g.AddNode(&Node{ID: "s2", Kind: KindService, Name: "S2", Package: "p"})
	g.AddNode(&Node{ID: "store", Kind: KindStore, Name: "Store", Package: "p"})

	g.AddEdge("h", "s1", EdgeDepends)
	g.AddEdge("h", "s2", EdgeDepends)
	g.AddEdge("s1", "store", EdgeDepends)
	g.AddEdge("s2", "store", EdgeDepends) // two paths to same store

	r := Trace("GET /x", g)
	if r.NotFound {
		t.Fatal("route not found")
	}

	seen := make(map[string]int)
	for _, n := range r.Chain {
		seen[n.ID]++
		if seen[n.ID] > 1 {
			t.Errorf("Node %q appears %d times in chain (should be deduped)", n.ID, seen[n.ID])
		}
	}
}

// ─── matchedRouteString ───────────────────────────────────────────────────────

func TestMatchedRouteString_ExactMatch(t *testing.T) {
	h := &Node{Meta: map[string]string{"routes": "GET /orders\nPOST /orders"}}
	got := matchedRouteString("GET /orders", h)
	if got != "GET /orders" {
		t.Errorf("matchedRouteString = %q, want %q", got, "GET /orders")
	}
}

func TestMatchedRouteString_SubstringFallback(t *testing.T) {
	h := &Node{Meta: map[string]string{"routes": "GET /orders"}}
	got := matchedRouteString("orders", h)
	if got != "GET /orders" {
		t.Errorf("matchedRouteString(substring) = %q, want %q", got, "GET /orders")
	}
}

func TestMatchedRouteString_EmptyRoutes(t *testing.T) {
	h := &Node{Meta: map[string]string{}}
	got := matchedRouteString("GET /x", h)
	// Falls back to the query itself
	if got != "GET /x" {
		t.Errorf("matchedRouteString(empty) = %q, want %q", got, "GET /x")
	}
}
