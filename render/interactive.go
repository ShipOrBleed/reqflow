package render

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	govis "github.com/thzgajendra/govis"
)

// InteractiveRenderer generates a self-contained HTML page with a layered
// architecture visualization — handlers at top, services in middle, stores
// and models at bottom, grouped by package.
type InteractiveRenderer struct{}

type layerNode struct {
	ID      string            `json:"id"`
	Label   string            `json:"label"`
	Kind    string            `json:"kind"`
	Pkg     string            `json:"pkg"`
	PkgName string            `json:"pkgName"`
	File    string            `json:"file"`
	Line    int               `json:"line"`
	Methods []string          `json:"methods,omitempty"`
	Fields  []string          `json:"fields,omitempty"`
	Meta    map[string]string `json:"meta,omitempty"`
}

type layerEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Kind   string `json:"kind"`
}

type layerData struct {
	Nodes []layerNode `json:"nodes"`
	Edges []layerEdge `json:"edges"`
}

func (ir *InteractiveRenderer) Render(g *govis.Graph, w io.Writer) error {
	ld := layerData{}

	for _, node := range g.Nodes {
		var fields []string
		for _, f := range node.Fields {
			fields = append(fields, fmt.Sprintf("%s %s", f.Name, f.Type))
		}
		pkgParts := strings.Split(node.Package, "/")
		pkgName := pkgParts[len(pkgParts)-1]

		ld.Nodes = append(ld.Nodes, layerNode{
			ID:      node.ID,
			Label:   node.Name,
			Kind:    string(node.Kind),
			Pkg:     node.Package,
			PkgName: pkgName,
			File:    node.File,
			Line:    node.Line,
			Methods: node.Methods,
			Fields:  fields,
			Meta:    node.Meta,
		})
	}

	// Sort nodes by kind then name for consistent layout
	sort.Slice(ld.Nodes, func(i, j int) bool {
		if ld.Nodes[i].Kind != ld.Nodes[j].Kind {
			return ld.Nodes[i].Kind < ld.Nodes[j].Kind
		}
		return ld.Nodes[i].Label < ld.Nodes[j].Label
	})

	nodeIDs := make(map[string]bool)
	for _, n := range ld.Nodes {
		nodeIDs[n.ID] = true
	}

	for _, edge := range g.Edges {
		if nodeIDs[edge.From] && nodeIDs[edge.To] {
			ld.Edges = append(ld.Edges, layerEdge{
				Source: edge.From,
				Target: edge.To,
				Kind:   string(edge.Kind),
			})
		}
	}

	graphJSON, err := json.Marshal(ld)
	if err != nil {
		return fmt.Errorf("marshalling graph: %w", err)
	}

	summaryHTML := govis.GetSummaryHTML(g)

	// Count by kind for filter checkboxes
	kindCounts := make(map[string]int)
	for _, n := range g.Nodes {
		kindCounts[string(n.Kind)]++
	}
	var filterHTML strings.Builder
	for kind, count := range kindCounts {
		checked := "checked"
		// Hide plain funcs/structs/interfaces by default
		if kind == "func" || kind == "struct" || kind == "interface" {
			checked = ""
		}
		filterHTML.WriteString(fmt.Sprintf(
			`<label class="filter-item"><input type="checkbox" %s data-kind="%s" onchange="applyFilters()"> %s (%d)</label>`,
			checked, kind, kind, count))
	}

	fmt.Fprintf(w, interactiveTemplate, summaryHTML, filterHTML.String(), string(graphJSON))
	return nil
}

var interactiveTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Govis Architecture</title>
    <style>
        :root {
            --bg: #0f172a; --surface: #1e293b; --surface2: #273449;
            --accent: #38bdf8; --text: #f1f5f9; --text-muted: #94a3b8; --border: #334155;
        }
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: 'Inter', system-ui, -apple-system, sans-serif; background: var(--bg); color: var(--text); height: 100vh; display: flex; flex-direction: column; overflow: hidden; }

        header { background: var(--surface); padding: 10px 20px; border-bottom: 1px solid var(--border); display: flex; justify-content: space-between; align-items: center; flex-shrink: 0; }
        .logo { font-size: 1.3rem; font-weight: 800; color: var(--accent); letter-spacing: -0.02em; }
        .logo span { color: var(--text); font-weight: 400; }

        .main { display: flex; flex: 1; overflow: hidden; }

        .sidebar { width: 260px; background: var(--surface); border-right: 1px solid var(--border); padding: 12px; overflow-y: auto; flex-shrink: 0; display: flex; flex-direction: column; gap: 14px; }
        .sidebar h4 { color: var(--text-muted); font-size: 0.65rem; text-transform: uppercase; letter-spacing: 0.06em; margin-bottom: 6px; }
        .stats-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 6px; }
        .stat-card { background: var(--bg); padding: 6px; border-radius: 6px; border: 1px solid var(--border); text-align: center; }
        .stat-card h3 { font-size: 1rem; color: var(--accent); }
        .stat-card p { font-size: 0.6rem; color: var(--text-muted); text-transform: uppercase; margin-top: 2px; }
        .filter-item { display: flex; align-items: center; gap: 6px; font-size: 0.75rem; cursor: pointer; padding: 2px 0; }
        .filter-item input { accent-color: var(--accent); }
        #search { width: 100%%; padding: 6px 8px; background: var(--bg); border: 1px solid var(--border); border-radius: 5px; color: var(--text); font-size: 0.8rem; }
        #search::placeholder { color: var(--text-muted); }

        .canvas-wrap { flex: 1; position: relative; overflow: hidden; }
        #canvas { position: absolute; inset: 0; overflow: auto; padding: 30px; }

        .layer { margin-bottom: 24px; }
        .layer-header { font-size: 0.7rem; font-weight: 700; text-transform: uppercase; letter-spacing: 0.08em; color: var(--text-muted); margin-bottom: 8px; padding-left: 4px; border-left: 3px solid; }
        .layer-header.handler { border-color: #28a745; color: #28a745; }
        .layer-header.service { border-color: #007bff; color: #007bff; }
        .layer-header.store { border-color: #ffc107; color: #ffc107; }
        .layer-header.model { border-color: #dc3545; color: #dc3545; }
        .layer-header.event { border-color: #adb5bd; color: #adb5bd; }
        .layer-header.grpc { border-color: #0dcaf0; color: #0dcaf0; }
        .layer-header.infra { border-color: #6c3483; color: #6c3483; }
        .layer-header.other { border-color: #6c757d; color: #6c757d; }

        .layer-grid { display: flex; flex-wrap: wrap; gap: 8px; }

        .node-card {
            background: var(--surface2); border: 1px solid var(--border); border-radius: 8px;
            padding: 8px 12px; cursor: pointer; transition: all 0.15s; min-width: 120px; max-width: 220px;
            position: relative;
        }
        .node-card:hover { border-color: var(--accent); transform: translateY(-1px); box-shadow: 0 4px 12px rgba(0,0,0,0.3); }
        .node-card.selected { border-color: var(--accent); background: #1a3a5c; }
        .node-card.search-match { border-color: #fbbf24; box-shadow: 0 0 0 2px rgba(251,191,36,0.3); }

        .node-name { font-size: 0.8rem; font-weight: 600; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
        .node-pkg { font-size: 0.6rem; color: var(--text-muted); margin-top: 2px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
        .node-badge { position: absolute; top: -4px; right: -4px; width: 10px; height: 10px; border-radius: 50%%; border: 2px solid var(--surface2); }
        .node-route { font-size: 0.6rem; color: #22c55e; margin-top: 3px; font-family: monospace; }

        .node-card.kind-handler { border-left: 3px solid #28a745; }
        .node-card.kind-service { border-left: 3px solid #007bff; }
        .node-card.kind-store { border-left: 3px solid #ffc107; }
        .node-card.kind-model { border-left: 3px solid #dc3545; }
        .node-card.kind-event { border-left: 3px solid #adb5bd; }
        .node-card.kind-grpc { border-left: 3px solid #0dcaf0; }
        .node-card.kind-infra { border-left: 3px solid #6c3483; }
        .node-card.kind-middleware { border-left: 3px solid #856404; }
        .node-card.kind-func { border-left: 3px solid #6c757d; }
        .node-card.kind-struct { border-left: 3px solid #6c757d; }
        .node-card.kind-interface { border-left: 3px solid #6c757d; }

        .detail-panel {
            display: none; position: fixed; top: 60px; right: 16px; width: 340px; max-height: calc(100vh - 80px);
            background: var(--surface); border: 1px solid var(--border); border-radius: 10px;
            padding: 16px; box-shadow: 0 16px 40px rgba(0,0,0,0.5); z-index: 50; overflow-y: auto;
        }
        .detail-panel h3 { color: var(--accent); font-size: 0.95rem; margin-bottom: 6px; }
        .detail-badge { display: inline-block; padding: 2px 8px; border-radius: 4px; font-size: 0.65rem; text-transform: uppercase; font-weight: 700; color: #fff; }
        .detail-section { margin-top: 10px; }
        .detail-section h4 { font-size: 0.65rem; color: var(--text-muted); text-transform: uppercase; margin-bottom: 4px; }
        .detail-item { font-size: 0.75rem; color: var(--text-muted); padding: 2px 0; }
        .detail-item strong { color: var(--text); }
        .detail-close { position: absolute; top: 8px; right: 12px; background: none; border: none; color: var(--text-muted); cursor: pointer; font-size: 1.1rem; }
        .conn-link { color: var(--accent); cursor: pointer; font-size: 0.75rem; }
        .conn-link:hover { text-decoration: underline; }

        ::-webkit-scrollbar { width: 5px; }
        ::-webkit-scrollbar-track { background: transparent; }
        ::-webkit-scrollbar-thumb { background: var(--border); border-radius: 3px; }
    </style>
</head>
<body>
    <header>
        <div class="logo">GOVIS<span>.arch</span></div>
        <div style="font-size:0.7rem;color:var(--text-muted);">Layered Architecture View</div>
    </header>

    <div class="main">
        <div class="sidebar">
            <section>
                <h4>Search</h4>
                <input id="search" type="text" placeholder="Search nodes..." oninput="searchNodes(this.value)">
            </section>
            <section>
                <h4>System Health</h4>
                %s
            </section>
            <section>
                <h4>Filter by Kind</h4>
                %s
            </section>
        </div>

        <div class="canvas-wrap">
            <div id="canvas"></div>
        </div>
    </div>

    <div class="detail-panel" id="detail">
        <button class="detail-close" onclick="closeDetail()">&times;</button>
        <div id="detail-content"></div>
    </div>

    <script>
    const data = %s;

    const kindColors = {
        handler:'#28a745', service:'#007bff', store:'#ffc107', model:'#dc3545',
        event:'#adb5bd', middleware:'#856404', grpc:'#0dcaf0', infra:'#6c3483',
        route:'#17a2b8', envvar:'#20c997', table:'#fd7e14', dependency:'#adb5bd',
        container:'#e83e8c', proto_rpc:'#6610f2', proto_msg:'#e83e8c',
        struct:'#6c757d', interface:'#6c757d', func:'#6c757d'
    };

    // Layer ordering (top to bottom)
    const layerOrder = ['handler','grpc','middleware','service','event','store','model','infra','route','envvar','table','container','proto_rpc','proto_msg','interface','struct','func'];

    // Build adjacency for connections
    const outEdges = {};  // nodeID -> [{target, kind}]
    const inEdges = {};   // nodeID -> [{source, kind}]
    data.edges.forEach(e => {
        if (!outEdges[e.source]) outEdges[e.source] = [];
        outEdges[e.source].push({target: e.target, kind: e.kind});
        if (!inEdges[e.target]) inEdges[e.target] = [];
        inEdges[e.target].push({source: e.source, kind: e.kind});
    });

    // Node lookup
    const nodeMap = {};
    data.nodes.forEach(n => nodeMap[n.id] = n);

    function renderGraph() {
        const canvas = document.getElementById('canvas');
        canvas.innerHTML = '';

        // Get active kind filters
        const activeKinds = new Set();
        document.querySelectorAll('[data-kind]').forEach(cb => {
            if (cb.checked) activeKinds.add(cb.dataset.kind);
        });

        // Group nodes by layer
        const layers = {};
        data.nodes.forEach(n => {
            if (!activeKinds.has(n.kind)) return;
            if (!layers[n.kind]) layers[n.kind] = [];
            layers[n.kind].push(n);
        });

        // Render layers in order
        layerOrder.forEach(kind => {
            const nodes = layers[kind];
            if (!nodes || nodes.length === 0) return;

            const layerDiv = document.createElement('div');
            layerDiv.className = 'layer';

            const layerName = kind.charAt(0).toUpperCase() + kind.slice(1) + 's';
            const headerClass = ['handler','service','store','model','event','grpc','infra'].includes(kind) ? kind : 'other';
            layerDiv.innerHTML = '<div class="layer-header ' + headerClass + '">' + layerName + ' (' + nodes.length + ')</div>';

            const grid = document.createElement('div');
            grid.className = 'layer-grid';

            // Sort by package then name
            nodes.sort((a,b) => (a.pkgName + a.label).localeCompare(b.pkgName + b.label));

            nodes.forEach(n => {
                const card = document.createElement('div');
                card.className = 'node-card kind-' + n.kind;
                card.dataset.id = n.id;
                card.onclick = () => showDetail(n.id);

                let html = '<div class="node-name" title="' + n.label + '">' + n.label + '</div>';
                html += '<div class="node-pkg" title="' + n.pkgName + '">' + n.pkgName + '</div>';

                if (n.meta && n.meta.route) {
                    html += '<div class="node-route">' + n.meta.route + '</div>';
                }

                const connections = (outEdges[n.id]||[]).length + (inEdges[n.id]||[]).length;
                if (connections > 0) {
                    html += '<div class="node-badge" style="background:' + (kindColors[n.kind]||'#6c757d') + '"></div>';
                }

                card.innerHTML = html;
                grid.appendChild(card);
            });

            layerDiv.appendChild(grid);
            canvas.appendChild(layerDiv);
        });
    }

    function showDetail(id) {
        // Clear previous selection
        document.querySelectorAll('.node-card.selected').forEach(c => c.classList.remove('selected'));
        const card = document.querySelector('[data-id="' + CSS.escape(id) + '"]');
        if (card) card.classList.add('selected');

        const n = nodeMap[id];
        if (!n) return;

        const panel = document.getElementById('detail');
        const content = document.getElementById('detail-content');

        let html = '<h3>' + n.label + '</h3>';
        html += '<div class="detail-badge" style="background:' + (kindColors[n.kind]||'#6c757d') + '">' + n.kind + '</div>';

        html += '<div class="detail-section"><h4>Location</h4>';
        html += '<div class="detail-item"><strong>Package:</strong> ' + n.pkg + '</div>';
        if (n.file) html += '<div class="detail-item"><strong>File:</strong> ' + n.file + ':' + n.line + '</div>';
        html += '</div>';

        // Metadata
        const meta = n.meta || {};
        const metaKeys = Object.keys(meta).filter(k => meta[k]);
        if (metaKeys.length > 0) {
            html += '<div class="detail-section"><h4>Metadata</h4>';
            metaKeys.forEach(k => {
                html += '<div class="detail-item"><strong>' + k + ':</strong> ' + meta[k] + '</div>';
            });
            html += '</div>';
        }

        // Methods
        if (n.methods && n.methods.length > 0) {
            html += '<div class="detail-section"><h4>Methods (' + n.methods.length + ')</h4>';
            n.methods.forEach(m => { html += '<div class="detail-item" style="font-family:monospace">' + m + '()</div>'; });
            html += '</div>';
        }

        // Fields
        if (n.fields && n.fields.length > 0) {
            html += '<div class="detail-section"><h4>Fields (' + n.fields.length + ')</h4>';
            n.fields.slice(0,10).forEach(f => { html += '<div class="detail-item" style="font-family:monospace">' + f + '</div>'; });
            if (n.fields.length > 10) html += '<div class="detail-item">... +' + (n.fields.length-10) + ' more</div>';
            html += '</div>';
        }

        // Connections
        const outs = outEdges[id] || [];
        const ins = inEdges[id] || [];
        if (outs.length > 0 || ins.length > 0) {
            html += '<div class="detail-section"><h4>Connections (' + ins.length + ' in, ' + outs.length + ' out)</h4>';
            ins.forEach(e => {
                const src = nodeMap[e.source];
                const label = src ? src.label : e.source;
                html += '<div class="detail-item"><span class="conn-link" onclick="showDetail(\'' + e.source.replace(/'/g,"\\'") + '\')">&larr; ' + label + '</span> <span style="color:#475569">(' + e.kind + ')</span></div>';
            });
            outs.forEach(e => {
                const tgt = nodeMap[e.target];
                const label = tgt ? tgt.label : e.target;
                html += '<div class="detail-item"><span class="conn-link" onclick="showDetail(\'' + e.target.replace(/'/g,"\\'") + '\')">&rarr; ' + label + '</span> <span style="color:#475569">(' + e.kind + ')</span></div>';
            });
            html += '</div>';
        }

        content.innerHTML = html;
        panel.style.display = 'block';
    }

    function closeDetail() {
        document.getElementById('detail').style.display = 'none';
        document.querySelectorAll('.node-card.selected').forEach(c => c.classList.remove('selected'));
    }

    function searchNodes(query) {
        document.querySelectorAll('.node-card').forEach(card => card.classList.remove('search-match'));
        if (!query) return;
        const q = query.toLowerCase();
        document.querySelectorAll('.node-card').forEach(card => {
            const name = card.querySelector('.node-name').textContent.toLowerCase();
            const pkg = card.querySelector('.node-pkg').textContent.toLowerCase();
            if (name.includes(q) || pkg.includes(q)) {
                card.classList.add('search-match');
                card.scrollIntoView({behavior:'smooth', block:'center'});
            }
        });
    }

    function applyFilters() {
        renderGraph();
    }

    // Initial render
    renderGraph();
    </script>
</body>
</html>`
