# reqflow

[![Go Reference](https://pkg.go.dev/badge/github.com/thzgajendra/reqflow.svg)](https://pkg.go.dev/github.com/thzgajendra/reqflow)
[![Go Report Card](https://goreportcard.com/badge/github.com/thzgajendra/reqflow)](https://goreportcard.com/report/github.com/thzgajendra/reqflow)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

**Trace any HTTP request through your Go codebase, statically.**

One command. No instrumentation. No runtime. Just point it at your code.

```bash
go install github.com/thzgajendra/reqflow/cmd/reqflow@latest
```

---

## Why

You just joined a team. There's a bug in `POST /orders`. Where do you even start?

You grep for the route, find the handler, cmd-click into the service, cmd-click again into the store, read struct tags to figure out the database table. You do this every time, for every repo, for every bug.

**reqflow does all of that in one command.**

---

## Quick Start

```bash
cd your-go-project/
reqflow trace "/orders" ./...
```

That's it. reqflow parses your code, finds every route registered on `/orders`, lets you pick one, and shows the complete request path with exact method names and file locations.

---

## Usage

### 1. Trace by path (most common)

Just type the path. reqflow finds all HTTP methods registered on it and lets you pick:

```
$ reqflow trace "/orders" ./...

Multiple routes match "/orders":

  1.  GET /orders
  2.  POST /orders

Enter number (1-2): 2
```

Then it shows the full trace for `POST /orders`.

### 2. Trace by full route

If you already know the method:

```bash
reqflow trace "POST /orders" ./...
```

### 3. Trace by substring

Don't remember the exact path? Use a keyword:

```bash
reqflow trace "budget" ./...
```

reqflow finds every route containing "budget" and lets you pick.

---

## What You Get

reqflow shows exactly what happens when a request hits your server — which method handles it, what it calls, and where the code lives:

```
GET /orgs/{orgID}/reports/metrics
───────────────────────────────────

  [H]  Handler   HTTP Handler · internal/handler/handler.go:700
       GetMetrics()
         → svc.GetMetricsByOrgPaginated()

  │
  ↓  delegates to
  │
  [S]  service   Service · internal/service/service.go:1768
       GetMetricsByOrgPaginated()
         → store.CountMetricSeriesByOrg()
         → store.GetMetricSeriesByOrgPaginated()

  │
  ↓  queries via
  │
  [D]  Store     Store / Repository · internal/store/store.go:32
       CountMetricSeriesByOrg()
       GetMetricSeriesByOrgPaginated()
```

Each node shows:

| Part | Meaning |
|------|---------|
| `[H]` `[S]` `[D]` `[M]` | Layer — Handler, Service, Store (DB), Model |
| `Handler` | Struct name |
| `internal/handler/handler.go:700` | Package, file, and **line of the method** (not the struct) |
| `GetMetrics()` | The specific method that handles this route |
| `→ svc.GetMetricsByOrgPaginated()` | What that method calls on the next layer |

---

## Real-World Examples

### Simple CRUD — 3 layers

```
$ reqflow trace "GET /orgs/{orgID}/budgets" ./...

  [H]  Handler   internal/handler/handler.go:350
       GetBudgets()
         → svc.GetBudgets()

  [S]  service   internal/service/service.go:4170
       GetBudgets()
         → store.GetBudgetsByResourceUIDs()
         → store.GetBudgetsByResourceGroupIDs()
         → store.GetBudgets()

  [D]  Store     internal/store/store.go:32
       GetBudgetsByResourceUIDs()
       GetBudgetsByResourceGroupIDs()
       GetBudgets()
```

### Service calling external clients

```
$ reqflow trace "POST /orgs/{orgID}/actions" ./...

  [H]  Handler        internal/handler.go:56
       ManualAction()
         → svc.GetPermissionLevel()
         → svc.GetGroupForResource()

  [I]  CredentialFetcher   internal/provider.go:26
       GetPermissionLevel()

  [I]  ResourceFetcher     internal/resource_fetcher.go:15
       GetGroupForResource()

  [D]  Store               internal/store.go:14
       ...

  [D]  ConfigGRPCClient    internal/config_grpc.go:18
       GetPermissionLevel()
       GetGroupForResource()
```

### Interactive route selection

```
$ reqflow trace "budgets" ./...

Multiple routes match "budgets":

  1.  GET /orgs/{orgID}/budgets/summary
  2.  GET /orgs/{orgID}/budgets
  3.  POST /orgs/{orgID}/budgets
  4.  PUT /orgs/{orgID}/budgets/{budgetID}
  5.  DELETE /orgs/{orgID}/budgets/{budgetID}

Enter number (1-5): 3
```

Pick a number, see the trace. One session, no re-running.

---

## Flags

```bash
reqflow trace [flags] <route> [packages]
```

| Flag | Description |
|------|-------------|
| `-format text` | Terminal output (default) |
| `-format html` | Self-contained HTML page |
| `-out <file>` | Write to file instead of stdout |
| `-tablemap` | Show database tables mapped from model struct tags |
| `-envmap` | Show environment variables read via `os.Getenv` / `viper` |

### Examples with flags

```bash
# Save trace as HTML for sharing
reqflow trace -format html -out trace.html "POST /orders" ./...

# See which DB tables a request touches
reqflow trace -tablemap "GET /users/{id}" ./...

# See which env vars a request path reads
reqflow trace -envmap "POST /orders" ./...
```

---

## How It Works

reqflow uses Go's type system — not grep, not regexes.

1. **Loads packages** with `golang.org/x/tools/go/packages` and walks the AST
2. **Classifies structs by structure** — a store is any struct holding a `*sql.DB` field; a handler is any struct whose methods accept `*gofr.Context` or `*gin.Context`
3. **Extracts routes** from `app.GET("/path", h.Method)` calls — including inline anonymous handlers like `app.GET("/health", func(ctx) { ... })`
4. **Builds a method-level call index** — knows `Handler.GetMetrics()` calls `svc.GetMetricsByOrgPaginated()`, not just "handler depends on service"
5. **Traces the precise path** — follows actual method calls, not struct dependencies. Only shows the nodes and methods touched by this specific request

---

## Supported Frameworks

| Framework | Handler signature |
|-----------|-------------------|
| [GoFr](https://gofr.dev) | `func(ctx *gofr.Context) (any, error)` |
| [Gin](https://gin-gonic.com) | `func(c *gin.Context)` |
| [Echo](https://echo.labstack.com) | `func(c echo.Context) error` |
| [Fiber](https://gofiber.io) | `func(c *fiber.Ctx) error` |
| net/http | `func(w http.ResponseWriter, r *http.Request)` |

### Store detection

Detected by struct field types, not naming conventions:

`*sql.DB`, `*sqlx.DB`, `*gorm.DB`, `*mongo.Client`, `*mongo.Database`, `*redis.Client`, `*redis.ClusterClient`, `*pgx.Conn`, `*pgxpool.Pool`, `*dynamodb.Client`, `*firestore.Client`, `*spanner.Client`, `*bigquery.Client`, `*elasticsearch.Client`

### Layer classification

reqflow classifies structs using both **name patterns** and **structural detection**:

| Layer | Name patterns | Structural detection |
|-------|--------------|---------------------|
| Handler | `*Handler`, `*Controller`, `*Endpoint` | Methods accept framework context |
| Service | `*Service`, `*UseCase`, `*Manager` | — |
| Store | `*Store`, `*Repository`, `*Repo`, `*Client` | Has DB client field |
| Model | `*Model`, `*Entity`, `*DTO` | — |

Override with `.reqflow.yml`:

```yaml
layers:
  service_pattern: ".*Service$|.*Processor$"
  store_pattern:   ".*Store$|.*Repository$|.*Client$"
  model_pattern:   ".*Model$|.*Entity$"
```

---

## Configuration

Optional `.reqflow.yml` in your project root:

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
```

---

## Use as a Go library

```go
import (
    "github.com/thzgajendra/reqflow"
)

graph, err := reqflow.Parse(reqflow.ParseOptions{Dir: "."})
result := reqflow.Trace("POST /orders", graph)

fmt.Println(result.Route)           // "POST /orders"
fmt.Println(result.Handler.Name)    // "OrderHandler"
for _, node := range result.Chain {
    fmt.Printf("[%s] %s\n", node.Kind, node.Name)
}
```

---

## License

Apache 2.0 — see [LICENSE](LICENSE)
