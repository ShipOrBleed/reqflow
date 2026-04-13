package render

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"

	govis "github.com/zopdev/govis"
)

// PDFRenderer generates a PDF by first rendering to DOT format and then
// invoking Graphviz's `dot` command to produce PDF output.
// Falls back to DOT output with instructions if Graphviz is not installed.
type PDFRenderer struct{}

func (p *PDFRenderer) Render(g *govis.Graph, w io.Writer) error {
	// First render to DOT format
	var dotBuf bytes.Buffer
	dot := &DOTRenderer{}
	if err := dot.Render(g, &dotBuf); err != nil {
		return fmt.Errorf("generating DOT: %w", err)
	}

	// Check if Graphviz is available
	if _, err := exec.LookPath("dot"); err != nil {
		// Graphviz not installed — output DOT with instructions
		fmt.Fprintln(w, "% Graphviz not found. Install it to generate PDF:")
		fmt.Fprintln(w, "% brew install graphviz    (macOS)")
		fmt.Fprintln(w, "% apt install graphviz     (Ubuntu)")
		fmt.Fprintln(w, "% Then run: govis -format dot | dot -Tpdf -o output.pdf")
		fmt.Fprintln(w, "%")
		fmt.Fprintln(w, "% DOT output follows:")
		fmt.Fprintln(w, "")
		_, err := w.Write(dotBuf.Bytes())
		return err
	}

	// Invoke dot -Tpdf
	cmd := exec.Command("dot", "-Tpdf")
	cmd.Stdin = &dotBuf
	cmd.Stdout = w

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("graphviz error: %s: %w", stderr.String(), err)
	}

	return nil
}
