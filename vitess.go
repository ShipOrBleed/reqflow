package structmap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// VSchema represents a simplified Vitess schema JSON definition
type VSchema struct {
	Sharded bool              `json:"sharded"`
	Tables  map[string]VTable `json:"tables"`
}

type VTable struct {
	ColumnVindexes []struct {
		Column string `json:"column"`
		Name   string `json:"name"`
	} `json:"column_vindexes"`
}

// parseVitessSchema enriches the graph with PlanetScale/Vitess topologies
func parseVitessSchema(dir string, graph *Graph) {
	vschemaPath := filepath.Join(dir, "vschema.json")
	
	bytes, err := os.ReadFile(vschemaPath)
	if err != nil {
		return // Silently pass if no vitess schema is found
	}
	
	var schema map[string]VSchema
	if err := json.Unmarshal(bytes, &schema); err != nil {
		return // Invalid JSON vschema
	}

	for keyspaceName, keyspaceDef := range schema {
		for _, n := range graph.Nodes {
			if n.Kind == KindModel || n.Kind == KindStore || n.Kind == KindStruct {
				for tableName, tableDef := range keyspaceDef.Tables {
					if equalFuzzy(n.Name, tableName) {
						n.Meta["vitess_keyspace"] = keyspaceName
						if keyspaceDef.Sharded {
							n.Meta["vitess_sharded"] = "true"
						} else {
							n.Meta["vitess_sharded"] = "false"
						}
						
						var vindexes []string
						for _, vidx := range tableDef.ColumnVindexes {
							vindexes = append(vindexes, vidx.Column)
						}
						if len(vindexes) > 0 {
							n.Meta["vitess_vindex"] = strings.Join(vindexes, ", ")
						}
					}
				}
			}
		}
	}
}

func equalFuzzy(structName, tableName string) bool {
	sn := strings.ToLower(structName)
	tn := strings.ToLower(tableName)
	if sn == tn || sn+"s" == tn || tn+"s" == sn || sn+"es" == tn {
		return true
	}
	// Direct containment heuristic
	if strings.Contains(tn, sn) && len(sn) > 3 {
		return true
	}
	return false
}
