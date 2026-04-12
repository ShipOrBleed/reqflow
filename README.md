# Govis 🔍📦

**Govis** is a powerful Go package and CLI tool designed to automatically analyze and visualize your Go backend architectures. It helps developers—from total newbies to seasoned veterans—understand how a codebase works, which components talk to each other, and how the entire backend is connected.

Whether you've just inherited a massive legacy codebase or want to automatically document your current project's architecture, Govis easily generates class diagrams and architecture charts out of your code!

---

## 🌟 Features

- **Blazing Fast Parsing:** Uses the robust `golang.org/x/tools/go/packages` toolchain to deeply understand your code's Syntax, Types, and Imports.
- **Smart Component Detection:** Automatically identifies and categorizes your backend components:
  - `Structs` and `Interfaces`
  - `Store` (Database layer detection)
  - `Functions` & `Constructors`
  - **Framework-Agnostic HTTP Handlers:** Detects handlers not just for standard `net/http`, but also natively parses handlers from popular frameworks including **Gin**, **Echo**, and **Fiber**!
- **Deep Relationship Mapping:** Govis maps out complex connections like:
  - Interface *Implementations* (both value and pointer receivers)
  - Struct *Embeds* (composition)
  - Dependency Injection (detects what components rely on others through constructors)
- **Multiple Export Formats:** Render your architecture natively to:
  - **Mermaid.js** (Perfect for Markdown files and GitHub PRs)
  - **DOT** (Perfect for Graphviz parsing)
  - **SVG** (Via Graphviz DOT piping)
  - **HTML Interactive Dashboard:** Directly generates a standalone web page with zoom, panning, and legend.
  - **JSON Data Export:** For building your own UI, pipelining, or integrations.

### 🛡️ Architecture Linter (`govis vet`)
Govis acts as a strict architecture guardrail. You can pass a rule like `-vet="handler!store"` and Govis will analyze your dependency structures. If any Junior Dev tries to import a Database Store directly into an HTTP Handler, it will print a `🚨 VIOLATION` error and exit with status 1, failing your CI/CD!

### 🔭 The Focus Feature
Got a massive 100,000-line monolith? Govis has a `--focus` flag! Provide the name of a `Service`, `Model`, or `Controller`, and Govis will strip away the noise—giving you an isolated, targeted map of just that specific component and its immediate dependencies.

### 🛣️ API Route Extraction
Govis's AST parser doesn't just find handlers—it detects your literal `router.GET("/api/v1/users")` definitions and automatically stamps those textual URLs right onto your generated visual nodes!

---

## 🚀 Installation

For any Go developer, installing Govis is as simple as running:

```bash
go install github.com/zopdev/govis/cmd/govis@latest
```

*(Make sure your `$(go env GOPATH)/bin` directory is in your system `$PATH`)*

---

## 💻 Usage

Run Govis directly in your project root to map your codebase!

### Generate a Mermaid Diagram
Mermaid diagrams are incredible because you can paste them directly into GitHub markdown or Notion!

```bash
govis -format mermaid ./... > architecture.md
```

### Generate a DOT Graph
For a deeper, detailed system graph, you can generate a `.dot` file:

```bash
govis -format dot ./... > graph.dot
```

### Generate an Interactive HTML Dashboard
The most powerful way to use Govis locally. This generates a standalone webpage displaying your architecture with drag, drop, and scroll-to-zoom:

```bash
govis -format html ./... > dashboard.html
open dashboard.html
```

### JSON Export API
Need raw structural data to pipe to another platform? 

```bash
govis -format json ./... > architecture.json
```

### Linting Architecture Violtations (Clean Architecture)
Run a CI check to ensure Handlers (`handler`) do not communicate straight to Database Repositories (`store`):

```bash
govis -vet="handler!store" ./...
```

### Focus on a Specific Component
Filter out all the noise and only map a targeted subset of your architecture (e.g. your `PaymentService`):

```bash
govis -format mermaid -focus="PaymentService" ./... > focus.md
```

### Pipe to SVG (Requires Graphviz)
You can directly convert the DOT mapping into a beautiful SVG graphic:

```bash
govis -format dot ./... | dot -Tsvg > architecture.svg
```

### Filtering Packages
If your project is huge, you can filter visualization to a specific sub-package namespace:

```bash
govis -format mermaid -filter="internal/api" ./...
```

---

## 🤖 GitHub Actions Integration

Govis includes a built-in Dockerfile and GitHub Action `entrypoint.sh` out of the box! You can integrate Govis into your CI/CD pipeline to **automatically comment Mermaid architecture diagrams on Pull Requests**.

**Example `.github/workflows/govis.yml`**:
```yaml
name: Map Go Architecture
on: [pull_request]

jobs:
  build-and-map:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout Code
        uses: actions/checkout@v3

      - name: Run Govis StructMap
        uses: zopdev/govis@main
        with:
          dir: './...'
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## Contributing

Pull requests are always welcome. Let's make backend architectures easier for everyone to understand!
