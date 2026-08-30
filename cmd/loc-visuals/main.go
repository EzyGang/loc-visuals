package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"loc-visuals/internal/report"
	"loc-visuals/internal/scan"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(arguments []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("loc-visuals", flag.ContinueOnError)
	flags.SetOutput(stderr)
	output := flags.String("o", "loc-report.html", "HTML artifact path")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: loc-visuals [-o report.html] [project]")
		flags.PrintDefaults()
	}
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	if flags.NArg() > 1 {
		flags.Usage()
		return 2
	}

	project := "."
	if flags.NArg() == 1 {
		project = flags.Arg(0)
	}
	result, err := scan.Analyze(project, *output)
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
