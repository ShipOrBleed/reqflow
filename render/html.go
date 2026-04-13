package render

import (
	"bytes"
	"fmt"
	"io"

	govis "github.com/thzgajendra/govis"
)

type HTMLRenderer struct{}

func (h *HTMLRenderer) Render(g *govis.Graph, w io.Writer) error {
	var buf bytes.Buffer
	m := &MermaidRenderer{}
	if err := m.Render(g, &buf); err != nil {
		return err
	}

	summaryHTML := govis.GetSummaryHTML(g)

	htmlTemplate := `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Govis Enterprise Dashboard</title>
    <style>
        :root {
            --bg: #0f172a;
            --surface: #1e293b;
            --accent: #38bdf8;
            --text: #f8fafc;
            --text-muted: #94a3b8;
            --border: #334155;
        }
        body { font-family: 'Inter', system-ui, sans-serif; background: var(--bg); margin: 0; display: flex; flex-direction: column; height: 100vh; color: var(--text); }
        header { 
            background: var(--surface); padding: 1rem 2rem; border-bottom: 1px solid var(--border);
            display: flex; justify-content: space-between; align-items: center; z-index: 100;
        }
        .logo { font-size: 1.5rem; font-weight: 800; color: var(--accent); letter-spacing: -0.025em; }
        .logo span { color: var(--text); font-weight: 400; }
        
        .main { display: flex; flex: 1; overflow: hidden; }
        
        .sidebar { 
            width: 320px; background: var(--surface); border-right: 1px solid var(--border);
            padding: 1.5rem; overflow-y: auto; display: flex; flex-direction: column; gap: 2rem;
        }
        
        .stats-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem; }
        .stat-card { 
            background: var(--bg); padding: 0.75rem; border-radius: 8px; border: 1px solid var(--border);
            text-align: center;
        }
        .stat-card h3 { margin: 0; font-size: 1.25rem; color: var(--accent); }
        .stat-card p { margin: 0.25rem 0 0 0; font-size: 0.75rem; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.05em; }
        
        #graph-container { flex: 1; position: relative; background: radial-gradient(circle at 2px 2px, #1e293b 1px, transparent 0); background-size: 24px 24px; }
        #mermaid-svg { width: 100%%; height: 100%%; }
        
        .node-details { 
            background: var(--surface); border: 1px solid var(--border); border-radius: 12px;
            padding: 1rem; position: absolute; bottom: 2rem; right: 2rem; width: 300px;
            box-shadow: 0 20px 25px -5px rgba(0,0,0,0.5); display: none;
        }
        
        .legend { display: flex; flex-direction: column; gap: 0.5rem; }
        .legend-item { display: flex; align-items: center; gap: 0.75rem; font-size: 0.875rem; }
        .dot { width: 12px; height: 12px; border-radius: 3px; }
        
        .controls { position: absolute; top: 1rem; right: 1rem; display: flex; gap: 0.5rem; }
        button { 
            background: var(--surface); border: 1px solid var(--border); color: var(--text);
            padding: 0.5rem; border-radius: 6px; cursor: pointer; transition: all 0.2s;
        }
        button:hover { background: var(--border); }

        /* Custom scrollbar */
        ::-webkit-scrollbar { width: 8px; }
        ::-webkit-scrollbar-track { background: var(--bg); }
        ::-webkit-scrollbar-thumb { background: var(--border); border-radius: 4px; }
    </style>
</head>
<body>
    <header>
        <div class="logo">GOVIS<span>.arch</span></div>
        <div style="display: flex; gap: 1rem; align-items: center;">
            <div style="font-size: 0.8rem; color: var(--text-muted);">ENTERPRISE EDITION</div>
            <div style="width: 8px; height: 8px; border-radius: 50%%; background: #22c55e; box-shadow: 0 0 8px #22c55e;"></div>
        </div>
    </header>
    
    <div class="main">
        <div class="sidebar">
            <section>
                <h4 style="margin: 0 0 1rem 0; color: var(--text-muted); font-size: 0.75rem; text-transform: uppercase;">System Health</h4>
                %s
            </section>
            
            <section>
                <h4 style="margin: 0 0 1rem 0; color: var(--text-muted); font-size: 0.75rem; text-transform: uppercase;">Architecture Layers</h4>
                <div class="legend">
                    <div class="legend-item"><div class="dot" style="background:#28a745"></div> 🌐 Handler / API</div>
                    <div class="legend-item"><div class="dot" style="background:#007bff"></div> ⚙️ Service / Logic</div>
                    <div class="legend-item"><div class="dot" style="background:#ffc107"></div> 🗄️ Store / Repository</div>
                    <div class="legend-item"><div class="dot" style="background:#dc3545"></div> 📄 Model / Entity</div>
                    <div class="legend-item"><div class="dot" style="background:#6c757d"></div> 📦 Other Types</div>
                </div>
            </section>

            <section style="margin-top: auto;">
                <p style="font-size: 0.75rem; color: var(--text-muted);">
                    Govis automatically infers these layers using DDD and framework naming heuristics.
                </p>
            </section>
        </div>
        
        <div id="graph-container">
            <div id="mermaid-svg" class="mermaid">
                %s
            </div>
            <div class="controls">
                <button onclick="resetZoom()">Reset View</button>
            </div>
        </div>
    </div>

    <script type="module">
        import mermaid from 'https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.esm.min.mjs';
        mermaid.initialize({ 
            startOnLoad: true, 
            theme: 'dark',
            securityLevel: 'loose'
        });

        setTimeout(() => {
            let svg = document.querySelector("#graph-container svg");
            if(svg) {
                svg.style.width = '100%%';
                svg.style.height = '100%%';
                
                let script = document.createElement('script');
                script.src = 'https://cdn.jsdelivr.net/npm/svg-pan-zoom@3.6.1/dist/svg-pan-zoom.min.js';
                script.onload = () => {
                    window.panZoom = svgPanZoom(svg, {
                        zoomEnabled: true,
                        controlIconsEnabled: false,
                        fit: true,
                        center: true,
                        minZoom: 0.05,
                        maxZoom: 20
                    });
                };
                document.head.appendChild(script);
            }
        }, 500);

        window.resetZoom = () => {
            if(window.panZoom) {
                window.panZoom.reset();
                window.panZoom.fit();
                window.panZoom.center();
            }
        };
    </script>
</body>
</html>`

	fmt.Fprintf(w, htmlTemplate, summaryHTML, buf.String())
	return nil
}
