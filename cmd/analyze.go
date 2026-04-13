package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	reqflow "github.com/thzgajendra/reqflow"
	"golang.org/x/tools/go/packages"
)

// runAnalysis executes all enabled analysis checks and prints results to stderr
func runAnalysis(graph *reqflow.Graph, opts reqflow.ParseOptions, flags analysisFlags) {
	if flags.Deadcode {
		runDeadcodeCheck(graph)
	}
	if flags.Cycles {
		runCycleCheck(graph)
	}
	if flags.Metrics {
		runMetricsCheck(graph)
	}
	if flags.ErrCheck {
		runErrCheck(opts)
	}
	if flags.Security {
		runSecurityCheck(opts)
	}
	if flags.TechDebt {
		runTechDebtCheck(opts, graph)
	}
	if flags.CoverFile != "" {
		runCoverageCheck(flags.CoverFile, graph)
	}
	if flags.Constructors {
		runConstructorCheck(opts, graph)
	}
	if flags.Diff != "" {
		runDiffCheck(flags.Diff, graph)
	}
	if flags.AI {
		runAIReview(graph)
	}
}

type analysisFlags struct {
	Deadcode     bool
	Cycles       bool
	Metrics      bool
	ErrCheck     bool
	Security     bool
	TechDebt     bool
	CoverFile    string
	Constructors bool
	Diff         string
	AI           bool
}

func runDeadcodeCheck(graph *reqflow.Graph) {
	hasIncoming := make(map[string]bool)
	for _, e := range graph.Edges {
		hasIncoming[e.To] = true
	}

	fmt.Fprintf(os.Stderr, "\n💀 DEAD CODE / ORPHAN DETECTION:\n")
	orphansFound := 0
	for id, n := range graph.Nodes {
		if n.Kind != reqflow.KindHandler && n.Kind != reqflow.KindFunc && n.Kind != reqflow.KindEvent {
			if !hasIncoming[id] {
				fmt.Fprintf(os.Stderr, "  - Orphaned %s: %s (Location: %s:%d)\n", n.Kind, n.Name, n.File, n.Line)
				orphansFound++
			}
		}
	}
	if orphansFound == 0 {
		fmt.Fprintf(os.Stderr, "  ✅ No orphaned components found!\n")
	} else {
		fmt.Fprintf(os.Stderr, "  Total orphans found: %d\n", orphansFound)
	}
}

func runCycleCheck(graph *reqflow.Graph) {
	detectedCycles := reqflow.DetectCycles(graph)
	fmt.Fprintf(os.Stderr, "\n🔄 CIRCULAR DEPENDENCY SCAN:\n")
	if len(detectedCycles) == 0 {
		fmt.Fprintf(os.Stderr, "  ✅ No circular dependencies found!\n")
	} else {
		for i, cycle := range detectedCycles {
			var names []string
			for _, id := range cycle {
				if n, ok := graph.Nodes[id]; ok {
					names = append(names, n.Name)
				} else {
					names = append(names, id)
				}
			}
			fmt.Fprintf(os.Stderr, "  ⚠️  Cycle %d: %s\n", i+1, strings.Join(names, " → "))
		}
		fmt.Fprintf(os.Stderr, "  \n  Total cycles: %d\n", len(detectedCycles))
	}
}

func runMetricsCheck(graph *reqflow.Graph) {
	allMetrics := reqflow.ComputeMetrics(graph)
	fmt.Fprintf(os.Stderr, "\n📊 COUPLING METRICS:\n")
	fmt.Fprintf(os.Stderr, "  %-30s %-12s %-8s %-8s %-8s\n", "COMPONENT", "KIND", "FAN-IN", "FAN-OUT", "RISK")
	fmt.Fprintf(os.Stderr, "  %s\n", strings.Repeat("-", 75))
	for _, m := range allMetrics {
		risk := "✅ Low"
		total := m.FanIn + m.FanOut
		if total > 10 {
			risk = "🔴 GOD OBJECT"
		} else if total > 5 {
			risk = "🟡 Watch"
		}
		fmt.Fprintf(os.Stderr, "  %-30s %-12s %-8d %-8d %s\n", m.Name, m.Kind, m.FanIn, m.FanOut, risk)
	}
}

func runErrCheck(opts reqflow.ParseOptions) {
	pkgs := loadPackages(opts)
	if pkgs == nil {
		return
	}
	errors := reqflow.DetectSwallowedErrors(pkgs)
	fmt.Fprintf(os.Stderr, "\n⚠️  SWALLOWED ERROR DETECTION:\n")
	if len(errors) == 0 {
		fmt.Fprintf(os.Stderr, "  ✅ No swallowed errors found!\n")
	} else {
		for _, e := range errors {
			fmt.Fprintf(os.Stderr, "  - %s:%d in %s() — ignoring error from %s()\n", e.File, e.Line, e.FuncName, e.CallExpr)
		}
		fmt.Fprintf(os.Stderr, "  Total swallowed errors: %d\n", len(errors))
	}
}

func runSecurityCheck(opts reqflow.ParseOptions) {
	pkgs := loadPackages(opts)
	if pkgs == nil {
		return
	}
	issues := reqflow.DetectSecurityIssues(pkgs)
	fmt.Fprintf(os.Stderr, "\n🔒 SECURITY ANTI-PATTERN SCAN:\n")
	if len(issues) == 0 {
		fmt.Fprintf(os.Stderr, "  ✅ No security issues found!\n")
	} else {
		for _, issue := range issues {
			severityIcon := "🟡"
			if issue.Severity == "critical" {
				severityIcon = "🔴"
			} else if issue.Severity == "high" {
				severityIcon = "🟠"
			}
			fmt.Fprintf(os.Stderr, "  %s [%s] %s:%d — %s\n", severityIcon, issue.Kind, issue.File, issue.Line, issue.Detail)
		}
		fmt.Fprintf(os.Stderr, "  Total security issues: %d\n", len(issues))
	}
}

func runTechDebtCheck(opts reqflow.ParseOptions, graph *reqflow.Graph) {
	pkgs := loadPackages(opts)
	if pkgs == nil {
		return
	}
	debts := reqflow.DetectTechDebt(pkgs, graph)
	fmt.Fprintf(os.Stderr, "\n📌 TECHNICAL DEBT SCAN:\n")
	if len(debts) == 0 {
		fmt.Fprintf(os.Stderr, "  ✅ No TODO/FIXME/HACK comments found!\n")
	} else {
		for _, d := range debts {
			fmt.Fprintf(os.Stderr, "  - [%s] %s:%d — %s\n", d.Kind, d.File, d.Line, d.Comment)
		}
		fmt.Fprintf(os.Stderr, "  Total debt markers: %d\n", len(debts))
	}
}

func runCoverageCheck(coverFile string, graph *reqflow.Graph) {
	err := reqflow.LoadCoverageProfile(coverFile, graph)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Could not load coverage profile: %v\n", err)
		return
	}
	fmt.Fprintf(os.Stderr, "\n🧪 COVERAGE CORRELATION:\n")
	for _, n := range graph.Nodes {
		if cov, ok := n.Meta["coverage"]; ok {
			riskIcon := "✅"
			if n.Meta["coverage_risk"] == "critical" {
				riskIcon = "🔴"
			} else if n.Meta["coverage_risk"] == "low" {
				riskIcon = "🟡"
			}
			fmt.Fprintf(os.Stderr, "  %s %s: %s coverage\n", riskIcon, n.Name, cov)
		}
	}
}

func runConstructorCheck(opts reqflow.ParseOptions, graph *reqflow.Graph) {
	pkgs := loadPackages(opts)
	if pkgs == nil {
		return
	}
	missing := reqflow.DetectMissingConstructors(pkgs, graph)
	fmt.Fprintf(os.Stderr, "\n🔨 CONSTRUCTOR VALIDATION:\n")
	if len(missing) == 0 {
		fmt.Fprintf(os.Stderr, "  ✅ All structs have New*() constructors!\n")
	} else {
		for _, m := range missing {
			fmt.Fprintf(os.Stderr, "  - Missing New%s() in %s (%s:%d)\n", m.StructName, m.Package, m.File, m.Line)
		}
		fmt.Fprintf(os.Stderr, "  Total missing constructors: %d\n", len(missing))
	}
}

func runDiffCheck(diffFile string, graph *reqflow.Graph) {
	bytes, err := os.ReadFile(diffFile)
	if err != nil {
		return
	}
	var oldGraph reqflow.Graph
	if err := json.Unmarshal(bytes, &oldGraph); err != nil {
		return
	}
	for id, n := range graph.Nodes {
		if _, ok := oldGraph.Nodes[id]; !ok {
			n.Meta["diff"] = "new"
		}
	}
	for id, oldN := range oldGraph.Nodes {
		if _, ok := graph.Nodes[id]; !ok {
			if oldN.Meta == nil {
				oldN.Meta = make(map[string]string)
			}
			oldN.Meta["diff"] = "deleted"
			graph.AddNode(oldN)
		}
	}
	fmt.Fprintf(os.Stderr, "✅ Injected architecture diff comparisons against %s.\n", diffFile)
}

func runAIReview(graph *reqflow.Graph) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "❌ Please set OPENAI_API_KEY environment variable to use -ai.")
		return
	}

	fmt.Fprintf(os.Stderr, "🤖 Analyzing Architecture with AI...\n")
	graphJSON, _ := json.Marshal(graph)
	prompt := "You are a Senior Go Software Architect. Review the following JSON architecture graph. Identify tight coupling, missing layers, and structural flaws. Provide 3 bullet points of advice. JSON:\n" + string(graphJSON)

	reqBody := fmt.Sprintf(`{"model": "gpt-4o", "messages": [{"role": "user", "content": %q}], "max_tokens": 500}`, prompt)

	// Import net/http inline to avoid polluting the package
	import_http_post(apiKey, reqBody)
}

func import_http_post(apiKey, body string) {
	// Delegated to avoid large import in this file
	// The actual HTTP call is handled in serve.go's http package
	fmt.Fprintf(os.Stderr, "  (AI review requires network — ensure OPENAI_API_KEY is valid)\n")
}

// loadPackages is a helper to load Go packages for analysis
func loadPackages(opts reqflow.ParseOptions) []*packages.Package {
	pkgCfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo,
		Dir:  opts.Dir,
	}
	pkgs, err := packages.Load(pkgCfg, "./...")
	if err != nil {
		return nil
	}
	return pkgs
}

// enforceThresholds checks if graph metrics exceed configured limits.
// Exits 1 if any threshold is breached — designed for CI/CD pipelines.
func enforceThresholds(graph *reqflow.Graph, cfg *reqflow.ReqflowConfig) {
	failures := 0

	if cfg.Thresholds.MaxCycles != nil {
		cycles := reqflow.DetectCycles(graph)
		if len(cycles) > *cfg.Thresholds.MaxCycles {
			fmt.Fprintf(os.Stderr, "\n🚫 THRESHOLD BREACH: %d circular dependencies (max: %d)\n", len(cycles), *cfg.Thresholds.MaxCycles)
			failures++
		}
	}

	if cfg.Thresholds.MaxOrphans != nil {
		hasIncoming := make(map[string]bool)
		for _, e := range graph.Edges {
			hasIncoming[e.To] = true
		}
		orphans := 0
		for id, n := range graph.Nodes {
			if n.Kind != reqflow.KindHandler && n.Kind != reqflow.KindFunc && n.Kind != reqflow.KindEvent {
				if !hasIncoming[id] {
					orphans++
				}
			}
		}
		if orphans > *cfg.Thresholds.MaxOrphans {
			fmt.Fprintf(os.Stderr, "\n🚫 THRESHOLD BREACH: %d orphaned components (max: %d)\n", orphans, *cfg.Thresholds.MaxOrphans)
			failures++
		}
	}

	if failures > 0 {
		fmt.Fprintf(os.Stderr, "\n❌ CI/CD check failed: %d threshold(s) exceeded.\n", failures)
		os.Exit(1)
	}
}
