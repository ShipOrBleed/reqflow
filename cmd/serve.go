package cmd

import (
	"fmt"
	"net/http"
	"os"

	structmap "github.com/zopdev/govis"
	"github.com/zopdev/govis/render"
)

// startServer launches the live HTTP visualization daemon
func startServer(addr string, opts structmap.ParseOptions) {
	fmt.Fprintf(os.Stderr, "\n🚀  Govis is LIVE! Watching codebase.\n    Open: http://localhost%s\n\n", addr)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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

	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Fprintf(os.Stderr, "Error binding server to %s: %v\n", addr, err)
		os.Exit(1)
	}
}
