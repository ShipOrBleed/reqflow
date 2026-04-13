package structmap

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

// ParseOptions allows configuring the parser
type ParseOptions struct {
	Dir    string
	Filter string
	Focus  string
	Config *GovisConfig
}

// Parse loads Go packages from the target directory and builds the
// full architecture graph through a multi-pass analysis pipeline.
func Parse(opts ParseOptions) (*Graph, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedImports,
		Dir: opts.Dir,
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("failed to load packages: %w", err)
	}

	graph := NewGraph()

	// Pass 1: Harvest structs, interfaces, and functions
	for _, pkg := range pkgs {
		// Apply ignore_packages filter from .govis.yml
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

	// Pass 4: Infrastructure & external topology
	parseVitessSchema(opts.Dir, graph)
	ExtractGoModDeps(opts.Dir, graph)

	// Pass 5: Runtime pattern detection
	DetectConcurrency(pkgs, graph)

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

		if hasDBTags && node.Kind == KindStruct {
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
				strings.Contains(typeStr, "fiber.Ctx") {
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
		if deps, ok := n.Meta["deps"]; ok && deps != "" {
			for _, d := range strings.Split(deps, ",") {
				cleanDep := strings.TrimLeft(d, "*")
				if _, targetExists := graph.Nodes[cleanDep]; targetExists {
					graph.AddEdge(n.ID, cleanDep, EdgeDepends)
				}
			}
		}
	}
}

var (
	serviceKeywords = []string{"service", "usecase", "interactor", "manager", "orchestrator", "worker", "processor", "biz"}
	storeKeywords   = []string{"repository", "repo", "store", "dao", "data", "gateway", "adapter", "persistence", "storage", "client"}
	modelKeywords   = []string{"model", "entity", "dto", "record", "schema", "domain", "aggregate"}
	handlerKeywords = []string{"handler", "controller", "endpoint", "transport", "api", "resource"}
)

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
