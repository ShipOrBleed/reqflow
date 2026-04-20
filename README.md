# reqflow

[![Go Reference](https://pkg.go.dev/badge/github.com/ShipOrBleed/reqflow.svg)](https://pkg.go.dev/github.com/ShipOrBleed/reqflow)
[![Go Report Card](https://goreportcard.com/badge/github.com/ShipOrBleed/reqflow)](https://goreportcard.com/report/github.com/ShipOrBleed/reqflow)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

**Trace any HTTP request through your Go codebase, statically.**

One command. No instrumentation. No runtime. Just point it at your code.

```bash
go install github.com/ShipOrBleed/reqflow/cmd/reqflow@latest
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

## Commands

### `reqflow trace` — Trace a request path

Just type the path. reqflow finds all HTTP methods registered on it and lets you pick:

```
$ reqflow trace "/orders" ./...

Multiple routes match "/orders":

  1.  GET /orders
  2.  POST /orders

Enter number (1-2): 2

POST /orders
──────────────

  [H]  OrderHandler   internal/handler/orders.go:45
       CreateOrder()
         → svc.Create()

  │
  ↓  delegates to
  │
  [S]  orderService   internal/service/orders.go:89
       Create()
         → store.Insert()

  │
  ↓  queries via
  │
  [D]  OrderStore     internal/store/orders.go:67
       Insert()
```

Other ways to trace:

```bash
reqflow trace "POST /orders" ./...    # exact route
reqflow trace "budget" ./...          # substring — shows all matches
```

### `reqflow routes` — List all routes in a service

```
$ reqflow routes ./...

  GET     /orgs/{orgID}/budgets              Handler.GetBudgets()        handler/handler.go:350
  POST    /orgs/{orgID}/budgets              Handler.CreateBudget()      handler/handler.go:398
  DELETE  /orgs/{orgID}/budgets/{budgetID}   Handler.DeleteBudget()      handler/handler.go:442
  GET     /orgs/{orgID}/reports/metrics      Handler.GetMetrics()        handler/handler.go:700
  ...

39 routes across 1 handlers
```

Export as JSON:

```bash
reqflow routes -format json ./...
```

---

## What You Get

Each node in the trace shows:

| Part | Meaning |
|------|---------|
| `[H]` `[S]` `[D]` `[C]` `[M]` | Layer — Handler, Service, Store, Client, Model |
| `Handler` | Struct name |
| `internal/handler/handler.go:700` | Package, file, and **line of the method** (not the struct) |
| `GetMetrics()` | The specific method that handles this route |
| `→ svc.GetMetricsByOrgPaginated()` | What that method calls on the next layer |

---

## Real-World Examples

### Handler → Service → Store (most common)

```
$ reqflow trace "GET /orgs/{orgID}/budgets" ./...

  [H]  Handler   internal/handler/handler.go:350
       GetBudgets()
         → svc.GetBudgets()

  [S]  service   internal/service/service.go:4258
       GetBudgets()
         → store.GetBudgetsByResourceUIDs()
         → store.GetBudgetsByResourceGroupIDs()
         → store.GetBudgets()

  [D]  Store     internal/store/store.go:32
       GetBudgetsByResourceUIDs()
       GetBudgetsByResourceGroupIDs()
       GetBudgets()
```

### Handler → External Clients (gRPC/HTTP)

```
$ reqflow trace "POST /orgs/{orgID}/actions" ./...

  [H]  Handler            internal/handler.go:80
       ManualAction()
         → cloudAccountFetcher.GetPermissionLevel()
         → resourceFetcher.GetGroupForResource()

  [C]  ConfigGRPCClient   internal/config_grpc.go:18
       GetPermissionLevel()
         → client.GetCloudAccountPermission()
       GetGroupForResource()

  [C]  ConfigServiceClient   internal/configclient/config_grpc.pb.go:35
       GetCloudAccountPermission()
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

### `reqflow trace`

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

```bash
reqflow trace -format html -out trace.html "POST /orders" ./...
reqflow trace -tablemap "GET /users/{id}" ./...
reqflow trace -envmap "POST /orders" ./...
```

### `reqflow routes`

```bash
reqflow routes [flags] [packages]
```

| Flag | Description |
|------|-------------|
| `-format text` | Aligned table output (default) |
| `-format json` | JSON array |
| `-out <file>` | Write to file instead of stdout |

---

## How It Works

reqflow uses Go's type system — not grep, not regexes.

1. **Loads packages** with `golang.org/x/tools/go/packages` and walks the AST
2. **Classifies structs by structure** — a store is any struct holding a `*sql.DB` field; a handler is any struct whose methods accept `*gofr.Context` or `*gin.Context`; an HTTP client is any struct named `*Client`
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

### Layer classification

| Layer | Badge | Name patterns | Structural detection |
|-------|-------|--------------|---------------------|
| Handler | `[H]` | `*Handler`, `*Controller`, `*Endpoint` | Methods accept framework context |
| Service | `[S]` | `*Service`, `*UseCase`, `*Manager` | — |
| Store | `[D]` | `*Store`, `*Repository`, `*Repo` | Has DB client field (`*sql.DB`, `*gorm.DB`, etc.) |
| Client | `[C]` | `*Client`, `*Caller`, `*Connector` | — |
| Model | `[M]` | `*Model`, `*Entity`, `*DTO` | Has DB struct tags |

Override with `.reqflow.yml`:

```yaml
layers:
  service_pattern: ".*Service$|.*Processor$"
  store_pattern:   ".*Store$|.*Repository$"
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
import "github.com/ShipOrBleed/reqflow"

// Parse codebase
graph, err := reqflow.Parse(reqflow.ParseOptions{Dir: "."})

// List all routes
routes := reqflow.ListRoutes(graph)
for _, r := range routes {
    fmt.Printf("%s %s → %s.%s()\n", r.Method, r.Path, r.HandlerName, r.MethodName)
}

// Trace a specific route
result := reqflow.Trace("POST /orders", graph)
fmt.Println(result.Route)
for _, node := range result.Chain {
    fmt.Printf("[%s] %s\n", node.Kind, node.Name)
}
```

---

## License

MIT — see [LICENSE](LICENSE)
