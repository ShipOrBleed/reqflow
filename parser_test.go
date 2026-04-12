package structmap

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDeadCodeAndEvents(t *testing.T) {
	// 1. Create a dummy package directory to parse
	dir, err := os.MkdirTemp("", "govistest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Create a dummy go.mod
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module dummy\n\ngo 1.21\n"), 0644)

	mainFile := filepath.Join(dir, "main.go")
	code := `package dummy
import "fmt"

type UserService struct {}

func (u *UserService) Process() {
	u.Publish("user_created")
}

func (u *UserService) Publish(topic string) {
	fmt.Println(topic)
}

// DeadService has no incoming calls
type DeadService struct {}
`
	os.WriteFile(mainFile, []byte(code), 0644)

	opts := ParseOptions{
		Dir: dir,
	}

	graph, err := Parse(opts)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if graph == nil {
		t.Fatal("Expected non-nil graph")
	}

	// Validate nodes exist
	if _, ok := graph.Nodes["dummy.UserService"]; !ok {
		t.Errorf("Expected dummy.UserService node to exist")
	}
	if _, ok := graph.Nodes["dummy.DeadService"]; !ok {
		t.Errorf("Expected dummy.DeadService node to exist")
	}

	// Validate Event Topic was captured
	if _, ok := graph.Nodes["eventbus.user_created"]; !ok {
		t.Errorf("Expected eventbus topic to be extracted natively")
	}

	// Check deadcode manually logic
	hasIncoming := make(map[string]bool)
	for _, e := range graph.Edges {
		hasIncoming[e.To] = true
	}

	if hasIncoming["dummy.DeadService"] {
		t.Errorf("DeadService should have ZERO incoming dependencies")
	}
}
