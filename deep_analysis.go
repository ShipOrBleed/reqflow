package structmap

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

// ============================================================
// CONCURRENCY PATTERN DETECTION
// ============================================================

// DetectConcurrency scans AST for goroutine launches, channels, mutexes,
// and WaitGroups. Tags nodes with concurrency metadata.
func DetectConcurrency(pkgs []*packages.Package, graph *Graph) {
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			var currentStructID string
			ast.Inspect(file, func(n ast.Node) bool {
				if fn, ok := n.(*ast.FuncDecl); ok {
					if fn.Recv != nil && len(fn.Recv.List) > 0 {
						if star, ok := fn.Recv.List[0].Type.(*ast.StarExpr); ok {
							if ident, ok := star.X.(*ast.Ident); ok {
								currentStructID = fmt.Sprintf("%s.%s", pkg.PkgPath, ident.Name)
							}
						} else if ident, ok := fn.Recv.List[0].Type.(*ast.Ident); ok {
							currentStructID = fmt.Sprintf("%s.%s", pkg.PkgPath, ident.Name)
						}
					}
				}

				if _, ok := n.(*ast.GoStmt); ok {
					tagConcurrency(graph, currentStructID, "goroutines")
				}

				if call, ok := n.(*ast.CallExpr); ok {
					if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "make" {
						if len(call.Args) > 0 {
							if _, ok := call.Args[0].(*ast.ChanType); ok {
								tagConcurrency(graph, currentStructID, "channels")
							}
						}
					}
				}

				if sel, ok := n.(*ast.SelectorExpr); ok {
					if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "sync" {
						switch sel.Sel.Name {
						case "Mutex", "RWMutex":
							tagConcurrency(graph, currentStructID, "mutexes")
						case "WaitGroup":
							tagConcurrency(graph, currentStructID, "waitgroups")
						}
					}
				}
				return true
			})
		}
	}
}

func tagConcurrency(graph *Graph, nodeID, kind string) {
	if nodeID == "" {
		return
	}
	node, exists := graph.Nodes[nodeID]
	if !exists {
		return
	}
	current := 0
	if val, ok := node.Meta["concurrency_"+kind]; ok {
		current, _ = strconv.Atoi(val)
	}
	current++
	node.Meta["concurrency_"+kind] = strconv.Itoa(current)
}

// ============================================================
// TEST COVERAGE CORRELATION
// ============================================================

// LoadCoverageProfile reads a Go coverage profile and maps coverage
// data to architectural nodes.
func LoadCoverageProfile(coverPath string, graph *Graph) error {
	file, err := os.Open(coverPath)
	if err != nil {
		return err
	}
	defer file.Close()

	fileCoverage := make(map[string]struct {
		total   int
		covered int
	})

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if lineNum == 1 {
			continue
		}
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) != 3 {
			continue
		}
		filePart := parts[0]
		colonIdx := strings.Index(filePart, ":")
		if colonIdx < 0 {
			continue
		}
		fileName := filePart[:colonIdx]
		statements, _ := strconv.Atoi(parts[1])
		count, _ := strconv.Atoi(parts[2])

		entry := fileCoverage[fileName]
		entry.total += statements
		if count > 0 {
			entry.covered += statements
		}
		fileCoverage[fileName] = entry
	}

	for _, node := range graph.Nodes {
		if node.File == "" {
			continue
		}
		for covFile, cov := range fileCoverage {
			if strings.HasSuffix(covFile, node.File) || strings.Contains(covFile, node.Package) {
				if cov.total > 0 {
					pct := float64(cov.covered) / float64(cov.total) * 100
					node.Meta["coverage"] = fmt.Sprintf("%.0f%%", pct)
					if pct < 30 {
						node.Meta["coverage_risk"] = "critical"
					} else if pct < 60 {
						node.Meta["coverage_risk"] = "low"
					} else {
						node.Meta["coverage_risk"] = "healthy"
					}
				}
				break
			}
		}
	}
	return nil
}

// ============================================================
// TECHNICAL DEBT SCANNER (TODO/FIXME/HACK)
// ============================================================

type TechDebt struct {
	File    string
	Line    int
	Kind    string
	Comment string
	NodeID  string
}

// DetectTechDebt scans AST comments for TODO/FIXME/HACK markers and
// maps them to the closest architectural node.
func DetectTechDebt(pkgs []*packages.Package, graph *Graph) []TechDebt {
	var results []TechDebt
	debtPatterns := regexp.MustCompile(`(?i)\b(TODO|FIXME|HACK|XXX|DEPRECATED)\b`)

	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			for _, cg := range file.Comments {
				for _, comment := range cg.List {
					matches := debtPatterns.FindAllString(comment.Text, -1)
					if len(matches) == 0 {
						continue
					}
					pos := pkg.Fset.Position(comment.Pos())
					closestNode := findClosestNode(graph, pos.Filename, pos.Line)

					results = append(results, TechDebt{
						File:    pos.Filename,
						Line:    pos.Line,
						Kind:    strings.ToUpper(matches[0]),
						Comment: strings.TrimSpace(comment.Text),
						NodeID:  closestNode,
					})

					if closestNode != "" {
						if node, ok := graph.Nodes[closestNode]; ok {
							current := 0
							if val, existed := node.Meta["tech_debt"]; existed {
								current, _ = strconv.Atoi(val)
							}
							current++
							node.Meta["tech_debt"] = strconv.Itoa(current)
						}
					}
				}
			}
		}
	}
	return results
}

func findClosestNode(graph *Graph, file string, line int) string {
	bestID := ""
	bestDist := 999999
	for id, n := range graph.Nodes {
		if n.File == file {
			dist := line - n.Line
			if dist >= 0 && dist < bestDist {
				bestDist = dist
				bestID = id
			}
		}
	}
	return bestID
}

// ============================================================
// CONSTRUCTOR PATTERN VALIDATION
// ============================================================

type MissingConstructor struct {
	StructName string
	Package    string
	File       string
	Line       int
}

// DetectMissingConstructors finds structs that lack a NewXxx() factory function.
func DetectMissingConstructors(pkgs []*packages.Package, graph *Graph) []MissingConstructor {
	constructors := make(map[string]bool)
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				if fn, ok := n.(*ast.FuncDecl); ok {
					if fn.Recv == nil && strings.HasPrefix(fn.Name.Name, "New") {
						constructors[strings.TrimPrefix(fn.Name.Name, "New")] = true
					}
				}
				return true
			})
		}
	}

	var missing []MissingConstructor
	for _, n := range graph.Nodes {
		if n.Kind == KindStruct || n.Kind == KindService || n.Kind == KindStore || n.Kind == KindHandler {
			if !constructors[n.Name] {
				missing = append(missing, MissingConstructor{
					StructName: n.Name,
					Package:    n.Package,
					File:       n.File,
					Line:       n.Line,
				})
				n.Meta["missing_constructor"] = "true"
			}
		}
	}
	return missing
}

// ============================================================
// SECURITY ANTI-PATTERN DETECTION
// ============================================================

type SecurityIssue struct {
	File     string
	Line     int
	Kind     string
	Detail   string
	Severity string
}

// DetectSecurityIssues scans for hardcoded secrets, SQL injection risk,
// and weak cryptographic hash imports.
func DetectSecurityIssues(pkgs []*packages.Package) []SecurityIssue {
	var issues []SecurityIssue

	secretPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(sk[-_]live|sk[-_]test)[a-zA-Z0-9]{20,}`),
		regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
		regexp.MustCompile(`(?i)(password|secret|apikey|api_key)\s*[:=]\s*"[^"]+`),
		regexp.MustCompile(`ghp_[a-zA-Z0-9]{36}`),
		regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9._-]{20,}`),
	}

	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
					for _, pattern := range secretPatterns {
						if pattern.MatchString(lit.Value) {
							issues = append(issues, SecurityIssue{
								File:     pkg.Fset.Position(lit.Pos()).Filename,
								Line:     pkg.Fset.Position(lit.Pos()).Line,
								Kind:     "hardcoded_secret",
								Detail:   "Potential hardcoded secret/credential in string literal",
								Severity: "critical",
							})
							break
						}
					}
				}

				if call, ok := n.(*ast.CallExpr); ok {
					if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
						method := sel.Sel.Name
						if method == "Query" || method == "Exec" || method == "QueryRow" {
							for _, arg := range call.Args {
								if binExpr, ok := arg.(*ast.BinaryExpr); ok && binExpr.Op == token.ADD {
									issues = append(issues, SecurityIssue{
										File:     pkg.Fset.Position(call.Pos()).Filename,
										Line:     pkg.Fset.Position(call.Pos()).Line,
										Kind:     "sql_injection",
										Detail:   fmt.Sprintf("String concatenation in %s() — use parameterized queries", method),
										Severity: "critical",
									})
								}
							}
						}
					}
				}

				if imp, ok := n.(*ast.ImportSpec); ok {
					importPath := strings.Trim(imp.Path.Value, "\"")
					if importPath == "crypto/md5" || importPath == "crypto/sha1" {
						issues = append(issues, SecurityIssue{
							File:     pkg.Fset.Position(imp.Pos()).Filename,
							Line:     pkg.Fset.Position(imp.Pos()).Line,
							Kind:     "weak_crypto",
							Detail:   fmt.Sprintf("Weak hash: %s — use crypto/sha256+", importPath),
							Severity: "high",
						})
					}
				}
				return true
			})
		}
	}
	return issues
}
