package structmap

import (
	"fmt"
	"go/ast"
	"strings"
	"unicode"

	"golang.org/x/tools/go/packages"
)

// ExtractTableMap inspects KindModel nodes for GORM/sqlx/bson struct tags
// and creates KindTable nodes representing the inferred database tables,
// linked via EdgeMapsTo edges.
func ExtractTableMap(pkgs []*packages.Package, graph *Graph) {
	// First, detect TableName() methods
	tableNameOverrides := make(map[string]string) // struct ID → table name
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				fn, ok := n.(*ast.FuncDecl)
				if !ok || fn.Recv == nil || fn.Name.Name != "TableName" {
					return true
				}
				if fn.Body == nil || len(fn.Body.List) == 0 {
					return true
				}

				// Get receiver type
				recvType := fn.Recv.List[0].Type
				if star, ok := recvType.(*ast.StarExpr); ok {
					recvType = star.X
				}
				ident, ok := recvType.(*ast.Ident)
				if !ok {
					return true
				}
				structID := fmt.Sprintf("%s.%s", pkg.PkgPath, ident.Name)

				// Extract return string literal
				for _, stmt := range fn.Body.List {
					if ret, ok := stmt.(*ast.ReturnStmt); ok && len(ret.Results) == 1 {
						if lit, ok := ret.Results[0].(*ast.BasicLit); ok {
							tableNameOverrides[structID] = strings.Trim(lit.Value, `"`)
						}
					}
				}
				return true
			})
		}
	}

	// Process model nodes
	for _, node := range graph.Nodes {
		if node.Kind != KindModel {
			continue
		}

		// Determine table name
		tableName := ""
		if override, ok := tableNameOverrides[node.ID]; ok {
			tableName = override
		} else {
			tableName = toSnakeCase(node.Name)
			// Simple pluralize
			if !strings.HasSuffix(tableName, "s") {
				tableName += "s"
			}
		}

		// Extract column info from fields
		var columns []string
		hasGormModel := false

		for _, field := range node.Fields {
			if strings.Contains(field.Type, "gorm.Model") || strings.Contains(field.Name, "gorm.Model") {
				hasGormModel = true
				continue
			}

			colName := ""

			// Check tags for explicit column names
			if field.Tag != "" {
				if col := extractTagValue(field.Tag, "gorm", "column"); col != "" {
					colName = col
				} else if col := extractTagValue(field.Tag, "db", ""); col != "" {
					colName = col
				} else if col := extractTagValue(field.Tag, "bson", ""); col != "" {
					colName = col
				}
			}

			// Fall back to snake_case of field name
			if colName == "" && field.Name != "" && !strings.Contains(field.Name, ".") {
				colName = toSnakeCase(field.Name)
			}

			if colName != "" && colName != "-" {
				columns = append(columns, colName)
			}
		}

		// Add standard gorm.Model columns
		if hasGormModel {
			columns = append([]string{"id", "created_at", "updated_at", "deleted_at"}, columns...)
		}

		// Create table node
		tableID := fmt.Sprintf("table.%s", tableName)
		tableNode := &Node{
			ID:      tableID,
			Kind:    KindTable,
			Name:    tableName,
			Package: "database",
			File:    node.File,
			Line:    node.Line,
			Meta: map[string]string{
				"table_name": tableName,
				"columns":    strings.Join(columns, ", "),
				"model":      node.Name,
			},
		}

		graph.AddNode(tableNode)
		graph.AddEdge(node.ID, tableID, EdgeMapsTo)
	}
}

// extractTagValue extracts a value from a struct tag.
// For gorm tags: `gorm:"column:name"` with key="gorm", subkey="column"
// For simple tags: `db:"name"` with key="db", subkey=""
func extractTagValue(tag, key, subkey string) string {
	// Remove backticks
	tag = strings.Trim(tag, "`")

	// Find the key
	idx := strings.Index(tag, key+`:"`)
	if idx < 0 {
		return ""
	}
	rest := tag[idx+len(key)+2:]
	endIdx := strings.Index(rest, `"`)
	if endIdx < 0 {
		return ""
	}
	tagContent := rest[:endIdx]

	if subkey == "" {
		// Return first value before comma
		if commaIdx := strings.Index(tagContent, ","); commaIdx >= 0 {
			return tagContent[:commaIdx]
		}
		return tagContent
	}

	// Search for subkey:value within the tag content
	for _, part := range strings.Split(tagContent, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, subkey+":") {
			return strings.TrimPrefix(part, subkey+":")
		}
	}
	return ""
}

// toSnakeCase converts CamelCase to snake_case.
func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result = append(result, '_')
			}
			result = append(result, unicode.ToLower(r))
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}
