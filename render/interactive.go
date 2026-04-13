package render

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	govis "github.com/thzgajendra/govis"
)

// InteractiveRenderer generates a self-contained HTML page with layered
// architecture visualization — handlers at top, services below, interfaces,
// stores, and models at bottom with vertical edge connections.
type InteractiveRenderer struct{}

type iNode struct {
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

type iEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Kind   string `json:"kind"`
}

type iData struct {
	Nodes []iNode `json:"nodes"`
	Edges []iEdge `json:"edges"`
}

func (ir *InteractiveRenderer) Render(g *govis.Graph, w io.Writer) error {
	d := iData{}

	for _, node := range g.Nodes {
		var fields []string
		for _, f := range node.Fields {
			fields = append(fields, f.Name)
		}
		pkgParts := strings.Split(node.Package, "/")
		pkgName := pkgParts[len(pkgParts)-1]

		d.Nodes = append(d.Nodes, iNode{
			ID: node.ID, Label: node.Name, Kind: string(node.Kind),
			Pkg: node.Package, PkgName: pkgName, File: node.File, Line: node.Line,
			Methods: node.Methods, Fields: fields, Meta: node.Meta,
		})
	}

	sort.Slice(d.Nodes, func(i, j int) bool {
		return d.Nodes[i].Label < d.Nodes[j].Label
	})

	nodeIDs := make(map[string]bool)
	for _, n := range d.Nodes {
		nodeIDs[n.ID] = true
	}
	for _, edge := range g.Edges {
		if nodeIDs[edge.From] && nodeIDs[edge.To] {
			d.Edges = append(d.Edges, iEdge{Source: edge.From, Target: edge.To, Kind: string(edge.Kind)})
		}
	}

	graphJSON, err := json.Marshal(d)
	if err != nil {
		return fmt.Errorf("marshalling graph: %w", err)
	}

	// Count by kind
	kindCounts := make(map[string]int)
	for _, n := range g.Nodes {
		kindCounts[string(n.Kind)]++
	}
	statsJSON, _ := json.Marshal(kindCounts)

	fmt.Fprintf(w, tmpl, string(graphJSON), string(statsJSON))
	return nil
}

var tmpl = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Govis Architecture</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
:root{--bg:#141820;--surface:#1c2029;--surface2:#242830;--card:#1e222a;--border:#2a2e38;--accent:#38bdf8;--text:#e2e8f0;--muted:#7a8394;--green:#34d399;--blue:#60a5fa;--yellow:#fbbf24;--red:#f87171;--purple:#a78bfa;--teal:#2dd4bf;--orange:#fb923c;--gray:#6b7280}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;background:var(--bg);color:var(--text);height:100vh;display:flex;flex-direction:column;overflow:hidden}
header{background:var(--surface);padding:10px 20px;border-bottom:1px solid var(--border);display:flex;justify-content:space-between;align-items:center;flex-shrink:0}
.logo{font-size:1.2rem;font-weight:800;color:var(--accent)}.logo span{color:var(--text);font-weight:400;opacity:.7}
.modes{display:flex;gap:4px}
.modes button{background:var(--surface2);border:1px solid var(--border);color:var(--muted);padding:6px 16px;border-radius:6px;cursor:pointer;font-size:.75rem;font-weight:600;transition:all .15s}
.modes button:hover{border-color:var(--accent);color:var(--text)}
.modes button.active{background:var(--accent);color:#0f172a;border-color:var(--accent)}
.wrap{display:flex;flex:1;overflow:hidden}
.sidebar{width:280px;background:var(--surface);border-right:1px solid var(--border);padding:16px;overflow-y:auto;flex-shrink:0;display:flex;flex-direction:column;gap:16px}
.sidebar h4{font-size:.65rem;text-transform:uppercase;letter-spacing:.08em;color:var(--muted);margin-bottom:8px}
#search{width:100%%;padding:8px 10px;background:var(--bg);border:1px solid var(--border);border-radius:6px;color:var(--text);font-size:.8rem;outline:none}
#search:focus{border-color:var(--accent)}
#search::placeholder{color:var(--muted)}
.stats{display:grid;grid-template-columns:1fr 1fr;gap:6px}
.stat{background:var(--bg);border:1px solid var(--border);border-radius:8px;padding:10px;cursor:pointer;transition:all .15s}
.stat:hover{border-color:var(--accent)}
.stat.active{border-color:var(--accent);background:#172033}
.stat h3{font-size:1.4rem;font-weight:800;line-height:1}
.stat p{font-size:.6rem;text-transform:uppercase;letter-spacing:.06em;margin-top:3px;display:flex;align-items:center;gap:4px}
.stat .dot{width:7px;height:7px;border-radius:50%%;display:inline-block}
.layer-toggles{display:flex;flex-direction:column;gap:4px}
.ltoggle{display:flex;align-items:center;gap:8px;padding:4px 0;cursor:pointer;font-size:.8rem}
.ltoggle input{accent-color:var(--accent);width:16px;height:16px}
.ltoggle .cnt{margin-left:auto;color:var(--muted);font-size:.75rem}
.detail-box{background:var(--bg);border:1px solid var(--border);border-radius:8px;padding:12px}
.detail-box .placeholder{color:var(--muted);font-size:.8rem}
.detail-box h3{font-size:.9rem;color:var(--accent);margin-bottom:4px}
.detail-badge{display:inline-block;padding:2px 8px;border-radius:4px;font-size:.6rem;text-transform:uppercase;font-weight:700;color:#fff}
.detail-section{margin-top:8px}
.detail-section h4{font-size:.6rem;text-transform:uppercase;color:var(--muted);margin-bottom:3px}
.detail-item{font-size:.75rem;color:var(--muted);padding:1px 0}
.detail-item strong{color:var(--text)}
.conn-link{color:var(--accent);cursor:pointer;font-size:.72rem}.conn-link:hover{text-decoration:underline}
.canvas-area{flex:1;overflow:auto;padding:24px 32px;position:relative}
.lane{margin-bottom:20px;position:relative}
.lane-label{font-size:.7rem;font-weight:700;text-transform:uppercase;letter-spacing:.1em;margin-bottom:8px;padding-left:2px}
.lane-label.handler{color:var(--green)}.lane-label.service{color:var(--blue)}.lane-label.store{color:var(--yellow)}.lane-label.model{color:var(--red)}.lane-label.interface{color:var(--purple)}.lane-label.grpc{color:var(--teal)}.lane-label.infra{color:var(--purple)}.lane-label.event{color:var(--gray)}.lane-label.other{color:var(--gray)}
.lane-grid{display:flex;flex-wrap:wrap;gap:10px}
.card{background:var(--card);border:1px solid var(--border);border-radius:8px;padding:10px 14px;min-width:140px;max-width:200px;cursor:pointer;transition:all .15s;position:relative}
.card:hover{border-color:var(--accent);transform:translateY(-2px);box-shadow:0 6px 20px rgba(0,0,0,.4)}
.card.selected{border-color:var(--accent);background:#172033;box-shadow:0 0 0 2px rgba(56,189,248,.2)}
.card.highlight{box-shadow:0 0 0 2px rgba(251,191,36,.4);border-color:var(--yellow)}
.card.dimmed{opacity:.2;pointer-events:none}
.card .name{font-size:.8rem;font-weight:700;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.card .sub{font-size:.6rem;color:var(--muted);margin-top:3px;line-height:1.3;max-height:2.6em;overflow:hidden}
.card .route{font-size:.6rem;color:var(--green);margin-top:3px;font-family:monospace}
.card.handler{border-left:3px solid var(--green)}.card.service{border-left:3px solid var(--blue)}.card.store{border-left:3px solid var(--yellow)}.card.model{border-left:3px solid var(--red)}.card.interface{border-left:3px solid var(--purple);border-style:dashed;border-left-style:dashed}.card.grpc{border-left:3px solid var(--teal)}.card.infra{border-left:3px solid var(--purple)}.card.event{border-left:3px solid var(--gray)}.card.middleware{border-left:3px solid var(--orange)}.card.func{border-left:3px solid var(--gray)}.card.struct{border-left:3px solid var(--gray)}
.edge-canvas{position:absolute;top:0;left:0;pointer-events:none;z-index:0}
.pkg-group{margin-bottom:16px}
.pkg-header{font-size:.7rem;font-weight:600;color:var(--muted);margin-bottom:6px;padding:4px 8px;background:var(--surface2);border-radius:4px;display:inline-block}
::-webkit-scrollbar{width:6px}::-webkit-scrollbar-track{background:transparent}::-webkit-scrollbar-thumb{background:var(--border);border-radius:3px}
</style>
</head>
<body>
<header>
<div class="logo">GOVIS<span>.arch</span></div>
<div class="modes">
<button class="active" onclick="setMode('layered')">Layered</button>
<button onclick="setMode('package')">Package</button>
<button onclick="setMode('focus')">Focus</button>
</div>
</header>
<div class="wrap">
<div class="sidebar">
<section>
<h4>Search</h4>
<input id="search" placeholder="Search nodes..." oninput="doSearch(this.value)">
</section>
<section id="stats-section">
<h4>Architecture</h4>
<div class="stats" id="stats-grid"></div>
</section>
<section>
<h4>Layer Visibility</h4>
<div class="layer-toggles" id="layer-toggles"></div>
</section>
<section>
<h4>Selected Node</h4>
<div class="detail-box" id="detail"><div class="placeholder">Click any node to inspect</div></div>
</section>
</div>
<div class="canvas-area" id="canvas"></div>
</div>
<script>
const D=%s;
const KC=%s;
const kindColor={handler:'var(--green)',service:'var(--blue)',store:'var(--yellow)',model:'var(--red)',interface:'var(--purple)',grpc:'var(--teal)',infra:'var(--purple)',event:'var(--gray)',middleware:'var(--orange)',func:'var(--gray)',struct:'var(--gray)',route:'var(--teal)',envvar:'var(--green)',table:'var(--orange)',container:'var(--purple)',dependency:'var(--gray)',proto_rpc:'var(--purple)',proto_msg:'var(--red)'};
const layerOrder=['handler','grpc','middleware','service','interface','event','store','model','infra','route','envvar','table','container','proto_rpc','proto_msg','struct','func'];
const layerLabels={handler:'HTTP Handlers',service:'Services',store:'Stores',model:'Models',interface:'Interfaces',grpc:'gRPC Services',infra:'Infrastructure',event:'Events',middleware:'Middleware',func:'Functions',struct:'Structs',route:'Routes',envvar:'Environment Variables',table:'Database Tables',container:'Containers',proto_rpc:'Proto RPCs',proto_msg:'Proto Messages'};
const archKinds=new Set(['handler','service','store','model','interface','grpc','infra','event','middleware']);
const NM={};D.nodes.forEach(n=>NM[n.id]=n);
const outE={},inE={};
D.edges.forEach(e=>{if(!outE[e.source])outE[e.source]=[];outE[e.source].push(e);if(!inE[e.target])inE[e.target]=[];inE[e.target].push(e)});
let mode='layered',selectedId=null,visibleKinds=new Set(layerOrder.filter(k=>archKinds.has(k)));

// Stats
const sg=document.getElementById('stats-grid');
const mainStats=[['handler','Handlers','var(--green)'],['service','Services','var(--blue)'],['store','Stores','var(--yellow)'],['model','Models','var(--red)']];
mainStats.forEach(([k,label,color])=>{
const c=KC[k]||0;if(!c)return;
const d=document.createElement('div');d.className='stat';d.dataset.kind=k;
d.innerHTML='<h3 style="color:'+color+'">'+c+'</h3><p><span class="dot" style="background:'+color+'"></span>'+label+'</p>';
d.onclick=()=>{document.querySelectorAll('.stat').forEach(s=>s.classList.remove('active'));d.classList.add('active');highlightKind(k)};
sg.appendChild(d)});

// Layer toggles
const lt=document.getElementById('layer-toggles');
layerOrder.forEach(k=>{const c=KC[k]||0;if(!c)return;
const l=document.createElement('label');l.className='ltoggle';
const ch=archKinds.has(k)?'checked':'';
l.innerHTML='<input type="checkbox" '+ch+' data-kind="'+k+'" onchange="toggleKind(this)"><span style="color:'+kindColor[k]+'">'+((layerLabels[k]||k).split(' ')[0])+'</span><span class="cnt">'+c+'</span>';
if(!archKinds.has(k))l.querySelector('input').checked=false;
lt.appendChild(l)});

function toggleKind(cb){if(cb.checked)visibleKinds.add(cb.dataset.kind);else visibleKinds.delete(cb.dataset.kind);render()}

function setMode(m){mode=m;document.querySelectorAll('.modes button').forEach(b=>b.classList.remove('active'));document.querySelector('.modes button[onclick*="'+m+'"]').classList.add('active');selectedId=null;render()}

function makeCard(n){
const c=document.createElement('div');c.className='card '+n.kind;c.dataset.id=n.id;
let html='<div class="name" title="'+n.label+'">'+n.label+'</div>';
const subs=[];
if(n.meta&&n.meta.route)html+='<div class="route">'+n.meta.route+'</div>';
else{
if(n.kind==='interface'&&n.methods&&n.methods.length)subs.push('«interface»');
if(n.methods&&n.methods.length)subs.push(n.methods.slice(0,3).join(' · ')+(n.methods.length>3?' +':''));
else if(n.fields&&n.fields.length)subs.push(n.fields.slice(0,3).join(' · ')+(n.fields.length>3?' +':''));
else subs.push(n.pkgName);
if(subs.length)html+='<div class="sub">'+subs.join('<br>')+'</div>';
}
c.innerHTML=html;
c.onclick=()=>{if(mode==='focus'){selectedId=n.id;render()}else showDetail(n.id)};
return c}

function render(){
const cv=document.getElementById('canvas');cv.innerHTML='';
if(mode==='layered')renderLayered(cv);
else if(mode==='package')renderPackage(cv);
else if(mode==='focus')renderFocus(cv)}

function renderLayered(cv){
const layers={};D.nodes.forEach(n=>{if(!visibleKinds.has(n.kind))return;if(!layers[n.kind])layers[n.kind]=[];layers[n.kind].push(n)});
layerOrder.forEach(k=>{const nodes=layers[k];if(!nodes||!nodes.length)return;
const lane=document.createElement('div');lane.className='lane';
const lbl=document.createElement('div');lbl.className='lane-label '+k;lbl.textContent=(layerLabels[k]||k)+' ('+nodes.length+')';
lane.appendChild(lbl);
const grid=document.createElement('div');grid.className='lane-grid';
nodes.sort((a,b)=>a.label.localeCompare(b.label));
nodes.forEach(n=>grid.appendChild(makeCard(n)));
lane.appendChild(grid);cv.appendChild(lane)})}

function renderPackage(cv){
const pkgs={};D.nodes.forEach(n=>{if(!visibleKinds.has(n.kind))return;if(!pkgs[n.pkgName])pkgs[n.pkgName]=[];pkgs[n.pkgName].push(n)});
Object.keys(pkgs).sort().forEach(pkg=>{
const grp=document.createElement('div');grp.className='pkg-group';
const hdr=document.createElement('div');hdr.className='pkg-header';hdr.textContent=pkg+' ('+pkgs[pkg].length+')';
grp.appendChild(hdr);
const grid=document.createElement('div');grid.className='lane-grid';
pkgs[pkg].sort((a,b)=>{const ki=layerOrder.indexOf(a.kind)-layerOrder.indexOf(b.kind);return ki||a.label.localeCompare(b.label)});
pkgs[pkg].forEach(n=>grid.appendChild(makeCard(n)));
grp.appendChild(grid);cv.appendChild(grp)})}

function renderFocus(cv){
if(!selectedId){cv.innerHTML='<div style="color:var(--muted);padding:40px;text-align:center;font-size:.9rem">Click any node to focus on its dependencies</div>';
const grid=document.createElement('div');grid.className='lane-grid';grid.style.marginTop='20px';
D.nodes.filter(n=>visibleKinds.has(n.kind)).sort((a,b)=>a.label.localeCompare(b.label)).forEach(n=>grid.appendChild(makeCard(n)));
cv.appendChild(grid);return}
const related=new Set([selectedId]);
(outE[selectedId]||[]).forEach(e=>related.add(e.target));
(inE[selectedId]||[]).forEach(e=>related.add(e.source));
// Second degree
related.forEach(id=>{if(id===selectedId)return;(outE[id]||[]).forEach(e=>related.add(e.target));(inE[id]||[]).forEach(e=>related.add(e.source))});
const focusNodes=D.nodes.filter(n=>related.has(n.id));
const layers={};focusNodes.forEach(n=>{if(!layers[n.kind])layers[n.kind]=[];layers[n.kind].push(n)});
layerOrder.forEach(k=>{const nodes=layers[k];if(!nodes)return;
const lane=document.createElement('div');lane.className='lane';
const lbl=document.createElement('div');lbl.className='lane-label '+k;lbl.textContent=(layerLabels[k]||k);lane.appendChild(lbl);
const grid=document.createElement('div');grid.className='lane-grid';
nodes.forEach(n=>{const c=makeCard(n);if(n.id===selectedId)c.classList.add('selected');grid.appendChild(c)});
lane.appendChild(grid);cv.appendChild(lane)});
showDetail(selectedId)}

function showDetail(id){
selectedId=id;
document.querySelectorAll('.card.selected').forEach(c=>c.classList.remove('selected'));
const el=document.querySelector('[data-id="'+CSS.escape(id)+'"]');if(el)el.classList.add('selected');
const n=NM[id];if(!n)return;
const db=document.getElementById('detail');
let h='<h3>'+n.label+'</h3><div class="detail-badge" style="background:'+kindColor[n.kind]+'">'+n.kind+'</div>';
h+='<div class="detail-section"><div class="detail-item"><strong>Package:</strong> '+n.pkgName+'</div>';
if(n.file)h+='<div class="detail-item"><strong>File:</strong> '+n.file.split('/').pop()+':'+n.line+'</div></div>';
const meta=n.meta||{};Object.keys(meta).filter(k=>meta[k]).forEach(k=>{h+='<div class="detail-item"><strong>'+k+':</strong> '+meta[k]+'</div>'});
if(n.methods&&n.methods.length){h+='<div class="detail-section"><h4>Methods ('+n.methods.length+')</h4>';n.methods.slice(0,8).forEach(m=>{h+='<div class="detail-item" style="font-family:monospace;font-size:.7rem">'+m+'()</div>'});if(n.methods.length>8)h+='<div class="detail-item">+'+(n.methods.length-8)+' more</div>';h+='</div>'}
if(n.fields&&n.fields.length){h+='<div class="detail-section"><h4>Fields ('+n.fields.length+')</h4>';n.fields.slice(0,6).forEach(f=>{h+='<div class="detail-item" style="font-family:monospace;font-size:.7rem">'+f+'</div>'});if(n.fields.length>6)h+='<div class="detail-item">+'+(n.fields.length-6)+' more</div>';h+='</div>'}
const ins=inE[id]||[],outs=outE[id]||[];
if(ins.length||outs.length){h+='<div class="detail-section"><h4>Connections ('+ins.length+' in, '+outs.length+' out)</h4>';
ins.forEach(e=>{const s=NM[e.source];h+='<div class="detail-item"><span class="conn-link" onclick="showDetail(\''+e.source.replace(/'/g,"\\'")+'\')">← '+(s?s.label:e.source)+'</span> <span style="color:var(--muted);font-size:.65rem">'+e.kind+'</span></div>'});
outs.forEach(e=>{const t=NM[e.target];h+='<div class="detail-item"><span class="conn-link" onclick="showDetail(\''+e.target.replace(/'/g,"\\'")+'\')">→ '+(t?t.label:e.target)+'</span> <span style="color:var(--muted);font-size:.65rem">'+e.kind+'</span></div>'});
h+='</div>'}
db.innerHTML=h}

function highlightKind(k){document.querySelectorAll('.card').forEach(c=>{if(c.classList.contains(k))c.classList.add('highlight');else c.classList.remove('highlight')});setTimeout(()=>document.querySelectorAll('.card.highlight').forEach(c=>c.classList.remove('highlight')),2000)}

function doSearch(q){document.querySelectorAll('.card').forEach(c=>c.classList.remove('highlight'));if(!q)return;q=q.toLowerCase();
document.querySelectorAll('.card').forEach(c=>{const nm=c.querySelector('.name').textContent.toLowerCase();if(nm.includes(q)){c.classList.add('highlight');c.scrollIntoView({behavior:'smooth',block:'center'})}});}

render();
</script>
</body>
</html>`
