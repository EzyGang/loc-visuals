package scan

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
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

type Summary struct {
	Categories   map[Category]Stats
	TotalLines   int
	TotalFiles   int
	SkippedFiles int
}

type RootResult struct {
	Path string
	Summary
}

type Result struct {
	Roots []RootResult
	Summary
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

func Analyze(roots []string, excludedPath string) (Result, error) {
	resolvedRoots, err := resolveRoots(roots)
	if err != nil {
		return Result{}, err
	}

	result := Result{Summary: newSummary()}
	excluded := cleanAbsolutePath(excludedPath)
	for _, root := range resolvedRoots {
		rootResult := RootResult{Path: root, Summary: newSummary()}
		if err := filepath.WalkDir(root, walkFile(root, excluded, &rootResult.Summary)); err != nil {
			return Result{}, err
		}
		mergeSummary(&result.Summary, rootResult.Summary)
		result.Roots = append(result.Roots, rootResult)
	}
	return result, nil
}

func newSummary() Summary {
	return Summary{Categories: make(map[Category]Stats, 3)}
}

func mergeSummary(destination *Summary, source Summary) {
	for category, stats := range source.Categories {
		combined := destination.Categories[category]
		combined.Lines += stats.Lines
		combined.Files += stats.Files
		destination.Categories[category] = combined
	}
	destination.TotalLines += source.TotalLines
	destination.TotalFiles += source.TotalFiles
	destination.SkippedFiles += source.SkippedFiles
}

func resolveRoots(roots []string) ([]string, error) {
	if len(roots) == 0 {
		roots = []string{"."}
	}

	resolved := make([]string, 0, len(roots))
	for _, root := range roots {
		absoluteRoot, err := filepath.Abs(root)
		if err != nil {
			return nil, fmt.Errorf("resolve project path %s: %w", root, err)
		}
		info, err := os.Stat(absoluteRoot)
		if err != nil {
			return nil, fmt.Errorf("inspect project path %s: %w", absoluteRoot, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("project path is not a directory: %s", absoluteRoot)
		}

		duplicate := false
		for _, existing := range resolved {
			if samePath(existing, absoluteRoot) {
				duplicate = true
				break
			}
		}
		if !duplicate {
			resolved = append(resolved, absoluteRoot)
		}
	}

	effective := make([]string, 0, len(resolved))
	for index, candidate := range resolved {
		covered := false
		for otherIndex, other := range resolved {
			if index != otherIndex && pathContains(other, candidate) {
				covered = true
				break
			}
		}
		if !covered {
			effective = append(effective, candidate)
		}
	}
	return effective, nil
}

func samePath(left string, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	return left == right || runtime.GOOS == "windows" && strings.EqualFold(left, right)
}

func pathContains(parent string, child string) bool {
	relative, err := filepath.Rel(parent, child)
	if err != nil || relative == "." {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func walkFile(root string, excluded string, result *Summary) fs.WalkDirFunc {
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
		category := classify(filepath.Join(filepath.Base(root), relative))
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
