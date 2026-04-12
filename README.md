# Govis 🔍📦

**Govis** is a blazing-fast, industry-grade Go architecture visualizer and linter. It parses Go ASTs natively to map dependencies, clean architecture layers, event bus topics, and framework usage, outputting highly interactive dashboards or enforcing structure directly in your CI/CD pipelines.

Whether you've inherited a chaotic massive codebase and need to find the dead code, or you're a strict Tech Lead wanting to mathematically block spaghetti-code during PR checks, Govis gives you X-Ray vision into your Go projects.

---

## 🌟 What Problem Does This Solve?

1. **"I can't read this diagram, it has 10,000 nodes."** Govis natively supports Click-To-Code deep-linking (click a node, open the exact line in VSCode), `--focus` micro-scoping, and beautiful interactive HTML dashboards.
2. **"Junior developers keep importing the Database into the HTTP layer."** Govis acts as a strict Architecture Guardrail using the `-vet` flag. 
3. **"What architectural changes did this Pull Request introduce?"** Govis can structurally `-diff` two branches and visually highlight what was built and what was destroyed.

---

## 🚀 Features & Usage

### 1. Interactive HTML Dashboard (Codebase Navigation)
**The Feature:** Generates a stunning, standalone frontend with deep dark-mode UI, legends, and completely draggable SVG pan & zoom canvases.
**The Problem Solved:** Huge codebases are impossible to view in console or Markdown. Click a node in this dashboard to instantly hyper-link into VS Code!
```bash
govis -format html ./... > dashboard.html
open dashboard.html
```

### 2. Architecture Linter (Clean Architecture Guardrails)
**The Feature:** Defends architectural boundaries by tracking dependency edges.
**The Problem Solved:** Stops logic leaking across layers. You can configure Govis to exit with `status 1` if it detects a forbidden map linkage. 
```bash
# Ensure Handlers never bypass Business Services to talk to Stores
govis -vet="handler!store" ./...
```

### 3. Dead Code Detection
**The Feature:** Mathematically proves what nodes lack incoming architectural references.
**The Problem Solved:** Eliminates the guesswork when deleting legacy systems. It tells you immediately if an entire `PaymentRepository` has been orphaned.
```bash
govis -deadcode ./...
```

### 4. Git Diff Architecture Visualization
**The Feature:** Ingests two AST states and creates a glowing Visual Difference Map. New systems glow Green, deprecated systems burn Red.
**The Problem Solved:** PR reviews show 400 lines of code changed, but completely fail to convey the massive structural shift introduced.
```bash
# Save main branch state
govis -format json ./... > old.json
git checkout my-new-feature
# View architectural diff
govis -diff old.json -format html ./... > diff.html
```

### 5. Automated AI Architect Review
**The Feature:** Instantly serializes the macro-graph to LLM endpoints to request an automated architectural critique.
**The Problem Solved:** Identifies circular dependencies, coupling issues, and poor API layer boundary separation programmatically.
```bash
export OPENAI_API_KEY="sk-..."
govis -ai ./...
```

### 6. Microservice Event-Bus Extraction
**The Feature:** Detects `.Publish("topic")` or `.Subscribe("topic")` signatures in AST execution.
**The Problem Solved:** Event-driven microservices are functionally invisible to traditional structural parsing. Govis physically draws `KindEvent` nodes displaying exactly what Services drop inputs into which Kafka/RabbitMQ Topics.

### 7. PlanetScale / Vitess Native Mapping
**The Feature:** Auto-discovers local `vschema.json` topologies and natively annotates `Store` models with shard metadata.
**The Problem Solved:** Cross-shard data operations are incredibly dangerous. Govis lets distributed systems engineers easily separate Sharded models from Unsharded keyspaces visually.

---

## 💻 Installation

Install Govis directly using standard Go tooling (ensure `GOPATH/bin` is in your environment PATH).

```bash
go install github.com/zopdev/govis/cmd/govis@latest
```

*(Note: Prior open-source publishing, customize the base `zopdev` github imports to match your namespace!)*
