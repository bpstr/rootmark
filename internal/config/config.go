package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Site       Site   `yaml:"site"`
	Build      Build  `yaml:"build"`
	Theme      string `yaml:"theme"`
	Navigation []Link `yaml:"navigation"`
	Primary    CTA    `yaml:"primary"`
	Footer     Footer `yaml:"footer"`
}

type Site struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Language    string `yaml:"language"`
	URL         string `yaml:"url"`
	Author      string `yaml:"author"`
	Favicon     string `yaml:"favicon"`
	Logo        string `yaml:"logo"`
	Repository  string `yaml:"repository"`
}

type Build struct {
	Source string `yaml:"source"`
	Output string `yaml:"output"`
}

type Link struct {
	Label string `yaml:"label"`
	URL   string `yaml:"url"`
}

type CTA struct {
	Label string `yaml:"label"`
	URL   string `yaml:"url"`
	Style string `yaml:"style"`
}

type Footer struct {
	Text          string `yaml:"text"`
	Links         []Link `yaml:"links"`
	GeneratedWith bool   `yaml:"generated_with"`
	DevelopedBy   Link   `yaml:"developed_by"`
}

func Default() Config {
	return Config{
		Site: Site{Language: "en"},
		Build: Build{
			Source: ".",
			Output: "_site",
		},
		Primary: CTA{Style: "button"},
		Footer:  Footer{GeneratedWith: true},
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
	if cfg.Primary.Style == "" {
		cfg.Primary.Style = "button"
	}
	cfg.Primary.Style = strings.ToLower(cfg.Primary.Style)
	if cfg.Primary.Style != "button" && cfg.Primary.Style != "link" {
		return Config{}, fmt.Errorf("primary.style must be button or link")
	}

	return cfg, nil
}
