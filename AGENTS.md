# Project rules

## Purpose

`loc-visuals` scans one project and creates a self-contained HTML artifact that shows the distribution of non-empty physical lines.
The three categories are documentation, tests, and code.

## Classification contract

Apply classification in this order:

1. Treat files under test or spec directories as tests.
2. Treat filenames with standalone `test`, `tests`, `spec`, or `specs` tokens as tests.
3. Treat Markdown, MDX, reStructuredText, and AsciiDoc files as documentation.
4. Treat all other scanned text files as code.

Do not count dependencies, caches, build output, lock files, minified files, source maps, symlinks, or binary files.
Exclude the requested report path from its own scan.

## Engineering

- Use the Go standard library unless a dependency removes demonstrated complexity.
- Keep scanning independent from HTML rendering.
- Keep the generated artifact self-contained and functional without a network connection.
- Preserve deterministic category order: documentation, tests, then code.
- Return operational errors with path context.
- Format changed Go files with `gofmt`.
- Add behavior tests for classification or counting changes.

## Verification

Run these commands from the project root:

```sh
go test ./...
go vet ./...
go build -o bin/loc-visuals ./cmd/loc-visuals
./bin/loc-visuals -o /tmp/loc-report.html .
```

Open the generated HTML in a browser after visual changes.
