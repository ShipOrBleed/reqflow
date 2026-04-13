# reqflow — Advanced Features

This document covers output formats, analysis passes, and multi-repo features. For the core trace feature, see [README.md](README.md).

---

## Output Formats

```bash
reqflow -format <fmt> [flags] [packages]
```

| Format | Description |
|--------|-------------|
| `interactive` | Clickable HTML explorer — Explore APIs / Architecture / Packages tabs |
| `mermaid` | Mermaid class diagram |
| `html` | Static HTML with embedded Mermaid |
| `json` | Raw graph JSON |
| `markdown` | Markdown table |
| `c4` | C4 PlantUML diagram |
| `dot` | Graphviz DOT |
| `dsm` | Dependency Structure Matrix |
| `excalidraw` | Excalidraw JSON (paste into excalidraw.com) |
| `pdf` | PDF via Graphviz (falls back to DOT if not installed) |
| `embed` | Embeddable HTML snippet — no CDN deps, paste into Notion/Confluence |
| `3d` | 3D force-directed graph (Three.js) |
| `apimap` | API surface map with request/response types |
| `dataflow` | Mermaid sequence diagram of request data flows |

### Common flags

```bash
-out <file>        Write output to file instead of stdout
-filter <pkg>      Filter nodes by package path substring
-focus <name>      Keep only a component and its direct neighbors
```

---

## Analysis Passes

These flags enable additional analysis during parsing:

```bash
-callgraph         Add function-to-function call edges (uses SSA + CHA)
-tablemap          Map struct tags to database table names (gorm/sqlx/bson)
-envmap            Map os.Getenv / viper calls to environment variable nodes
-deptree           Include full go.mod transitive dependency tree as nodes
-infratopo         Parse Dockerfile, docker-compose.yml, K8s manifests
-heatmap           Overlay Go coverage data as node colors (-cover profile.out)
```

---

## Git-Based Analysis

```bash
-churn             Overlay git commit frequency as a heatmap on nodes
-contributors      Tag each node with its primary contributor and bus-factor risk
-pr-impact <ref>   Show which request paths are affected by the current PR
                   Example: -pr-impact main
```

---

## Evolution Timeline

Track how your architecture changes across git tags:

```bash
reqflow -evolution v1.0,v1.5,v2.0 -format html -out timeline.html ./...
```

Creates an HTML timeline showing nodes added/removed between versions.

---

## Multi-Repo / Microservice

### Stitch multiple repos

Combine multiple services into one graph:

```bash
# First, export each service
reqflow -format json -out svc-a.json ./services/a
reqflow -format json -out svc-b.json ./services/b

# Then stitch them together
reqflow -stitch svc-a.json,svc-b.json -format interactive -out combined.html

# With cross-service edge detection (HTTP client → matching handler routes)
reqflow -stitch svc-a.json,svc-b.json -service-map -format interactive -out combined.html
```

---

## OpenTelemetry Overlay

Overlay real latency and error data from a live trace export onto the static graph:

```bash
reqflow -otel-trace trace.json -format interactive -out annotated.html ./...
```

Expects an OTLP JSON export. Span operations are matched to graph nodes by route/function name. Nodes are tagged with `otel_avg_duration`, `otel_p99`, `otel_error_rate`.

---

## Proto / gRPC

Parse `.proto` files and cross-reference with Go gRPC registrations:

```bash
reqflow -proto -format interactive -out grpc.html ./...
```

Creates `KindProtoRPC` and `KindProtoMsg` nodes linked to Go handler nodes.

---

## Architecture Linting

```bash
# Fail if any handler directly calls a store (bypassing the service layer)
reqflow -vet "handler!store" ./...

# Package-level rule
reqflow -vet "pkg:handler!pkg:store" ./...

# Detect circular dependencies
reqflow -cycles ./...

# Detect unused/orphaned components
reqflow -deadcode ./...
```

Exit code 1 on violations — suitable for CI.

---

## Live Server

Watch for file changes and serve a live-updating visualization:

```bash
reqflow -serve :8080 ./...
# Open http://localhost:8080
```

---

## Configuration Reference

`.reqflow.yml` full schema:

```yaml
parser:
  ignore_packages:
    - vendor
    - _test
    - mock

layers:
  service_pattern: ".*Service$"
  store_pattern:   ".*Store$|.*Repository$|.*Repo$"
  model_pattern:   ".*Model$|.*Entity$"

linter:
  vet_rules:
    - "handler!store"
  thresholds:
    max_fan_in: 10
    max_fan_out: 10
    max_instability: 0.9
```

JSON Schema for IDE autocomplete: [reqflow.schema.json](reqflow.schema.json)
