package reqflow

import (
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// ReqflowConfig represents the .reqflow.yml configuration schema.
type ReqflowConfig struct {
	Linter struct {
		VetRules []string `yaml:"vet_rules"`
	} `yaml:"linter"`
	Parser struct {
		IgnorePackages []string `yaml:"ignore_packages"`
		DomainNaming   struct {
			ServiceMatch string `yaml:"service_match"`
			StoreMatch   string `yaml:"store_match"`
			ModelMatch   string `yaml:"model_match"`
		} `yaml:"domain_naming"`
	} `yaml:"parser"`
	Thresholds struct {
		MaxCycles         *int `yaml:"max_cycles"`
		MaxOrphans        *int `yaml:"max_orphans"`
		MaxSecurityIssues *int `yaml:"max_security_issues"`
	} `yaml:"thresholds"`

	// Compiled regexes (not from YAML)
	ServiceRegex *regexp.Regexp `yaml:"-"`
	StoreRegex   *regexp.Regexp `yaml:"-"`
	ModelRegex   *regexp.Regexp `yaml:"-"`
}

// LoadConfig reads and parses a .reqflow.yml configuration file.
func LoadConfig(path string) (*ReqflowConfig, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg ReqflowConfig
	if err := yaml.Unmarshal(bytes, &cfg); err != nil {
		return nil, err
	}

	if cfg.Parser.DomainNaming.ServiceMatch != "" {
		cfg.ServiceRegex = regexp.MustCompile(cfg.Parser.DomainNaming.ServiceMatch)
	}
	if cfg.Parser.DomainNaming.StoreMatch != "" {
		cfg.StoreRegex = regexp.MustCompile(cfg.Parser.DomainNaming.StoreMatch)
	}
	if cfg.Parser.DomainNaming.ModelMatch != "" {
		cfg.ModelRegex = regexp.MustCompile(cfg.Parser.DomainNaming.ModelMatch)
	}

	return &cfg, nil
}
