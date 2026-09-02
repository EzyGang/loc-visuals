package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"loc-visuals/internal/report"
	"loc-visuals/internal/scan"
	"loc-visuals/internal/update"
	"loc-visuals/internal/version"
)

func main() {
	code := run(os.Args[1:], os.Stdout, os.Stderr)
	if code == 0 && shouldCheckForUpdate(os.Args[1:], os.Stdout) {
		reportAvailableUpdate(os.Stderr)
	}
	os.Exit(code)
}

func run(arguments []string, stdout io.Writer, stderr io.Writer) int {
	if len(arguments) == 1 && (arguments[0] == "version" || arguments[0] == "--version") {
		fmt.Fprintf(stdout, "loc-visuals %s\n", version.Current)
		return 0
	}

	flags := flag.NewFlagSet("loc-visuals", flag.ContinueOnError)
	flags.SetOutput(stderr)
	output := flags.String("o", "loc-report.html", "HTML artifact path")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: loc-visuals [-o report.html] [path ...]")
		fmt.Fprintln(stderr, "       loc-visuals version")
		flags.PrintDefaults()
	}
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	result, err := scan.Analyze(flags.Args(), *output)
	if err != nil {
		fmt.Fprintf(stderr, "loc-visuals: %v\n", err)
		return 1
	}
	if err := report.Write(result, *output); err != nil {
		fmt.Fprintf(stderr, "loc-visuals: %v\n", err)
		return 1
	}

	absoluteOutput, err := filepath.Abs(*output)
	if err != nil {
		absoluteOutput = *output
	}
	fmt.Fprintf(stdout, "Created %s\n", absoluteOutput)
	fmt.Fprintf(stdout, "%d non-empty lines across %d text files\n", result.TotalLines, result.TotalFiles)
	return 0
}

func shouldCheckForUpdate(arguments []string, stdout *os.File) bool {
	if os.Getenv("LOC_VISUALS_NO_UPDATE_CHECK") != "" {
		return false
	}
	if len(arguments) == 1 && (arguments[0] == "version" || arguments[0] == "--version") {
		return false
	}
	info, err := stdout.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func reportAvailableUpdate(stderr io.Writer) {
	available, err := update.Check(version.Current)
	if err != nil {
		fmt.Fprintf(stderr, "loc-visuals: update check failed: %v\n", err)
		return
	}
	if available != nil {
		fmt.Fprintf(stderr, "loc-visuals %s is available; update at %s\n", available.Version, available.URL)
	}
}
