package cmd

import (
	"flag"
	"fmt"
	"os"

	reqflow "github.com/thzgajendra/reqflow"
	"github.com/thzgajendra/reqflow/render"
)

// Execute is the main CLI entrypoint for reqflow.
func Execute() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version":
			fmt.Printf("reqflow %s\n", reqflow.Version)
			return
		case "trace":
			runTrace(os.Args[2:])
			return
		case "routes":
			runRoutes(os.Args[2:])
			return
		}
	}

	// Default: show usage
	fmt.Fprintf(os.Stderr, "Usage: reqflow <command> [flags] [packages]\n\n")
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  trace    Trace a request path through your Go codebase\n")
	fmt.Fprintf(os.Stderr, "  routes   List all registered routes in a service\n")
	fmt.Fprintf(os.Stderr, "  version  Show version\n")
	fmt.Fprintf(os.Stderr, "\nExamples:\n")
	fmt.Fprintf(os.Stderr, "  reqflow trace \"/orders\" ./...\n")
	fmt.Fprintf(os.Stderr, "  reqflow trace \"GET /orders\" ./...\n")
	fmt.Fprintf(os.Stderr, "  reqflow trace -format html -out trace.html \"POST /orders\" ./...\n")
	fmt.Fprintf(os.Stderr, "  reqflow routes ./...\n")
}

// runRoutes implements the `reqflow routes [dir]` subcommand.
func runRoutes(args []string) {
	fs := flag.NewFlagSet("routes", flag.ExitOnError)
	outPath := fs.String("out", "", "Output file (default: stdout)")
	format := fs.String("format", "text", "Output format: text, json")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: reqflow routes [flags] [packages]\n\n")
		fmt.Fprintf(os.Stderr, "List all registered routes in a service.\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  reqflow routes ./...\n")
		fmt.Fprintf(os.Stderr, "  reqflow routes -format json -out routes.json ./...\n\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	dir := "./..."
	if fs.NArg() >= 1 {
		dir = fs.Arg(0)
	}

	opts := reqflow.ParseOptions{Dir: dir}

	fmt.Fprintf(os.Stderr, "Analyzing %s...\n", dir)
	graph, err := reqflow.Parse(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse error: %v\n", err)
		os.Exit(1)
	}

	routes := reqflow.ListRoutes(graph)

	var output string
	switch *format {
	case "json":
		output = reqflow.FormatRoutesJSON(routes)
	default:
		output = reqflow.FormatRoutesText(routes)
	}

	w := os.Stdout
	if *outPath != "" {
		f, err := os.Create(*outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		w = f
	}

	fmt.Fprint(w, output)
}

// runTrace implements the `reqflow trace <route> [dir]` subcommand.
func runTrace(args []string) {
	fs := flag.NewFlagSet("trace", flag.ExitOnError)
	outPath := fs.String("out", "", "Output file (default: stdout)")
	format := fs.String("format", "text", "Output format: text, html")
	tableMap := fs.Bool("tablemap", false, "Resolve model → database table mappings")
	envMap := fs.Bool("envmap", false, "Resolve environment variable reads")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: reqflow trace [flags] <route> [packages]\n\n")
		fmt.Fprintf(os.Stderr, "Trace a request path through your Go codebase.\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  reqflow trace \"/orders\" ./...\n")
		fmt.Fprintf(os.Stderr, "  reqflow trace \"GET /orders\" ./...\n")
		fmt.Fprintf(os.Stderr, "  reqflow trace -format html -out trace.html \"POST /orders\" ./...\n\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Error: route argument required")
		fs.Usage()
		os.Exit(1)
	}

	route := fs.Arg(0)
	dir := "./..."
	if fs.NArg() >= 2 {
		dir = fs.Arg(1)
	}

	opts := reqflow.ParseOptions{
		Dir:      dir,
		TableMap: *tableMap,
		EnvMap:   *envMap,
	}

	fmt.Fprintf(os.Stderr, "Analyzing %s...\n", dir)
	graph, err := reqflow.Parse(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse error: %v\n", err)
		os.Exit(1)
	}

	result := reqflow.Trace(route, graph)

	// Interactive selection: if multiple routes match, let user pick
	if len(result.MultiMatch) > 0 {
		fmt.Fprintf(os.Stderr, "\nMultiple routes match %q:\n\n", route)
		for i, r := range result.MultiMatch {
			fmt.Fprintf(os.Stderr, "  %d.  %s\n", i+1, r)
		}
		fmt.Fprintf(os.Stderr, "\nEnter number (1-%d): ", len(result.MultiMatch))

		var choice int
		if _, err := fmt.Fscan(os.Stdin, &choice); err != nil || choice < 1 || choice > len(result.MultiMatch) {
			fmt.Fprintf(os.Stderr, "\nInvalid selection.\n")
			os.Exit(1)
		}

		// Re-trace with the selected route
		result = reqflow.Trace(result.MultiMatch[choice-1], graph)
	}

	w := os.Stdout
	if *outPath != "" {
		f, err := os.Create(*outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		w = f
	}

	renderer := &render.TraceRenderer{Format: *format}
	if err := renderer.RenderTrace(result, w); err != nil {
		fmt.Fprintf(os.Stderr, "Render error: %v\n", err)
		os.Exit(1)
	}
}
