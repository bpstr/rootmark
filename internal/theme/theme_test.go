package theme

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDirectoryRendersTemplatesAndCopiesAssets(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "templates"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "theme.yml"), []byte("name: Test theme\nformat: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "templates", "default.html"), []byte("<main>{{.}}</main>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "templates", "landing.html"), []byte("<article>{{.}}</article>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "assets", "theme.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	activeTheme, err := loadDirectory(root)
	if err != nil {
		t.Fatal(err)
	}
	if activeTheme.Name() != "Test theme" {
		t.Fatalf("unexpected theme name %q", activeTheme.Name())
	}

	var rendered bytes.Buffer
	if err := activeTheme.Execute(&rendered, "landing", "Hello"); err != nil {
		t.Fatal(err)
	}
	if rendered.String() != "<article>Hello</article>" {
		t.Fatalf("unexpected render %q", rendered.String())
	}

	output := filepath.Join(root, "site")
	if err := activeTheme.CopyAssets(output); err != nil {
		t.Fatal(err)
	}
	asset, err := os.ReadFile(filepath.Join(output, "_rootmark", "theme.css"))
	if err != nil {
		t.Fatal(err)
	}
	if string(asset) != "body{}" {
		t.Fatalf("unexpected copied asset %q", asset)
	}
}

func TestLoadDirectoryRequiresSupportedManifestAndDefaultTemplate(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "templates"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "theme.yml"), []byte("name: Broken\nformat: 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := loadDirectory(root)
	if err == nil || !strings.Contains(err.Error(), "unsupported theme format") {
		t.Fatalf("expected unsupported format error, got %v", err)
	}
}

func TestLoadRejectsNonHTTPSThemeURL(t *testing.T) {
	_, err := Load("git@github.com:example/theme.git")
	if err == nil || !strings.Contains(err.Error(), "public https repository URL") {
		t.Fatalf("expected public URL error, got %v", err)
	}
}
