# reqflow

[![Go Reference](https://pkg.go.dev/badge/github.com/thzgajendra/reqflow.svg)](https://pkg.go.dev/github.com/thzgajendra/reqflow)
[![Go Report Card](https://goreportcard.com/badge/github.com/thzgajendra/reqflow)](https://goreportcard.com/report/github.com/thzgajendra/reqflow)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

**Trace any HTTP request through your Go codebase, statically.**

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

No instrumentation. No runtime. Just point it at your code.

---

## The Problem

You just joined a team. There's a bug in `POST /orders`. Where do you even start?

You grep for the route, find the handler, cmd-click into the service, cmd-click again into the repo. You do this every time, for every repo, for every bug.

**reqflow does that in one command.**

---

## Install

```bash
go install github.com/thzgajendra/reqflow/cmd/reqflow@latest
```

---

## Usage

### Trace a request

```bash
# Type just the path — reqflow shows available methods, you pick one
reqflow trace "/orders" ./...

# Or specify the full route directly
reqflow trace "POST /orders" ./...

# Substring match
reqflow trace "orders" ./...
```

### What you get

For each step in the request path, reqflow shows:
- The **struct name** and **layer** (Handler / Service / Store)
- The **exact method** that handles this request
- The **file and line number** where that method is defined
- The **sub-calls** — which methods it calls on the next layer

### Flags

```bash
reqflow trace -tablemap "/orders" ./...   # include database tables
reqflow trace -envmap "/orders" ./...     # include environment variables read
reqflow trace -format html -out trace.html "/orders" ./...  # HTML output
```

| Flag | Description |
|------|-------------|
| `-format text` | Terminal output (default) |
| `-format html` | Self-contained HTML page |
| `-out <file>` | Write to file instead of stdout |
| `-tablemap` | Show model-to-database table mappings |
| `-envmap` | Show environment variable reads |

---

## How It Works

reqflow uses Go's type system — not grep, not regexes. It loads your packages with `golang.org/x/tools/go/packages`, walks the AST, and:

1. **Classifies structs** — a store is any struct holding a `*sql.DB`; a handler is any struct whose methods accept a framework context
2. **Extracts routes** from `app.GET("/path", h.Method)` calls, including inline anonymous handlers
3. **Builds a method-level call index** — knows that `Handler.GetMetrics()` calls `svc.GetMetricsByOrgPaginated()`, not just "handler depends on service"
4. **Traces the precise path** — only the methods actually called for this specific route, not every method on the struct

---

## Supported Frameworks

| Framework | Detection |
|-----------|-----------|
| [GoFr](https://gofr.dev) | `func(ctx *gofr.Context) (any, error)` |
| [Gin](https://gin-gonic.com) | `func(c *gin.Context)` |
| [Echo](https://echo.labstack.com) | `func(c echo.Context) error` |
| [Fiber](https://gofiber.io) | `func(c *fiber.Ctx) error` |
| net/http | `func(w http.ResponseWriter, r *http.Request)` |

Store detection: `*sql.DB`, `*sqlx.DB`, `*gorm.DB`, `*mongo.Client`, `*redis.Client`, `*pgxpool.Pool`, and more — from struct field types, not naming conventions.

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

## License

Apache 2.0 — see [LICENSE](LICENSE)
