package scan

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
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
			if path == root {
				return fmt.Errorf("walk %s: %w", path, walkErr)
			}
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			result.SkippedFiles++
			return nil
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
			result.SkippedFiles++
			return nil
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

	buffer := make([]byte, 64*1024+utf8.UTFMax)
	buffered := 0
	lines := 0
	lineHasContent := false
	for {
		read, readErr := file.Read(buffer[buffered:])
		content := buffer[:buffered+read]
		buffered = 0

		for len(content) > 0 {
			if !utf8.FullRune(content) {
				if readErr != nil {
					return 0, true, nil
				}
				buffered = len(content)
				copy(buffer[:buffered], content)
				break
			}

			character, size := utf8.DecodeRune(content)
			if character == utf8.RuneError && size == 1 || isBinaryControl(character) {
				return 0, true, nil
			}
			if character == '\n' {
				if lineHasContent {
					lines++
				}
				lineHasContent = false
			} else if !unicode.IsSpace(character) {
				lineHasContent = true
			}
			content = content[size:]
		}

		if readErr != nil {
			if !errors.Is(readErr, io.EOF) {
				return 0, false, readErr
			}
			if lineHasContent {
				lines++
			}
			return lines, false, nil
		}
	}
}

func isBinaryControl(character rune) bool {
	return unicode.IsControl(character) && !unicode.IsSpace(character)
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
