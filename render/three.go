package render

import (
	"encoding/json"
	"fmt"
	"io"

	reqflow "github.com/thzgajendra/reqflow"
)

// ThreeRenderer generates a self-contained HTML page with Three.js and
// 3d-force-graph for interactive 3D architecture visualization.
type ThreeRenderer struct{}

type threeNode struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Kind  string `json:"kind"`
	Pkg   string `json:"pkg"`
	Val   int    `json:"val"` // size based on connections
}

type threeEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Kind   string `json:"kind"`
}

type threeGraph struct {
	Nodes []threeNode `json:"nodes"`
	Links []threeEdge `json:"links"`
}

func (t *ThreeRenderer) Render(g *reqflow.Graph, w io.Writer) error {
	tg := threeGraph{}

	// Count connections per node for sizing
	connCount := make(map[string]int)
	for _, edge := range g.Edges {
		connCount[edge.From]++
		connCount[edge.To]++
	}

	for _, node := range g.Nodes {
		val := connCount[node.ID] + 1
		if val < 2 {
			val = 2
		}
		tg.Nodes = append(tg.Nodes, threeNode{
			ID:   node.ID,
			Name: node.Name,
			Kind: string(node.Kind),
			Pkg:  node.Package,
			Val:  val,
		})
	}

	for _, edge := range g.Edges {
		tg.Links = append(tg.Links, threeEdge{
			Source: edge.From,
			Target: edge.To,
			Kind:   string(edge.Kind),
		})
	}

	graphJSON, err := json.Marshal(tg)
	if err != nil {
		return fmt.Errorf("marshalling graph: %w", err)
	}

	fmt.Fprintf(w, threeTemplate, string(graphJSON))
	return nil
}

var threeTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Reqflow 3D Architecture</title>
    <style>
        body { margin: 0; background: #0f172a; overflow: hidden; font-family: system-ui, sans-serif; }
        #info {
            position: fixed; top: 1rem; left: 1rem; z-index: 10;
            background: rgba(30,41,59,0.9); padding: 1rem; border-radius: 10px;
            color: #f8fafc; border: 1px solid #334155; min-width: 200px;
        }
        #info h2 { margin: 0 0 0.5rem 0; font-size: 1.2rem; color: #38bdf8; }
        #info p { margin: 0.2rem 0; font-size: 0.8rem; color: #94a3b8; }
        #detail {
            position: fixed; bottom: 1rem; right: 1rem; z-index: 10;
            background: rgba(30,41,59,0.95); padding: 1rem; border-radius: 10px;
            color: #f8fafc; border: 1px solid #334155; display: none; max-width: 300px;
        }
        #detail h3 { margin: 0 0 0.5rem 0; color: #38bdf8; }
        #detail .badge { display: inline-block; padding: 2px 8px; border-radius: 4px; font-size: 0.7rem; font-weight: 600; }
    </style>
</head>
<body>
    <div id="info">
        <h2>REQFLOW.3d</h2>
        <p>Drag to rotate, scroll to zoom</p>
        <p>Click a node for details</p>
    </div>
    <div id="detail">
        <h3 id="detail-name"></h3>
        <span class="badge" id="detail-kind"></span>
        <p id="detail-pkg" style="margin-top:0.5rem;font-size:0.8rem;color:#94a3b8;"></p>
    </div>
    <div id="graph"></div>

    <script src="https://unpkg.com/three@0.160.0/build/three.min.js"></script>
    <script src="https://unpkg.com/three-spritetext@1.8.2/dist/three-spritetext.min.js"></script>
    <script src="https://unpkg.com/3d-force-graph@1.73.3/dist/3d-force-graph.min.js"></script>

    <script>
    const data = %s;

    const kindColors = {
        handler: 0x28a745, service: 0x007bff, store: 0xffc107, model: 0xdc3545,
        event: 0xadb5bd, middleware: 0x856404, grpc: 0x0c5460, infra: 0x6c3483,
        route: 0x17a2b8, envvar: 0x20c997, table: 0xfd7e14, dependency: 0xadb5bd,
        container: 0xe83e8c, proto_rpc: 0x6610f2, proto_msg: 0xe83e8c,
        struct: 0x6c757d, interface: 0x6c757d, func: 0x6c757d
    };

    const edgeColors = {
        depends: '#64748b', implements: '#38bdf8', embeds: '#a78bfa',
        calls: '#f472b6', flows: '#22c55e', reads: '#facc15',
        maps_to: '#fb923c', publishes: '#34d399', subscribes: '#f87171',
        rpc: '#818cf8', transitive: '#475569'
    };

    const Graph = ForceGraph3D()
        (document.getElementById('graph'))
        .graphData(data)
        .backgroundColor('#0f172a')
        .nodeLabel(n => n.name + ' [' + n.kind + ']')
        .nodeColor(n => '#' + (kindColors[n.kind] || 0x6c757d).toString(16).padStart(6, '0'))
        .nodeVal(n => n.val)
        .nodeOpacity(0.9)
        .linkColor(l => edgeColors[l.kind] || '#64748b')
        .linkOpacity(0.4)
        .linkWidth(1)
        .linkDirectionalArrowLength(4)
        .linkDirectionalArrowRelPos(1)
        .onNodeClick(node => {
            const detail = document.getElementById('detail');
            document.getElementById('detail-name').textContent = node.name;
            const badge = document.getElementById('detail-kind');
            badge.textContent = node.kind;
            badge.style.background = '#' + (kindColors[node.kind] || 0x6c757d).toString(16).padStart(6, '0');
            badge.style.color = '#fff';
            document.getElementById('detail-pkg').textContent = node.pkg;
            detail.style.display = 'block';

            // Focus camera on node
            const distance = 120;
            const distRatio = 1 + distance/Math.hypot(node.x, node.y, node.z);
            Graph.cameraPosition(
                { x: node.x * distRatio, y: node.y * distRatio, z: node.z * distRatio },
                node, 1000
            );
        })
        .onBackgroundClick(() => {
            document.getElementById('detail').style.display = 'none';
        });

    // Add node text labels
    Graph.nodeThreeObject(node => {
        const sprite = new SpriteText(node.name);
        sprite.material.depthWrite = false;
        sprite.color = '#f8fafc';
        sprite.textHeight = 3;
        return sprite;
    });
    Graph.nodeThreeObjectExtend(true);
    </script>
</body>
</html>`
