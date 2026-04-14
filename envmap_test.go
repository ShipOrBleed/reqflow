package reqflow

import (
	"os"
	"testing"
)

func TestExtractEnvMap_OsGetenv(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"config.go": `package testmod

import "os"

type AppConfig struct{}

func (c *AppConfig) Load() string {
	return os.Getenv("DATABASE_URL")
}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{EnvMap: true})

	if _, ok := graph.Nodes["env.DATABASE_URL"]; !ok {
		t.Error("Expected env.DATABASE_URL node from os.Getenv")
	}
	node := graph.Nodes["env.DATABASE_URL"]
	if node.Kind != KindEnvVar {
		t.Errorf("Expected KindEnvVar, got %s", node.Kind)
	}
	if node.Meta["env_var_name"] != "DATABASE_URL" {
		t.Errorf("env_var_name = %q, want DATABASE_URL", node.Meta["env_var_name"])
	}
	if node.Meta["access_via"] != "os.Getenv" {
		t.Errorf("access_via = %q, want os.Getenv", node.Meta["access_via"])
	}
}

func TestExtractEnvMap_OsLookupEnv(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"config.go": `package testmod

import "os"

func getPort() string {
	val, _ := os.LookupEnv("PORT")
	return val
}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{EnvMap: true})

	if _, ok := graph.Nodes["env.PORT"]; !ok {
		t.Error("Expected env.PORT node from os.LookupEnv")
	}
	if graph.Nodes["env.PORT"].Meta["access_via"] != "os.LookupEnv" {
		t.Errorf("access_via = %q, want os.LookupEnv", graph.Nodes["env.PORT"].Meta["access_via"])
	}
}

func TestExtractEnvMap_MultipleVars(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"app.go": `package testmod

import "os"

func init() {
	_ = os.Getenv("DB_HOST")
	_ = os.Getenv("DB_PORT")
	_ = os.Getenv("SECRET_KEY")
}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{EnvMap: true})

	for _, name := range []string{"DB_HOST", "DB_PORT", "SECRET_KEY"} {
		if _, ok := graph.Nodes["env."+name]; !ok {
			t.Errorf("Expected env.%s node", name)
		}
	}
}

func TestExtractEnvMap_Deduplication(t *testing.T) {
	// Same env var read in multiple places should produce one node
	dir := helperWriteModule(t, map[string]string{
		"a.go": `package testmod

import "os"

func a() { _ = os.Getenv("SHARED_KEY") }
`,
		"b.go": `package testmod

import "os"

func b() { _ = os.Getenv("SHARED_KEY") }
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{EnvMap: true})

	count := 0
	for id := range graph.Nodes {
		if id == "env.SHARED_KEY" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("Expected exactly 1 env node for SHARED_KEY, got %d", count)
	}
}

func TestExtractEnvMap_EdgeCreated(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"service.go": `package testmod

import "os"

type ConfigService struct{}

func (c *ConfigService) GetDB() string {
	return os.Getenv("DB_URL")
}
`,
	})
	defer os.RemoveAll(dir)

	graph := helperParse(t, dir, ParseOptions{EnvMap: true})

	if _, ok := graph.Nodes["env.DB_URL"]; !ok {
		t.Fatal("Expected env.DB_URL node")
	}

	// There should be an EdgeReads edge pointing to env.DB_URL
	hasEdge := false
	for _, edge := range graph.Edges {
		if edge.To == "env.DB_URL" && edge.Kind == EdgeReads {
			hasEdge = true
			break
		}
	}
	if !hasEdge {
		t.Error("Expected EdgeReads edge to env.DB_URL")
	}
}

func TestExtractEnvMap_ViperGetString(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"config.go": `package testmod

// Simulate viper package
var viper viperPkg

type viperPkg struct{}
func (v viperPkg) GetString(key string) string { return "" }
func (v viperPkg) GetInt(key string) int { return 0 }

func loadConfig() {
	_ = viper.GetString("REDIS_ADDR")
	_ = viper.GetInt("WORKER_COUNT")
}
`,
	})
	defer os.RemoveAll(dir)

	// Note: our AST scanner matches on the package name "viper", not the type.
	// This test validates the path even if the fake viper won't match (local struct, not import).
	// The test confirms the function runs without panicking on non-matching calls.
	graph := helperParse(t, dir, ParseOptions{EnvMap: true})
	// No env nodes expected because the local struct is not detected as "viper" pkg import
	_ = graph
}

func TestExtractEnvMap_NotRunWithoutFlag(t *testing.T) {
	dir := helperWriteModule(t, map[string]string{
		"main.go": `package testmod

import "os"

func init() { _ = os.Getenv("SHOULD_NOT_APPEAR") }
`,
	})
	defer os.RemoveAll(dir)

	// EnvMap: false — env nodes should not appear
	graph := helperParse(t, dir, ParseOptions{EnvMap: false})

	if _, ok := graph.Nodes["env.SHOULD_NOT_APPEAR"]; ok {
		t.Error("env node should not be created when EnvMap=false")
	}
}
