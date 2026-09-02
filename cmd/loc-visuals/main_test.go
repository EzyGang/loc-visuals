package main

import (
	"bytes"
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
