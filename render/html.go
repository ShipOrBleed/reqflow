package render

import (
	"bytes"
	"fmt"
	"io"

	"github.com/zopdev/govis"
)

type HTMLRenderer struct{}

func (h *HTMLRenderer) Render(g *structmap.Graph, w io.Writer) error {
	var buf bytes.Buffer
	m := &MermaidRenderer{}
	if err := m.Render(g, &buf); err != nil {
		return err
	}

	htmlTemplate := `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Govis Interactive Architecture Map</title>
    <!-- Modern Styling -->
    <style>
        body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; background: #0f172a; margin: 0; display: flex; flex-direction: column; height: 100vh; color: #f8fafc; }
        header { background: #1e293b; padding: 1.5rem; box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.5); z-index: 10; display: flex; justify-content: space-between; align-items: center; }
        h2 { margin: 0; font-size: 1.5rem; font-weight: 600; color: #38bdf8; }
        p { margin: 0.5rem 0 0 0; color: #94a3b8; font-size: 0.9rem; }
        .badge { background: #3b82f6; color: white; padding: 0.25rem 0.75rem; border-radius: 999px; font-size: 0.8rem; font-weight: bold; }
        
        .container { flex-grow: 1; overflow: hidden; display: flex; position: relative; }
        
        /* The graph container */
        #mermaid-container { width: 100%%; height: 100%%; display: flex; align-items: center; justify-content: center; cursor: grab; }
        #mermaid-container:active { cursor: grabbing; }
        
        /* Legend */
        .legend { position: absolute; bottom: 2rem; left: 2rem; background: rgba(30, 41, 59, 0.9); padding: 1.5rem; border-radius: 12px; box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.5); backdrop-filter: blur(4px); border: 1px solid #334155; }
        .legend h4 { margin: 0 0 1rem 0; color: #f1f5f9; }
        .legend-item { display: flex; align-items: center; margin-bottom: 0.5rem; font-size: 0.85rem; color: #cbd5e1; }
        .dot { width: 12px; height: 12px; border-radius: 50%%; margin-right: 8px; }
        
        .c-handler { background: #28a745; box-shadow: 0 0 8px #28a745; }
        .c-service { background: #007bff; box-shadow: 0 0 8px #007bff; }
        .c-store { background: #ffc107; box-shadow: 0 0 8px #ffc107; }
        .c-model { background: #dc3545; box-shadow: 0 0 8px #dc3545; }
    </style>
</head>
<body>
    <header>
        <div>
            <h2>Govis Architecture Dashboard 🗺️</h2>
            <p>Interactive graph mapped from your Go AST. Pan around and zoom to explore.</p>
        </div>
        <div class="badge">Live Map</div>
    </header>
    
    <div class="container">
        <!-- SVG injection inside a flex container allows panning -->
        <div id="mermaid-container" class="mermaid">
%s
        </div>
        
        <div class="legend">
            <h4>Architecture Legend</h4>
            <div class="legend-item"><div class="dot c-handler"></div> HTTP Handler / Controller</div>
            <div class="legend-item"><div class="dot c-service"></div> Business Logic / Service</div>
            <div class="legend-item"><div class="dot c-store"></div> Database Store / Repository</div>
            <div class="legend-item"><div class="dot c-model"></div> Data Model / Entity</div>
            <div class="legend-item"><div class="dot" style="background:#6c757d; border-radius:2px"></div> Interface Integration</div>
        </div>
    </div>
    
    <!-- Render Mermaid natively mapping to standard SVGs -->
    <script type="module">
      import mermaid from 'https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.esm.min.mjs';
      mermaid.initialize({ 
          startOnLoad: true, 
          maxTextSize: 900000,
          theme: 'dark', // Gorgeous dark mode
          securityLevel: 'loose' // Vital to allow our vscode:// custom URI schemes!
      });
      
      // Delay initialization of SVG pan-zoom until Mermaid injects the literal SVG node
      setTimeout(() => {
          let svgElem = document.querySelector("#mermaid-container svg");
          if(svgElem) {
              svgElem.style.width = '100%%';
              svgElem.style.height = '100%%';
              svgElem.style.maxWidth = 'none';
              
              // Load svg-pan-zoom dynamically
              let script = document.createElement('script');
              script.src = 'https://cdn.jsdelivr.net/npm/svg-pan-zoom@3.6.1/dist/svg-pan-zoom.min.js';
              script.onload = () => {
                  window.panZoomSetup = svgPanZoom(svgElem, {
                      zoomEnabled: true,
                      controlIconsEnabled: true,
                      fit: true,
                      center: true,
                      minZoom: 0.1
                  });
              };
              document.head.appendChild(script);
          }
      }, 500);
    </script>
</body>
</html>`

	// Note: We escape any format string markers in javascript or css utilizing percents inside string template
	fmt.Fprintf(w, htmlTemplate, buf.String())
	return nil
}
