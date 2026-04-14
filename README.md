# reqflow

[![Go Reference](https://pkg.go.dev/badge/github.com/thzgajendra/reqflow.svg)](https://pkg.go.dev/github.com/thzgajendra/reqflow)
[![Go Report Card](https://goreportcard.com/badge/github.com/thzgajendra/reqflow)](https://goreportcard.com/report/github.com/thzgajendra/reqflow)
[![CI](https://github.com/thzgajendra/reqflow/actions/workflows/ci.yml/badge.svg)](https://github.com/thzgajendra/reqflow/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

**Trace any HTTP request through your Go codebase, statically.**

```bash
reqflow trace "POST /orders" ./...
```

Shows you the complete path: handler -> service -> store -> database table -> external calls.
No instrumentation. No runtime. Just point it at your code.

---

## The Problem

You just joined a team. There's a bug in `POST /orders`. Where do you even start?

You grep for the route, find the handler, cmd-click into the service, cmd-click again into the repo, look at struct tags to figure out what database table it's writing to. You do this every time, for every repo, for every bug.

**reqflow automates that entire workflow.**

---

## Install

```bash
go install github.com/thzgajendra/reqflow/cmd/reqflow@latest
```

Or download a binary from [Releases](https://github.com/thzgajendra/reqflow/releases).

---

## Usage

```bash
reqflow trace "POST /orders" ./...
reqflow trace "/orders" ./...                        # path-only, any method
reqflow trace -format html -out trace.html "orders"  # partial match, HTML output
reqflow trace -tablemap "GET /users/{id}" ./...      # include database table mappings
reqflow trace -envmap "POST /orders" ./...            # include environment variable reads
```

### Flags

| Flag | Description |
|------|-------------|
| `-format text` | Terminal output (default) |
| `-format html` | Self-contained HTML page |
| `-out <file>` | Write output to a file instead of stdout |
| `-tablemap` | Resolve model-to-database table mappings |
| `-envmap` | Resolve environment variable reads |

### Example output

```
POST /orders
--------------

  [H]  OrderHandler          HTTP Handler . handler/orders.go:45
       CreateOrder()

  |
  v  delegates to
  |
  [S]  OrderService          Service . service/orders.go:23
       Create()

  |
  v  queries via
  |
  [D]  OrderStore            Store / Repository . store/orders.go:67
       Insert()

  |
  v  maps to model
  |
  [M]  Order                 Data Model . model/order.go:12
       Fields: ID, CustomerID, Status, Total, CreatedAt

  +- Database tables
  |   orders
  +-
```

---

## How It Works

reqflow uses Go's type system -- not grep, not regexes. It loads your packages with `golang.org/x/tools/go/packages`, walks the AST, and:

1. **Classifies types structurally** -- a store is any struct holding a `*sql.DB` (not one named `*Store`); a handler is any struct whose methods accept a framework context
2. **Builds dependency edges** from struct fields and constructor parameters
3. **Extracts route registrations** from `app.GET("/path", h.Handler)` calls
4. **Traces the path** from the matched handler through all reachable dependencies, ordered by architectural layer

Works offline, in CI, on codebases you've never seen before.

---

## Supported Frameworks

| Framework | Detection |
|-----------|-----------|
| [GoFr](https://gofr.dev) | `func(ctx *gofr.Context) (any, error)` |
| [Gin](https://gin-gonic.com) | `func(c *gin.Context)` |
| [Echo](https://echo.labstack.com) | `func(c echo.Context) error` |
| [Fiber](https://gofiber.io) | `func(c *fiber.Ctx) error` |
| net/http | `func(w http.ResponseWriter, r *http.Request)` |

Store detection: `*sql.DB`, `*sqlx.DB`, `*gorm.DB`, `*mongo.Client`, `*redis.Client`, `*pgxpool.Pool`, and more -- from struct field types, not naming conventions.

---

## Configuration

Create `.reqflow.yml` in your project root to customize layer detection:

```yaml
parser:
  ignore_packages:
    - vendor
    - _test

  # Override automatic layer detection with explicit regex patterns:
  # domain_naming:
  #   service_match: ".*Service$"
  #   store_match:   ".*Store$|.*Repository$|.*Repo$"
  #   model_match:   ".*Model$|.*Entity$"
```

---

## License

Apache 2.0 -- see [LICENSE](LICENSE)
