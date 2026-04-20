package reqflow

import (
	"fmt"
	"sort"
	"strings"
)

// RouteInfo holds information about a single registered HTTP route,
// including the handler struct, method name, and source location.
type RouteInfo struct {
	Method      string // HTTP method: GET, POST, PUT, DELETE, PATCH
	Path        string // URL path, e.g. "/orgs/{orgID}/budgets"
	HandlerName string // Handler struct name, e.g. "OrderHandler"
	MethodName  string // Handler method name, e.g. "GetBudgets"
	File        string // Absolute path to the source file
	Line        int    // Line number where the handler method is defined
}

// ListRoutes extracts all registered HTTP routes from the graph.
// Routes are sorted by path, then by HTTP method.
//
// Example:
//
//	graph, _ := reqflow.Parse(reqflow.ParseOptions{Dir: "."})
//	routes := reqflow.ListRoutes(graph)
//	for _, r := range routes {
//	    fmt.Printf("%s %s → %s.%s()\n", r.Method, r.Path, r.HandlerName, r.MethodName)
//	}
func ListRoutes(g *Graph) []RouteInfo {
	if g == nil {
		return nil
	}

	var routes []RouteInfo

	for _, node := range g.Nodes {
		if node.Kind != KindHandler {
			continue
		}

		rawRoutes := node.Meta["routes"]
		if rawRoutes == "" {
			rawRoutes = node.Meta["route"]
		}
		if rawRoutes == "" {
			continue
		}

		for _, r := range strings.Split(rawRoutes, "\n") {
			r = strings.TrimSpace(r)
			if r == "" {
				continue
			}

			parts := strings.SplitN(r, " ", 2)
			method := ""
			path := r
			if len(parts) == 2 {
				method = parts[0]
				path = parts[1]
			}

			// Resolve handler method name for this route
			methodName := node.Meta["route_method:"+r]
			handlerName := node.Name

			// Get file and line from method metadata
			file := node.File
			line := node.Line
			if methodName != "" {
				if mf := node.Meta["method_file:"+methodName]; mf != "" {
					file = mf
				}
				if ml := node.Meta["method_line:"+methodName]; ml != "" {
					fmt.Sscanf(ml, "%d", &line)
				}
			}

			routes = append(routes, RouteInfo{
				Method:      method,
				Path:        path,
				HandlerName: handlerName,
				MethodName:  methodName,
				File:        file,
				Line:        line,
			})
		}
	}

	// Sort by path, then by method
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Path != routes[j].Path {
			return routes[i].Path < routes[j].Path
		}
		return routes[i].Method < routes[j].Method
	})

	return routes
}

// FormatRoutesText renders route info as an aligned text table suitable for terminal output.
// Includes a summary line showing total routes and handler count.
func FormatRoutesText(routes []RouteInfo) string {
	if len(routes) == 0 {
		return "No routes found.\n"
	}

	// Calculate column widths
	maxMethod := 0
	maxPath := 0
	maxHandler := 0
	for _, r := range routes {
		if len(r.Method) > maxMethod {
			maxMethod = len(r.Method)
		}
		if len(r.Path) > maxPath {
			maxPath = len(r.Path)
		}
		h := handlerDisplay(r)
		if len(h) > maxHandler {
			maxHandler = len(h)
		}
	}

	var sb strings.Builder
	sb.WriteString("\n")

	for _, r := range routes {
		h := handlerDisplay(r)
		loc := shortLocation(r.File, r.Line)
		fmt.Fprintf(&sb, "  %-*s  %-*s  %-*s  %s\n",
			maxMethod, r.Method,
			maxPath, r.Path,
			maxHandler, h,
			loc,
		)
	}

	// Summary
	handlers := make(map[string]bool)
	for _, r := range routes {
		handlers[r.HandlerName] = true
	}
	fmt.Fprintf(&sb, "\n%d routes across %d handlers\n", len(routes), len(handlers))

	return sb.String()
}

// FormatRoutesJSON renders route info as a JSON array of objects.
// Each object has fields: method, path, handler, method_name, file, line.
func FormatRoutesJSON(routes []RouteInfo) string {
	var sb strings.Builder
	sb.WriteString("[\n")
	for i, r := range routes {
		sb.WriteString(fmt.Sprintf(
			`  {"method":%q,"path":%q,"handler":%q,"method_name":%q,"file":%q,"line":%d}`,
			r.Method, r.Path, r.HandlerName, r.MethodName, r.File, r.Line,
		))
		if i < len(routes)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}
	sb.WriteString("]\n")
	return sb.String()
}

func handlerDisplay(r RouteInfo) string {
	if r.MethodName != "" {
		return r.HandlerName + "." + r.MethodName + "()"
	}
	return r.HandlerName + "()"
}

func shortLocation(file string, line int) string {
	if file == "" {
		return ""
	}
	parts := strings.Split(file, "/")
	if len(parts) > 2 {
		parts = parts[len(parts)-2:]
	}
	if line > 0 {
		return strings.Join(parts, "/") + fmt.Sprintf(":%d", line)
	}
	return strings.Join(parts, "/")
}
