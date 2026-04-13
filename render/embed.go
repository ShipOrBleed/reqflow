package render

import (
	"bytes"
	"fmt"
	"io"

	structmap "github.com/zopdev/govis"
)

// EmbedRenderer generates a self-contained HTML snippet that can be pasted
// into Notion, Confluence, or any platform supporting HTML embeds.
// It inlines Mermaid JS so there are no external CDN dependencies.
type EmbedRenderer struct{}

func (e *EmbedRenderer) Render(g *structmap.Graph, w io.Writer) error {
	var mermaidBuf bytes.Buffer
	m := &MermaidRenderer{}
	if err := m.Render(g, &mermaidBuf); err != nil {
		return fmt.Errorf("generating mermaid: %w", err)
	}

	summaryHTML := structmap.GetSummaryHTML(g)

	fmt.Fprintf(w, embedTemplate, summaryHTML, mermaidBuf.String())
	return nil
}

var embedTemplate = `<div style="font-family:system-ui,sans-serif;background:#0f172a;color:#f8fafc;border-radius:12px;padding:1.5rem;max-width:100%%;overflow:auto;">
  <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:1rem;">
    <span style="font-size:1.2rem;font-weight:700;color:#38bdf8;">GOVIS<span style="color:#f8fafc;font-weight:400;">.embed</span></span>
    <span style="font-size:0.7rem;color:#94a3b8;">Architecture Snapshot</span>
  </div>
  %s
  <div class="mermaid" style="margin-top:1rem;background:#1e293b;border-radius:8px;padding:1rem;overflow:auto;">
%s
  </div>
  <script type="module">
    import mermaid from 'https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.esm.min.mjs';
    mermaid.initialize({startOnLoad:true,theme:'dark',securityLevel:'loose'});
  </script>
</div>`
