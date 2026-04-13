package govis

import (
	"fmt"
	"strings"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/cha"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// ExtractCallGraph builds an SSA program from the loaded packages, runs
// Class Hierarchy Analysis (CHA) to construct the call graph, and adds
// EdgeCalls edges between function/method nodes in the graph.
func ExtractCallGraph(pkgs []*packages.Package, graph *Graph, modulePath string) {
	// Build SSA program
	prog, ssaPkgs := ssautil.AllPackages(pkgs, ssa.InstantiateGenerics)
	prog.Build()

	// Filter to only our module packages
	var relevantPkgs []*ssa.Package
	for _, p := range ssaPkgs {
		if p != nil && strings.HasPrefix(p.Pkg.Path(), modulePath) {
			relevantPkgs = append(relevantPkgs, p)
		}
	}

	if len(relevantPkgs) == 0 {
		return
	}

	// Run CHA
	cg := cha.CallGraph(prog)
	if cg == nil {
		return
	}

	// Walk call graph edges and add EdgeCalls for internal calls
	callgraph.GraphVisitEdges(cg, func(edge *callgraph.Edge) error {
		caller := edge.Caller.Func
		callee := edge.Callee.Func

		if caller == nil || callee == nil {
			return nil
		}

		callerPkg := caller.Package()
		calleePkg := callee.Package()
		if callerPkg == nil || calleePkg == nil {
			return nil
		}

		// Only include edges within our module
		if !strings.HasPrefix(callerPkg.Pkg.Path(), modulePath) ||
			!strings.HasPrefix(calleePkg.Pkg.Path(), modulePath) {
			return nil
		}

		// Map SSA functions to graph nodes
		callerID := resolveNodeID(caller, graph)
		calleeID := resolveNodeID(callee, graph)

		if callerID != "" && calleeID != "" && callerID != calleeID {
			graph.AddEdge(callerID, calleeID, EdgeCalls)
		}

		return nil
	})

	// Tag nodes with call counts
	inCount := make(map[string]int)
	outCount := make(map[string]int)
	for _, edge := range graph.Edges {
		if edge.Kind == EdgeCalls {
			outCount[edge.From]++
			inCount[edge.To]++
		}
	}
	for id, node := range graph.Nodes {
		if c := inCount[id]; c > 0 {
			node.Meta["callgraph_in"] = fmt.Sprintf("%d", c)
		}
		if c := outCount[id]; c > 0 {
			node.Meta["callgraph_out"] = fmt.Sprintf("%d", c)
		}
	}
}

// resolveNodeID maps an SSA function to the best matching graph node ID.
// For methods, it maps to the receiver type. For package-level functions,
// it maps to the function node.
func resolveNodeID(fn *ssa.Function, graph *Graph) string {
	if fn.Signature.Recv() != nil {
		// Method — map to receiver type
		recv := fn.Signature.Recv().Type().String()
		recv = strings.TrimLeft(recv, "*")
		if _, exists := graph.Nodes[recv]; exists {
			return recv
		}
	}

	// Package-level function
	if fn.Package() != nil {
		funcID := fmt.Sprintf("%s.%s", fn.Package().Pkg.Path(), fn.Name())
		if _, exists := graph.Nodes[funcID]; exists {
			return funcID
		}
	}

	return ""
}
