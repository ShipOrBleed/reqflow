package reqflow

import (
	"strings"
)

// TraceResult holds the complete request path for a single HTTP route.
type TraceResult struct {
	Route    string  // matched route, e.g. "POST /orders"
	Handler  *Node   // the handler node that owns this route
	Chain    []*Node // ordered nodes: handler → service → store → model
	Tables   []string // database tables touched by this path
	EnvVars  []string // environment variables read along this path
	NotFound bool     // true when no handler matched the given route
}

// layerRank assigns a sort priority to each node kind so the chain
// is always displayed in the natural request-processing order.
var layerRank = map[NodeKind]int{
	KindHandler:    0,
	KindMiddleware: 1,
	KindGRPC:       2,
	KindService:    3,
	KindInterface:  4,
	KindEvent:      5,
	KindStore:      6,
	KindInfra:      7,
	KindModel:      8,
	KindTable:      9,
}

// Trace finds the handler matching route and returns the complete
// static request path through the codebase — handler → service →
// store → tables, plus any environment variables read along the way.
//
// route may be:
//   - exact match: "POST /orders"
//   - path only: "/orders" (matches any HTTP method)
//   - partial: "orders" (substring match against all registered routes)
func Trace(route string, g *Graph) *TraceResult {
	handler := findHandlerForRoute(route, g)
	if handler == nil {
		return &TraceResult{Route: route, NotFound: true}
	}

	// Determine which specific route string was matched
	matchedRoute := matchedRouteString(route, handler)

	chain, tables, envVars := buildChain(handler.ID, g)

	return &TraceResult{
		Route:   matchedRoute,
		Handler: handler,
		Chain:   chain,
		Tables:  tables,
		EnvVars: envVars,
	}
}

// findHandlerForRoute finds the handler node whose registered routes
// match the given route query using exact → path → substring fallback.
func findHandlerForRoute(query string, g *Graph) *Node {
	query = strings.TrimSpace(query)
	queryLower := strings.ToLower(query)

	// Exact match first
	for _, n := range g.Nodes {
		if n.Kind != KindHandler {
			continue
		}
		for _, r := range nodeRoutes(n) {
			if strings.EqualFold(r, query) {
				return n
			}
		}
	}

	// Path-only or partial match
	for _, n := range g.Nodes {
		if n.Kind != KindHandler {
			continue
		}
		for _, r := range nodeRoutes(n) {
			rLower := strings.ToLower(r)
			// Strip method prefix for path-only queries
			parts := strings.SplitN(rLower, " ", 2)
			routePath := parts[len(parts)-1]
			if routePath == queryLower || strings.Contains(rLower, queryLower) {
				return n
			}
		}
	}

	return nil
}

// nodeRoutes returns the slice of routes registered on a handler node.
func nodeRoutes(n *Node) []string {
	raw := n.Meta["routes"]
	if raw == "" {
		raw = n.Meta["route"]
	}
	if raw == "" {
		return nil
	}
	var out []string
	for _, r := range strings.Split(raw, "\n") {
		if r = strings.TrimSpace(r); r != "" {
			out = append(out, r)
		}
	}
	return out
}

// matchedRouteString returns the specific route string from the handler
// that best matches the query.
func matchedRouteString(query string, handler *Node) string {
	routes := nodeRoutes(handler)
	if len(routes) == 0 {
		return query
	}
	queryLower := strings.ToLower(query)
	for _, r := range routes {
		if strings.EqualFold(r, query) || strings.Contains(strings.ToLower(r), queryLower) {
			return r
		}
	}
	return routes[0]
}

// buildChain performs a BFS from the given node ID, following dependency
// edges to collect all reachable nodes. It returns the chain sorted by
// architectural layer, plus tables and env vars found along the way.
func buildChain(startID string, g *Graph) (chain []*Node, tables []string, envVars []string) {
	visited := make(map[string]bool)
	queue := []string{startID}
	visited[startID] = true

	var nodes []*Node
	tableSet := make(map[string]bool)
	envSet := make(map[string]bool)

	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]

		node := g.Nodes[id]
		if node == nil {
			continue
		}

		switch node.Kind {
		case KindTable:
			if !tableSet[node.Name] {
				tableSet[node.Name] = true
				tables = append(tables, node.Name)
			}
		case KindEnvVar:
			if !envSet[node.Name] {
				envSet[node.Name] = true
				envVars = append(envVars, node.Name)
			}
		default:
			nodes = append(nodes, node)
		}

		// Follow outgoing edges
		for _, edge := range g.Edges {
			if edge.From == id && !visited[edge.To] {
				visited[edge.To] = true
				queue = append(queue, edge.To)
			}
		}
	}

	// Sort by architectural layer
	sortByLayer(nodes)

	// Deduplicate while preserving order
	seen := make(map[string]bool)
	for _, n := range nodes {
		if !seen[n.ID] {
			seen[n.ID] = true
			chain = append(chain, n)
		}
	}

	return chain, tables, envVars
}

func sortByLayer(nodes []*Node) {
	for i := 1; i < len(nodes); i++ {
		for j := i; j > 0; j-- {
			ri := rankOf(nodes[j])
			rj := rankOf(nodes[j-1])
			if ri < rj {
				nodes[j], nodes[j-1] = nodes[j-1], nodes[j]
			} else {
				break
			}
		}
	}
}

func rankOf(n *Node) int {
	if r, ok := layerRank[n.Kind]; ok {
		return r
	}
	return 99
}

// EdgeLabel returns a human-readable description of the transition
// between two adjacent steps in the trace chain.
func EdgeLabel(from, to *Node) string {
	switch {
	case from.Kind == KindHandler && to.Kind == KindService:
		return "delegates to"
	case from.Kind == KindHandler && to.Kind == KindStore:
		return "queries via"
	case from.Kind == KindService && to.Kind == KindStore:
		return "queries via"
	case from.Kind == KindService && to.Kind == KindInterface:
		return "uses interface"
	case from.Kind == KindStore && to.Kind == KindModel:
		return "maps to model"
	case from.Kind == KindStore && to.Kind == KindTable:
		return "writes to"
	case to.Kind == KindEvent:
		return "publishes event"
	case to.Kind == KindGRPC:
		return "calls gRPC"
	default:
		return "→"
	}
}
