# govis

[![Go Reference](https://pkg.go.dev/badge/github.com/thzgajendra/govis.svg)](https://pkg.go.dev/github.com/thzgajendra/govis)
[![Go Report Card](https://goreportcard.com/badge/github.com/thzgajendra/govis)](https://goreportcard.com/report/github.com/thzgajendra/govis)
[![CI](https://github.com/thzgajendra/govis/actions/workflows/ci.yml/badge.svg)](https://github.com/thzgajendra/govis/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

**Trace any HTTP request through your Go codebase, statically.**

```bash
govis trace "POST /orders" ./...
```

Shows you the complete path: handler → service → store → database table → external calls.
No instrumentation. No runtime. Just point it at your code.

---

## The Problem

You just joined a team. There's a bug in `POST /orders`. Where do you even start?

You grep for the route, find the handler, cmd-click into the service, cmd-click again into the repo, then go look at struct tags to figure out what database table it's writing to. You do this every time, for every repo, for every bug.

**govis automates that entire workflow.**

---

## Quick Start

```bash
go install github.com/thzgajendra/govis/cmd/govis@latest
```

### Trace a request path

```bash
# Show what happens when POST /orders is called
govis trace "POST /orders" ./...

# Partial match — finds any route containing "orders"
govis trace "orders" ./...

# Output as self-contained HTML
govis trace -format html -out trace.html "POST /orders" ./...
```

**Example output:**

```
POST /orders
────────────

  [H]  OrderHandler          HTTP Handler · handler/orders.go:45
       …/internal/handler
       Methods: CreateOrder(), GetOrder(), ListOrders()

  │
  ↓  delegates to
  │
  [S]  OrderService          Service · service/orders.go:23
       …/internal/service
       Methods: Create(), FindByID(), List(), Cancel()

  │
  ↓  queries via
  │
  [D]  OrderStore            Store / Repository · store/orders.go:67
       …/internal/store
       Methods: Insert(), Select(), Update(), Delete()

  │
  ↓  maps to model
  │
  [M]  Order                 Data Model · model/order.go:12
       …/internal/model
       Fields: ID, CustomerID, Status, Total, CreatedAt

  ┌─ Database tables
  │   orders
  └─
```

---

## Interactive Explorer

For a full visual overview of your codebase:

```bash
govis -format interactive -out explorer.html ./...
open explorer.html
```

**Explore APIs tab** (default) — lists every HTTP endpoint. Click any to trace its flow step-by-step through the codebase.

**Architecture tab** — layered view: Handlers → Services → Stores → Models, click any node for full detail.

**Packages tab** — browse by Go package, see what components live in each one.

---

## All Commands

### Trace (primary feature)

```bash
govis trace [flags] <route> [packages]

Flags:
  -format text|html    Output format (default: text)
  -out <file>          Write to file instead of stdout
  -tablemap            Resolve model → database table mappings
  -envmap              Resolve environment variable reads

Examples:
  govis trace "POST /orders" ./...
  govis trace "/orders" ./...                     # path-only, any method
  govis trace -format html -out t.html "orders"   # partial match, HTML output
```

### Visualize

```bash
govis [flags] [packages]

Output formats (-format):
  interactive   Clickable explorer — Explore APIs / Architecture / Packages (default for browsers)
  mermaid       Mermaid class diagram
  html          Static HTML with embedded Mermaid
  json          Raw graph JSON
  markdown      Markdown table
  c4            C4 PlantUML
  dot           Graphviz DOT
  dsm           Dependency Structure Matrix
  excalidraw    Excalidraw JSON
  pdf           PDF via Graphviz (falls back to DOT if not installed)
  embed         Embeddable HTML snippet (no CDN deps)
  3d            3D force-directed graph (Three.js)
  apimap        API surface map with request/response types
  dataflow      Mermaid sequence diagram of request flows

Common flags:
  -out <file>          Write output to file
  -filter <pkg>        Filter by package path substring
  -focus <name>        Focus on one component and its neighbors
  -callgraph           Include function-to-function call edges
  -tablemap            Resolve model → database table mappings
  -envmap              Map environment variable usage
  -deptree             Include full go.mod transitive dependency tree
  -infratopo           Parse Docker/K8s topology
  -churn               Overlay git commit frequency
  -contributors        Show primary contributor per component
  -pr-impact <ref>     Show which paths are affected by this PR
  -stitch <files>      Combine multiple repos into one graph
  -service-map         Detect cross-service HTTP/gRPC calls
```

---

## Supported Frameworks

Route and handler detection works out of the box for:

| Framework | Handler signature |
|-----------|-------------------|
| [GoFr](https://gofr.dev) | `func(ctx *gofr.Context) (any, error)` |
| [Gin](https://gin-gonic.com) | `func(c *gin.Context)` |
| [Echo](https://echo.labstack.com) | `func(c echo.Context) error` |
| [Fiber](https://gofiber.io) | `func(c *fiber.Ctx) error` |
| net/http | `func(w http.ResponseWriter, r *http.Request)` |

Store detection works for: `*sql.DB`, `*sqlx.DB`, `*gorm.DB`, `*mongo.Client`, `*redis.Client`, `*pgxpool.Pool`, and more — detected from struct field types, not naming conventions.

---

## How It Works

govis uses Go's type system — not grep, not regexes. It loads your packages with `golang.org/x/tools/go/packages`, walks the AST, and:

1. **Classifies types structurally**: a store is a struct that holds a `*sql.DB` (not one named `*Store`), a handler is a struct whose methods accept a framework context
2. **Builds dependency edges** from struct fields and constructor parameters
3. **Extracts route registrations** from `app.GET("/path", h.Handler)` calls
4. **Traces the path** from the matched handler through all reachable dependencies, ordered by architectural layer

Works offline, in CI, and on codebases you've never seen before.

---

## Configuration

Create `.govis.yml` in your project root:

```yaml
parser:
  ignore_packages:
    - vendor
    - _test

layers:
  service_pattern: ".*Service$"
  store_pattern:   ".*Store$|.*Repository$|.*Repo$"
  model_pattern:   ".*Model$|.*Entity$"
```

Generate a starter config:

```bash
govis init
```

---

## Install

```bash
# Latest
go install github.com/thzgajendra/govis/cmd/govis@latest

# Specific version
go install github.com/thzgajendra/govis/cmd/govis@v0.2.0
```

Or download a binary from [Releases](https://github.com/thzgajendra/govis/releases).

---

## License

Apache 2.0 — see [LICENSE](LICENSE)
