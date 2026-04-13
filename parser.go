package reqflow

import (
	"fmt"
	"go/ast"
	"go/types"
	"os"
	"strings"

	"golang.org/x/tools/go/packages"
)

// ParseOptions configures the Parse function. Dir is the Go module directory
// to analyze. Boolean flags enable optional analysis passes (call graph,
// data flow, env map, etc.).
type ParseOptions struct {
	Dir        string
	Filter     string
	Focus      string
	Config     *ReqflowConfig
	APIMap     bool
	Heatmap    bool
	CallGraph  bool
	DataFlow   bool
	EnvMap       bool
	TableMap     bool
	DepTree      bool
	InfraTopo    bool
	Churn        bool
	Contributors bool
	PRImpact     string // base ref for PR impact (e.g. "main")
	Evolution    string // comma-separated git tags
	Proto        bool
	ServiceMap   bool
	OtelTrace    string // path to OTLP JSON export file
}

// Parse loads Go packages from the target directory and builds the
// full architecture graph through a multi-pass analysis pipeline.
func Parse(opts ParseOptions) (*Graph, error) {
	mode := packages.NeedName |
		packages.NeedSyntax |
		packages.NeedTypes |
		packages.NeedTypesInfo |
		packages.NeedImports

	if opts.CallGraph {
		mode |= packages.NeedDeps
	}

	dir := opts.Dir
	pattern := "./..."
	if dir == "./..." || dir == "" {
		dir = "."
	} else if strings.HasSuffix(dir, "/...") {
		dir = strings.TrimSuffix(dir, "/...")
	}

	cfg := &packages.Config{
		Mode: mode,
		Dir:  dir,
	}

	pkgs, err := packages.Load(cfg, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to load packages: %w", err)
	}

	graph := NewGraph()

	// Pass 1: Harvest structs, interfaces, and functions
	for _, pkg := range pkgs {
		// Apply ignore_packages filter from .reqflow.yml
		if opts.Config != nil && shouldIgnorePackage(pkg.PkgPath, opts.Config.Parser.IgnorePackages) {
			continue
		}
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				switch t := n.(type) {
				case *ast.TypeSpec:
					handleTypeSpec(t, pkg, graph, opts)
				case *ast.FuncDecl:
					handleFuncDecl(t, pkg, graph)
				}
				return true
			})
		}
	}

	// Pass 2: Resolve structural relationships
	resolveInterfaces(pkgs, graph)
	resolveDependencies(graph)

	// Pass 3: Framework-level enrichments
	extractRoutes(pkgs, graph)
	extractEvents(pkgs, graph)
	ExtractMiddleware(pkgs, graph)
	ExtractGRPC(pkgs, graph)

	// Pass 3b: API surface map (request/response type extraction)
	if opts.APIMap {
		ExtractAPIMap(pkgs, graph)
	}

	// Pass 3c: Call graph visualization
	if opts.CallGraph {
		modulePath := getModulePath(opts.Dir)
		if modulePath != "" {
			ExtractCallGraph(pkgs, graph, modulePath)
		}
	}

	// Pass 3d: Data flow extraction
	if opts.DataFlow {
		ExtractDataFlows(graph)
	}

	// Pass 4: Infrastructure & external topology
	parseVitessSchema(dir, graph)
	ExtractGoModDeps(dir, graph)

	// Pass 4b: Environment variable map
	if opts.EnvMap {
		ExtractEnvMap(pkgs, graph)
	}

	// Pass 4c: Model-to-table mapping
	if opts.TableMap {
		ExtractTableMap(pkgs, graph)
	}

	// Pass 4d: Full go.mod dependency tree
	if opts.DepTree {
		ExtractDepTree(dir, graph)
	}

	// Pass 4e: Docker/K8s infrastructure topology
	if opts.InfraTopo {
		ExtractInfraTopo(dir, graph)
	}

	// Pass 4f: Proto/gRPC contract graph
	if opts.Proto {
		ExtractProto(dir, graph)
	}

	// Pass 4g: OpenTelemetry trace overlay
	if opts.OtelTrace != "" {
		ExtractOtelTrace(opts.OtelTrace, graph)
	}

	// Pass 5: Runtime pattern detection
	DetectConcurrency(pkgs, graph)

	// Pass 5b: Git-based analysis
	if opts.Churn {
		ExtractChurn(dir, graph)
	}
	if opts.Contributors {
		ExtractContributors(dir, graph)
	}
	if opts.PRImpact != "" {
		ExtractPRImpact(dir, opts.PRImpact, graph)
	}

	// Pass 6: Scope filtering (always last)
	if opts.Focus != "" {
		applyFocus(graph, opts.Focus)
	}

	return graph, nil
}

// shouldIgnorePackage checks if a package path matches any ignore pattern.
func shouldIgnorePackage(pkgPath string, ignorePatterns []string) bool {
	for _, pattern := range ignorePatterns {
		if strings.Contains(pkgPath, pattern) {
			return true
		}
	}
	return false
}

// handleTypeSpec processes a single type declaration (struct or interface).
func handleTypeSpec(t *ast.TypeSpec, pkg *packages.Package, graph *Graph, opts ParseOptions) {
	lowerName := strings.ToLower(t.Name.Name)
	if strings.Contains(lowerName, "mock") || strings.Contains(lowerName, "test") {
		return
	}

	obj := pkg.TypesInfo.Defs[t.Name]
	if obj == nil {
		return
	}

	id := fmt.Sprintf("%s.%s", pkg.PkgPath, t.Name.Name)
	node := &Node{
		ID:      id,
		Name:    t.Name.Name,
		Package: pkg.PkgPath,
		File:    pkg.Fset.Position(t.Pos()).Filename,
		Line:    pkg.Fset.Position(t.Pos()).Line,
	}

	switch structOrIface := t.Type.(type) {
	case *ast.StructType:
		node.Kind = KindStruct

		isService := matchLayer(t.Name.Name, pkg.PkgPath, serviceKeywords)
		isStore := matchLayer(t.Name.Name, pkg.PkgPath, storeKeywords)
		isModel := matchLayer(t.Name.Name, pkg.PkgPath, modelKeywords)
		isHandler := matchLayer(t.Name.Name, pkg.PkgPath, handlerKeywords)

		if opts.Config != nil {
			if opts.Config.ServiceRegex != nil {
				isService = opts.Config.ServiceRegex.MatchString(t.Name.Name)
			}
			if opts.Config.StoreRegex != nil {
				isStore = opts.Config.StoreRegex.MatchString(t.Name.Name)
			}
			if opts.Config.ModelRegex != nil {
				isModel = opts.Config.ModelRegex.MatchString(t.Name.Name)
			}
		}

		if isStore {
			node.Kind = KindStore
		} else if isService {
			node.Kind = KindService
		} else if isModel {
			node.Kind = KindModel
		} else if isHandler {
			node.Kind = KindHandler
		}

		hasDBTags := false
		hasDBField := false
		for _, field := range structOrIface.Fields.List {
			tag := ""
			if field.Tag != nil {
				tag = field.Tag.Value
				if strings.Contains(tag, `gorm:`) || strings.Contains(tag, `db:`) || strings.Contains(tag, `bson:`) || strings.Contains(tag, `json:`) {
					hasDBTags = true
				}
			}
			typStr := ""
			if typObj := pkg.TypesInfo.TypeOf(field.Type); typObj != nil {
				typStr = typObj.String()
				if strings.Contains(typStr, "gorm.Model") {
					hasDBTags = true
				}
				// Structural store detection: field of a known DB client type
				if isDBClientType(typStr) {
					hasDBField = true
				}
			}

			if len(field.Names) == 0 {
				node.Fields = append(node.Fields, Field{Name: typStr, Type: typStr, Tag: tag})
				graph.AddEdge(node.ID, typStr, EdgeEmbeds)
			} else {
				for _, name := range field.Names {
					node.Fields = append(node.Fields, Field{Name: name.Name, Type: typStr, Tag: tag})
				}
			}
		}

		// Structural override: DB client field → always a store
		if hasDBField {
			node.Kind = KindStore
		} else if hasDBTags && node.Kind == KindStruct {
			node.Kind = KindModel
		}

	case *ast.InterfaceType:
		node.Kind = KindInterface
		if matchLayer(t.Name.Name, pkg.PkgPath, storeKeywords) {
			node.Kind = KindStore
		} else if matchLayer(t.Name.Name, pkg.PkgPath, serviceKeywords) {
			node.Kind = KindService
		} else if matchLayer(t.Name.Name, pkg.PkgPath, handlerKeywords) {
			node.Kind = KindHandler
		} else if matchLayer(t.Name.Name, pkg.PkgPath, modelKeywords) {
			node.Kind = KindModel
		}

		for _, method := range structOrIface.Methods.List {
			if len(method.Names) > 0 {
				node.Methods = append(node.Methods, method.Names[0].Name)
			} else if embType := pkg.TypesInfo.TypeOf(method.Type); embType != nil {
				graph.AddEdge(node.ID, embType.String(), EdgeEmbeds)
			}
		}
	default:
		return
	}

	graph.AddNode(node)
}

// handleFuncDecl processes a function declaration, detecting HTTP handlers
// and constructor patterns.
func handleFuncDecl(fn *ast.FuncDecl, pkg *packages.Package, graph *Graph) {
	if fn.Recv != nil {
		recvType := fn.Recv.List[0].Type
		if star, ok := recvType.(*ast.StarExpr); ok {
			recvType = star.X
		}
		if ident, ok := recvType.(*ast.Ident); ok {
			id := fmt.Sprintf("%s.%s", pkg.PkgPath, ident.Name)
			if node, ok := graph.Nodes[id]; ok {
				node.Methods = append(node.Methods, fn.Name.Name)
				if isHTTPHandler(fn, pkg.TypesInfo) {
					node.Kind = KindHandler
					node.Meta["http_method"] = fn.Name.Name
				}
			}
		}
	} else {
		id := fmt.Sprintf("%s.%s", pkg.PkgPath, fn.Name.Name)
		node := &Node{
			ID:      id,
			Kind:    KindFunc,
			Name:    fn.Name.Name,
			Package: pkg.PkgPath,
			File:    pkg.Fset.Position(fn.Pos()).Filename,
			Line:    pkg.Fset.Position(fn.Pos()).Line,
			Meta:    make(map[string]string),
		}

		if isHTTPHandler(fn, pkg.TypesInfo) {
			node.Kind = KindHandler
		}
		if strings.HasPrefix(fn.Name.Name, "New") {
			node.Meta["is_constructor"] = "true"
			extractDependencies(fn, pkg.TypesInfo, node)
		}

		graph.AddNode(node)
	}
}

// isHTTPHandler detects HTTP handler functions for net/http, Gin, Echo, and Fiber.
func isHTTPHandler(fn *ast.FuncDecl, info *types.Info) bool {
	if fn.Type.Params == nil || len(fn.Type.Params.List) == 0 {
		return false
	}

	if len(fn.Type.Params.List) == 1 {
		t0 := info.TypeOf(fn.Type.Params.List[0].Type)
		if t0 != nil {
			typeStr := t0.String()
			if strings.Contains(typeStr, "gin.Context") ||
				strings.Contains(typeStr, "echo.Context") ||
				strings.Contains(typeStr, "fiber.Ctx") ||
				strings.Contains(typeStr, "gofr.Context") {
				return true
			}
		}
	}

	if len(fn.Type.Params.List) == 2 {
		t0 := info.TypeOf(fn.Type.Params.List[0].Type)
		t1 := info.TypeOf(fn.Type.Params.List[1].Type)
		if t0 != nil && t1 != nil {
			if strings.HasSuffix(t0.String(), "net/http.ResponseWriter") && strings.HasSuffix(t1.String(), "net/http.Request") {
				return true
			}
		}
	}

	return false
}

func extractDependencies(fn *ast.FuncDecl, info *types.Info, node *Node) {
	if fn.Type.Params == nil {
		return
	}
	var deps []string
	for _, p := range fn.Type.Params.List {
		typ := info.TypeOf(p.Type)
		if typ != nil {
			deps = append(deps, typ.String())
		}
	}
	node.Meta["deps"] = strings.Join(deps, ",")
}

func resolveInterfaces(pkgs []*packages.Package, graph *Graph) {
	structTypes := make(map[string]types.Type)
	ifaceTypes := make(map[string]*types.Interface)

	for _, p := range pkgs {
		scope := p.Types.Scope()
		for _, name := range scope.Names() {
			obj := scope.Lookup(name)
			if t, ok := obj.(*types.TypeName); ok {
				if typ, ok := t.Type().Underlying().(*types.Struct); ok {
					structTypes[fmt.Sprintf("%s.%s", p.PkgPath, name)] = t.Type()
					_ = typ
				} else if iface, ok := t.Type().Underlying().(*types.Interface); ok {
					ifaceTypes[fmt.Sprintf("%s.%s", p.PkgPath, name)] = iface
				}
			}
		}
	}

	for sID, sTyp := range structTypes {
		pTyp := types.NewPointer(sTyp)
		for iID, iTyp := range ifaceTypes {
			if iTyp.NumMethods() == 0 {
				continue
			}
			if types.Implements(sTyp, iTyp) || types.Implements(pTyp, iTyp) {
				if _, sok := graph.Nodes[sID]; sok {
					if _, iok := graph.Nodes[iID]; iok {
						graph.AddEdge(sID, iID, EdgeImplements)
					}
				}
			}
		}
	}
}

func resolveDependencies(graph *Graph) {
	for _, n := range graph.Nodes {
		// Constructor deps (New* functions): link constructor → dependency
		if deps, ok := n.Meta["deps"]; ok && deps != "" {
			for _, d := range strings.Split(deps, ",") {
				cleanDep := strings.TrimLeft(d, "*")
				if _, targetExists := graph.Nodes[cleanDep]; targetExists {
					graph.AddEdge(n.ID, cleanDep, EdgeDepends)
				}
			}
		}

		// Struct field deps: if a field type is a known node, add an edge.
		// This handles patterns like:  svc service.Service  or  store *UserStore
		// which are the most common ways Go layers wire together.
		if n.Kind == KindHandler || n.Kind == KindService || n.Kind == KindStore ||
			n.Kind == KindGRPC || n.Kind == KindMiddleware {
			for _, f := range n.Fields {
				cleanType := strings.TrimLeft(f.Type, "*")
				// Skip basic/stdlib types and empty
				if cleanType == "" || !strings.Contains(cleanType, ".") {
					continue
				}
				if _, exists := graph.Nodes[cleanType]; exists {
					graph.AddEdge(n.ID, cleanType, EdgeDepends)
				}
			}
		}
	}
}

// getModulePath extracts the module path from go.mod in the given directory.
func getModulePath(dir string) string {
	modDir := dir
	if modDir == "./..." || modDir == "" {
		modDir = "."
	}
	data, err := os.ReadFile(modDir + "/go.mod")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module"))
		}
	}
	return ""
}

var (
	serviceKeywords = []string{"service", "usecase", "interactor", "manager", "orchestrator", "worker", "processor", "biz"}
	storeKeywords   = []string{"repository", "repo", "store", "dao", "data", "gateway", "adapter", "persistence", "storage", "client"}
	modelKeywords   = []string{"model", "entity", "dto", "record", "schema", "domain", "aggregate"}
	handlerKeywords = []string{"handler", "controller", "endpoint", "transport", "api", "resource"}

	// dbClientTypes are field type suffixes that indicate a struct is a database store.
	dbClientTypes = []string{
		"sql.DB", "sqlx.DB", "gorm.DB", "pgx.Conn", "pgxpool.Pool",
		"mongo.Client", "redis.Client", "redis.ClusterClient",
		"cassandra.Session", "dynamodb.Client",
		"sqlx.NamedStmt", "sql.Tx", "sqlx.Tx",
	}
)

// isDBClientType returns true if the type string contains a known database client type.
func isDBClientType(typStr string) bool {
	for _, dbType := range dbClientTypes {
		if strings.HasSuffix(typStr, dbType) || strings.Contains(typStr, dbType) {
			return true
		}
	}
	return false
}

// matchLayer checks if a node belongs to an architectural layer based on its name or package
func matchLayer(name string, pkgPath string, keywords []string) bool {
	lowerName := strings.ToLower(name)
	for _, kw := range keywords {
		if strings.HasSuffix(lowerName, kw) {
			return true
		}
	}
	parts := strings.Split(pkgPath, "/")
	if len(parts) > 0 {
		pkgName := strings.ToLower(parts[len(parts)-1])
		for _, kw := range keywords {
			if pkgName == kw || pkgName == kw+"s" {
				return true
			}
		}
	}
	return false
}
