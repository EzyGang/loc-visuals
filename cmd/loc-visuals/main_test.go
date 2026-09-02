package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"loc-visuals/internal/version"
)

func TestVersionCommand(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := run([]string{"version"}, &stdout, &stderr); code != 0 {
		t.Fatalf("run(version) code = %d, want 0; stderr = %q", code, stderr.String())
	}
	want := "loc-visuals " + version.Current + "\n"
	if stdout.String() != want {
		t.Errorf("run(version) output = %q, want %q", stdout.String(), want)
	}
}

func TestVersionFlag(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := run([]string{"--version"}, &stdout, &stderr); code != 0 {
		t.Fatalf("run(--version) code = %d, want 0; stderr = %q", code, stderr.String())
	}
	want := "loc-visuals " + version.Current + "\n"
	if stdout.String() != want {
		t.Errorf("run(--version) output = %q, want %q", stdout.String(), want)
	}
}

func TestRunAcceptsMultipleProjectPaths(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	sourceRoot := filepath.Join(root, "src")
	testRoot := filepath.Join(root, "tests")
	for _, path := range []string{sourceRoot, testRoot} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", path, err)
		}
	}
	if err := os.WriteFile(filepath.Join(sourceRoot, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(source) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(testRoot, "main_test.go"), []byte("package tests\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(test) error = %v", err)
	}

	output := filepath.Join(root, "report.html")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := run([]string{"-o", output, sourceRoot, testRoot}, &stdout, &stderr); code != 0 {
		t.Fatalf("run(multiple paths) code = %d, want 0; stderr = %q", code, stderr.String())
	}
	content, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("ReadFile(report) error = %v", err)
	}
	report := string(content)
	for _, expected := range []string{"2 selected paths", sourceRoot, testRoot} {
		if !strings.Contains(report, expected) {
			t.Errorf("report does not contain %q", expected)
		}
	}
}
