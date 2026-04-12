package structmap

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

// Options allows configuring the parser
type Options struct {
	Dir    string
	Filter string
}

// Parse loads packages and builds the graph
func Parse(opts Options) (*Graph, error) {
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

	// 1. Initial pass: harvest basic structures and funcs
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				switch t := n.(type) {
				case *ast.TypeSpec:
					handleTypeSpec(t, pkg, graph)
				case *ast.FuncDecl:
					handleFuncDecl(t, pkg, graph)
				}
				return true
			})
		}
	}

	// 2. Resolve relationships
	resolveInterfaces(pkgs, graph)
	resolveDependencies(graph)

	return graph, nil
}

func handleTypeSpec(t *ast.TypeSpec, pkg *packages.Package, graph *Graph) {
	// Skip mocks and generated tests
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
		
		if strings.HasSuffix(lowerName, "repository") || strings.HasSuffix(lowerName, "store") || strings.HasSuffix(lowerName, "dao") {
			node.Kind = KindStore
		} else if strings.HasSuffix(lowerName, "service") || strings.HasSuffix(lowerName, "usecase") {
			node.Kind = KindService
		} else if strings.HasSuffix(lowerName, "model") {
			node.Kind = KindModel
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
				// Embedded field
				node.Fields = append(node.Fields, Field{Name: typStr, Type: typStr, Tag: tag})
				// Create embeds edge
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
		if strings.HasSuffix(lowerName, "repository") || strings.HasSuffix(lowerName, "store") || strings.HasSuffix(lowerName, "dao") {
			node.Kind = KindStore
		} else if strings.HasSuffix(lowerName, "service") || strings.HasSuffix(lowerName, "usecase") {
			node.Kind = KindService
		}

		for _, method := range structOrIface.Methods.List {
			if len(method.Names) > 0 {
				node.Methods = append(node.Methods, method.Names[0].Name)
			} else if embType := pkg.TypesInfo.TypeOf(method.Type); embType != nil {
				// Embedded interface
				graph.AddEdge(node.ID, embType.String(), EdgeEmbeds)
			}
		}
	default:
		return
	}

	graph.AddNode(node)
}

func handleFuncDecl(fn *ast.FuncDecl, pkg *packages.Package, graph *Graph) {
	if fn.Recv != nil {
		// It's a method attached to a struct pointer/value
		recvType := fn.Recv.List[0].Type
		if star, ok := recvType.(*ast.StarExpr); ok {
			recvType = star.X
		}
		if ident, ok := recvType.(*ast.Ident); ok {
			id := fmt.Sprintf("%s.%s", pkg.PkgPath, ident.Name)
			if node, ok := graph.Nodes[id]; ok {
				node.Methods = append(node.Methods, fn.Name.Name)
				// Check if it's an HTTP handler
				if isHTTPHandler(fn, pkg.TypesInfo) {
					node.Kind = KindHandler
					node.Meta["http_method"] = fn.Name.Name
				}
			}
		}
	} else {
		// Package block function
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

func isHTTPHandler(fn *ast.FuncDecl, info *types.Info) bool {
	if fn.Type.Params == nil || len(fn.Type.Params.List) == 0 {
		return false
	}

	// Single parameter handlers (Gin, Echo, Fiber)
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

	// Two parameter handlers (net/http standard)
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
	// Pre-collect interfaces and structs for lookup
	structTypes := make(map[string]types.Type)
	ifaceTypes := make(map[string]*types.Interface)

	for _, p := range pkgs {
		scope := p.Types.Scope()
		for _, name := range scope.Names() {
			obj := scope.Lookup(name)
			if t, ok := obj.(*types.TypeName); ok {
				if typ, ok := t.Type().Underlying().(*types.Struct); ok {
					structTypes[fmt.Sprintf("%s.%s", p.PkgPath, name)] = t.Type()
					_ = typ // keep compiler happy
				} else if iface, ok := t.Type().Underlying().(*types.Interface); ok {
					ifaceTypes[fmt.Sprintf("%s.%s", p.PkgPath, name)] = iface
				}
			}
		}
	}

	// Calculate implementations
	for sID, sTyp := range structTypes {
		pTyp := types.NewPointer(sTyp)
		for iID, iTyp := range ifaceTypes {
			// Ignore `interface{}`
			if iTyp.NumMethods() == 0 {
				continue
			}

			if types.Implements(sTyp, iTyp) || types.Implements(pTyp, iTyp) {
				// We only add edges if both are in our graph
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
				// Simple heuristic: if the dependency is a pointer or interface that matches another node
				cleanDep := strings.TrimLeft(d, "*")
				if _, targetExists := graph.Nodes[cleanDep]; targetExists {
					graph.AddEdge(n.ID, cleanDep, EdgeDepends)
				}
			}
		}
	}
}
