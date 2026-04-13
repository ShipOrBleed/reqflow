package render

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	govis "github.com/thzgajendra/govis"
)

// InteractiveRenderer generates a self-contained HTML page focused on helping
// developers understand how a backend works — starting from API endpoints and
// tracing request flows through handlers, services, stores, and models.
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
<meta charset="utf-8"><title>Govis — Understand Your Backend</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
:root{--bg:#0f1117;--surface:#181b23;--surface2:#1f232d;--card:#1a1e28;--border:#262a36;--accent:#38bdf8;--text:#e2e8f0;--muted:#6b7280;--green:#34d399;--blue:#60a5fa;--yellow:#fbbf24;--red:#f87171;--purple:#a78bfa;--teal:#2dd4bf;--orange:#fb923c}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;background:var(--bg);color:var(--text);height:100vh;display:flex;flex-direction:column;overflow:hidden}
a{color:var(--accent);text-decoration:none}a:hover{text-decoration:underline}

/* Header */
.hdr{background:var(--surface);padding:10px 20px;border-bottom:1px solid var(--border);display:flex;align-items:center;gap:16px;flex-shrink:0}
.logo{font-size:1.15rem;font-weight:800;color:var(--accent)}.logo span{color:var(--text);font-weight:400;opacity:.6}
.tabs{display:flex;gap:2px;background:var(--bg);border-radius:8px;padding:2px}
.tab{padding:6px 16px;border-radius:6px;cursor:pointer;font-size:.75rem;font-weight:600;color:var(--muted);transition:all .15s;border:none;background:none}
.tab:hover{color:var(--text)}.tab.on{background:var(--accent);color:#0f172a}
.hdr-stats{margin-left:auto;display:flex;gap:12px;font-size:.7rem;color:var(--muted)}
.hdr-stats b{color:var(--text);margin-right:2px}

/* Layout */
.body{display:flex;flex:1;overflow:hidden}
.panel{width:320px;background:var(--surface);border-right:1px solid var(--border);display:flex;flex-direction:column;flex-shrink:0;overflow:hidden}
.panel-head{padding:14px 16px 10px;border-bottom:1px solid var(--border)}
.panel-head h3{font-size:.85rem;font-weight:700;margin-bottom:8px}
.panel-search{width:100%%;padding:7px 10px;background:var(--bg);border:1px solid var(--border);border-radius:6px;color:var(--text);font-size:.8rem;outline:none}
.panel-search:focus{border-color:var(--accent)}.panel-search::placeholder{color:var(--muted)}
.panel-list{flex:1;overflow-y:auto;padding:8px}
.main-area{flex:1;overflow-y:auto;padding:28px 36px}

/* Endpoint list items */
.ep{padding:10px 12px;border-radius:8px;cursor:pointer;margin-bottom:4px;transition:all .12s;border:1px solid transparent}
.ep:hover{background:var(--surface2);border-color:var(--border)}
.ep.active{background:var(--surface2);border-color:var(--accent)}
.ep .method{font-size:.6rem;font-weight:700;padding:2px 6px;border-radius:3px;color:#fff;display:inline-block;margin-right:6px}
.ep .method.GET{background:#059669}.ep .method.POST{background:#2563eb}.ep .method.PUT{background:#d97706}.ep .method.DELETE{background:#dc2626}.ep .method.PATCH{background:#7c3aed}
.ep .path{font-size:.8rem;font-family:monospace;color:var(--text)}
.ep .handler-name{font-size:.65rem;color:var(--muted);margin-top:3px}

/* Flow visualization */
.flow{max-width:700px}
.flow-title{font-size:1rem;font-weight:700;margin-bottom:4px}
.flow-subtitle{font-size:.75rem;color:var(--muted);margin-bottom:24px}
.step{position:relative;padding-left:48px;padding-bottom:24px}
.step:last-child{padding-bottom:0}
.step::before{content:'';position:absolute;left:18px;top:36px;bottom:0;width:2px;background:var(--border)}
.step:last-child::before{display:none}
.step-icon{position:absolute;left:4px;top:4px;width:30px;height:30px;border-radius:50%%;display:flex;align-items:center;justify-content:center;font-size:.75rem;font-weight:800;color:#fff;z-index:1}
.step-box{background:var(--card);border:1px solid var(--border);border-radius:10px;padding:14px 16px;transition:all .15s;cursor:pointer}
.step-box:hover{border-color:var(--accent);box-shadow:0 4px 16px rgba(0,0,0,.3)}
.step-box.active{border-color:var(--accent);background:#131a2a}
.step-kind{font-size:.55rem;text-transform:uppercase;font-weight:700;letter-spacing:.06em;margin-bottom:4px}
.step-name{font-size:.9rem;font-weight:700}
.step-detail{font-size:.72rem;color:var(--muted);margin-top:6px;line-height:1.5}
.step-detail code{background:var(--surface2);padding:1px 5px;border-radius:3px;font-size:.68rem}
.step-methods{display:flex;flex-wrap:wrap;gap:4px;margin-top:6px}
.step-methods span{background:var(--surface2);padding:2px 8px;border-radius:4px;font-size:.65rem;font-family:monospace;color:var(--text)}
.step-fields{margin-top:6px;display:flex;flex-wrap:wrap;gap:4px}
.step-fields span{font-size:.62rem;color:var(--muted);font-family:monospace;background:var(--bg);padding:2px 6px;border-radius:3px}
.step-arrow{color:var(--muted);font-size:.6rem;padding-left:48px;padding-bottom:4px;margin-top:-4px}
.step-used-by{margin-top:8px;padding-top:8px;border-top:1px solid var(--border)}
.step-used-by h5{font-size:.6rem;text-transform:uppercase;color:var(--muted);margin-bottom:4px}
.step-used-by a{font-size:.7rem;display:inline-block;margin-right:8px}

/* Overview grid */
.overview-section{margin-bottom:24px}
.overview-section h3{font-size:.8rem;font-weight:700;margin-bottom:10px;display:flex;align-items:center;gap:8px}
.overview-section h3 .badge{font-size:.65rem;background:var(--surface2);padding:2px 8px;border-radius:10px;color:var(--muted);font-weight:400}
.overview-grid{display:flex;flex-wrap:wrap;gap:8px}
.ov-card{background:var(--card);border:1px solid var(--border);border-radius:8px;padding:10px 14px;width:175px;cursor:pointer;transition:all .15s}
.ov-card:hover{border-color:var(--accent);transform:translateY(-1px)}
.ov-card .ov-name{font-size:.8rem;font-weight:600}
.ov-card .ov-sub{font-size:.6rem;color:var(--muted);margin-top:2px}
.ov-card.handler{border-left:3px solid var(--green)}.ov-card.service{border-left:3px solid var(--blue)}.ov-card.store{border-left:3px solid var(--yellow)}.ov-card.model{border-left:3px solid var(--red)}.ov-card.interface{border-left:3px solid var(--purple);border-style:dashed}.ov-card.grpc{border-left:3px solid var(--teal)}.ov-card.infra{border-left:3px solid var(--purple)}.ov-card.event{border-left:3px solid var(--muted)}

/* Empty state */
.empty{text-align:center;padding:60px 40px;color:var(--muted)}
.empty h2{font-size:1.1rem;color:var(--text);margin-bottom:8px}
.empty p{font-size:.85rem;line-height:1.6}

::-webkit-scrollbar{width:5px}::-webkit-scrollbar-track{background:transparent}::-webkit-scrollbar-thumb{background:var(--border);border-radius:3px}
</style>
</head>
<body>
<div class="hdr">
<div class="logo">GOVIS<span>.arch</span></div>
<div class="tabs">
<div class="tab on" onclick="setTab('explore')">Explore APIs</div>
<div class="tab" onclick="setTab('overview')">Architecture</div>
<div class="tab" onclick="setTab('packages')">Packages</div>
</div>
<div class="hdr-stats" id="hdr-stats"></div>
</div>
<div class="body">
<div class="panel" id="panel">
<div class="panel-head">
<h3 id="panel-title">API Endpoints</h3>
<input class="panel-search" id="panel-search" placeholder="Filter..." oninput="filterPanel()">
</div>
<div class="panel-list" id="panel-list"></div>
</div>
<div class="main-area" id="main"></div>
</div>
<script>
const D=%s,KC=%s;
const kindColor={handler:'#34d399',service:'#60a5fa',store:'#fbbf24',model:'#f87171',interface:'#a78bfa',grpc:'#2dd4bf',infra:'#a78bfa',event:'#6b7280',middleware:'#fb923c',func:'#6b7280',struct:'#6b7280'};
const kindEmoji={handler:'H',service:'S',store:'D',model:'M',interface:'I',grpc:'G',infra:'X',event:'E'};
const layerOrder=['handler','grpc','middleware','service','interface','event','store','model','infra'];
const NM={};D.nodes.forEach(n=>NM[n.id]=n);
const outE={},inE={};
D.edges.forEach(e=>{if(!outE[e.source])outE[e.source]=[];outE[e.source].push(e);if(!inE[e.target])inE[e.target]=[];inE[e.target].push(e)});

// Header stats
const hs=document.getElementById('hdr-stats');
['handler','service','store','model'].forEach(k=>{if(KC[k])hs.innerHTML+='<span><b style="color:'+kindColor[k]+'">'+KC[k]+'</b> '+k+'s</span>'});

let tab='explore',selectedEp=null,selectedNode=null,selectedRoute=null;

// Get full downstream chain
function getChain(id,visited){
if(!visited)visited=new Set();if(visited.has(id))return[];visited.add(id);
let chain=[id];(outE[id]||[]).forEach(e=>{if(!visited.has(e.target))chain=chain.concat(getChain(e.target,visited))});
return chain}

// Get reverse chain
function getUsedBy(id){const users=[];(inE[id]||[]).forEach(e=>{const n=NM[e.source];if(n)users.push(n)});return users}

// === TABS ===
function setTab(t){tab=t;document.querySelectorAll('.tab').forEach(b=>b.classList.remove('on'));document.querySelector('.tab[onclick*="'+t+'"]').classList.add('on');
document.getElementById('panel-search').value='';render()}

function render(){
if(tab==='explore')renderExplore();
else if(tab==='overview')renderOverview();
else if(tab==='packages')renderPackages()}

// === EXPLORE APIs ===
function renderExplore(){
document.getElementById('panel-title').textContent='API Endpoints';
const pl=document.getElementById('panel-list');pl.innerHTML='';
const main=document.getElementById('main');

// Collect all handlers with routes — each route becomes its own entry
const endpoints=[];
D.nodes.forEach(n=>{if(n.kind==='handler'){
const routes=n.meta?.routes||n.meta?.route||'';
if(routes){
routes.split('\n').forEach(route=>{
if(!route.trim())return;
const parts=route.trim().split(' ');
endpoints.push({node:n,method:parts[0]||'',path:parts.slice(1).join(' ')||route,routeLabel:route})
})}
else endpoints.push({node:n,method:'',path:n.label,routeLabel:n.label})}});

endpoints.sort((a,b)=>a.path.localeCompare(b.path));

if(endpoints.length===0){
pl.innerHTML='<div style="padding:12px;color:var(--muted);font-size:.8rem">No HTTP handlers found</div>';
main.innerHTML='<div class="empty"><h2>No API Endpoints Detected</h2><p>This service has no HTTP handlers.<br>Try the Architecture tab to explore components.</p></div>';
return}

endpoints.forEach((ep,i)=>{
const div=document.createElement('div');div.className='ep'+(selectedEp===ep.node.id?' active':'');div.dataset.id=ep.node.id;
let mclass=ep.method||'GET';
div.innerHTML=(ep.method?'<span class="method '+mclass+'">'+ep.method+'</span>':'')+
'<span class="path">'+ep.path+'</span>'+
'<div class="handler-name">'+ep.node.label+'</div>';
div.onclick=()=>{selectedEp=ep.node.id;selectedRoute=ep.routeLabel;renderExplore()};
pl.appendChild(div)});

if(!selectedEp&&endpoints.length>0){selectedEp=endpoints[0].node.id;selectedRoute=endpoints[0].routeLabel;}
// Highlight active
pl.querySelectorAll('.ep').forEach(e=>{if(e.dataset.id===selectedEp)e.classList.add('active');else e.classList.remove('active')});

if(selectedEp)renderFlow(selectedEp);
else main.innerHTML='<div class="empty"><h2>Select an Endpoint</h2><p>Click any API endpoint on the left to trace<br>its request flow through the backend.</p></div>'}

function renderFlow(handlerId){
const main=document.getElementById('main');main.innerHTML='';
const handler=NM[handlerId];if(!handler)return;
const route=selectedRoute||handler.meta?.route||handler.label;

const flow=document.createElement('div');flow.className='flow';
flow.innerHTML='<div class="flow-title">'+route+'</div><div class="flow-subtitle">Request flow — trace how this endpoint processes a request from entry to database</div>';

// Build the chain
const chainIds=getChain(handlerId);
// Order by layer
const chainNodes=chainIds.map(id=>NM[id]).filter(Boolean);
chainNodes.sort((a,b)=>layerOrder.indexOf(a.kind)-layerOrder.indexOf(b.kind));

// Remove duplicates keeping order
const seen=new Set();const ordered=[];
chainNodes.forEach(n=>{if(!seen.has(n.id)){seen.add(n.id);ordered.push(n)}});

ordered.forEach((n,i)=>{
const color=kindColor[n.kind]||'#6b7280';
const step=document.createElement('div');

// Arrow between steps
if(i>0){const arrow=document.createElement('div');arrow.className='step-arrow';
const prevKind=ordered[i-1].kind;const curKind=n.kind;
let label='calls';
if(prevKind==='handler'&&curKind==='service')label='delegates to';
else if(prevKind==='service'&&curKind==='store')label='queries via';
else if(prevKind==='service'&&curKind==='interface')label='uses interface';
else if(prevKind==='store'&&curKind==='model')label='maps to';
else if(curKind==='event')label='publishes to';
arrow.textContent='↓ '+label;
flow.appendChild(arrow)}

step.className='step';
step.innerHTML='<div class="step-icon" style="background:'+color+'">'+(kindEmoji[n.kind]||'?')+'</div>';

const box=document.createElement('div');box.className='step-box'+(n.id===selectedNode?' active':'');
box.onclick=()=>{selectedNode=n.id;renderFlow(handlerId)};

let html='<div class="step-kind" style="color:'+color+'">'+n.kind+'</div>';
html+='<div class="step-name">'+n.label+'</div>';

// Context-specific details
const details=[];
if(n.meta?.route)details.push('Route: <code>'+n.meta.route+'</code>');
if(n.pkgName)details.push('Package: <code>'+n.pkgName+'</code>');
if(n.file)details.push('File: <code>'+n.file.split('/').pop()+':'+n.line+'</code>');
if(details.length)html+='<div class="step-detail">'+details.join(' · ')+'</div>';

// Methods
if(n.methods&&n.methods.length){
html+='<div class="step-methods">';
n.methods.slice(0,6).forEach(m=>{html+='<span>'+m+'()</span>'});
if(n.methods.length>6)html+='<span>+' +(n.methods.length-6)+' more</span>';
html+='</div>'}

// Fields for models
if(n.kind==='model'&&n.fields&&n.fields.length){
html+='<div class="step-fields">';
n.fields.slice(0,8).forEach(f=>{html+='<span>'+f+'</span>'});
if(n.fields.length>8)html+='<span>+' +(n.fields.length-8)+'</span>';
html+='</div>'}

// Meta (non-standard)
const metaKeys=Object.keys(n.meta||{}).filter(k=>!['route','http_method','is_constructor','deps'].includes(k)&&n.meta[k]);
if(metaKeys.length){
html+='<div class="step-detail" style="margin-top:6px">';
metaKeys.slice(0,3).forEach(k=>{html+=k+': <code>'+n.meta[k]+'</code> '});
html+='</div>'}

// "Also used by" — show other handlers that use this component
if(n.id!==handlerId){
const usedBy=getUsedBy(n.id).filter(u=>u.id!==handlerId&&u.kind==='handler');
if(usedBy.length){html+='<div class="step-used-by"><h5>Also used by</h5>';
usedBy.slice(0,4).forEach(u=>{html+='<a href="#" onclick="event.preventDefault();selectedEp=\''+u.id.replace(/'/g,"\\'")+'\';renderExplore()">'+u.label+'</a>'});
if(usedBy.length>4)html+='<span style="color:var(--muted);font-size:.65rem">+' +(usedBy.length-4)+' more</span>';
html+='</div>'}}

box.innerHTML=html;step.appendChild(box);flow.appendChild(step)});

if(ordered.length<=1){
flow.innerHTML+='<div style="color:var(--muted);padding:20px 48px;font-size:.8rem">No downstream dependencies detected for this handler.<br>It may call external services directly or use patterns not yet detected by GoVis.</div>'}

main.appendChild(flow)}

// === OVERVIEW ===
function renderOverview(){
document.getElementById('panel-title').textContent='Components';
const pl=document.getElementById('panel-list');pl.innerHTML='';
const main=document.getElementById('main');main.innerHTML='';

// Panel: list all arch nodes
const archNodes=D.nodes.filter(n=>['handler','service','store','model','interface','grpc','infra','event','middleware'].includes(n.kind));
archNodes.sort((a,b)=>{const ki=layerOrder.indexOf(a.kind)-layerOrder.indexOf(b.kind);return ki||a.label.localeCompare(b.label)});
archNodes.forEach(n=>{
const div=document.createElement('div');div.className='ep'+(selectedNode===n.id?' active':'');div.dataset.id=n.id;
div.innerHTML='<span style="display:inline-block;width:8px;height:8px;border-radius:50%%;background:'+kindColor[n.kind]+';margin-right:8px"></span><span class="path" style="font-family:inherit">'+n.label+'</span><div class="handler-name">'+n.kind+' · '+n.pkgName+'</div>';
div.onclick=()=>{selectedNode=n.id;renderOverview()};
pl.appendChild(div)});

// Main: layered overview
layerOrder.forEach(k=>{
const nodes=D.nodes.filter(n=>n.kind===k);if(!nodes.length)return;
const sec=document.createElement('div');sec.className='overview-section';
sec.innerHTML='<h3 style="color:'+kindColor[k]+'">'+({'handler':'HTTP Handlers','service':'Services','store':'Stores','model':'Models','interface':'Interfaces','grpc':'gRPC Services','infra':'Infrastructure','event':'Events','middleware':'Middleware'}[k]||k)+'<span class="badge">'+nodes.length+'</span></h3>';
const grid=document.createElement('div');grid.className='overview-grid';
nodes.sort((a,b)=>a.label.localeCompare(b.label));
nodes.forEach(n=>{
const card=document.createElement('div');card.className='ov-card '+n.kind;
const conns=(outE[n.id]||[]).length+(inE[n.id]||[]).length;
let sub=n.pkgName;if(n.methods&&n.methods.length)sub=n.methods.slice(0,2).join(', ')+(n.methods.length>2?' +':'');
if(n.meta?.route)sub=n.meta.route;
card.innerHTML='<div class="ov-name">'+n.label+(conns?' <span style="color:var(--muted);font-size:.6rem">'+conns+'</span>':'')+'</div><div class="ov-sub">'+sub+'</div>';
card.onclick=()=>{selectedNode=n.id;renderNodeDetail(n.id)};
grid.appendChild(card)});
sec.appendChild(grid);main.appendChild(sec)});

if(selectedNode)renderNodeDetail(selectedNode)}

function renderNodeDetail(id){
const n=NM[id];if(!n)return;
// Show detail in main area below overview — scroll to it
let detail=document.getElementById('node-detail');
if(!detail){detail=document.createElement('div');detail.id='node-detail';document.getElementById('main').appendChild(detail)}
const color=kindColor[n.kind]||'#6b7280';
let h='<div style="margin-top:24px;padding:20px;background:var(--card);border:1px solid var(--border);border-radius:12px;border-left:4px solid '+color+'">';
h+='<div style="display:flex;align-items:center;gap:8px;margin-bottom:8px"><span style="background:'+color+';color:#fff;padding:2px 8px;border-radius:4px;font-size:.6rem;font-weight:700;text-transform:uppercase">'+n.kind+'</span><span style="font-size:1rem;font-weight:700">'+n.label+'</span></div>';
h+='<div style="font-size:.75rem;color:var(--muted)">'+n.pkg+(n.file?' · '+n.file.split('/').pop()+':'+n.line:'')+'</div>';
if(n.methods&&n.methods.length){h+='<div style="margin-top:10px"><span style="font-size:.6rem;color:var(--muted);text-transform:uppercase">Methods</span><div style="margin-top:4px;display:flex;flex-wrap:wrap;gap:4px">';n.methods.forEach(m=>{h+='<span style="background:var(--surface2);padding:2px 8px;border-radius:4px;font-size:.68rem;font-family:monospace">'+m+'()</span>'});h+='</div></div>'}
if(n.fields&&n.fields.length){h+='<div style="margin-top:8px"><span style="font-size:.6rem;color:var(--muted);text-transform:uppercase">Fields</span><div style="margin-top:4px;display:flex;flex-wrap:wrap;gap:4px">';n.fields.slice(0,12).forEach(f=>{h+='<span style="background:var(--bg);padding:2px 6px;border-radius:3px;font-size:.65rem;font-family:monospace;color:var(--muted)">'+f+'</span>'});h+='</div></div>'}
const outs=outE[id]||[],ins=inE[id]||[];
if(outs.length){h+='<div style="margin-top:10px"><span style="font-size:.6rem;color:var(--muted);text-transform:uppercase">Depends on ('+outs.length+')</span><div style="margin-top:4px">';outs.forEach(e=>{const t=NM[e.target];if(t)h+='<a href="#" style="margin-right:10px;font-size:.75rem" onclick="event.preventDefault();selectedNode=\''+e.target.replace(/'/g,"\\'")+'\';renderOverview()"><span style="color:'+kindColor[t.kind]+'">●</span> '+t.label+'</a>'});h+='</div></div>'}
if(ins.length){h+='<div style="margin-top:8px"><span style="font-size:.6rem;color:var(--muted);text-transform:uppercase">Used by ('+ins.length+')</span><div style="margin-top:4px">';ins.forEach(e=>{const s=NM[e.source];if(s)h+='<a href="#" style="margin-right:10px;font-size:.75rem" onclick="event.preventDefault();selectedNode=\''+e.source.replace(/'/g,"\\'")+'\';renderOverview()"><span style="color:'+kindColor[s.kind]+'">●</span> '+s.label+'</a>'});h+='</div></div>'}
h+='</div>';detail.innerHTML=h;detail.scrollIntoView({behavior:'smooth'})}

// === PACKAGES ===
function renderPackages(){
document.getElementById('panel-title').textContent='Packages';
const pl=document.getElementById('panel-list');pl.innerHTML='';
const main=document.getElementById('main');main.innerHTML='';
const pkgs={};D.nodes.forEach(n=>{if(!pkgs[n.pkgName])pkgs[n.pkgName]=[];pkgs[n.pkgName].push(n)});
Object.keys(pkgs).sort().forEach(pkg=>{
const nodes=pkgs[pkg];
const div=document.createElement('div');div.className='ep'+(selectedNode===pkg?' active':'');div.dataset.id=pkg;
const kinds={};nodes.forEach(n=>{kinds[n.kind]=(kinds[n.kind]||0)+1});
const summary=Object.entries(kinds).map(([k,v])=>'<span style="color:'+kindColor[k]+'">'+v+' '+k+'</span>').join(' · ');
div.innerHTML='<span class="path" style="font-family:inherit;font-weight:600">'+pkg+'</span><div class="handler-name">'+nodes.length+' components · '+summary+'</div>';
div.onclick=()=>{selectedNode=pkg;renderPackages()};
pl.appendChild(div)});

if(selectedNode&&pkgs[selectedNode]){
const nodes=pkgs[selectedNode];
const sec=document.createElement('div');sec.className='overview-section';
sec.innerHTML='<h3>'+selectedNode+'<span class="badge">'+nodes.length+' components</span></h3>';
// Group by kind
const byKind={};nodes.forEach(n=>{if(!byKind[n.kind])byKind[n.kind]=[];byKind[n.kind].push(n)});
layerOrder.concat(['struct','func']).forEach(k=>{
if(!byKind[k])return;
const sub=document.createElement('div');sub.style.marginBottom='16px';
sub.innerHTML='<div style="font-size:.7rem;font-weight:600;color:'+kindColor[k]+';margin-bottom:6px;text-transform:uppercase">'+k+'s ('+byKind[k].length+')</div>';
const grid=document.createElement('div');grid.className='overview-grid';
byKind[k].sort((a,b)=>a.label.localeCompare(b.label)).forEach(n=>{
const card=document.createElement('div');card.className='ov-card '+n.kind;
let sub=n.pkgName;if(n.methods?.length)sub=n.methods.slice(0,2).join(', ');if(n.meta?.route)sub=n.meta.route;
card.innerHTML='<div class="ov-name">'+n.label+'</div><div class="ov-sub">'+sub+'</div>';
card.onclick=()=>{selectedNode=n.id;tab='overview';setTab('overview')};
grid.appendChild(card)});
sub.appendChild(grid);sec.appendChild(sub)});
main.appendChild(sec)}
else{main.innerHTML='<div class="empty"><h2>Select a Package</h2><p>Click a package on the left to explore its components.</p></div>'}}

function filterPanel(){
const q=document.getElementById('panel-search').value.toLowerCase();
document.querySelectorAll('.panel-list .ep').forEach(el=>{
const text=el.textContent.toLowerCase();el.style.display=text.includes(q)?'':'none'})}

render();
</script>
</body>
</html>`
