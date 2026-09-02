package scan

import (
	"os"
	"path/filepath"
	"strings"
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

	result, err := Analyze([]string{root}, output)
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

func TestAnalyzeSkipsNonTextFilesByContent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeFixture(t, root, "src/main.rs", "fn main() {}\n")
	writeFixture(t, root, "notes.rlib", "readable text\n")
	writeFixture(t, root, ".runtime/voice-test-target/debug/deps/libgodot_core.rlib", string([]byte{0xff, 0xfe, 0x01}))

	result, err := Analyze([]string{root}, "")
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	assertStats(t, result.Categories[Code], Stats{Lines: 2, Files: 2})
	if result.SkippedFiles != 1 {
		t.Errorf("SkippedFiles = %d, want 1", result.SkippedFiles)
	}
}

func TestAnalyzeAggregatesMultipleRoots(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	sourceRoot := filepath.Join(project, "src")
	testRoot := filepath.Join(project, "tests")
	writeFixture(t, project, "src/main.go", "package main\n")
	writeFixture(t, project, "tests/main.go", "package tests\n")

	result, err := Analyze([]string{sourceRoot, testRoot}, "")
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	assertStats(t, result.Categories[Code], Stats{Lines: 1, Files: 1})
	assertStats(t, result.Categories[Tests], Stats{Lines: 1, Files: 1})
	if result.TotalLines != 2 || result.TotalFiles != 2 {
		t.Errorf("combined totals = %d lines in %d files, want 2 lines in 2 files", result.TotalLines, result.TotalFiles)
	}
	if len(result.Roots) != 2 || result.Roots[0].Path != sourceRoot || result.Roots[1].Path != testRoot {
		t.Errorf("Roots = %+v, want paths [%q %q]", result.Roots, sourceRoot, testRoot)
	}
	assertStats(t, result.Roots[0].Categories[Code], Stats{Lines: 1, Files: 1})
	assertStats(t, result.Roots[1].Categories[Tests], Stats{Lines: 1, Files: 1})
	if result.Roots[0].TotalLines != 1 || result.Roots[1].TotalLines != 1 {
		t.Errorf("per-root lines = [%d %d], want [1 1]", result.Roots[0].TotalLines, result.Roots[1].TotalLines)
	}
}

func TestAnalyzeDoesNotRescanNestedOrDuplicateRoots(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	nested := filepath.Join(root, "src")
	writeFixture(t, root, "README.md", "# Project\n")
	writeFixture(t, root, "src/main.go", "package main\n")

	result, err := Analyze([]string{nested, root, root}, "")
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if result.TotalFiles != 2 || result.TotalLines != 2 {
		t.Errorf("totals = %d files and %d lines, want 2 files and 2 lines", result.TotalFiles, result.TotalLines)
	}
	if len(result.Roots) != 1 || result.Roots[0].Path != root {
		t.Errorf("Roots = %+v, want path %q", result.Roots, root)
	}
}

func TestCountLinesHandlesOversizedTextLine(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "generated.txt")
	writeFixture(t, root, "generated.txt", strings.Repeat("x", 17*1024*1024)+"\n\nnext\n")

	lines, binary, err := countLines(path)
	if err != nil {
		t.Fatalf("countLines() error = %v", err)
	}
	if binary {
		t.Fatal("countLines() classified text as binary")
	}
	if lines != 2 {
		t.Errorf("countLines() = %d, want 2", lines)
	}
}

func TestWalkFileSkipsUnreadableFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "vanished.go")
	writeFixture(t, root, "vanished.go", "package vanished\n")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	result := newSummary()
	if err := walkFile(root, "", &result)(path, entries[0], nil); err != nil {
		t.Fatalf("walkFile() error = %v, want unreadable file to be skipped", err)
	}
	if result.SkippedFiles != 1 {
		t.Errorf("SkippedFiles = %d, want 1", result.SkippedFiles)
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
