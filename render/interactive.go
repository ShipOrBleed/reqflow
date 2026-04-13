package render

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	govis "github.com/zopdev/govis"
)

// InteractiveRenderer generates a self-contained HTML page with Cytoscape.js
// for interactive, force-directed graph visualization.
type InteractiveRenderer struct{}

type cyNode struct {
	ID      string            `json:"id"`
	Label   string            `json:"label"`
	Kind    string            `json:"kind"`
	Pkg     string            `json:"pkg"`
	File    string            `json:"file"`
	Line    int               `json:"line"`
	Methods []string          `json:"methods,omitempty"`
	Fields  []string          `json:"fields,omitempty"`
	Meta    map[string]string `json:"meta,omitempty"`
	Parent  string            `json:"parent,omitempty"`
}

type cyEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Kind   string `json:"kind"`
}

type cyGraph struct {
	Nodes    []cyNode `json:"nodes"`
	Edges    []cyEdge `json:"edges"`
	Clusters []string `json:"clusters"`
}

func (ir *InteractiveRenderer) Render(g *govis.Graph, w io.Writer) error {
	cg := cyGraph{}

	// Add compound parent nodes for each cluster/package
	for pkg := range g.Clusters {
		cg.Clusters = append(cg.Clusters, pkg)
	}

	// Add nodes
	for _, node := range g.Nodes {
		var fields []string
		for _, f := range node.Fields {
			fields = append(fields, fmt.Sprintf("%s %s", f.Name, f.Type))
		}
		cg.Nodes = append(cg.Nodes, cyNode{
			ID:      node.ID,
			Label:   node.Name,
			Kind:    string(node.Kind),
			Pkg:     node.Package,
			File:    node.File,
			Line:    node.Line,
			Methods: node.Methods,
			Fields:  fields,
			Meta:    node.Meta,
			Parent:  node.Package,
		})
	}

	// Add edges
	for _, edge := range g.Edges {
		cg.Edges = append(cg.Edges, cyEdge{
			Source: edge.From,
			Target: edge.To,
			Kind:   string(edge.Kind),
		})
	}

	graphJSON, err := json.Marshal(cg)
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
		filterHTML.WriteString(fmt.Sprintf(
			`<label class="filter-item"><input type="checkbox" checked data-kind="%s" onchange="toggleKind()"> %s (%d)</label>`,
			kind, kind, count))
	}

	fmt.Fprintf(w, interactiveTemplate, summaryHTML, filterHTML.String(), string(graphJSON))
	return nil
}

var interactiveTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Govis Interactive Architecture</title>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/cytoscape/3.28.1/cytoscape.min.js"></script>
    <style>
        :root {
            --bg: #0f172a;
            --surface: #1e293b;
            --accent: #38bdf8;
            --text: #f8fafc;
            --text-muted: #94a3b8;
            --border: #334155;
        }
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: 'Inter', system-ui, sans-serif; background: var(--bg); color: var(--text); height: 100vh; display: flex; flex-direction: column; }

        header {
            background: var(--surface); padding: 0.75rem 1.5rem; border-bottom: 1px solid var(--border);
            display: flex; justify-content: space-between; align-items: center;
        }
        .logo { font-size: 1.4rem; font-weight: 800; color: var(--accent); }
        .logo span { color: var(--text); font-weight: 400; }

        .main { display: flex; flex: 1; overflow: hidden; }

        .sidebar {
            width: 300px; background: var(--surface); border-right: 1px solid var(--border);
            padding: 1rem; overflow-y: auto; display: flex; flex-direction: column; gap: 1.25rem;
        }
        .sidebar h4 { color: var(--text-muted); font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.05em; margin-bottom: 0.5rem; }

        .stats-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 0.5rem; }
        .stat-card { background: var(--bg); padding: 0.5rem; border-radius: 6px; border: 1px solid var(--border); text-align: center; }
        .stat-card h3 { font-size: 1.1rem; color: var(--accent); }
        .stat-card p { font-size: 0.65rem; color: var(--text-muted); text-transform: uppercase; margin-top: 0.2rem; }

        .filter-item { display: flex; align-items: center; gap: 0.5rem; font-size: 0.8rem; cursor: pointer; padding: 0.15rem 0; }
        .filter-item input { accent-color: var(--accent); }

        #search { width: 100%%; padding: 0.5rem; background: var(--bg); border: 1px solid var(--border); border-radius: 6px; color: var(--text); font-size: 0.85rem; }
        #search::placeholder { color: var(--text-muted); }

        #cy { flex: 1; }

        .detail-panel {
            display: none; position: fixed; bottom: 1rem; right: 1rem; width: 340px;
            background: var(--surface); border: 1px solid var(--border); border-radius: 10px;
            padding: 1rem; box-shadow: 0 16px 40px rgba(0,0,0,0.5); z-index: 50; max-height: 50vh; overflow-y: auto;
        }
        .detail-panel h3 { color: var(--accent); margin-bottom: 0.5rem; font-size: 1rem; }
        .detail-panel .kind-badge { display: inline-block; padding: 0.15rem 0.5rem; border-radius: 4px; font-size: 0.7rem; text-transform: uppercase; font-weight: 600; margin-bottom: 0.5rem; }
        .detail-panel .meta-item { font-size: 0.8rem; color: var(--text-muted); padding: 0.2rem 0; }
        .detail-panel .meta-item strong { color: var(--text); }
        .detail-panel .close-btn { position: absolute; top: 0.5rem; right: 0.75rem; background: none; border: none; color: var(--text-muted); cursor: pointer; font-size: 1.2rem; }

        .controls { position: absolute; top: 0.75rem; right: 0.75rem; display: flex; gap: 0.4rem; z-index: 10; }
        .controls button {
            background: var(--surface); border: 1px solid var(--border); color: var(--text);
            padding: 0.4rem 0.75rem; border-radius: 6px; cursor: pointer; font-size: 0.8rem;
        }
        .controls button:hover { background: var(--border); }

        .legend { display: flex; flex-direction: column; gap: 0.3rem; }
        .legend-item { display: flex; align-items: center; gap: 0.5rem; font-size: 0.8rem; }
        .legend-dot { width: 10px; height: 10px; border-radius: 3px; flex-shrink: 0; }

        ::-webkit-scrollbar { width: 6px; }
        ::-webkit-scrollbar-track { background: var(--bg); }
        ::-webkit-scrollbar-thumb { background: var(--border); border-radius: 3px; }
    </style>
</head>
<body>
    <header>
        <div class="logo">GOVIS<span>.interactive</span></div>
        <div style="font-size:0.75rem;color:var(--text-muted);">Force-Directed Graph</div>
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
            <section>
                <h4>Layers</h4>
                <div class="legend">
                    <div class="legend-item"><div class="legend-dot" style="background:#28a745"></div> Handler</div>
                    <div class="legend-item"><div class="legend-dot" style="background:#007bff"></div> Service</div>
                    <div class="legend-item"><div class="legend-dot" style="background:#ffc107"></div> Store</div>
                    <div class="legend-item"><div class="legend-dot" style="background:#dc3545"></div> Model</div>
                    <div class="legend-item"><div class="legend-dot" style="background:#e2e3e5"></div> Event</div>
                    <div class="legend-item"><div class="legend-dot" style="background:#0c5460"></div> gRPC</div>
                    <div class="legend-item"><div class="legend-dot" style="background:#6c3483"></div> Infra</div>
                    <div class="legend-item"><div class="legend-dot" style="background:#6c757d"></div> Other</div>
                </div>
            </section>
        </div>

        <div id="cy" style="position:relative;">
            <div class="controls">
                <button onclick="cy.fit(null, 50)">Fit</button>
                <button onclick="cy.zoom(cy.zoom()*1.3);cy.center()">Zoom +</button>
                <button onclick="cy.zoom(cy.zoom()*0.7);cy.center()">Zoom -</button>
                <button onclick="runLayout()">Re-layout</button>
            </div>
        </div>
    </div>

    <div class="detail-panel" id="detail">
        <button class="close-btn" onclick="document.getElementById('detail').style.display='none'">&times;</button>
        <div id="detail-content"></div>
    </div>

    <script>
    const graphData = %s;

    const kindColors = {
        handler: '#28a745', service: '#007bff', store: '#ffc107', model: '#dc3545',
        event: '#e2e3e5', middleware: '#856404', grpc: '#0c5460', infra: '#6c3483',
        route: '#17a2b8', envvar: '#20c997', table: '#fd7e14', dependency: '#adb5bd',
        container: '#e83e8c', proto_rpc: '#6610f2', proto_msg: '#e83e8c',
        struct: '#6c757d', interface: '#6c757d', func: '#6c757d'
    };

    const kindShapes = {
        handler: 'round-rectangle', service: 'ellipse', store: 'barrel',
        model: 'diamond', event: 'tag', middleware: 'round-hexagon',
        grpc: 'hexagon', infra: 'octagon', interface: 'cut-rectangle',
        route: 'round-triangle', table: 'barrel', container: 'rectangle',
        struct: 'rectangle', func: 'round-rectangle'
    };

    const edgeColors = {
        depends: '#64748b', implements: '#38bdf8', embeds: '#a78bfa',
        calls: '#f472b6', flows: '#22c55e', reads: '#facc15',
        maps_to: '#fb923c', publishes: '#34d399', subscribes: '#f87171',
        rpc: '#818cf8', transitive: '#475569'
    };

    const elements = [];

    // Add compound parent nodes for clusters
    const clusters = new Set(graphData.nodes.map(n => n.pkg));
    clusters.forEach(pkg => {
        elements.push({ data: { id: 'cluster_' + pkg, label: pkg.split('/').pop() }, classes: 'cluster' });
    });

    graphData.nodes.forEach(n => {
        elements.push({
            data: {
                id: n.id, label: n.label, kind: n.kind, pkg: n.pkg,
                file: n.file, line: n.line, parent: 'cluster_' + n.pkg,
                meta: n.meta || {}, methods: n.methods || [], fields: n.fields || []
            }
        });
    });

    graphData.edges.forEach(e => {
        elements.push({
            data: { source: e.source, target: e.target, kind: e.kind, label: e.kind }
        });
    });

    const cy = cytoscape({
        container: document.getElementById('cy'),
        elements: elements,
        style: [
            {
                selector: 'node[kind]',
                style: {
                    'label': 'data(label)',
                    'background-color': function(ele) { return kindColors[ele.data('kind')] || '#6c757d'; },
                    'shape': function(ele) { return kindShapes[ele.data('kind')] || 'rectangle'; },
                    'color': '#f8fafc',
                    'text-outline-color': '#0f172a',
                    'text-outline-width': 2,
                    'font-size': '11px',
                    'width': 40,
                    'height': 40,
                    'text-valign': 'bottom',
                    'text-margin-y': 6,
                    'border-width': 2,
                    'border-color': function(ele) { return kindColors[ele.data('kind')] || '#6c757d'; }
                }
            },
            {
                selector: ':parent',
                style: {
                    'background-color': '#1e293b',
                    'background-opacity': 0.6,
                    'border-color': '#334155',
                    'border-width': 1,
                    'label': 'data(label)',
                    'color': '#94a3b8',
                    'font-size': '10px',
                    'text-valign': 'top',
                    'text-halign': 'center',
                    'padding': '12px',
                    'shape': 'round-rectangle'
                }
            },
            {
                selector: 'edge',
                style: {
                    'width': 1.5,
                    'line-color': function(ele) { return edgeColors[ele.data('kind')] || '#64748b'; },
                    'target-arrow-color': function(ele) { return edgeColors[ele.data('kind')] || '#64748b'; },
                    'target-arrow-shape': 'triangle',
                    'curve-style': 'bezier',
                    'arrow-scale': 0.8,
                    'opacity': 0.7
                }
            },
            {
                selector: 'edge[kind="implements"]',
                style: { 'line-style': 'dashed', 'line-dash-pattern': [6, 3] }
            },
            {
                selector: 'edge[kind="embeds"]',
                style: { 'width': 2.5 }
            },
            {
                selector: 'edge[kind="calls"]',
                style: { 'line-style': 'dotted' }
            },
            {
                selector: 'node:selected',
                style: { 'border-width': 4, 'border-color': '#38bdf8', 'background-color': '#38bdf8' }
            },
            {
                selector: '.highlighted',
                style: { 'border-width': 3, 'border-color': '#fbbf24', 'background-color': '#fbbf24' }
            },
            {
                selector: '.dimmed',
                style: { 'opacity': 0.15 }
            },
            {
                selector: '.neighbor',
                style: { 'opacity': 1, 'border-width': 3, 'border-color': '#22c55e' }
            },
            {
                selector: '.hidden',
                style: { 'display': 'none' }
            }
        ],
        layout: { name: 'cose', animate: false, nodeRepulsion: 8000, idealEdgeLength: 120, edgeElasticity: 100, gravity: 0.25, numIter: 500 },
        wheelSensitivity: 0.3,
        minZoom: 0.05,
        maxZoom: 5
    });

    function runLayout() {
        cy.layout({ name: 'cose', animate: true, animationDuration: 800, nodeRepulsion: 8000, idealEdgeLength: 120, edgeElasticity: 100, gravity: 0.25, numIter: 500 }).run();
    }

    // Click to show detail panel
    cy.on('tap', 'node[kind]', function(evt) {
        const d = evt.target.data();
        const panel = document.getElementById('detail');
        const content = document.getElementById('detail-content');

        let html = '<h3>' + d.label + '</h3>';
        html += '<div class="kind-badge" style="background:' + (kindColors[d.kind]||'#6c757d') + ';color:#fff;">' + d.kind + '</div>';
        html += '<div class="meta-item"><strong>Package:</strong> ' + d.pkg + '</div>';
        if (d.file) html += '<div class="meta-item"><strong>File:</strong> ' + d.file + ':' + d.line + '</div>';

        if (d.methods && d.methods.length > 0) {
            html += '<div class="meta-item"><strong>Methods:</strong></div>';
            d.methods.forEach(m => { html += '<div class="meta-item" style="padding-left:1rem;">' + m + '()</div>'; });
        }
        if (d.fields && d.fields.length > 0) {
            html += '<div class="meta-item"><strong>Fields:</strong></div>';
            d.fields.forEach(f => { html += '<div class="meta-item" style="padding-left:1rem;">' + f + '</div>'; });
        }

        const meta = d.meta || {};
        const metaKeys = Object.keys(meta);
        if (metaKeys.length > 0) {
            html += '<div class="meta-item" style="margin-top:0.5rem;"><strong>Metadata:</strong></div>';
            metaKeys.forEach(k => { html += '<div class="meta-item" style="padding-left:1rem;"><strong>' + k + ':</strong> ' + meta[k] + '</div>'; });
        }

        // Show connections
        const neighborhood = evt.target.neighborhood();
        const incoming = neighborhood.filter('edge[target="' + d.id + '"]');
        const outgoing = neighborhood.filter('edge[source="' + d.id + '"]');
        html += '<div class="meta-item" style="margin-top:0.5rem;"><strong>Connections:</strong> ' + incoming.length + ' in, ' + outgoing.length + ' out</div>';

        content.innerHTML = html;
        panel.style.display = 'block';

        // Highlight neighbors
        cy.elements().removeClass('dimmed neighbor');
        cy.elements().not(evt.target.closedNeighborhood()).addClass('dimmed');
        evt.target.neighborhood().addClass('neighbor');
    });

    cy.on('tap', function(evt) {
        if (evt.target === cy) {
            cy.elements().removeClass('dimmed neighbor');
            document.getElementById('detail').style.display = 'none';
        }
    });

    function searchNodes(query) {
        cy.elements().removeClass('highlighted');
        if (!query) return;
        const q = query.toLowerCase();
        cy.nodes().forEach(n => {
            if (n.data('label') && n.data('label').toLowerCase().includes(q)) {
                n.addClass('highlighted');
            }
        });
    }

    function toggleKind() {
        const checkboxes = document.querySelectorAll('[data-kind]');
        const hidden = new Set();
        checkboxes.forEach(cb => { if (!cb.checked) hidden.add(cb.dataset.kind); });

        cy.nodes('[kind]').forEach(n => {
            if (hidden.has(n.data('kind'))) {
                n.addClass('hidden');
            } else {
                n.removeClass('hidden');
            }
        });
    }
    </script>
</body>
</html>`
