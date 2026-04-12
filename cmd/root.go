package cmd

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/zopdev/govis"
	"github.com/zopdev/govis/render"
)

func Execute() {
	format := flag.String("format", "mermaid", "Output format: mermaid, dot, html, json, svg")
	out := flag.String("out", "", "Output file (default: stdout)")
	filter := flag.String("filter", "", "Filter by package path")
	focus := flag.String("focus", "", "Focus strictly on a specific struct, interface, or service name")
	vet := flag.String("vet", "", "Lint architecture rules (e.g. 'handler!store' means handler cannot depend on store)")
	
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: govis [flags] [packages]\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	dir := "./..."
	if flag.NArg() > 0 {
		dir = flag.Arg(0)
	}

	opts := structmap.Options{
		Dir:    dir,
		Filter: *filter,
		Focus:  *focus,
	}

	graph, err := structmap.Parse(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing packages: %v\n", err)
		os.Exit(1)
	}

	// 🚨 Architecture Linter Feature
	if *vet != "" {
		parts := strings.Split(*vet, "!")
		if len(parts) == 2 {
			fromKind := structmap.NodeKind(parts[0])
			toKind := structmap.NodeKind(parts[1])
			
			violations := 0
			for _, edge := range graph.Edges {
				fromNode := graph.Nodes[edge.From]
				toNode := graph.Nodes[edge.To]
				if fromNode != nil && toNode != nil {
					if fromNode.Kind == fromKind && toNode.Kind == toKind {
						fmt.Fprintf(os.Stderr, "🚨 ARCHITECTURE VIOLATION: '%s' (%s) directly depends on '%s' (%s)!\n", fromNode.Name, fromNode.Kind, toNode.Name, toNode.Kind)
						violations++
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
		} else {
			fmt.Fprintf(os.Stderr, "Invalid vet rule format. Use 'from!to' (e.g., 'handler!store')\n")
			os.Exit(1)
		}
	}

	var r render.Renderer
	switch *format {
	case "json":
		r = &render.JSONRenderer{}
	case "html":
		r = &render.HTMLRenderer{}
	case "mermaid":
		r = &render.MermaidRenderer{}
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
