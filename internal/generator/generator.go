package generator

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bpstr/rootmark/internal/config"
	"github.com/bpstr/rootmark/internal/site"
)

type Result = site.Result

func Build(cfg config.Config) (Result, error) {
	result, err := site.Build(cfg)
	if err != nil {
		return Result{}, err
	}

	output, err := filepath.Abs(result.Output)
	if err != nil {
		return Result{}, fmt.Errorf("resolve output: %w", err)
	}
	if err := enhancePreset(cfg, output); err != nil {
		return Result{}, fmt.Errorf("apply %s preset: %w", cfg.Preset, err)
	}

	if cfg.SitemapEnabled() {
		if err := writeRobots(cfg, output); err != nil {
			return Result{}, fmt.Errorf("generate robots.txt: %w", err)
		}
	}
	return result, nil
}

func writeRobots(cfg config.Config, output string) error {
	path := filepath.Join(output, "robots.txt")
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	contents := "User-agent: *\nAllow: /\nSitemap: " + canonicalURL(cfg.Site.URL, "/sitemap.xml") + "\n"
	return os.WriteFile(path, []byte(contents), 0o644)
}

func canonicalURL(siteURL, route string) string {
	for len(siteURL) > 0 && siteURL[len(siteURL)-1] == '/' {
		siteURL = siteURL[:len(siteURL)-1]
	}
	for len(route) > 0 && route[0] == '/' {
		route = route[1:]
	}
	return siteURL + "/" + route
}
