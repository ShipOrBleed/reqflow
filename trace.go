package reqflow

import (
	"strings"
)

// TraceResult holds the complete request path for a single HTTP route.
//
// When a route is found, Chain contains the ordered list of nodes from
// handler through service to store, CalledMethods maps each node ID to
// the specific methods called on it, and MethodCalls provides the full
// call index for rendering sub-calls.
//
// When multiple routes match a path-only query (e.g. "/orders" matches
// both GET and POST), MultiMatch is populated instead of Chain, allowing
// the caller to present a selection to the user.
type TraceResult struct {
	// Route is the matched route string, e.g. "POST /orders".
	Route string

	// Handler is the handler node that owns this route.
	Handler *Node

	// Chain is the ordered list of nodes in the request path:
	// handler → service → store → model, sorted by architectural layer.
	Chain []*Node

	// Tables lists database table names touched by this request path.
	// Populated when -tablemap flag is used.
	Tables []string

	// EnvVars lists environment variable names read along this request path.
	// Populated when -envmap flag is used.
	EnvVars []string

	// NotFound is true when no handler matched the given route query.
	NotFound bool

	// CalledMethods maps node ID → list of specific methods called on that node
	// in this request path. Only methods actually invoked are included, not all
	// methods on the struct.
	CalledMethods map[string][]string

	// MethodCalls is the full method-level call index, used by renderers
	// to display sub-calls (e.g. → svc.GetMetrics()).
	MethodCalls MethodCallIndex

	// MultiMatch is populated when a path-only query matches multiple HTTP
	// methods (e.g. "/orders" → GET /orders, POST /orders). The caller should
	// present these options for the user to select one.
	MultiMatch []string
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
	KindClient:     7,
	KindInfra:      8,
	KindModel:      9,
	KindTable:      10,
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
	route = strings.TrimSpace(route)
	if route == "" {
		return &TraceResult{Route: route, NotFound: true}
	}

	// Check if the query has an HTTP method prefix
	hasMethod := false
	upper := strings.ToUpper(route)
	for _, m := range []string{"GET ", "POST ", "PUT ", "DELETE ", "PATCH "} {
		if strings.HasPrefix(upper, m) {
			hasMethod = true
			break
		}
	}

	// If no method, find all matching routes and let user pick
	if !hasMethod {
		matches := findAllMatchingRoutes(route, g)
		if len(matches) == 0 {
			return &TraceResult{Route: route, NotFound: true}
		}
		if len(matches) > 1 {
			return &TraceResult{Route: route, MultiMatch: matches}
		}
		// Exactly one match — use it
		route = matches[0]
	}

	handler := findHandlerForRoute(route, g)
	if handler == nil {
		return &TraceResult{Route: route, NotFound: true}
	}

	// Determine which specific route string was matched
	matchedRoute := matchedRouteString(route, handler)

	// Try method-level precise chain first (only nodes actually called)
	calledMethods, preciseChain, tables, envVars := buildPreciseChain(g, handler, matchedRoute)

	if len(preciseChain) > 1 {
		// Method-level resolution worked — use the precise chain
		return &TraceResult{
			Route:         matchedRoute,
			Handler:       handler,
			Chain:         preciseChain,
			Tables:        tables,
			EnvVars:       envVars,
			CalledMethods: calledMethods,
			MethodCalls:   g.MethodCalls,
		}
	}

	// Fallback to struct-level BFS when method calls can't be resolved
	chain, tables, envVars := buildChain(handler.ID, g)
	calledMethods = resolveTraceMethodCalls(g, handler, matchedRoute, chain)

	return &TraceResult{
		Route:         matchedRoute,
		Handler:       handler,
		Chain:         chain,
		Tables:        tables,
		EnvVars:       envVars,
		MethodCalls:   g.MethodCalls,
		CalledMethods: calledMethods,
	}
}

// buildPreciseChain builds the chain by following actual method calls instead of
// struct-level dependencies. This ensures only the nodes touched by this specific
// request are included — not every field on the struct.
func buildPreciseChain(g *Graph, handler *Node, route string) (calledMethods map[string][]string, chain []*Node, tables []string, envVars []string) {
	calledMethods = make(map[string][]string)

	if g.MethodCalls == nil {
		return calledMethods, nil, nil, nil
	}

	handlerMethod := handler.Meta["route_method:"+route]
	if handlerMethod == "" {
		return calledMethods, nil, nil, nil
	}

	// BFS through method calls to find all reachable nodes
	type pending struct {
		structID string
		method   string
	}

	visited := make(map[string]bool)
	nodeSet := make(map[string]bool)
	tableSet := make(map[string]bool)
	envSet := make(map[string]bool)

	// Start with handler
	nodeSet[handler.ID] = true
	queue := []pending{{handler.ID, handlerMethod}}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		key := cur.structID + "." + cur.method
		if visited[key] {
			continue
		}
		visited[key] = true

		calls := g.MethodCalls[key]
		sourceNode := g.Nodes[cur.structID]
		if sourceNode == nil {
			continue
		}

		for _, call := range calls {
			targetNodeID := resolveFieldTarget(sourceNode, call.FieldName, g)
			if targetNodeID == "" {
				continue
			}

			targetNode := g.Nodes[targetNodeID]
			if targetNode == nil {
				continue
			}

			// Classify the target
			switch targetNode.Kind {
			case KindTable:
				if !tableSet[targetNode.Name] {
					tableSet[targetNode.Name] = true
					tables = append(tables, targetNode.Name)
				}
				continue
			case KindEnvVar:
				if !envSet[targetNode.Name] {
					envSet[targetNode.Name] = true
					envVars = append(envVars, targetNode.Name)
				}
				continue
			}

			nodeSet[targetNodeID] = true
			addCalledMethod(calledMethods, targetNodeID, call.TargetMethod)
			queue = append(queue, pending{targetNodeID, call.TargetMethod})

			// If target is an interface, also follow to concrete implementations
			if targetNode.Kind == KindInterface || targetNode.Kind == KindService {
				for _, edge := range g.Edges {
					if edge.To == targetNodeID && edge.Kind == EdgeImplements {
						implID := edge.From
						nodeSet[implID] = true
						addCalledMethod(calledMethods, implID, call.TargetMethod)
						queue = append(queue, pending{implID, call.TargetMethod})
					}
				}
			}
		}
	}

	// Remove interface nodes when their concrete implementation is also in the chain.
	// e.g., if both Service (interface) and service (struct) are present with the same
	// called methods, keep only the concrete struct — showing both is redundant.
	skipSet := make(map[string]bool)
	for id := range nodeSet {
		n := g.Nodes[id]
		if n == nil {
			continue
		}
		// If this is an interface/service-interface, check if a concrete impl is also in chain
		if n.Kind == KindInterface || (n.Kind == KindService && isInterfaceNode(n, g)) {
			for _, edge := range g.Edges {
				if edge.To == id && edge.Kind == EdgeImplements && nodeSet[edge.From] {
					// Concrete impl exists in chain — skip the interface
					skipSet[id] = true
					break
				}
			}
		}
	}

	// Build chain from nodeSet, sorted by layer
	var nodes []*Node
	for id := range nodeSet {
		if skipSet[id] {
			continue
		}
		if n := g.Nodes[id]; n != nil {
			nodes = append(nodes, n)
		}
	}
	sortByLayer(nodes)

	// Deduplicate
	seen := make(map[string]bool)
	for _, n := range nodes {
		if !seen[n.ID] {
			seen[n.ID] = true
			chain = append(chain, n)
		}
	}

	return calledMethods, chain, tables, envVars
}

// resolveTraceMethodCalls walks the chain and determines which specific methods
// are called at each layer. Starting from the handler method, it follows the
// call index to find what service methods are called, then what store methods, etc.
func resolveTraceMethodCalls(g *Graph, handler *Node, route string, chain []*Node) map[string][]string {
	result := make(map[string][]string)
	if g.MethodCalls == nil {
		return result
	}

	// Start with the handler method for this route
	handlerMethod := handler.Meta["route_method:"+route]
	if handlerMethod == "" {
		return result
	}

	// BFS through the chain, tracking which methods to look for at each level
	type pending struct {
		structID string
		method   string
	}
	queue := []pending{{handler.ID, handlerMethod}}
	visited := make(map[string]bool)

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		key := cur.structID + "." + cur.method
		if visited[key] {
			continue
		}
		visited[key] = true

		calls := g.MethodCalls[key]
		if len(calls) == 0 {
			continue
		}

		// For each call, find which chain node it targets
		sourceNode := g.Nodes[cur.structID]
		if sourceNode == nil {
			continue
		}

		for _, call := range calls {
			// Find the target node: match the field name to a field on the source struct
			// then find the chain node whose type matches that field
			targetNodeID := resolveFieldTarget(sourceNode, call.FieldName, g)
			if targetNodeID == "" {
				continue
			}

			addCalledMethod(result, targetNodeID, call.TargetMethod)

			// Continue tracing on the target
			queue = append(queue, pending{targetNodeID, call.TargetMethod})

			// If target is an interface, also trace into concrete implementations
			targetNode := g.Nodes[targetNodeID]
			if targetNode != nil && (targetNode.Kind == KindInterface || targetNode.Kind == KindService) {
				for _, edge := range g.Edges {
					if edge.To == targetNodeID && edge.Kind == EdgeImplements {
						implID := edge.From
						addCalledMethod(result, implID, call.TargetMethod)
						queue = append(queue, pending{implID, call.TargetMethod})
					}
				}
			}
		}
	}

	return result
}

// isInterfaceNode checks if a service-kind node is actually an interface definition
// (has concrete implementations pointing to it via EdgeImplements).
func isInterfaceNode(n *Node, g *Graph) bool {
	for _, edge := range g.Edges {
		if edge.To == n.ID && edge.Kind == EdgeImplements {
			return true
		}
	}
	return false
}

func addCalledMethod(result map[string][]string, nodeID, method string) {
	for _, m := range result[nodeID] {
		if m == method {
			return
		}
	}
	result[nodeID] = append(result[nodeID], method)
}

// resolveFieldTarget finds the graph node ID that a struct field points to.
// Given a source node and a field name, it looks up the field's type in the graph.
func resolveFieldTarget(source *Node, fieldName string, g *Graph) string {
	for _, f := range source.Fields {
		if strings.EqualFold(f.Name, fieldName) {
			cleanType := strings.TrimLeft(f.Type, "*")
			if _, exists := g.Nodes[cleanType]; exists {
				return cleanType
			}
		}
	}

	// Fallback: check outgoing edges and match by node name similarity
	for _, edge := range g.Edges {
		if edge.From == source.ID && (edge.Kind == EdgeDepends || edge.Kind == EdgeImplements) {
			targetNode := g.Nodes[edge.To]
			if targetNode != nil && strings.EqualFold(targetNode.Name, fieldName) {
				return edge.To
			}
		}
	}

	return ""
}

// findAllMatchingRoutes returns all registered routes whose path matches the query.
// Used when the user provides a path without an HTTP method (e.g. "/orgs/{orgID}/budgets").
func findAllMatchingRoutes(query string, g *Graph) []string {
	queryLower := strings.ToLower(strings.TrimSpace(query))
	var exact, prefix []string
	seen := make(map[string]bool)

	for _, n := range g.Nodes {
		if n.Kind != KindHandler {
			continue
		}
		for _, r := range nodeRoutes(n) {
			if seen[r] {
				continue
			}
			rLower := strings.ToLower(r)
			parts := strings.SplitN(rLower, " ", 2)
			routePath := parts[len(parts)-1]

			if routePath == queryLower {
				// Exact path match (e.g. query="/budgets", route="GET /budgets")
				seen[r] = true
				exact = append(exact, r)
			} else if strings.Contains(routePath, queryLower) {
				seen[r] = true
				prefix = append(prefix, r)
			}
		}
	}

	if len(exact) > 0 {
		return exact
	}
	return prefix
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

	// Path-only match: query is just the path, match against any method
	for _, n := range g.Nodes {
		if n.Kind != KindHandler {
			continue
		}
		for _, r := range nodeRoutes(n) {
			rLower := strings.ToLower(r)
			parts := strings.SplitN(rLower, " ", 2)
			routePath := parts[len(parts)-1]
			if routePath == queryLower {
				return n
			}
		}
	}

	// Substring match: find the shortest matching route (most specific)
	var bestNode *Node
	bestLen := int(^uint(0) >> 1) // max int
	for _, n := range g.Nodes {
		if n.Kind != KindHandler {
			continue
		}
		for _, r := range nodeRoutes(n) {
			rLower := strings.ToLower(r)
			if strings.Contains(rLower, queryLower) && len(r) < bestLen {
				bestNode = n
				bestLen = len(r)
			}
		}
	}

	return bestNode
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
	// Prefer exact match
	for _, r := range routes {
		if strings.EqualFold(r, query) {
			return r
		}
	}
	// Then shortest substring match
	best := ""
	for _, r := range routes {
		if strings.Contains(strings.ToLower(r), queryLower) {
			if best == "" || len(r) < len(best) {
				best = r
			}
		}
	}
	if best != "" {
		return best
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

		// Follow outgoing edges (depends, calls, flows, mapsTo)
		for _, edge := range g.Edges {
			if edge.From == id && !visited[edge.To] {
				visited[edge.To] = true
				queue = append(queue, edge.To)
			}
		}

		// If this is an interface, also follow to its concrete implementations.
		// Pattern: Handler depends on Service (interface) → service struct implements Service.
		// Without this, the trace stops at the interface and never reaches the store layer.
		if node.Kind == KindInterface || node.Kind == KindService {
			for _, edge := range g.Edges {
				if edge.To == id && edge.Kind == EdgeImplements && !visited[edge.From] {
					visited[edge.From] = true
					queue = append(queue, edge.From)
				}
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
	case to.Kind == KindClient:
		return "calls client"
	case from.Kind == KindClient:
		return "calls external"
	default:
		return "→"
	}
}
