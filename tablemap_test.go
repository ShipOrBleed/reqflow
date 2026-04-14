package reqflow

import (
	"os"
	"testing"
)

func TestExtractTableMap_BasicModel(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"model.go": `package testmod

type OrderModel struct {
	ID    int
	Total float64
}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{TableMap: true})

	// OrderModel → orders table (snake_case + pluralize)
	if _, ok := graph.Nodes["table.order_models"]; !ok {
		t.Error("Expected table.order_models node")
	}
}

func TestExtractTableMap_PluralizesName(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"model.go": `package testmod

type UserModel struct {
	ID   int
	Name string
}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{TableMap: true})

	if _, ok := graph.Nodes["table.user_models"]; !ok {
		t.Error("Expected table.user_models node")
	}
}

func TestExtractTableMap_GormColumnTag(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"model.go": `package testmod

type InvoiceModel struct {
	ID        int    ` + "`" + `gorm:"column:invoice_id"` + "`" + `
	Reference string ` + "`" + `gorm:"column:ref_no"` + "`" + `
}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{TableMap: true})

	// Table should be created
	var tableNode *Node
	for _, n := range graph.Nodes {
		if n.Kind == KindTable {
			tableNode = n
			break
		}
	}
	if tableNode == nil {
		t.Fatal("Expected a KindTable node")
	}
	if tableNode.Meta["columns"] == "" {
		t.Error("Expected columns metadata on table node")
	}
}

func TestExtractTableMap_TableNameMethod(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"model.go": `package testmod

type PaymentModel struct {
	ID int
}

func (PaymentModel) TableName() string {
	return "payments"
}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{TableMap: true})

	if _, ok := graph.Nodes["table.payments"]; !ok {
		t.Error("Expected table.payments from TableName() override")
	}
}

func TestExtractTableMap_EdgeMapsTo(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"model.go": `package testmod

type ProductModel struct {
	ID int
}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{TableMap: true})

	hasEdge := false
	for _, edge := range graph.Edges {
		if edge.Kind == EdgeMapsTo && edge.From == "testmod.ProductModel" {
			hasEdge = true
			break
		}
	}
	if !hasEdge {
		t.Error("Expected EdgeMapsTo from ProductModel to its table")
	}
}

func TestExtractTableMap_OnlyModels(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"app.go": `package testmod

type UserHandler struct{}
type UserService struct{}
type UserModel struct{ ID int }
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{TableMap: true})

	for _, node := range graph.Nodes {
		if node.Kind == KindTable {
			// Table should be linked to the model, not handler/service
			var linkedFrom string
			for _, edge := range graph.Edges {
				if edge.To == node.ID && edge.Kind == EdgeMapsTo {
					linkedFrom = edge.From
				}
			}
			if linkedFrom == "" {
				t.Errorf("Table %s has no incoming EdgeMapsTo", node.ID)
			}
			modelNode := graph.Nodes[linkedFrom]
			if modelNode.Kind != KindModel {
				t.Errorf("Table %s is linked from non-model node %s (%s)", node.ID, linkedFrom, modelNode.Kind)
			}
		}
	}
}

func TestExtractTableMap_DbTag(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"model.go": `package testmod

type AccountModel struct {
	UserID int    ` + "`" + `db:"user_id"` + "`" + `
	Email  string ` + "`" + `db:"email_address"` + "`" + `
}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{TableMap: true})

	var tableNode *Node
	for _, n := range graph.Nodes {
		if n.Kind == KindTable {
			tableNode = n
			break
		}
	}
	if tableNode == nil {
		t.Fatal("Expected KindTable node")
	}
}

func TestExtractTableMap_NotRunWithoutFlag(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"model.go": `package testmod

type ItemModel struct{ ID int }
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{TableMap: false})

	for _, n := range graph.Nodes {
		if n.Kind == KindTable {
			t.Errorf("Found unexpected KindTable node %s when TableMap=false", n.ID)
		}
	}
}
