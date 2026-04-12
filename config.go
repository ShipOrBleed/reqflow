package structmap

import (
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

type GovisConfig struct {
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

	ServiceRegex *regexp.Regexp
	StoreRegex   *regexp.Regexp
	ModelRegex   *regexp.Regexp
}

func LoadConfig(path string) (*GovisConfig, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg GovisConfig
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
