package cmd

import (
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
	format := flag.String("format", "mermaid", "Output format: mermaid, dot, html, json, markdown, svg")
	out := flag.String("out", "", "Output file (default: stdout)")
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

	// ---- Render Output ----
	var r render.Renderer
	switch *format {
	case "json":
		r = &render.JSONRenderer{}
	case "html":
		r = &render.HTMLRenderer{}
	case "mermaid":
		r = &render.MermaidRenderer{}
	case "markdown", "md":
		r = &render.MarkdownRenderer{}
	case "dot":
		r = &render.DOTRenderer{}
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

// runVetRules checks architecture rules and exits 1 on violations
func runVetRules(rules []string, graph *structmap.Graph) {
	violations := 0
	for _, rule := range rules {
		parts := strings.Split(rule, "!")
		if len(parts) != 2 {
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
