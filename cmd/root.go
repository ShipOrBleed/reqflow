package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	structmap "github.com/zopdev/govis"
	"github.com/zopdev/govis/render"
)

// Execute is the main CLI entrypoint for govis.
func Execute() {
	// Handle subcommands before flag parsing
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version":
			fmt.Printf("govis %s\n", structmap.Version)
			return
		case "init":
			generateInitConfig()
			return
		}
	}

	// ---- Flag Definitions ----
	format := flag.String("format", "mermaid", "Output format: mermaid, c4, dsm, dot, html, interactive, json, markdown, svg, excalidraw, pdf, embed, 3d, apimap, dataflow")
	out := flag.String("out", "", "Output file (default: stdout)")
	stitch := flag.String("stitch", "", "Stitch multiple JSON architecture exports (comma-separated files)")
	serve := flag.String("serve", "", "Start a live HTTP visualization server (e.g., ':8080')")
	filter := flag.String("filter", "", "Filter by package path")
	focus := flag.String("focus", "", "Focus on a specific component name")
	vet := flag.String("vet", "", "Lint rules (e.g. 'handler!store')")
	deadcode := flag.Bool("deadcode", false, "Detect orphaned/unused components")
	diff := flag.String("diff", "", "Diff against an older JSON architecture export")
	aiReview := flag.Bool("ai", false, "AI architecture review (requires OPENAI_API_KEY)")
	cycles := flag.Bool("cycles", false, "Detect circular dependencies")
	metrics := flag.Bool("metrics", false, "Print coupling metrics (fan-in/fan-out)")
	errCheck := flag.Bool("errcheck", false, "Detect swallowed error return values")
	security := flag.Bool("security", false, "Detect security anti-patterns")
	techDebt := flag.Bool("techdebt", false, "Scan TODO/FIXME/HACK comments")
	coverFile := flag.String("cover", "", "Path to Go coverage profile (cover.out)")
	constructors := flag.Bool("constructors", false, "Detect missing New*() constructors")
	fullAudit := flag.Bool("audit", false, "Run ALL analysis checks at once")
	apiMap := flag.Bool("apimap", false, "Generate full API surface map with request/response types")
	heatmap := flag.Bool("heatmap", false, "Overlay coverage data as heatmap colors on graph nodes")
	callGraph := flag.Bool("callgraph", false, "Visualize function-to-function call graph")
	dataFlow := flag.Bool("dataflow", false, "Visualize request data flow (handler→service→store)")
	envMap := flag.Bool("envmap", false, "Visualize environment variable usage across the codebase")
	tableMap := flag.Bool("tablemap", false, "Visualize model-to-database-table mappings")
	depTree := flag.Bool("deptree", false, "Visualize full go.mod transitive dependency tree")
	infraTopo := flag.Bool("infratopo", false, "Visualize Docker/K8s infrastructure topology")
	churn := flag.Bool("churn", false, "Overlay code churn heatmap (git commit frequency)")
	contributors := flag.Bool("contributors", false, "Show contributor map per component")
	prImpact := flag.String("pr-impact", "", "Visualize PR impact against a base ref (e.g. 'main')")
	evolution := flag.String("evolution", "", "Architecture evolution timeline across git tags (comma-separated)")
	proto := flag.Bool("proto", false, "Parse .proto files for gRPC service/RPC/message graph")
	serviceMap := flag.Bool("service-map", false, "Detect cross-service edges during stitch")
	otelTrace := flag.String("otel-trace", "", "Overlay OpenTelemetry trace data from OTLP JSON export")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: govis [flags] [packages]\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// ---- Target Directory ----
	dir := "./..."
	if flag.NArg() > 0 {
		dir = flag.Arg(0)
	}

	// ---- Configuration Loading ----
	var loadedConfig *structmap.GovisConfig
	if cfg, err := structmap.LoadConfig(".govis.yml"); err == nil {
		loadedConfig = cfg
		fmt.Fprintf(os.Stderr, "⚙️  Loaded configuration from .govis.yml\n")
	}

	opts := structmap.ParseOptions{
		Dir:    dir,
		Filter: *filter,
		Focus:  *focus,
		Config: loadedConfig,
		APIMap:    *apiMap,
		Heatmap:   *heatmap,
		CallGraph:  *callGraph,
		DataFlow:   *dataFlow,
		EnvMap:     *envMap,
		TableMap:   *tableMap,
		DepTree:    *depTree,
		InfraTopo:    *infraTopo,
		Churn:        *churn,
		Contributors: *contributors,
		PRImpact:     *prImpact,
		Evolution:    *evolution,
		Proto:        *proto,
		ServiceMap:   *serviceMap,
		OtelTrace:    *otelTrace,
	}

	// ---- Full Audit Mode ----
	if *fullAudit {
		*cycles = true
		*metrics = true
		*deadcode = true
		*errCheck = true
		*security = true
		*techDebt = true
		*constructors = true
	}

	// ---- Stitch Mode ----
	if *stitch != "" {
		handleStitch(*stitch, *format, *out, *serviceMap)
		return
	}

	// ---- Parse ----
	graph, err := structmap.Parse(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing packages: %v\n", err)
		os.Exit(1)
	}

	// ---- Always Print Summary ----
	structmap.PrintSummary(graph, os.Stderr)

	// ---- Architecture Linting ----
	var activeVetRules []string
	if *vet != "" {
		activeVetRules = strings.Split(*vet, ",")
	}
	if loadedConfig != nil && len(loadedConfig.Linter.VetRules) > 0 {
		activeVetRules = append(activeVetRules, loadedConfig.Linter.VetRules...)
	}
	if len(activeVetRules) > 0 {
		runVetRules(activeVetRules, graph)
	}

	// ---- Analysis Checks ----
	runAnalysis(graph, opts, analysisFlags{
		Deadcode:     *deadcode,
		Cycles:       *cycles,
		Metrics:      *metrics,
		ErrCheck:     *errCheck,
		Security:     *security,
		TechDebt:     *techDebt,
		CoverFile:    *coverFile,
		Constructors: *constructors,
		Diff:         *diff,
		AI:           *aiReview,
	})

	// ---- CI/CD Threshold Enforcement ----
	if *fullAudit && loadedConfig != nil {
		enforceThresholds(graph, loadedConfig)
	}

	// ---- Live Server Mode ----
	if *serve != "" {
		startServer(*serve, opts)
		return
	}

	// ---- Evolution Timeline ----
	if *evolution != "" {
		refs := strings.Split(*evolution, ",")
		snapshots := structmap.ExtractEvolution(dir, refs, opts)
		tr := &render.TimelineRenderer{Snapshots: snapshots}
		w := os.Stdout
		if *out != "" {
			f, err := os.Create(*out)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
				os.Exit(1)
			}
			defer f.Close()
			w = f
		}
		tr.Render(graph, w)
		return
	}

	// ---- Render Output ----
	var r render.Renderer
	switch *format {
	case "apimap":
		r = &render.APIMapRenderer{}
	case "dataflow":
		r = &render.DataFlowRenderer{}
	case "json":
		r = &render.JSONRenderer{}
	case "html":
		r = &render.HTMLRenderer{}
	case "interactive":
		r = &render.InteractiveRenderer{}
	case "mermaid":
		r = &render.MermaidRenderer{}
	case "markdown", "md":
		r = &render.MarkdownRenderer{}
	case "c4":
		r = &render.C4Renderer{}
	case "dsm":
		r = &render.DSMRenderer{}
	case "dot":
		r = &render.DOTRenderer{}
	case "excalidraw":
		r = &render.ExcalidrawRenderer{}
	case "pdf":
		r = &render.PDFRenderer{}
	case "embed":
		r = &render.EmbedRenderer{}
	case "3d":
		r = &render.ThreeRenderer{}
	case "svg":
		fmt.Fprintf(os.Stderr, "SVG: pipe dot into graphviz: govis -format dot | dot -Tsvg\n")
		r = &render.DOTRenderer{}
	default:
		fmt.Fprintf(os.Stderr, "Unknown format: %s\n", *format)
		os.Exit(1)
	}

	w := os.Stdout
	if *out != "" {
		f, err := os.Create(*out)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		w = f
	}

	if err := r.Render(graph, w); err != nil {
		fmt.Fprintf(os.Stderr, "Error rendering graph: %v\n", err)
		os.Exit(1)
	}
}

func handleStitch(filesStr, format, outPath string, withServiceMap bool) {
	files := strings.Split(filesStr, ",")
	var graphs []*structmap.Graph

	for _, f := range files {
		data, err := os.ReadFile(strings.TrimSpace(f))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stitch file %s: %v\n", f, err)
			os.Exit(1)
		}
		var g structmap.Graph
		if err := json.Unmarshal(data, &g); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing JSON from %s: %v\n", f, err)
			os.Exit(1)
		}
		graphs = append(graphs, &g)
	}

	var merged *structmap.Graph
	if withServiceMap {
		merged = structmap.StitchWithServiceMap(graphs)
	} else {
		merged = structmap.Stitch(graphs)
	}
	
	// Print summary of merged graph
	structmap.PrintSummary(merged, os.Stderr)

	var r render.Renderer
	switch format {
	case "json": r = &render.JSONRenderer{}
	case "html": r = &render.HTMLRenderer{}
	case "interactive": r = &render.InteractiveRenderer{}
	case "mermaid": r = &render.MermaidRenderer{}
	case "markdown", "md": r = &render.MarkdownRenderer{}
	case "c4": r = &render.C4Renderer{}
	case "dsm": r = &render.DSMRenderer{}
	case "dot": r = &render.DOTRenderer{}
	default:
		r = &render.MermaidRenderer{}
	}

	w := os.Stdout
	if outPath != "" {
		f, _ := os.Create(outPath)
		defer f.Close()
		w = f
	}
	r.Render(merged, w)
}

// runVetRules checks architecture rules and exits 1 on violations
func runVetRules(rules []string, graph *structmap.Graph) {
	violations := 0
	for _, rule := range rules {
		parts := strings.Split(rule, "!")
		if len(parts) != 2 {
			continue
		}

		isPkgRule := strings.HasPrefix(parts[0], "pkg:") && strings.HasPrefix(parts[1], "pkg:")
		
		if isPkgRule {
			fromPkg := strings.TrimPrefix(parts[0], "pkg:")
			toPkg := strings.TrimPrefix(parts[1], "pkg:")

			for _, edge := range graph.Edges {
				fromNode := graph.Nodes[edge.From]
				toNode := graph.Nodes[edge.To]
				if fromNode != nil && toNode != nil {
					if strings.Contains(fromNode.Package, fromPkg) && strings.Contains(toNode.Package, toPkg) {
						fmt.Fprintf(os.Stderr, "🚨 VIOLATION [%s!%s]: '%s' → '%s'\n", parts[0], parts[1], fromNode.ID, toNode.ID)
						violations++
					}
				}
			}
			continue
		}

		fromKind := structmap.NodeKind(parts[0])
		toKind := structmap.NodeKind(parts[1])

		for _, edge := range graph.Edges {
			fromNode := graph.Nodes[edge.From]
			toNode := graph.Nodes[edge.To]
			if fromNode != nil && toNode != nil {
				if fromNode.Kind == fromKind && toNode.Kind == toKind {
					fmt.Fprintf(os.Stderr, "🚨 VIOLATION [%s!%s]: '%s' → '%s'\n", parts[0], parts[1], fromNode.Name, toNode.Name)
					violations++
				}
			}
		}
	}

	if violations > 0 {
		fmt.Fprintf(os.Stderr, "\n❌ Vet failed: %d violations.\n", violations)
		os.Exit(1)
	} else if len(rules) > 0 {
		fmt.Fprintf(os.Stderr, "✅ Architecture vet passed.\n")
	}
}
