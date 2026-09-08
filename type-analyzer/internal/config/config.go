package config

import (
	"fmt"
	"os"
	"path/filepath"
)

import "gopkg.in/yaml.v3"

type Config struct {
	Version int `yaml:"version"`
	Target  struct {
		Root     string   `yaml:"root"`
		Mode     string   `yaml:"mode"`
		Patterns []string `yaml:"patterns"`
		Build    struct {
			Tags []string          `yaml:"tags"`
			Env  map[string]string `yaml:"env"`
		} `yaml:"build"`
		Vendor struct {
			LegacyImportPath string `yaml:"legacy_import_path"`
		} `yaml:"vendor"`
	} `yaml:"target"`
	Analysis struct {
		IncludeStdlib       bool `yaml:"include_stdlib"`
		IncludeDependencies bool `yaml:"include_dependencies"`
		Strict              bool `yaml:"strict"`
	} `yaml:"analysis"`
	Neo4j struct {
		URI, Database, User, PasswordEnv string
		BatchSize                        int    `yaml:"batch_size"`
		GraphID                          string `yaml:"graph_id"`
	} `yaml:"neo4j"`
	Output struct {
		CSVDir string `yaml:"csv_dir"`
	} `yaml:"output"`
}

func Load(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return c, fmt.Errorf("parse config: %w", err)
	}
	if c.Target.Root == "" {
		return c, fmt.Errorf("target.root is required")
	}
	abs, err := filepath.Abs(c.Target.Root)
	if err != nil {
		return c, err
	}
	c.Target.Root = abs
	if c.Target.Mode == "" {
		c.Target.Mode = "auto"
	}
	if c.Target.Mode != "auto" && c.Target.Mode != "module" && c.Target.Mode != "vendor" {
		return c, fmt.Errorf("target.mode must be auto, module, or vendor")
	}
	if len(c.Target.Patterns) == 0 {
		c.Target.Patterns = []string{"./..."}
	}
	if c.Neo4j.BatchSize <= 0 {
		c.Neo4j.BatchSize = 1000
	}
	if c.Neo4j.GraphID == "" {
		c.Neo4j.GraphID = "default"
	}
	return c, nil
}
