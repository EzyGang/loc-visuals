package report

import (
	_ "embed"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"loc-visuals/internal/scan"
)

//go:embed template.html
var reportTemplate string

type categoryView struct {
	Name       string
	Key        string
	Lines      string
	Files      string
	Percentage string
	Dash       string
	Gap        string
	Offset     string
	X          string
	Width      string
}

type sourceView struct {
	Name         string
	Path         string
	TotalLines   string
	TotalFiles   string
	SkippedFiles string
	Categories   []categoryView
	AriaLabel    string
}

type templateData struct {
	Project       string
	SummaryTitle  string
	RootCount     string
	MultipleRoots bool
	Sources       []sourceView
	Generated     string
	TotalLines    string
	TotalFiles    string
	SkippedFiles  string
	Categories    []categoryView
	AriaLabel     string
}

func Write(result scan.Result, outputPath string) error {
	absoluteOutput, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(absoluteOutput), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	parsed, err := template.New("report").Parse(reportTemplate)
	if err != nil {
		return fmt.Errorf("parse report template: %w", err)
	}

	file, err := os.Create(absoluteOutput)
	if err != nil {
		return fmt.Errorf("create report: %w", err)
	}
	defer file.Close()

	if err := parsed.Execute(file, makeTemplateData(result)); err != nil {
		return fmt.Errorf("render report: %w", err)
	}
	return nil
}

func makeTemplateData(result scan.Result) templateData {
	categories := makeCategoryViews(result.Summary)
	multipleRoots := len(result.Roots) > 1
	summaryTitle := "Line distribution"
	if multipleRoots {
		summaryTitle = "Combined totals"
	}

	return templateData{
		Project:       projectName(result.Roots),
		SummaryTitle:  summaryTitle,
		RootCount:     formatNumber(len(result.Roots)),
		MultipleRoots: multipleRoots,
		Sources:       makeSourceViews(result.Roots),
		Generated:     time.Now().Format("2 January 2006, 15:04"),
		TotalLines:    formatNumber(result.TotalLines),
		TotalFiles:    formatNumber(result.TotalFiles),
		SkippedFiles:  formatNumber(result.SkippedFiles),
		Categories:    categories,
		AriaLabel:     categoryAriaLabel(categories),
	}
}

func projectName(roots []scan.RootResult) string {
	if len(roots) == 1 {
		return filepath.Base(roots[0].Path)
	}
	return fmt.Sprintf("%d selected paths", len(roots))
}

func makeSourceViews(roots []scan.RootResult) []sourceView {
	views := make([]sourceView, 0, len(roots))
	for _, root := range roots {
		categories := makeCategoryViews(root.Summary)
		views = append(views, sourceView{
			Name:         filepath.Base(root.Path),
			Path:         root.Path,
			TotalLines:   formatNumber(root.TotalLines),
			TotalFiles:   formatNumber(root.TotalFiles),
			SkippedFiles: formatNumber(root.SkippedFiles),
			Categories:   categories,
			AriaLabel:    categoryAriaLabel(categories),
		})
	}
	return views
}

func categoryAriaLabel(categories []categoryView) string {
	labels := make([]string, 0, len(categories))
	for _, category := range categories {
		labels = append(labels, fmt.Sprintf("%s %s percent", category.Name, category.Percentage))
	}
	return strings.Join(labels, ", ")
}

func makeCategoryViews(result scan.Summary) []categoryView {
	definitions := []struct {
		category scan.Category
		name     string
		key      string
	}{
		{scan.Documentation, "Documentation", "docs"},
		{scan.Tests, "Tests", "tests"},
		{scan.Code, "Code", "code"},
	}

	views := make([]categoryView, 0, len(definitions))
	offset := 0.0
	for _, definition := range definitions {
		stats := result.Categories[definition.category]
		percentage := percent(stats.Lines, result.TotalLines)
		views = append(views, categoryView{
			Name:       definition.name,
			Key:        definition.key,
			Lines:      formatNumber(stats.Lines),
			Files:      formatNumber(stats.Files),
			Percentage: formatDecimal(percentage),
			Dash:       formatDecimal(percentage),
			Gap:        formatDecimal(100 - percentage),
			Offset:     formatDecimal(-offset),
			X:          formatDecimal(offset),
			Width:      formatDecimal(percentage),
		})
		offset += percentage
	}
	return views
}

func percent(value int, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(value) * 100 / float64(total)
}

func formatDecimal(value float64) string {
	return strconv.FormatFloat(value, 'f', 1, 64)
}

func formatNumber(value int) string {
	digits := strconv.Itoa(value)
	for index := len(digits) - 3; index > 0; index -= 3 {
		digits = digits[:index] + "," + digits[index:]
	}
	return digits
}
