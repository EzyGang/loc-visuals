package scan

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Category string

const (
	Documentation Category = "documentation"
	Tests         Category = "tests"
	Code          Category = "code"
)

type Stats struct {
	Lines int
	Files int
}

type Result struct {
	Root         string
	Categories   map[Category]Stats
	TotalLines   int
	TotalFiles   int
	SkippedFiles int
}

var ignoredDirectories = map[string]struct{}{
	".git": {}, ".gradle": {}, ".hg": {}, ".mypy_cache": {}, ".next": {}, ".nuxt": {},
	".pytest_cache": {}, ".ruff_cache": {}, ".svn": {}, ".tox": {}, ".turbo": {}, ".venv": {},
	"__pycache__": {}, "build": {}, "coverage": {}, "dist": {}, "htmlcov": {}, "node_modules": {},
	"pods": {}, "target": {}, "vendor": {}, "venv": {},
}

var ignoredFiles = map[string]struct{}{
	"cargo.lock": {}, "composer.lock": {}, "package-lock.json": {}, "pnpm-lock.yaml": {},
	"poetry.lock": {}, "uv.lock": {}, "yarn.lock": {},
}

func Analyze(root string, excludedPath string) (Result, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return Result{}, fmt.Errorf("resolve project path: %w", err)
	}

	info, err := os.Stat(absoluteRoot)
	if err != nil {
		return Result{}, fmt.Errorf("inspect project path: %w", err)
	}
	if !info.IsDir() {
		return Result{}, fmt.Errorf("project path is not a directory: %s", absoluteRoot)
	}

	result := Result{Root: absoluteRoot, Categories: make(map[Category]Stats, 3)}
	excluded := cleanAbsolutePath(excludedPath)
	err = filepath.WalkDir(absoluteRoot, walkFile(absoluteRoot, excluded, &result))
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

func walkFile(root string, excluded string, result *Result) fs.WalkDirFunc {
	return func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk %s: %w", path, walkErr)
		}
		if path == excluded {
			return nil
		}
		if entry.IsDir() {
			if path != root && isIgnoredDirectory(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 || isIgnoredFile(entry.Name()) {
			result.SkippedFiles++
			return nil
		}

		lines, binary, err := countLines(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		if binary {
			result.SkippedFiles++
			return nil
		}

		relative, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("resolve relative path for %s: %w", path, err)
		}
		category := classify(relative)
		stats := result.Categories[category]
		stats.Lines += lines
		stats.Files++
		result.Categories[category] = stats
		result.TotalLines += lines
		result.TotalFiles++
		return nil
	}
}

func countLines(path string) (int, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	lines := 0
	for scanner.Scan() {
		line := scanner.Bytes()
		if bytes.IndexByte(line, 0) >= 0 {
			return 0, true, nil
		}
		if len(bytes.TrimSpace(line)) > 0 {
			lines++
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, false, err
	}
	return lines, false, nil
}

func classify(path string) Category {
	if isTestPath(path) {
		return Tests
	}

	extension := strings.ToLower(filepath.Ext(path))
	switch extension {
	case ".adoc", ".md", ".mdx", ".rst":
		return Documentation
	default:
		return Code
	}
}

func isTestPath(path string) bool {
	parts := strings.FieldsFunc(strings.ToLower(filepath.ToSlash(path)), func(character rune) bool {
		return character == '/' || character == '.' || character == '_' || character == '-'
	})
	for _, part := range parts {
		switch part {
		case "test", "tests", "spec", "specs", "__tests__":
			return true
		}
	}
	return false
}

func isIgnoredDirectory(name string) bool {
	_, ignored := ignoredDirectories[strings.ToLower(name)]
	return ignored
}

func isIgnoredFile(name string) bool {
	lowerName := strings.ToLower(name)
	if _, ignored := ignoredFiles[lowerName]; ignored {
		return true
	}
	return strings.HasSuffix(lowerName, ".map") || strings.Contains(lowerName, ".min.")
}

func cleanAbsolutePath(path string) string {
	if path == "" {
		return ""
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return ""
	}
	return filepath.Clean(absolute)
}
