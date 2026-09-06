package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Site  Site  `yaml:"site"`
	Build Build `yaml:"build"`
}

type Site struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Language    string `yaml:"language"`
}

type Build struct {
	Source string `yaml:"source"`
	Output string `yaml:"output"`
}

func Default() Config {
	return Config{
		Site: Site{Language: "en"},
		Build: Build{
			Source: ".",
			Output: "_site",
		},
	}
}

func Load(path string) (Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	if cfg.Site.Language == "" {
		cfg.Site.Language = "en"
	}
	if cfg.Build.Source == "" {
		cfg.Build.Source = "."
	}
	if cfg.Build.Output == "" {
		cfg.Build.Output = "_site"
	}

	return cfg, nil
}
