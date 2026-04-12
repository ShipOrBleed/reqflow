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
