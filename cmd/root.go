package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/zopdev/govis"
	"github.com/zopdev/govis/render"
	"golang.org/x/tools/go/packages"
)

func Execute() {
	format := flag.String("format", "mermaid", "Output format: mermaid, dot, html, json, svg")
	out := flag.String("out", "", "Output file (default: stdout)")
	serve := flag.String("serve", "", "Start a live HTTP visualization server on this port (e.g., ':8080')")
	filter := flag.String("filter", "", "Filter by package path")
	focus := flag.String("focus", "", "Focus strictly on a specific struct, interface, or service name")
	vet := flag.String("vet", "", "Lint architecture rules (e.g. 'handler!store' means handler cannot depend on store)")
	deadcode := flag.Bool("deadcode", false, "Detect and print potentially dead/unused architectural nodes (orphans)")
	diff := flag.String("diff", "", "Compare current code against an older JSON architecture export (filepath)")
	aiReview := flag.Bool("ai", false, "Analyze architecture using AI (Requires OPENAI_API_KEY env var)")
	cycles := flag.Bool("cycles", false, "Detect circular dependencies in the architecture")
	metrics := flag.Bool("metrics", false, "Print coupling metrics (fan-in/fan-out) for all components")
	errCheck := flag.Bool("errcheck", false, "Detect swallowed/ignored error return values")
	
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: govis [flags] [packages]\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	dir := "./..."
	if flag.NArg() > 0 {
		dir = flag.Arg(0)
	}

	// 💼 Load Enterprise .govis.yml
	var loadedConfig *structmap.GovisConfig
	if cfg, err := structmap.LoadConfig(".govis.yml"); err == nil {
		loadedConfig = cfg
		fmt.Fprintf(os.Stderr, "⚙️  Loaded enterprise configuration from .govis.yml\n")
	}

	opts := structmap.ParseOptions{
		Dir:    dir,
		Filter: *filter,
		Focus:  *focus,
		Config: loadedConfig,
	}

	graph, err := structmap.Parse(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing packages: %v\n", err)
		os.Exit(1)
	}

	// Aggregate Rules
	var activeVetRules []string
	if *vet != "" {
		activeVetRules = strings.Split(*vet, ",")
	}
	if loadedConfig != nil && len(loadedConfig.Linter.VetRules) > 0 {
		activeVetRules = append(activeVetRules, loadedConfig.Linter.VetRules...)
	}

	// 🚨 Architecture Linter Feature
	if len(activeVetRules) > 0 {
		violations := 0
		for _, rule := range activeVetRules {
			parts := strings.Split(rule, "!")
			if len(parts) == 2 {
				fromKind := structmap.NodeKind(parts[0])
				toKind := structmap.NodeKind(parts[1])
				
				for _, edge := range graph.Edges {
					fromNode := graph.Nodes[edge.From]
					toNode := graph.Nodes[edge.To]
					if fromNode != nil && toNode != nil {
						if fromNode.Kind == fromKind && toNode.Kind == toKind {
							fmt.Fprintf(os.Stderr, "🚨 RULE VIOLATION [%s!%s]: '%s' directly depends on '%s'!\n", string(fromKind), string(toKind), fromNode.Name, toNode.Name)
							violations++
						}
					}
				}
			}
		}
		
		if violations > 0 {
			fmt.Fprintf(os.Stderr, "\n❌ Architecture vet failed! Found %d forbidden dependency violations.\n", violations)
			os.Exit(1)
		} else {
			fmt.Fprintf(os.Stderr, "✅ Architecture vet passed perfectly! No forbidden dependencies.\n")
			// If no file processing is explicitly wanted beyond vetting, return
			if *format == "" {
				return
			}
		}
	}

	// 🚨 V2 Feature: Dead Code Detection
	if *deadcode {
		hasIncoming := make(map[string]bool)
		for _, e := range graph.Edges {
			hasIncoming[e.To] = true
		}
		
		fmt.Fprintf(os.Stderr, "\n💀 DEAD CODE / ORPHAN DETECTION:\n")
		orphansFound := 0
		for id, n := range graph.Nodes {
			if n.Kind != structmap.KindHandler && n.Kind != structmap.KindFunc && n.Kind != structmap.KindEvent {
				if !hasIncoming[id] {
					fmt.Fprintf(os.Stderr, "  - Orphaned %s: %s (Location: %s:%d)\n", n.Kind, n.Name, n.File, n.Line)
					orphansFound++
				}
			}
		}
		if orphansFound == 0 {
			fmt.Fprintf(os.Stderr, "  ✅ No orphaned components found!\n")
		} else {
			fmt.Fprintf(os.Stderr, "  Total orphans found: %d\n", orphansFound)
		}
	}

	// 🔄 V3: Circular Dependency Detection
	if *cycles {
		detectedCycles := structmap.DetectCycles(graph)
		fmt.Fprintf(os.Stderr, "\n🔄 CIRCULAR DEPENDENCY SCAN:\n")
		if len(detectedCycles) == 0 {
			fmt.Fprintf(os.Stderr, "  ✅ No circular dependencies found!\n")
		} else {
			for i, cycle := range detectedCycles {
				var names []string
				for _, id := range cycle {
					if n, ok := graph.Nodes[id]; ok {
						names = append(names, n.Name)
					} else {
						names = append(names, id)
					}
				}
				fmt.Fprintf(os.Stderr, "  ⚠️  Cycle %d: %s\n", i+1, strings.Join(names, " → "))
			}
			fmt.Fprintf(os.Stderr, "  \n  Total cycles: %d\n", len(detectedCycles))
		}
	}

	// 📊 V3: Coupling Metrics
	if *metrics {
		allMetrics := structmap.ComputeMetrics(graph)
		fmt.Fprintf(os.Stderr, "\n📊 COUPLING METRICS:\n")
		fmt.Fprintf(os.Stderr, "  %-30s %-12s %-8s %-8s %-8s\n", "COMPONENT", "KIND", "FAN-IN", "FAN-OUT", "RISK")
		fmt.Fprintf(os.Stderr, "  %s\n", strings.Repeat("-", 75))
		for _, m := range allMetrics {
			risk := "✅ Low"
			total := m.FanIn + m.FanOut
			if total > 10 {
				risk = "🔴 GOD OBJECT"
			} else if total > 5 {
				risk = "🟡 Watch"
			}
			fmt.Fprintf(os.Stderr, "  %-30s %-12s %-8d %-8d %s\n", m.Name, m.Kind, m.FanIn, m.FanOut, risk)
		}
	}

	// ⚠️ V3: Swallowed Error Detection
	if *errCheck {
		// Re-load packages for deeper inspection
		pkgCfg := &packages.Config{
			Mode: packages.NeedName | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo,
			Dir:  opts.Dir,
		}
		pkgs, err := packages.Load(pkgCfg, "./...")
		if err == nil {
			errors := structmap.DetectSwallowedErrors(pkgs)
			fmt.Fprintf(os.Stderr, "\n⚠️  SWALLOWED ERROR DETECTION:\n")
			if len(errors) == 0 {
				fmt.Fprintf(os.Stderr, "  ✅ No swallowed errors found!\n")
			} else {
				for _, e := range errors {
					fmt.Fprintf(os.Stderr, "  - %s:%d in %s() — ignoring error from %s()\n", e.File, e.Line, e.FuncName, e.CallExpr)
				}
				fmt.Fprintf(os.Stderr, "  Total swallowed errors: %d\n", len(errors))
			}
		}
	}

	// 🚨 V2 Feature: Architecture Diffing
	if *diff != "" {
		bytes, err := os.ReadFile(*diff)
		if err == nil {
			var oldGraph structmap.Graph
			if err := json.Unmarshal(bytes, &oldGraph); err == nil {
				// Identify new nodes
				for id, n := range graph.Nodes {
					if _, ok := oldGraph.Nodes[id]; !ok {
						n.Meta["diff"] = "new"
					}
				}
				// Identify deleted nodes
				for id, oldN := range oldGraph.Nodes {
					if _, ok := graph.Nodes[id]; !ok {
						if oldN.Meta == nil {
							oldN.Meta = make(map[string]string)
						}
						oldN.Meta["diff"] = "deleted"
						graph.AddNode(oldN) // Add purely for visualization
					}
				}
				fmt.Fprintf(os.Stderr, "✅ Injected architecture diff comparisons against %s.\n", *diff)
			}
		}
	}

	// 🚨 V2 Feature: Native AI Code Reviewer
	if *aiReview {
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			fmt.Fprintln(os.Stderr, "❌ Please set OPENAI_API_KEY environment variable to use -ai.")
		} else {
			fmt.Fprintf(os.Stderr, "🤖 Analyzing Architecture with AI...\n")
			graphJSON, _ := json.Marshal(graph)
			prompt := "You are a Senior Go Software Architect. Review the following JSON architecture graph of a codebase. Identify any tight coupling, missing layers (like a handler calling a database store), and structural flaws. Provide 3 direct bullet points of specific code review advice. JSON Data:\n" + string(graphJSON)
			reqBody := fmt.Sprintf(`{"model": "gpt-4o", "messages": [{"role": "user", "content": %q}], "max_tokens": 500}`, prompt)
			req, _ := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", strings.NewReader(reqBody))
			req.Header.Set("Authorization", "Bearer "+apiKey)
			req.Header.Set("Content-Type", "application/json")
			client := &http.Client{}
			resp, err := client.Do(req)
			if err == nil {
				defer resp.Body.Close()
				var resStruct struct {
					Choices []struct {
						Message struct {
							Content string `json:"content"`
						} `json:"message"`
					} `json:"choices"`
				}
				json.NewDecoder(resp.Body).Decode(&resStruct)
				if len(resStruct.Choices) > 0 {
					fmt.Printf("\n--- 🤖 AI ARCHITECT FEEDBACK ---\n%s\n--------------------------------\n", resStruct.Choices[0].Message.Content)
				}
			}
		}
	}

	// 🚨 V2 Feature: Live Web Server Daemon
	if *serve != "" {
		fmt.Fprintf(os.Stderr, "\n🚀  Govis is LIVE! Watching codebase.\n    Open: http://localhost%s\n\n", *serve)
		
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Re-parse the AST continuously on every browser refresh!
			liveGraph, err := structmap.Parse(opts)
			if err != nil {
				fmt.Fprintf(w, "<html><body><h1>AST Parsing Error: %v</h1></body></html>", err)
				return
			}
			
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			renderer := &render.HTMLRenderer{}
			if err := renderer.Render(liveGraph, w); err != nil {
				fmt.Fprintf(w, "Internal rendering error: %v", err)
			}
		})
		
		if err := http.ListenAndServe(*serve, nil); err != nil {
			fmt.Fprintf(os.Stderr, "Error automatically binding server to %s: %v\n", *serve, err)
			os.Exit(1)
		}
		return
	}

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
		fmt.Fprintf(os.Stderr, "SVG rendering is not natively supported yet, pipe dot format into graphviz: govis -format dot | dot -Tsvg\n")
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
