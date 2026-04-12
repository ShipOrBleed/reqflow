package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/zopdev/govis"
	"github.com/zopdev/govis/render"
)

func Execute() {
	format := flag.String("format", "mermaid", "Output format: mermaid, dot, svg")
	out := flag.String("out", "", "Output file (default: stdout)")
	filter := flag.String("filter", "", "Filter by package path")
	
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
	}

	graph, err := structmap.Parse(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing packages: %v\n", err)
		os.Exit(1)
	}

	var r render.Renderer
	switch *format {
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
