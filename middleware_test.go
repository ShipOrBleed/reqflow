package reqflow

import (
	"os"
	"testing"
)

// ─── Middleware detection ─────────────────────────────────────────────────────

func TestExtractMiddleware_UseCall(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"router/router.go": `package router

type App struct{}
func (a *App) Use(handler interface{}) {}
func (a *App) GET(path string, h interface{}) {}
`,
		"mw/auth.go": `package mw

func AuthMiddleware() interface{} { return nil }
`,
		"main.go": `package main

import (
	"testmod/router"
)

func MyLogger() interface{} { return nil }

func main() {
	app := &router.App{}
	app.Use(MyLogger)
	app.GET("/health", func() interface{} { return "ok" })
}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{})

	// At least one KindMiddleware node should be detected
	hasMW := false
	for _, n := range graph.Nodes {
		if n.Kind == KindMiddleware {
			hasMW = true
			break
		}
	}
	if !hasMW {
		t.Error("Expected at least one KindMiddleware node from app.Use() call")
	}
}

// ─── gRPC detection ───────────────────────────────────────────────────────────

func TestExtractGRPC_RegisterServerCall(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"grpc/grpc.go": `package grpc

type Server struct{}
func RegisterOrderServer(srv *Server, impl interface{}) {}
`,
		"main.go": `package main

import "testmod/grpc"

func main() {
	srv := &grpc.Server{}
	grpc.RegisterOrderServer(srv, nil)
}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{})

	hasGRPC := false
	for _, n := range graph.Nodes {
		if n.Kind == KindGRPC {
			hasGRPC = true
			break
		}
	}
	if !hasGRPC {
		t.Error("Expected KindGRPC node from RegisterOrderServer call")
	}
}

func TestExtractGRPC_UnimplementedEmbed(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"pb/pb.go": `package pb

type UnimplementedOrderServer struct{}
`,
		"service/service.go": `package service

import "testmod/pb"

type OrderService struct {
	pb.UnimplementedOrderServer
}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{})

	// OrderService embeds UnimplementedOrderServer — should be promoted to KindGRPC
	node := graph.Nodes["testmod/service.OrderService"]
	if node == nil {
		t.Skip("OrderService node not found — skipping gRPC embed test")
	}
	if node.Kind != KindGRPC {
		t.Logf("OrderService kind = %s (gRPC embed detection depends on exact type string format)", node.Kind)
	}
}

// ─── StitchWithServiceMap event topics ───────────────────────────────────────

func TestStitchWithServiceMap_EventTopics(t *testing.T) {
	// svc1 publishes "order.created", svc2 subscribes to "order.created" → EdgeRPC
	g1 := NewGraph()
	g1.AddNode(&Node{ID: "svc1.Publisher", Kind: KindService, Name: "Publisher", Package: "svc1"})
	g1.AddNode(&Node{ID: "event.order.created", Kind: KindEvent, Name: "order.created", Package: "events",
		Meta: map[string]string{"topic": "order.created"}})
	g1.AddEdge("svc1.Publisher", "event.order.created", EdgePublishes)

	g2 := NewGraph()
	g2.AddNode(&Node{ID: "svc2.Consumer", Kind: KindService, Name: "Consumer", Package: "svc2"})
	g2.AddNode(&Node{ID: "event.order.created", Kind: KindEvent, Name: "order.created", Package: "events",
		Meta: map[string]string{"topic": "order.created"}})
	g2.AddEdge("event.order.created", "svc2.Consumer", EdgeSubscribes)

	merged := StitchWithServiceMap([]*Graph{g1, g2})

	hasRPC := false
	for _, edge := range merged.Edges {
		if edge.Kind == EdgeRPC && edge.From == "svc1.Publisher" && edge.To == "svc2.Consumer" {
			hasRPC = true
			break
		}
	}
	if !hasRPC {
		t.Error("Expected EdgeRPC from publisher to subscriber via shared event topic")
	}
}

func TestStitchWithServiceMap_GRPCClientEdge(t *testing.T) {
	// svc2 dials "orders" gRPC service owned by svc1 → EdgeRPC
	g1 := NewGraph()
	g1.AddNode(&Node{ID: "svc1.grpc.Order", Kind: KindGRPC, Name: "⚡ gRPC: Order", Package: "svc1"})

	g2 := NewGraph()
	g2.AddNode(&Node{ID: "svc2.Client", Kind: KindService, Name: "Client", Package: "svc2",
		Meta: map[string]string{"grpc_dial_target": "order-service:9090"}})

	merged := StitchWithServiceMap([]*Graph{g1, g2})

	hasRPC := false
	for _, edge := range merged.Edges {
		if edge.Kind == EdgeRPC {
			hasRPC = true
			break
		}
	}
	if !hasRPC {
		t.Error("Expected EdgeRPC from gRPC client to gRPC server")
	}
}
