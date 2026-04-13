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
#search:focus{border-color:var(--accent)}#search::placeholder{color:var(--muted)}
.stats{display:grid;grid-template-columns:1fr 1fr;gap:6px}
.stat{background:var(--bg);border:1px solid var(--border);border-radius:8px;padding:10px;cursor:pointer;transition:all .15s}
.stat:hover{border-color:var(--accent)}.stat.active{border-color:var(--accent);background:#172033}
.stat h3{font-size:1.4rem;font-weight:800;line-height:1}
.stat p{font-size:.6rem;text-transform:uppercase;letter-spacing:.06em;margin-top:3px;display:flex;align-items:center;gap:4px}
.stat .dot{width:7px;height:7px;border-radius:50%%;display:inline-block}
.layer-toggles{display:flex;flex-direction:column;gap:4px}
.ltoggle{display:flex;align-items:center;gap:8px;padding:4px 0;cursor:pointer;font-size:.8rem}
.ltoggle input{accent-color:var(--accent);width:16px;height:16px}
.ltoggle .cnt{margin-left:auto;color:var(--muted);font-size:.75rem}
.detail-box{background:var(--bg);border:1px solid var(--border);border-radius:8px;padding:12px;max-height:40vh;overflow-y:auto}
.detail-box .placeholder{color:var(--muted);font-size:.8rem}
.detail-box h3{font-size:.9rem;color:var(--accent);margin-bottom:4px}
.detail-badge{display:inline-block;padding:2px 8px;border-radius:4px;font-size:.6rem;text-transform:uppercase;font-weight:700;color:#fff}
.detail-section{margin-top:8px}.detail-section h4{font-size:.6rem;text-transform:uppercase;color:var(--muted);margin-bottom:3px}
.detail-item{font-size:.75rem;color:var(--muted);padding:1px 0}.detail-item strong{color:var(--text)}
.conn-link{color:var(--accent);cursor:pointer;font-size:.72rem}.conn-link:hover{text-decoration:underline}
.canvas-area{flex:1;overflow:auto;padding:24px 32px;position:relative}
.lane{margin-bottom:8px;position:relative}
.lane-label{font-size:.7rem;font-weight:700;text-transform:uppercase;letter-spacing:.1em;margin-bottom:8px;padding-left:2px}
.lane-label.handler{color:var(--green)}.lane-label.service{color:var(--blue)}.lane-label.store{color:var(--yellow)}.lane-label.model{color:var(--red)}.lane-label.interface{color:var(--purple)}.lane-label.grpc{color:var(--teal)}.lane-label.infra{color:var(--purple)}.lane-label.event{color:var(--gray)}.lane-label.other{color:var(--gray)}
.lane-grid{display:flex;flex-wrap:wrap;gap:8px;padding-bottom:12px;border-bottom:1px solid var(--border);margin-bottom:8px}
.card{background:var(--card);border:1px solid var(--border);border-radius:8px;padding:10px 14px;width:180px;cursor:pointer;transition:all .15s;position:relative}
.card:hover{border-color:var(--accent);transform:translateY(-2px);box-shadow:0 6px 20px rgba(0,0,0,.4)}
.card.selected{border-color:var(--accent);background:#172033;box-shadow:0 0 0 2px rgba(56,189,248,.2)}
.card.highlight{box-shadow:0 0 0 2px rgba(251,191,36,.4);border-color:var(--yellow)}
.card.chain{box-shadow:0 0 0 2px rgba(56,189,248,.35);border-color:var(--accent);background:#15202e}
.card.dimmed{opacity:.15}
.card .name{font-size:.8rem;font-weight:700;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.card .sub{font-size:.6rem;color:var(--muted);margin-top:3px;line-height:1.3;max-height:2.6em;overflow:hidden}
.card .route{font-size:.6rem;color:var(--green);margin-top:3px;font-family:monospace}
.card .conn-count{position:absolute;top:6px;right:8px;font-size:.55rem;color:var(--muted);background:var(--surface2);padding:1px 5px;border-radius:8px}
.card.handler{border-left:3px solid var(--green)}.card.service{border-left:3px solid var(--blue)}.card.store{border-left:3px solid var(--yellow)}.card.model{border-left:3px solid var(--red)}.card.interface{border-left:3px solid var(--purple);border-style:dashed;border-left-style:dashed}.card.grpc{border-left:3px solid var(--teal)}.card.infra{border-left:3px solid var(--purple)}.card.event{border-left:3px solid var(--gray)}.card.middleware{border-left:3px solid var(--orange)}.card.func{border-left:3px solid var(--gray)}.card.struct{border-left:3px solid var(--gray)}
svg.edges{position:absolute;top:0;left:0;width:100%%;height:100%%;pointer-events:none;z-index:1}
svg.edges line{stroke-width:1.5;opacity:.35}
svg.edges line.active{stroke-width:2.5;opacity:.9}
.flow-item{display:flex;align-items:center;gap:6px;padding:6px 8px;background:var(--surface2);border-radius:6px;cursor:pointer;margin-bottom:4px;font-size:.75rem;transition:all .15s}
.flow-item:hover{background:var(--border)}.flow-item .arrow{color:var(--muted);font-size:.6rem}
.flow-item .fkind{font-size:.55rem;padding:1px 5px;border-radius:3px;color:#fff}
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
<button onclick="setMode('flows')">API Flows</button>
</div>
</header>
<div class="wrap">
<div class="sidebar">
<section><h4>Search</h4><input id="search" placeholder="Search nodes..." oninput="doSearch(this.value)"></section>
<section><h4>Architecture</h4><div class="stats" id="stats-grid"></div></section>
<section><h4>Layer Visibility</h4><div class="layer-toggles" id="layer-toggles"></div></section>
<section><h4>Selected Node</h4><div class="detail-box" id="detail"><div class="placeholder">Click any node to inspect</div></div></section>
</div>
<div class="canvas-area" id="canvas"></div>
</div>
<script>
const D=%s;
const KC=%s;
const kindColor={handler:'#34d399',service:'#60a5fa',store:'#fbbf24',model:'#f87171',interface:'#a78bfa',grpc:'#2dd4bf',infra:'#a78bfa',event:'#6b7280',middleware:'#fb923c',func:'#6b7280',struct:'#6b7280',route:'#2dd4bf',envvar:'#34d399',table:'#fb923c',container:'#a78bfa',dependency:'#6b7280',proto_rpc:'#a78bfa',proto_msg:'#f87171'};
const layerOrder=['handler','grpc','middleware','service','interface','event','store','model','infra','route','envvar','table','container','proto_rpc','proto_msg','struct','func'];
const layerLabels={handler:'HTTP Handlers',service:'Services',store:'Stores',model:'Models',interface:'Interfaces',grpc:'gRPC Services',infra:'Infrastructure',event:'Events',middleware:'Middleware',func:'Functions',struct:'Structs',route:'Routes',envvar:'Env Vars',table:'Tables',container:'Containers',proto_rpc:'Proto RPCs',proto_msg:'Proto Messages'};
const archKinds=new Set(['handler','service','store','model','interface','grpc','infra','event','middleware']);
const NM={};D.nodes.forEach(n=>NM[n.id]=n);
const outE={},inE={};
D.edges.forEach(e=>{if(!outE[e.source])outE[e.source]=[];outE[e.source].push(e);if(!inE[e.target])inE[e.target]=[];inE[e.target].push(e)});
let mode='layered',selectedId=null,visibleKinds=new Set(layerOrder.filter(k=>archKinds.has(k)));

// Build full dependency chain from a node (BFS downward)
function getChain(id){
const chain=new Set([id]);const q=[id];
while(q.length){const cur=q.shift();(outE[cur]||[]).forEach(e=>{if(!chain.has(e.target)){chain.add(e.target);q.push(e.target)}})}
return chain}

// Build reverse chain (who depends on this)
function getReverseChain(id){
const chain=new Set([id]);const q=[id];
while(q.length){const cur=q.shift();(inE[cur]||[]).forEach(e=>{if(!chain.has(e.source)){chain.add(e.source);q.push(e.source)}})}
return chain}

// Stats
const sg=document.getElementById('stats-grid');
[['handler','Handlers'],['service','Services'],['store','Stores'],['model','Models']].forEach(([k,label])=>{
const c=KC[k]||0;if(!c)return;
const d=document.createElement('div');d.className='stat';d.dataset.kind=k;
d.innerHTML='<h3 style="color:'+kindColor[k]+'">'+c+'</h3><p><span class="dot" style="background:'+kindColor[k]+'"></span>'+label+'</p>';
d.onclick=()=>{document.querySelectorAll('.stat').forEach(s=>s.classList.remove('active'));d.classList.add('active');highlightKind(k)};
sg.appendChild(d)});

// Layer toggles
const lt=document.getElementById('layer-toggles');
layerOrder.forEach(k=>{const c=KC[k]||0;if(!c)return;
const l=document.createElement('label');l.className='ltoggle';
l.innerHTML='<input type="checkbox" '+(archKinds.has(k)?'checked':'')+' data-kind="'+k+'" onchange="toggleKind(this)"><span style="color:'+kindColor[k]+'">'+(layerLabels[k]||k).split(' ')[0]+'</span><span class="cnt">'+c+'</span>';
lt.appendChild(l)});

function toggleKind(cb){if(cb.checked)visibleKinds.add(cb.dataset.kind);else visibleKinds.delete(cb.dataset.kind);render()}
function setMode(m){mode=m;document.querySelectorAll('.modes button').forEach(b=>b.classList.remove('active'));document.querySelector('.modes button[onclick*="'+m+'"]').classList.add('active');selectedId=null;render()}

function makeCard(n,extra){
const c=document.createElement('div');c.className='card '+n.kind+(extra||'');c.dataset.id=n.id;
const conns=(outE[n.id]||[]).length+(inE[n.id]||[]).length;
let html='<div class="name" title="'+n.label+'">'+n.label+'</div>';
if(conns>0)html+='<span class="conn-count">'+conns+'</span>';
if(n.meta&&n.meta.route)html+='<div class="route">'+n.meta.route+'</div>';
else{const subs=[];
if(n.kind==='interface')subs.push('«interface»');
if(n.methods&&n.methods.length)subs.push(n.methods.slice(0,3).join(' · ')+(n.methods.length>3?' +'+( n.methods.length-3):''));
else if(n.fields&&n.fields.length)subs.push(n.fields.slice(0,3).join(' · ')+(n.fields.length>3?' +':''));
if(subs.length)html+='<div class="sub">'+subs.join(' ')+'</div>';
else html+='<div class="sub">'+n.pkgName+'</div>';}
c.innerHTML=html;
c.onclick=()=>{showDetail(n.id);if(mode==='focus'){selectedId=n.id;render()}};
c.onmouseenter=()=>{if(mode==='layered')traceChain(n.id)};
c.onmouseleave=()=>{if(mode==='layered')clearTrace()};
return c}

function traceChain(id){
const chain=getChain(id);const rev=getReverseChain(id);
const full=new Set([...chain,...rev]);
document.querySelectorAll('.card').forEach(c=>{if(!full.has(c.dataset.id))c.classList.add('dimmed');else c.classList.add('chain')});
document.querySelectorAll('svg.edges line').forEach(l=>{
if(full.has(l.dataset.from)&&full.has(l.dataset.to))l.classList.add('active')})}

function clearTrace(){
document.querySelectorAll('.card.dimmed').forEach(c=>c.classList.remove('dimmed'));
document.querySelectorAll('.card.chain').forEach(c=>c.classList.remove('chain'));
document.querySelectorAll('svg.edges line.active').forEach(l=>l.classList.remove('active'))}

function render(){
const cv=document.getElementById('canvas');cv.innerHTML='';
if(mode==='layered')renderLayered(cv);
else if(mode==='package')renderPackage(cv);
else if(mode==='focus')renderFocus(cv);
else if(mode==='flows')renderFlows(cv)}

function renderLayered(cv){
const layers={};D.nodes.forEach(n=>{if(!visibleKinds.has(n.kind))return;if(!layers[n.kind])layers[n.kind]=[];layers[n.kind].push(n)});
layerOrder.forEach(k=>{const nodes=layers[k];if(!nodes||!nodes.length)return;
const lane=document.createElement('div');lane.className='lane';
lane.innerHTML='<div class="lane-label '+k+'">'+(layerLabels[k]||k)+' ('+nodes.length+')</div>';
const grid=document.createElement('div');grid.className='lane-grid';
nodes.sort((a,b)=>a.label.localeCompare(b.label));
nodes.forEach(n=>grid.appendChild(makeCard(n)));
lane.appendChild(grid);cv.appendChild(lane)});
// Draw SVG edges after layout
requestAnimationFrame(()=>drawEdges(cv))}

function drawEdges(cv){
const existing=cv.querySelector('svg.edges');if(existing)existing.remove();
const rect=cv.getBoundingClientRect();
const svg=document.createElementNS('http://www.w3.org/2000/svg','svg');
svg.setAttribute('class','edges');svg.style.width=cv.scrollWidth+'px';svg.style.height=cv.scrollHeight+'px';
D.edges.forEach(e=>{
const fromEl=cv.querySelector('[data-id="'+CSS.escape(e.source)+'"]');
const toEl=cv.querySelector('[data-id="'+CSS.escape(e.target)+'"]');
if(!fromEl||!toEl)return;
const fr=fromEl.getBoundingClientRect();const tr=toEl.getBoundingClientRect();
const sx=fr.left+fr.width/2-rect.left+cv.scrollLeft;
const sy=fr.top+fr.height-rect.top+cv.scrollTop;
const tx=tr.left+tr.width/2-rect.left+cv.scrollLeft;
const ty=tr.top-rect.top+cv.scrollTop;
const line=document.createElementNS('http://www.w3.org/2000/svg','line');
line.setAttribute('x1',sx);line.setAttribute('y1',sy);
line.setAttribute('x2',tx);line.setAttribute('y2',ty);
line.setAttribute('stroke',kindColor[NM[e.source]?.kind]||'#475569');
line.dataset.from=e.source;line.dataset.to=e.target;
svg.appendChild(line)});
cv.style.position='relative';cv.insertBefore(svg,cv.firstChild)}

function renderPackage(cv){
const pkgs={};D.nodes.forEach(n=>{if(!visibleKinds.has(n.kind))return;if(!pkgs[n.pkgName])pkgs[n.pkgName]=[];pkgs[n.pkgName].push(n)});
Object.keys(pkgs).sort().forEach(pkg=>{
const grp=document.createElement('div');grp.className='pkg-group';
grp.innerHTML='<div class="pkg-header">'+pkg+' ('+pkgs[pkg].length+')</div>';
const grid=document.createElement('div');grid.className='lane-grid';
pkgs[pkg].sort((a,b)=>{const ki=layerOrder.indexOf(a.kind)-layerOrder.indexOf(b.kind);return ki||a.label.localeCompare(b.label)});
pkgs[pkg].forEach(n=>grid.appendChild(makeCard(n)));
grp.appendChild(grid);cv.appendChild(grp)})}

function renderFocus(cv){
if(!selectedId){cv.innerHTML='<div style="color:var(--muted);padding:40px;text-align:center;font-size:.9rem">Click any node to see its full dependency chain</div>';
const grid=document.createElement('div');grid.className='lane-grid';grid.style.marginTop='20px';
D.nodes.filter(n=>visibleKinds.has(n.kind)).sort((a,b)=>a.label.localeCompare(b.label)).forEach(n=>grid.appendChild(makeCard(n)));
cv.appendChild(grid);return}
const down=getChain(selectedId);const up=getReverseChain(selectedId);
const all=new Set([...down,...up]);
const focusNodes=D.nodes.filter(n=>all.has(n.id));
const layers={};focusNodes.forEach(n=>{if(!layers[n.kind])layers[n.kind]=[];layers[n.kind].push(n)});
// Title
cv.innerHTML='<div style="margin-bottom:16px"><span style="color:var(--accent);font-weight:700">'+NM[selectedId].label+'</span><span style="color:var(--muted);margin-left:8px">dependency chain ('+all.size+' nodes)</span><button onclick="selectedId=null;render()" style="margin-left:12px;background:var(--surface2);border:1px solid var(--border);color:var(--muted);padding:3px 10px;border-radius:4px;cursor:pointer;font-size:.7rem">Clear</button></div>';
layerOrder.forEach(k=>{const nodes=layers[k];if(!nodes)return;
const lane=document.createElement('div');lane.className='lane';
lane.innerHTML='<div class="lane-label '+k+'">'+(layerLabels[k]||k)+'</div>';
const grid=document.createElement('div');grid.className='lane-grid';
nodes.forEach(n=>grid.appendChild(makeCard(n,n.id===selectedId?' selected':'')));
lane.appendChild(grid);cv.appendChild(lane)});
requestAnimationFrame(()=>drawEdges(cv));
showDetail(selectedId)}

function renderFlows(cv){
// Show each handler's full request flow as a visual chain
const handlers=D.nodes.filter(n=>n.kind==='handler');
if(!handlers.length){cv.innerHTML='<div style="color:var(--muted);padding:40px">No HTTP handlers found</div>';return}
cv.innerHTML='<div style="margin-bottom:16px;color:var(--muted);font-size:.8rem">Click any API endpoint to trace the full request flow through services, stores, and models.</div>';
handlers.sort((a,b)=>{const ar=a.meta?.route||a.label;const br=b.meta?.route||b.label;return ar.localeCompare(br)});
handlers.forEach(h=>{
const chain=getChain(h.id);
const chainNodes=D.nodes.filter(n=>chain.has(n.id)&&n.id!==h.id);
const div=document.createElement('div');div.style.cssText='margin-bottom:16px;background:var(--surface);border:1px solid var(--border);border-radius:10px;padding:14px;cursor:pointer';
div.onclick=()=>{selectedId=h.id;setMode('focus')};
let title=h.meta?.route||h.label;
div.innerHTML='<div style="font-weight:700;font-size:.85rem;color:var(--green);margin-bottom:8px">'+title+'</div>';
if(chainNodes.length===0){div.innerHTML+='<div style="color:var(--muted);font-size:.75rem">No downstream dependencies detected</div>';cv.appendChild(div);return}
// Show the chain as a flow: handler → service → store → model
const flow=document.createElement('div');flow.style.cssText='display:flex;flex-wrap:wrap;align-items:center;gap:6px';
flow.innerHTML='<span class="fkind" style="background:'+kindColor.handler+'">'+h.label+'</span>';
// Group chain by kind
const byKind={};chainNodes.forEach(n=>{if(!byKind[n.kind])byKind[n.kind]=[];byKind[n.kind].push(n)});
layerOrder.forEach(k=>{if(!byKind[k])return;
flow.innerHTML+='<span class="arrow" style="color:var(--muted)">→</span>';
byKind[k].forEach(n=>{flow.innerHTML+='<span class="fkind" style="background:'+kindColor[k]+'">'+n.label+'</span>'})});
div.appendChild(flow);cv.appendChild(div)})}

function showDetail(id){
selectedId=id;
document.querySelectorAll('.card.selected').forEach(c=>c.classList.remove('selected'));
const el=document.querySelector('[data-id="'+CSS.escape(id)+'"]');if(el)el.classList.add('selected');
const n=NM[id];if(!n)return;
const db=document.getElementById('detail');
let h='<h3>'+n.label+'</h3><div class="detail-badge" style="background:'+kindColor[n.kind]+'">'+n.kind+'</div>';
h+='<div class="detail-section"><div class="detail-item"><strong>Pkg:</strong> '+n.pkgName+'</div>';
if(n.file)h+='<div class="detail-item"><strong>File:</strong> '+n.file.split('/').pop()+':'+n.line+'</div></div>';
const meta=n.meta||{};const mk=Object.keys(meta).filter(k=>meta[k]);
if(mk.length){h+='<div class="detail-section"><h4>Metadata</h4>';mk.forEach(k=>{h+='<div class="detail-item"><strong>'+k+':</strong> '+meta[k]+'</div>'});h+='</div>'}
if(n.methods&&n.methods.length){h+='<div class="detail-section"><h4>Methods ('+n.methods.length+')</h4>';n.methods.slice(0,8).forEach(m=>{h+='<div class="detail-item" style="font-family:monospace;font-size:.7rem">'+m+'()</div>'});if(n.methods.length>8)h+='<div class="detail-item">+'+(n.methods.length-8)+' more</div>';h+='</div>'}
if(n.fields&&n.fields.length){h+='<div class="detail-section"><h4>Fields ('+n.fields.length+')</h4>';n.fields.slice(0,6).forEach(f=>{h+='<div class="detail-item" style="font-family:monospace;font-size:.7rem">'+f+'</div>'});if(n.fields.length>6)h+='<div class="detail-item">+'+(n.fields.length-6)+' more</div>';h+='</div>'}
const ins=inE[id]||[],outs=outE[id]||[];
if(ins.length||outs.length){h+='<div class="detail-section"><h4>Connections ('+ins.length+' in, '+outs.length+' out)</h4>';
ins.forEach(e=>{const s=NM[e.source];h+='<div class="detail-item"><span class="conn-link" onclick="event.stopPropagation();showDetail(\''+e.source.replace(/'/g,"\\'")+'\')">← '+(s?s.label:e.source)+'</span> <span style="color:var(--muted);font-size:.6rem">'+e.kind+'</span></div>'});
outs.forEach(e=>{const t=NM[e.target];h+='<div class="detail-item"><span class="conn-link" onclick="event.stopPropagation();showDetail(\''+e.target.replace(/'/g,"\\'")+'\')">→ '+(t?t.label:e.target)+'</span> <span style="color:var(--muted);font-size:.6rem">'+e.kind+'</span></div>'});
h+='</div>'}
db.innerHTML=h}

function highlightKind(k){document.querySelectorAll('.card').forEach(c=>{if(c.classList.contains(k))c.classList.add('highlight');else c.classList.remove('highlight')});setTimeout(()=>document.querySelectorAll('.card.highlight').forEach(c=>c.classList.remove('highlight')),2000)}

function doSearch(q){document.querySelectorAll('.card').forEach(c=>c.classList.remove('highlight'));if(!q)return;q=q.toLowerCase();
let first=true;document.querySelectorAll('.card').forEach(c=>{const nm=c.querySelector('.name').textContent.toLowerCase();if(nm.includes(q)){c.classList.add('highlight');if(first){c.scrollIntoView({behavior:'smooth',block:'center'});first=false}}})}

render();
</script>
</body>
</html>`
