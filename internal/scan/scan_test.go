package scan

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClassifyUsesTestPrecedenceAndFilenameTokens(t *testing.T) {
	t.Parallel()

	cases := map[string]Category{
		"README.md":                   Documentation,
		"docs/guide.mdx":              Documentation,
		"tests/fixtures/README.md":    Tests,
		"src/session_tests.py":        Tests,
		"src/tests_location.rs":       Tests,
		"web/account.spec.ts":         Tests,
		"src/contest.py":              Code,
		"src/latest.go":               Code,
		"configuration/settings.toml": Code,
	}
	for path, expected := range cases {
		if actual := classify(path); actual != expected {
			t.Errorf("classify(%q) = %q, want %q", path, actual, expected)
		}
	}
}

func TestAnalyzeCountsNonEmptyTextAndSkipsGeneratedContent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeFixture(t, root, "README.md", "# Project\n\nDetails\n")
	writeFixture(t, root, "src/main.go", "package main\n\nfunc main() {}\n")
	writeFixture(t, root, "src/main_tests.go", "package main\nfunc TestMain() {}\n")
	writeFixture(t, root, "tests/guide.md", "# Fixture\ntext\n")
	writeFixture(t, root, "target/generated.rs", "one\ntwo\nthree\n")
	writeFixture(t, root, "pnpm-lock.yaml", "one\ntwo\n")
	writeFixture(t, root, "image.bin", "text\x00binary\n")
	output := filepath.Join(root, "loc-report.html")
	writeFixture(t, root, "loc-report.html", "old\nreport\n")

	result, err := Analyze(root, output)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	assertStats(t, result.Categories[Documentation], Stats{Lines: 2, Files: 1})
	assertStats(t, result.Categories[Tests], Stats{Lines: 4, Files: 2})
	assertStats(t, result.Categories[Code], Stats{Lines: 2, Files: 1})
	if result.TotalLines != 8 || result.TotalFiles != 4 {
		t.Errorf("totals = %d lines in %d files, want 8 lines in 4 files", result.TotalLines, result.TotalFiles)
	}
	if result.SkippedFiles != 2 {
		t.Errorf("SkippedFiles = %d, want 2", result.SkippedFiles)
	}
}

func writeFixture(t *testing.T, root string, relativePath string, content string) {
	t.Helper()
	path := filepath.Join(root, relativePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func assertStats(t *testing.T, actual Stats, expected Stats) {
	t.Helper()
	if actual != expected {
		t.Errorf("stats = %+v, want %+v", actual, expected)
	}
}
