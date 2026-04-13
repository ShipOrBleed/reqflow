package structmap

import (
	"fmt"
	"io"
	"strings"
)

// Version is set at build time via ldflags
var Version = "dev"

// PrintSummary outputs a quick human-readable stats overview of the graph.
func PrintSummary(g *Graph, w io.Writer) {
	counts := make(map[NodeKind]int)
	for _, n := range g.Nodes {
		counts[n.Kind]++
	}

	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "📊 Govis Analysis Summary\n")
	fmt.Fprintf(w, "   Packages: %d\n", len(g.Clusters))

	// Layer counts
	var parts []string
	if c := counts[KindHandler]; c > 0 {
		parts = append(parts, fmt.Sprintf("Handlers: %d", c))
	}
	if c := counts[KindGRPC]; c > 0 {
		parts = append(parts, fmt.Sprintf("gRPC: %d", c))
	}
	if c := counts[KindMiddleware]; c > 0 {
		parts = append(parts, fmt.Sprintf("Middleware: %d", c))
	}
	if c := counts[KindService]; c > 0 {
		parts = append(parts, fmt.Sprintf("Services: %d", c))
	}
	if c := counts[KindStore]; c > 0 {
		parts = append(parts, fmt.Sprintf("Stores: %d", c))
	}
	if c := counts[KindModel]; c > 0 {
		parts = append(parts, fmt.Sprintf("Models: %d", c))
	}
	if c := counts[KindEvent]; c > 0 {
		parts = append(parts, fmt.Sprintf("Events: %d", c))
	}
	if c := counts[KindInfra]; c > 0 {
		parts = append(parts, fmt.Sprintf("Infra: %d", c))
	}
	if c := counts[KindRoute]; c > 0 {
		parts = append(parts, fmt.Sprintf("Routes: %d", c))
	}
	if c := counts[KindEnvVar]; c > 0 {
		parts = append(parts, fmt.Sprintf("EnvVars: %d", c))
	}
	if c := counts[KindTable]; c > 0 {
		parts = append(parts, fmt.Sprintf("Tables: %d", c))
	}
	if c := counts[KindDep]; c > 0 {
		parts = append(parts, fmt.Sprintf("Deps: %d", c))
	}
	if c := counts[KindContainer]; c > 0 {
		parts = append(parts, fmt.Sprintf("Containers: %d", c))
	}
	if c := counts[KindStruct] + counts[KindInterface] + counts[KindFunc]; c > 0 {
		parts = append(parts, fmt.Sprintf("Other: %d", c))
	}
	if len(parts) > 0 {
		fmt.Fprintf(w, "   %s\n", strings.Join(parts, "  |  "))
	}

	// Routes
	var routes []string
	for _, n := range g.Nodes {
		if route, ok := n.Meta["route"]; ok {
			routes = append(routes, route)
		}
	}
	if len(routes) > 0 {
		display := routes
		if len(display) > 5 {
			display = display[:5]
			display = append(display, fmt.Sprintf("... +%d more", len(routes)-5))
		}
		fmt.Fprintf(w, "   Routes: %s\n", strings.Join(display, ", "))
	}

	// Quick health
	fmt.Fprintf(w, "   Nodes: %d  |  Edges: %d\n", len(g.Nodes), len(g.Edges))
	fmt.Fprintln(w, "")
}

// GetSummaryHTML returns the graph statistics formatted as HTML for use in dashboards.
func GetSummaryHTML(g *Graph) string {
	counts := make(map[NodeKind]int)
	for _, n := range g.Nodes {
		counts[n.Kind]++
	}

	var sb strings.Builder
	sb.WriteString("<div class='stats-grid'>")
	
	addStat := func(label string, val int, icon string) {
		if val > 0 {
			sb.WriteString(fmt.Sprintf("<div class='stat-card'><h3>%d</h3><p>%s %s</p></div>", val, icon, label))
		}
	}

	addStat("Packages", len(g.Clusters), "📦")
	addStat("Handlers", counts[KindHandler], "🌐")
	addStat("Services", counts[KindService], "⚙️")
	addStat("Stores", counts[KindStore], "🗄️")
	addStat("Models", counts[KindModel], "📄")
	addStat("Events", counts[KindEvent], "📢")
	addStat("Infra", counts[KindInfra], "☁️")
	addStat("EnvVars", counts[KindEnvVar], "🔑")
	addStat("Tables", counts[KindTable], "🗃️")
	addStat("Deps", counts[KindDep], "📎")
	addStat("Containers", counts[KindContainer], "🐳")
	
	sb.WriteString("</div>")
	return sb.String()
}

