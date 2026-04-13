package render

import (
	"fmt"
	"io"
	"sort"
	"strings"

	structmap "github.com/zopdev/govis"
)

// APIMapRenderer generates a focused table view of all API endpoints
// with their request/response types.
type APIMapRenderer struct{}

func (a *APIMapRenderer) Render(g *structmap.Graph, w io.Writer) error {
	type endpoint struct {
		Method   string
		Path     string
		Handler  string
		Request  string
		Response string
		File     string
		Line     int
	}

	var endpoints []endpoint

	for _, node := range g.Nodes {
		if node.Kind != structmap.KindRoute && node.Meta["route"] == "" {
			continue
		}

		route := node.Meta["route"]
		method, path := "", ""
		if route != "" {
			parts := strings.SplitN(route, " ", 2)
			if len(parts) == 2 {
				method, path = parts[0], parts[1]
			} else {
				path = route
			}
		}

		ep := endpoint{
			Method:   method,
			Path:     path,
			Handler:  node.Name,
			Request:  node.Meta["request_types"],
			Response: node.Meta["response_types"],
			File:     node.File,
			Line:     node.Line,
		}
		endpoints = append(endpoints, ep)
	}

	// Sort by method then path
	sort.Slice(endpoints, func(i, j int) bool {
		if endpoints[i].Method != endpoints[j].Method {
			return endpoints[i].Method < endpoints[j].Method
		}
		return endpoints[i].Path < endpoints[j].Path
	})

	if len(endpoints) == 0 {
		fmt.Fprintln(w, "No API endpoints detected.")
		return nil
	}

	// Markdown table output
	fmt.Fprintln(w, "# API Surface Map")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "| Method | Path | Handler | Request | Response |")
	fmt.Fprintln(w, "|--------|------|---------|---------|----------|")

	for _, ep := range endpoints {
		req := ep.Request
		if req == "" {
			req = "—"
		}
		resp := ep.Response
		if resp == "" {
			resp = "—"
		}
		fmt.Fprintf(w, "| `%s` | `%s` | `%s` | `%s` | `%s` |\n",
			ep.Method, ep.Path, ep.Handler, req, resp)
	}

	fmt.Fprintf(w, "\n**Total endpoints: %d**\n", len(endpoints))
	return nil
}
