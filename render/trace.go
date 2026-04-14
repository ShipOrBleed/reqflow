package render

import (
	"fmt"
	"io"
	"strings"

	reqflow "github.com/thzgajendra/reqflow"
)

// TraceRenderer renders a single request-path trace as a focused,
// readable terminal output or HTML page.
type TraceRenderer struct {
	Format string // "text" (default) or "html"
}

// traceIcon returns a single-letter badge for each node kind (used in trace output).
func traceIcon(k reqflow.NodeKind) string {
	switch k {
	case reqflow.KindHandler:
		return "H"
	case reqflow.KindService:
		return "S"
	case reqflow.KindStore:
		return "D"
	case reqflow.KindModel:
		return "M"
	case reqflow.KindInterface:
		return "I"
	case reqflow.KindGRPC:
		return "G"
	case reqflow.KindEvent:
		return "E"
	case reqflow.KindMiddleware:
		return "MW"
	default:
		return "?"
	}
}

// kindLabel returns a human-readable layer name.
func kindLabel(k reqflow.NodeKind) string {
	switch k {
	case reqflow.KindHandler:
		return "HTTP Handler"
	case reqflow.KindService:
		return "Service"
	case reqflow.KindStore:
		return "Store / Repository"
	case reqflow.KindModel:
		return "Data Model"
	case reqflow.KindInterface:
		return "Interface"
	case reqflow.KindGRPC:
		return "gRPC Service"
	case reqflow.KindEvent:
		return "Event / Topic"
	case reqflow.KindMiddleware:
		return "Middleware"
	default:
		return string(k)
	}
}

// RenderTrace writes a TraceResult to w in either text or HTML format.
func (tr *TraceRenderer) RenderTrace(result *reqflow.TraceResult, w io.Writer) error {
	if tr.Format == "html" {
		return renderTraceHTML(result, w)
	}
	return renderTraceText(result, w)
}

// ─── Terminal (text) renderer ────────────────────────────────────────────────

const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	dim    = "\033[2m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	blue   = "\033[34m"
	cyan   = "\033[36m"
	white  = "\033[37m"
	gray   = "\033[90m"
)

var kindColor = map[reqflow.NodeKind]string{
	reqflow.KindHandler:   green,
	reqflow.KindService:   blue,
	reqflow.KindStore:     yellow,
	reqflow.KindModel:     red,
	reqflow.KindInterface: cyan,
	reqflow.KindGRPC:      cyan,
	reqflow.KindEvent:     gray,
}

func color(k reqflow.NodeKind) string {
	if c, ok := kindColor[k]; ok {
		return c
	}
	return white
}

func renderTraceText(r *reqflow.TraceResult, w io.Writer) error {
	if r.NotFound {
		fmt.Fprintf(w, "\n%s✗ No handler found matching: %q%s\n\n", red, r.Route, reset)
		fmt.Fprintf(w, "  Hint: run  reqflow -format json ./...  and search Meta[\"routes\"]\n")
		fmt.Fprintf(w, "        to see all registered routes.\n\n")
		return nil
	}

	// Multiple routes match this path — show options
	if len(r.MultiMatch) > 0 {
		fmt.Fprintf(w, "\n%sMultiple routes match %q:%s\n\n", bold, r.Route, reset)
		for i, route := range r.MultiMatch {
			fmt.Fprintf(w, "  %s%d.%s  %s\n", cyan, i+1, reset, route)
		}
		fmt.Fprintf(w, "\n%sRun with the full route:%s\n", gray, reset)
		fmt.Fprintf(w, "  reqflow trace %q ./...\n\n", r.MultiMatch[0])
		return nil
	}

	fmt.Fprintf(w, "\n%s%s%s\n", bold, r.Route, reset)
	fmt.Fprintf(w, "%s%s%s\n\n", gray, strings.Repeat("─", len(r.Route)+2), reset)

	for i, node := range r.Chain {
		c := color(node.Kind)
		icon := traceIcon(node.Kind)
		label := kindLabel(node.Kind)

		// Arrow between steps
		if i > 0 {
			prev := r.Chain[i-1]
			edgeLabel := reqflow.EdgeLabel(prev, node)
			fmt.Fprintf(w, "  %s│%s\n", gray, reset)
			fmt.Fprintf(w, "  %s↓  %s%s\n", gray, edgeLabel, reset)
			fmt.Fprintf(w, "  %s│%s\n", gray, reset)
		}

		// Node header — show method location if we know which method is called
		file, line := node.File, node.Line
		if node.Kind == reqflow.KindHandler {
			if method := node.Meta["route_method:"+r.Route]; method != "" {
				if mf := node.Meta["method_file:"+method]; mf != "" {
					file = mf
				}
				if ml := node.Meta["method_line:"+method]; ml != "" {
					fmt.Sscanf(ml, "%d", &line)
				}
			}
		} else if calledMethods, ok := r.CalledMethods[node.ID]; ok && len(calledMethods) == 1 {
			method := calledMethods[0]
			if mf := node.Meta["method_file:"+method]; mf != "" {
				file = mf
			}
			if ml := node.Meta["method_line:"+method]; ml != "" {
				fmt.Sscanf(ml, "%d", &line)
			}
		}

		fmt.Fprintf(w, "  %s[%s]%s  %s%s%s", c, icon, reset, bold, node.Name, reset)
		fmt.Fprintf(w, "  %s%s · %s%s\n", gray, label, pkgFile(node.Package, file, line), reset)

		// Show the methods relevant to this trace + what they call
		if node.Kind == reqflow.KindHandler {
			if method, ok := node.Meta["route_method:"+r.Route]; ok && method != "" {
				fmt.Fprintf(w, "       %s%s()%s\n", cyan, method, reset)
				renderSubCalls(w, node.ID, method, r)
			}
		} else if calledMethods, ok := r.CalledMethods[node.ID]; ok && len(calledMethods) > 0 {
			for _, m := range calledMethods {
				fmt.Fprintf(w, "       %s%s()%s\n", cyan, m, reset)
				renderSubCalls(w, node.ID, m, r)
			}
		} else if len(node.Methods) > 0 {
			// Fallback: show all methods if we couldn't resolve the specific ones
			methods := node.Methods
			if len(methods) > 6 {
				methods = append(methods[:6], fmt.Sprintf("+%d more", len(node.Methods)-6))
			}
			fmt.Fprintf(w, "       Methods: %s%s%s\n", cyan, strings.Join(methods, "(), ")+"()", reset)
		}

		// Fields for models
		if node.Kind == reqflow.KindModel && len(node.Fields) > 0 {
			fields := make([]string, 0, len(node.Fields))
			for _, f := range node.Fields {
				if f.Name != "" && !strings.Contains(f.Name, ".") {
					fields = append(fields, f.Name)
				}
			}
			if len(fields) > 8 {
				fields = append(fields[:8], fmt.Sprintf("+%d", len(fields)-8))
			}
			if len(fields) > 0 {
				fmt.Fprintf(w, "       Fields:  %s%s%s\n", gray, strings.Join(fields, ", "), reset)
			}
		}

		fmt.Fprintln(w)
	}

	// Tables
	if len(r.Tables) > 0 {
		fmt.Fprintf(w, "  %s┌─ Database tables%s\n", yellow, reset)
		for _, t := range r.Tables {
			fmt.Fprintf(w, "  %s│   %s%s%s\n", yellow, bold, t, reset)
		}
		fmt.Fprintf(w, "  %s└─%s\n\n", yellow, reset)
	}

	// Env vars
	if len(r.EnvVars) > 0 {
		fmt.Fprintf(w, "  %s┌─ Environment variables%s\n", cyan, reset)
		for _, v := range r.EnvVars {
			fmt.Fprintf(w, "  %s│   %s%s\n", cyan, v, reset)
		}
		fmt.Fprintf(w, "  %s└─%s\n\n", cyan, reset)
	}

	if len(r.Chain) <= 1 {
		fmt.Fprintf(w, "  %sNote: Only the handler was found. Dependencies may use patterns%s\n", gray, reset)
		fmt.Fprintf(w, "  %s      not yet detected by reqflow (interface injection, closures).%s\n\n", gray, reset)
	}

	return nil
}

// renderSubCalls shows what a specific method calls (e.g. → svc.GetBudgets(), → store.Insert())
func renderSubCalls(w io.Writer, nodeID, method string, r *reqflow.TraceResult) {
	if r.MethodCalls == nil {
		return
	}
	key := nodeID + "." + method
	calls := r.MethodCalls[key]
	if len(calls) == 0 {
		return
	}
	for _, call := range calls {
		fmt.Fprintf(w, "         %s→ %s.%s()%s\n", gray, call.FieldName, call.TargetMethod, reset)
	}
}

// pkgFile shows "package/file.go:line" — e.g. "handler/handler.go:692"
func pkgFile(pkg, file string, line int) string {
	if file == "" {
		return ""
	}
	// Extract last 2-3 path segments from the package for context
	shortPkg := pkg
	parts := strings.Split(pkg, "/")
	if len(parts) > 2 {
		shortPkg = strings.Join(parts[len(parts)-2:], "/")
	}

	// Get just the filename
	fileParts := strings.Split(file, "/")
	fileName := fileParts[len(fileParts)-1]

	if line > 0 {
		return fmt.Sprintf("%s/%s:%d", shortPkg, fileName, line)
	}
	return fmt.Sprintf("%s/%s", shortPkg, fileName)
}

func shortFile(file string, line int) string {
	if file == "" {
		return ""
	}
	parts := strings.Split(file, "/")
	if len(parts) > 2 {
		parts = parts[len(parts)-2:]
	}
	if line > 0 {
		return strings.Join(parts, "/") + fmt.Sprintf(":%d", line)
	}
	return strings.Join(parts, "/")
}

func routeList(n *reqflow.Node) []string {
	raw := n.Meta["routes"]
	if raw == "" {
		raw = n.Meta["route"]
	}
	var out []string
	for _, r := range strings.Split(raw, "\n") {
		if r = strings.TrimSpace(r); r != "" {
			out = append(out, r)
		}
	}
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ─── HTML renderer ────────────────────────────────────────────────────────────

func renderTraceHTML(r *reqflow.TraceResult, w io.Writer) error {
	if r.NotFound {
		fmt.Fprintf(w, `<!DOCTYPE html><html><body style="font-family:monospace;padding:40px">
<h2 style="color:#f87171">✗ No handler found matching %q</h2>
<p>Run <code>reqflow -format json ./...</code> and search Meta["routes"] to see all registered routes.</p>
</body></html>`, r.Route)
		return nil
	}

	steps := buildHTMLSteps(r)
	extras := buildHTMLExtras(r)

	fmt.Fprintf(w, traceHTMLTmpl, r.Route, r.Route, steps, extras)
	return nil
}

func buildHTMLSteps(r *reqflow.TraceResult) string {
	var sb strings.Builder
	for i, node := range r.Chain {
		c := htmlKindColor(node.Kind)
		icon := traceIcon(node.Kind)
		label := kindLabel(node.Kind)

		if i > 0 {
			prev := r.Chain[i-1]
			edge := reqflow.EdgeLabel(prev, node)
			sb.WriteString(`<div class="arrow">↓ ` + edge + `</div>`)
		}

		sb.WriteString(`<div class="step">`)
		sb.WriteString(fmt.Sprintf(`<div class="step-icon" style="background:%s">%s</div>`, c, icon))
		sb.WriteString(`<div class="step-body">`)
		sb.WriteString(fmt.Sprintf(`<div class="step-kind" style="color:%s">%s</div>`, c, label))
		sb.WriteString(fmt.Sprintf(`<div class="step-name">%s</div>`, node.Name))

		var details []string
		if node.Package != "" {
			pkgParts := strings.Split(node.Package, "/")
			pkg := node.Package
			if len(pkgParts) > 3 {
				pkg = "…/" + strings.Join(pkgParts[len(pkgParts)-2:], "/")
			}
			details = append(details, `<code>`+pkg+`</code>`)
		}
		if node.File != "" {
			details = append(details, `<code>`+shortFile(node.File, node.Line)+`</code>`)
		}
		if len(details) > 0 {
			sb.WriteString(`<div class="step-meta">` + strings.Join(details, " · ") + `</div>`)
		}

		// Methods
		if len(node.Methods) > 0 {
			sb.WriteString(`<div class="step-methods">`)
			methods := node.Methods
			if len(methods) > 8 {
				methods = methods[:8]
			}
			for _, m := range methods {
				sb.WriteString(`<span class="method-badge">` + m + `()</span>`)
			}
			if len(node.Methods) > 8 {
				sb.WriteString(fmt.Sprintf(`<span class="method-badge muted">+%d more</span>`, len(node.Methods)-8))
			}
			sb.WriteString(`</div>`)
		}

		// Fields for models
		if node.Kind == reqflow.KindModel && len(node.Fields) > 0 {
			sb.WriteString(`<div class="step-fields">`)
			for i, f := range node.Fields {
				if i >= 10 || strings.Contains(f.Name, ".") {
					break
				}
				sb.WriteString(`<span class="field-badge">` + f.Name + `</span>`)
			}
			if len(node.Fields) > 10 {
				sb.WriteString(fmt.Sprintf(`<span class="field-badge muted">+%d</span>`, len(node.Fields)-10))
			}
			sb.WriteString(`</div>`)
		}

		sb.WriteString(`</div></div>`) // close step-body, step
	}
	return sb.String()
}

func buildHTMLExtras(r *reqflow.TraceResult) string {
	if len(r.Tables) == 0 && len(r.EnvVars) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(`<div class="extras">`)
	if len(r.Tables) > 0 {
		sb.WriteString(`<div class="extra-section"><h4>📦 Database Tables</h4>`)
		for _, t := range r.Tables {
			sb.WriteString(`<span class="table-badge">` + t + `</span>`)
		}
		sb.WriteString(`</div>`)
	}
	if len(r.EnvVars) > 0 {
		sb.WriteString(`<div class="extra-section"><h4>🔑 Environment Variables</h4>`)
		for _, v := range r.EnvVars {
			sb.WriteString(`<span class="env-badge">` + v + `</span>`)
		}
		sb.WriteString(`</div>`)
	}
	sb.WriteString(`</div>`)
	return sb.String()
}

func htmlKindColor(k reqflow.NodeKind) string {
	switch k {
	case reqflow.KindHandler:
		return "#34d399"
	case reqflow.KindService:
		return "#60a5fa"
	case reqflow.KindStore:
		return "#fbbf24"
	case reqflow.KindModel:
		return "#f87171"
	case reqflow.KindInterface:
		return "#a78bfa"
	case reqflow.KindGRPC:
		return "#2dd4bf"
	default:
		return "#6b7280"
	}
}

var traceHTMLTmpl = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>GoVis Trace — %s</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;background:#0f1117;color:#e2e8f0;min-height:100vh;padding:48px 24px}
.container{max-width:680px;margin:0 auto}
.header{margin-bottom:40px}
.header .label{font-size:.7rem;font-weight:700;text-transform:uppercase;letter-spacing:.1em;color:#6b7280;margin-bottom:8px}
.header h1{font-size:1.6rem;font-weight:800;color:#e2e8f0;font-family:monospace}
.header p{font-size:.8rem;color:#6b7280;margin-top:8px}
.step{display:flex;gap:16px;padding:20px;background:#1a1e28;border:1px solid #262a36;border-radius:12px;position:relative}
.step-icon{width:36px;height:36px;border-radius:50%%;display:flex;align-items:center;justify-content:center;font-size:.75rem;font-weight:800;color:#fff;flex-shrink:0;margin-top:2px}
.step-body{flex:1;min-width:0}
.step-kind{font-size:.6rem;text-transform:uppercase;font-weight:700;letter-spacing:.08em;margin-bottom:4px}
.step-name{font-size:1rem;font-weight:700;margin-bottom:6px}
.step-meta{font-size:.7rem;color:#6b7280;margin-bottom:8px}
.step-meta code{background:#262a36;padding:2px 6px;border-radius:3px;font-size:.65rem}
.step-methods{display:flex;flex-wrap:wrap;gap:4px;margin-top:4px}
.method-badge{background:#1f2937;border:1px solid #374151;padding:2px 8px;border-radius:4px;font-size:.65rem;font-family:monospace;color:#d1d5db}
.step-fields{display:flex;flex-wrap:wrap;gap:4px;margin-top:6px}
.field-badge{background:#0f1117;padding:2px 6px;border-radius:3px;font-size:.62rem;font-family:monospace;color:#6b7280}
.muted{opacity:.6}
.arrow{padding:12px 0 12px 52px;color:#374151;font-size:.72rem;font-style:italic}
.extras{margin-top:40px;padding-top:24px;border-top:1px solid #1f2937}
.extra-section{margin-bottom:20px}
.extra-section h4{font-size:.75rem;font-weight:700;margin-bottom:10px;color:#9ca3af}
.table-badge{display:inline-block;background:#292524;border:1px solid #57534e;padding:4px 12px;border-radius:6px;font-size:.75rem;font-family:monospace;color:#fbbf24;margin-right:8px;margin-bottom:6px}
.env-badge{display:inline-block;background:#172554;border:1px solid #1e3a8a;padding:4px 12px;border-radius:6px;font-size:.75rem;font-family:monospace;color:#93c5fd;margin-right:8px;margin-bottom:6px}
.reqflow-link{margin-top:48px;text-align:right;font-size:.65rem;color:#374151}
.reqflow-link a{color:#374151;text-decoration:none}
</style>
</head>
<body>
<div class="container">
<div class="header">
<div class="label">REQFLOW TRACE</div>
<h1>%s</h1>
<p>Complete static request path through your Go codebase</p>
</div>
%s
%s
<div class="reqflow-link"><a href="https://github.com/thzgajendra/reqflow">reqflow</a></div>
</div>
</body>
</html>`
