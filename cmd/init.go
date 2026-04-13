package cmd

import (
	"fmt"
	"os"
)

const govisYAMLTemplate = `# ===========================================
# Reqflow Configuration File
# ===========================================
# Place this file at the root of your Go project.
# Reqflow will auto-detect and load it on every run.
#
# Documentation: https://github.com/thzgajendra/reqflow

# Architecture linting rules.
# Format: "from_kind!to_kind" — fails CI if this dependency exists.
# Available kinds: handler, service, store, model, event, middleware, grpc
linter:
  vet_rules:
    - "handler!store"    # Handlers must not bypass services to call stores directly
    # - "handler!model"  # Uncomment to block handlers from accessing models

# Parser configuration
parser:
  # Packages to completely ignore during analysis
  ignore_packages:
    - "mocks"
    - "vendor"
    - "testdata"
    - "generated"

  # Reqflow automatically detects layers based on struct suffixes AND package names
  # (e.g., standard "Service", "Repository" logic OR "biz/", "adapter/" directories).
  # You can bypass the auto-detection completely by specifying strict regex rules below:
  # domain_naming:
  #   service_match: ".*(Service|UseCase|Manager|Interactor|Biz)$"
  #   store_match: ".*(Repository|Store|Adapter|Gateway|DAO|Data)$"
  #   model_match: ".*(Model|Entity|DTO|Record|Domain)$"

# CI/CD thresholds — govis exits 1 if any threshold is exceeded.
# Use with: govis -audit ./...
thresholds:
  max_cycles: 0          # Maximum allowed circular dependencies
  max_orphans: 5         # Maximum allowed dead/orphaned components
  # max_security_issues: 0 # Uncomment to fail on any security finding
`

// generateInitConfig creates a starter .reqflow.yml in the current directory.
func generateInitConfig() {
	path := ".reqflow.yml"

	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(os.Stderr, "⚠️  .reqflow.yml already exists. Remove it first to regenerate.\n")
		return
	}

	if err := os.WriteFile(path, []byte(govisYAMLTemplate), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create .reqflow.yml: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "✅ Created .reqflow.yml with default configuration.\n")
	fmt.Fprintf(os.Stderr, "   Edit it to customize architecture rules for your project.\n")
}
