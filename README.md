# loc-visuals

[![CI](https://github.com/EzyGang/loc-visuals/actions/workflows/ci.yml/badge.svg)](https://github.com/EzyGang/loc-visuals/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-5de5ff.svg)](LICENSE)

`loc-visuals` scans a project and creates a self-contained HTML report showing how its non-empty physical lines are distributed across documentation, tests, and code.

The generated report has no external assets or network dependencies. Open it locally, attach it to a build, or publish it as a CI artifact.

## Install

### Linux and macOS

```sh
curl -fsSL https://raw.githubusercontent.com/EzyGang/loc-visuals/main/scripts/install.sh | sh
```

The installer selects the current operating system and architecture, verifies the release checksum, and installs the binary to `~/.local/bin` by default.

Set `LOC_VISUALS_VERSION` to install a specific release or `LOC_VISUALS_INSTALL_DIR` to change the destination:

```sh
curl -fsSL https://raw.githubusercontent.com/EzyGang/loc-visuals/main/scripts/install.sh | LOC_VISUALS_VERSION=0.1.1 LOC_VISUALS_INSTALL_DIR="$HOME/bin" sh
```

### Windows

```powershell
irm https://raw.githubusercontent.com/EzyGang/loc-visuals/main/scripts/install.ps1 | iex
```

The PowerShell installer verifies the release checksum, copies the executable to `%USERPROFILE%\.local\bin`, and adds that directory to both the persistent user `PATH` and the current PowerShell session. It installs the executable itself rather than a shortcut.

```powershell
$env:LOC_VISUALS_VERSION="0.1.1"; $env:LOC_VISUALS_INSTALL_DIR="$HOME\bin"; irm https://raw.githubusercontent.com/EzyGang/loc-visuals/main/scripts/install.ps1 | iex
```

### Build from source

Go 1.25 or newer is required.

```sh
git clone https://github.com/EzyGang/loc-visuals.git && cd loc-visuals && go build -o bin/loc-visuals ./cmd/loc-visuals
```

## Usage

Scan the current project and write `loc-report.html`:

```sh
loc-visuals .
```

Choose another output path or project:

```sh
loc-visuals -o artifacts/lines.html ../project
```

Print the installed version:

```sh
loc-visuals version
```

Successful interactive runs check the GitHub Releases API at most once every 24 hours and report when a newer version is available. Set `LOC_VISUALS_NO_UPDATE_CHECK=1` to disable this check.

## Classification

Files are classified in this order:

1. Files under test or spec directories are tests.
2. Filenames containing a standalone `test`, `tests`, `spec`, or `specs` token are tests.
3. Markdown, MDX, reStructuredText, and AsciiDoc files are documentation.
4. All other scanned text files are code.

Dependencies, caches, build output, lock files, minified files, source maps, symlinks, binary files, and the requested report path are excluded.

## Development

```sh
go test ./... && go vet ./... && go build -o bin/loc-visuals ./cmd/loc-visuals && ./bin/loc-visuals -o /tmp/loc-report.html .
```

CI runs tests, vet, and builds on Linux, macOS, and Windows.

## Releases

Releases are manual:

1. Update `Current` in [`internal/version/version.go`](internal/version/version.go).
2. Run the **Release** workflow from the intended branch or commit in GitHub Actions.

The workflow reads the package version, builds Linux, macOS, and Windows archives for amd64 and arm64, creates the matching `vMAJOR.MINOR.PATCH` release, and attaches the archives, installers, and SHA-256 checksums.

## License

[MIT](LICENSE)
