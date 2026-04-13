# Govis

[![Go Reference](https://pkg.go.dev/badge/github.com/zopdev/govis.svg)](https://pkg.go.dev/github.com/zopdev/govis)
[![Go Report Card](https://goreportcard.com/badge/github.com/zopdev/govis)](https://goreportcard.com/report/github.com/zopdev/govis)
[![CI](https://github.com/thzgajendra/govis/actions/workflows/ci.yml/badge.svg)](https://github.com/thzgajendra/govis/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

**Go architecture visualizer.** Parses Go ASTs to build dependency graphs, detect architectural layers, and render interactive visualizations in 14 output formats.

```bash
go install github.com/zopdev/govis/cmd/govis@latest
cd your-go-project
govis -format interactive ./...
```

Open the generated HTML in your browser to see a force-directed, clickable architecture graph of your entire codebase.

---

## What It Does

Govis statically analyzes your Go codebase and produces:

- **Dependency graphs** with auto-detected layers (handlers, services, stores, models)
- **Interactive HTML dashboards** with drag, zoom, filter, and click-to-inspect
- **Call graphs** via SSA analysis (function-to-function call paths)
- **Data flow diagrams** (request lifecycle: handler -> service -> store)
- **API surface maps** with request/response types
- **Infrastructure topology** (Docker, K8s, env vars, database tables, go.mod tree)
- **Git insights** (code churn heatmap, contributor map, PR impact analysis)
- **Multi-repo views** (stitch microservices, proto/gRPC contracts, event topology)
- **Architecture diffs** between git branches

No competing tool combines visualization + analysis + multi-format output in a single binary with zero config.

---

## Quick Start

### Install

```bash
go install github.com/zopdev/govis/cmd/govis@latest
```

### Basic Usage

```bash
# Interactive force-directed graph (open in browser)
govis -format interactive -out arch.html ./...

# Mermaid diagram (renders in GitHub, Notion, etc.)
govis -format mermaid ./...

# Live development server (auto-reloads on refresh)
govis -serve=":8080" ./...

# Full architecture audit
govis -audit ./...
```

### Use as a Library

```go
import "github.com/zopdev/govis"

graph, err := govis.Parse(govis.ParseOptions{Dir: "."})
// graph.Nodes, graph.Edges, graph.Clusters are ready to use
```

---

## Output Formats

| Format | Flag | Description |
|--------|------|-------------|
| **Interactive** | `-format interactive` | Cytoscape.js force-directed graph with drag, zoom, filter, click-to-inspect |
| **3D** | `-format 3d` | Three.js 3D visualization with orbit controls |
| **HTML** | `-format html` | Mermaid-based dashboard with dark theme and pan/zoom |
| **Mermaid** | `-format mermaid` | Class diagram with color-coded layers and VS Code deep links |
| **Excalidraw** | `-format excalidraw` | Editable `.excalidraw` diagram for whiteboard sessions |
| **C4** | `-format c4` | PlantUML C4 model for enterprise architecture docs |
| **Markdown** | `-format markdown` | Documentation with tables, icons, and metrics |
| **DOT** | `-format dot` | Graphviz format for SVG/PNG export |
| **DSM** | `-format dsm` | Dependency Structure Matrix for coupling analysis |
| **JSON** | `-format json` | Raw graph for programmatic consumption |
| **PDF** | `-format pdf` | Graphviz-rendered PDF (requires `dot` installed) |
| **Embed** | `-format embed` | Self-contained HTML snippet for Notion/Confluence |
| **API Map** | `-format apimap` | Table of all endpoints with request/response types |
| **Data Flow** | `-format dataflow` | Mermaid sequence diagram of request lifecycle |

---

## Analysis Features

### Architecture Visualization
```bash
govis -format interactive ./...     # Force-directed graph
govis -callgraph -format mermaid    # Function call graph (SSA + CHA)
govis -dataflow -format dataflow    # Handler -> Service -> Store flows
govis -apimap -format apimap        # API surface with req/resp types
```

### Infrastructure Mapping
```bash
govis -envmap ./...                 # Environment variable usage map
govis -tablemap ./...               # Model-to-database-table mapping (GORM/sqlx)
govis -deptree ./...                # Full go.mod transitive dependency tree
govis -infratopo ./...              # Docker/K8s infrastructure topology
govis -proto ./...                  # Parse .proto files for gRPC service graph
```

### Git & Evolution
```bash
govis -churn ./...                  # Code churn heatmap (hot/warm/cold)
govis -contributors ./...           # Contributor map with bus-factor risk
govis -pr-impact main ./...         # PR impact: direct + indirect affected nodes
govis -evolution v1.0,v2.0 ./...    # Architecture timeline across git tags
```

### Architecture Linting
```bash
govis -vet="handler!store" ./...              # Block handlers from importing stores
govis -vet="pkg:cmd/api!pkg:internal/db"      # Package-level rules
```

### Multi-Repo / Microservice
```bash
# Export each service as JSON
govis -format json ./... > svc-a.json
govis -format json ./... > svc-b.json

# Stitch into one graph with cross-service edge detection
govis -stitch svc-a.json,svc-b.json -service-map -format interactive
```

### Diff & Review
```bash
# Save baseline, then compare
govis -format json ./... > baseline.json
govis -diff baseline.json -format html ./...  # Green=new, Red=removed

# AI architecture review
export OPENAI_API_KEY="sk-..."
govis -ai ./...
```

### Coverage & Quality Overlays
```bash
govis -cover cover.out -heatmap ./...   # Red/Yellow/Green coverage overlay
govis -deadcode ./...                   # Find orphaned components
govis -cycles ./...                     # Detect circular dependencies
govis -errcheck ./...                   # Find swallowed errors
govis -security ./...                   # Security anti-pattern detection
govis -techdebt ./...                   # TODO/FIXME/HACK scanner
govis -otel-trace trace.json ./...      # OpenTelemetry latency overlay
```

---

## Configuration

Create `.govis.yml` in your project root (optional):

```yaml
linter:
  vet_rules:
    - "handler!store"
    - "route!model"

parser:
  ignore_packages:
    - "vendor"
    - "generated"
  domain_naming:
    service_match: ".*(Service|UseCase|Manager)$"
    store_match: ".*Store$"
    model_match: ".*Model$"

thresholds:
  max_cycles: 5
  max_orphans: 10
  max_security_issues: 3
```

Run `govis init` to generate a starter config.

---

## GitHub Actions

Add to your CI pipeline:

```yaml
- name: Architecture Check
  run: |
    go install github.com/zopdev/govis/cmd/govis@latest
    govis -vet="handler!store" -audit ./...
```

Or use the Docker-based action:

```yaml
- uses: thzgajendra/govis@main
  with:
    dir: './...'
```

---

## Auto-Detected Layers

Govis automatically classifies nodes using naming heuristics and AST analysis:

| Layer | Detection | Color |
|-------|-----------|-------|
| Handler | `*Handler`, `*Controller`, Gin/Echo/Fiber context params | Green |
| Service | `*Service`, `*UseCase`, `*Manager` | Blue |
| Store | `*Store`, `*Repository`, `*DAO` | Yellow |
| Model | `*Model`, `*Entity`, GORM/DB tags | Red |
| Event | `.Publish()`, `.Subscribe()`, Kafka/NATS/AMQP | Gray |
| gRPC | `Register*Server()`, `Unimplemented*Server` embeds | Teal |
| Middleware | `.Use()` registrations | Amber |
| Infra | AWS/GCP/Azure SDKs from go.mod | Purple |

Override with custom regex in `.govis.yml`.

---

## Examples

Pre-generated outputs from running govis on itself:

- [`examples/govis-interactive.html`](examples/govis-interactive.html) — Interactive Cytoscape.js graph
- [`examples/govis-mermaid.md`](examples/govis-mermaid.md) — Mermaid class diagram
- [`examples/govis-docs.md`](examples/govis-docs.md) — Markdown documentation
- [`examples/govis-graph.json`](examples/govis-graph.json) — Raw JSON graph

---

## How It Works

Govis runs a **6-pass AST analysis pipeline**:

1. **Type Harvesting** — Extract structs, interfaces, functions from `go/ast`
2. **Type Resolution** — Resolve interface implementations and constructor dependencies
3. **Framework Enrichment** — Detect HTTP routes, events, middleware, gRPC, API maps
4. **Infrastructure Mapping** — Parse go.mod, Vitess schemas, Docker/K8s, env vars, DB tables
5. **Runtime Detection** — Concurrency patterns, coverage correlation, git analysis
6. **Scope Filtering** — Apply focus/filter constraints

The entire pipeline runs in a single `go/packages.Load` call with zero external dependencies beyond `golang.org/x/tools` and `gopkg.in/yaml.v3`.

---

## License

[Apache License 2.0](LICENSE)
