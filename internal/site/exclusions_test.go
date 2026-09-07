package site

import "testing"

func TestRepositoryAndAgentFilesAreExcluded(t *testing.T) {
	cases := []string{
		"README.md",
		"docs/README.md",
		"AGENTS.md",
		"packages/api/AGENTS.md",
		"CLAUDE.md",
		"nested/CLAUDE.md",
		"CODEX.md",
		"GEMINI.md",
		"skills/example/SKILL.md",
		"LICENSE",
		"CHANGELOG.md",
		"CONTRIBUTING.md",
	}

	for _, path := range cases {
		if !isRepositoryFile(path) {
			t.Fatalf("expected %s to be excluded", path)
		}
	}

	for _, path := range []string{"index.md", "posts/agents.md-notes.md", "skills.md", "docs/security-guide.md"} {
		if isRepositoryFile(path) {
			t.Fatalf("expected %s to remain content", path)
		}
	}
}
