package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/CaliLuke/loom/internal/openapiimport"
)

const defaultImportOutput = "design"

type (
	openAPIImportArgs struct {
		input            string
		output           string
		allowLossy       bool
		selection        openapiimport.Selection
		listTags         bool
		report           bool
		skipUnrenderable bool
	}

	diagnosticGroup struct {
		code     string
		count    int
		messages []diagnosticMessageGroup
	}

	diagnosticMessageGroup struct {
		message string
		count   int
	}
)

func parseOpenAPIImportArgs(args []string) (openAPIImportArgs, error) {
	if len(args) < 2 || args[0] != "openapi" || strings.TrimSpace(args[1]) == "" {
		return openAPIImportArgs{}, fmt.Errorf("usage: loom import openapi INPUT [-o FILE-OR-DIRECTORY] [--allow-lossy] [FILTERS]")
	}
	if args[1] == "-" {
		return openAPIImportArgs{}, fmt.Errorf("openapi import input must be a file; stdin is not supported")
	}

	flags := flag.NewFlagSet("import openapi", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	shortOutput := flags.String("o", "", "output file or directory")
	output := flags.String("output", defaultImportOutput, "output file or directory")
	allowLossy := flags.Bool("allow-lossy", false, "allow explicitly lossy metadata omissions")
	listTags := flags.Bool("list-tags", false, "list operation counts by tag")
	report := flags.Bool("report", false, "report import blockers without writing a design")
	skipUnrenderable := flags.Bool("skip-unrenderable", false, "write only operations that can be rendered")
	var tags, pathPrefixes, paths []string
	flags.Func(
		"tag",
		"select operations with this tag; repeat to form a union",
		appendImportFilter(&tags),
	)
	flags.Func(
		"path-prefix",
		"select operations below this path prefix; repeat to form a union",
		appendImportFilter(&pathPrefixes),
	)
	flags.Func(
		"path",
		"select operations matching this path pattern; repeat to form a union",
		appendImportFilter(&paths),
	)
	if err := flags.Parse(args[2:]); err != nil {
		return openAPIImportArgs{}, fmt.Errorf("parse import flags: %w", err)
	}
	if flags.NArg() > 0 {
		return openAPIImportArgs{}, fmt.Errorf("unexpected import arguments: %s", strings.Join(flags.Args(), " "))
	}
	if *shortOutput != "" {
		*output = *shortOutput
	}
	if strings.TrimSpace(*output) == "" {
		return openAPIImportArgs{}, fmt.Errorf("import output path must not be empty")
	}
	selection := openapiimport.Selection{Tags: tags, PathPrefixes: pathPrefixes, Paths: paths}
	if err := selection.Validate(); err != nil {
		return openAPIImportArgs{}, err
	}
	if *listTags && (selection.Active() || *report || *skipUnrenderable) {
		return openAPIImportArgs{}, fmt.Errorf("--list-tags cannot be combined with filters, --report, or --skip-unrenderable")
	}
	if *report && *skipUnrenderable {
		return openAPIImportArgs{}, fmt.Errorf("--report and --skip-unrenderable cannot be combined")
	}
	return openAPIImportArgs{
		input:            args[1],
		output:           *output,
		allowLossy:       *allowLossy,
		selection:        selection,
		listTags:         *listTags,
		report:           *report,
		skipUnrenderable: *skipUnrenderable,
	}, nil
}

func appendImportFilter(values *[]string) func(string) error {
	return func(value string) error {
		value = strings.TrimSpace(value)
		if value == "" {
			return fmt.Errorf("OpenAPI import filters must not be empty")
		}
		*values = append(*values, value)
		return nil
	}
}

func importOpenAPIDesign(input, output string, allowLossy bool) (string, openapiimport.Diagnostics, error) {
	target, warnings, _, err := importOpenAPIDesignSelected(input, output, allowLossy, openapiimport.Selection{})
	return target, warnings, err
}

func importOpenAPIDesignSelected(
	input, output string,
	allowLossy bool,
	selection openapiimport.Selection,
) (string, openapiimport.Diagnostics, openapiimport.SelectionReport, error) {
	source, err := os.ReadFile(input)
	if err != nil {
		return "", nil, openapiimport.SelectionReport{}, fmt.Errorf("read OpenAPI input %q: %w", input, err)
	}
	document, diagnostics, report, err := openapiimport.AnalyzeSelected(source, selection)
	if err != nil {
		return "", nil, report, fmt.Errorf("analyze OpenAPI input %q: %w", input, err)
	}
	fatal, warnings := diagnostics.Classify(allowLossy)
	if len(fatal) > 0 {
		return "", nil, report, fmt.Errorf("OpenAPI import cannot preserve the input contract:\n%s", fatal.Error())
	}
	if document == nil {
		return "", nil, report, fmt.Errorf("OpenAPI analysis did not produce a document")
	}
	if selection.Active() && len(document.Operations) == 0 {
		return "", nil, report, fmt.Errorf("OpenAPI selection matched no operations")
	}

	target, err := installOpenAPIDocument(document, output)
	if err != nil {
		return "", nil, report, err
	}
	return target, warnings, report, nil
}

func analyzeOpenAPIPartial(
	input string,
	allowLossy bool,
	selection openapiimport.Selection,
) (*openapiimport.PartialAnalysis, openapiimport.SelectionReport, error) {
	source, err := os.ReadFile(input)
	if err != nil {
		return nil, openapiimport.SelectionReport{}, fmt.Errorf("read OpenAPI input %q: %w", input, err)
	}
	analysis, report, err := openapiimport.AnalyzePartial(source, selection, allowLossy)
	if err != nil {
		return nil, report, fmt.Errorf("analyze OpenAPI input %q: %w", input, err)
	}
	return analysis, report, nil
}

func importOpenAPIPartial(
	input, output string,
	allowLossy bool,
	selection openapiimport.Selection,
) (string, *openapiimport.PartialAnalysis, openapiimport.SelectionReport, error) {
	analysis, report, err := analyzeOpenAPIPartial(input, allowLossy, selection)
	if err != nil {
		return "", nil, report, err
	}
	if analysis.Document == nil || len(analysis.Document.Operations) == 0 {
		return "", analysis, report, nil
	}
	target, err := installOpenAPIDocument(analysis.Document, output)
	if err != nil {
		return "", analysis, report, err
	}
	return target, analysis, report, nil
}

func installOpenAPIDocument(document *openapiimport.Document, output string) (string, error) {
	target, packageName, err := resolveImportTarget(output)
	if err != nil {
		return "", err
	}
	rendered, err := openapiimport.Render(document, openapiimport.Options{PackageName: packageName})
	if err != nil {
		return "", err
	}
	if err := installImportFile(target, rendered); err != nil {
		return "", err
	}
	return target, nil
}

func inspectOpenAPITags(input string) ([]openapiimport.TagSummary, error) {
	source, err := os.ReadFile(input)
	if err != nil {
		return nil, fmt.Errorf("read OpenAPI input %q: %w", input, err)
	}
	document, diagnostics, report, err := openapiimport.AnalyzeSelected(source, openapiimport.Selection{})
	if err != nil {
		return nil, fmt.Errorf("analyze OpenAPI input %q: %w", input, err)
	}
	if document == nil {
		return nil, fmt.Errorf("inspect OpenAPI tags:\n%s", diagnostics.Error())
	}
	return report.Tags, nil
}

func writePartialReport(writer io.Writer, analysis *openapiimport.PartialAnalysis) error {
	importedOperations, importedSchemas := 0, 0
	if analysis.Document != nil {
		importedOperations = len(analysis.Document.Operations)
		importedSchemas = len(analysis.Document.Components.Schemas)
	}
	if _, err := fmt.Fprintf(
		writer,
		"importable: %d/%d operations, %d/%d schemas\n",
		importedOperations,
		analysis.TotalOperations,
		importedSchemas,
		analysis.TotalSchemas,
	); err != nil {
		return err
	}
	if err := writeDiagnosticGroups(writer, "blocked", analysis.Blocked); err != nil {
		return err
	}
	if err := writeDiagnosticGroups(writer, "warnings", analysis.Warnings); err != nil {
		return err
	}
	if len(analysis.Skipped) == 0 {
		return nil
	}
	if _, err := fmt.Fprintln(writer, "skipped:"); err != nil {
		return err
	}
	for _, operation := range analysis.Skipped {
		if _, err := fmt.Fprintf(writer, "  %s %s\n", operation.Method, operation.Path); err != nil {
			return err
		}
		for _, diagnostic := range operation.Diagnostics {
			if _, err := fmt.Fprintf(writer, "    %s: %s\n", diagnostic.Code, diagnostic.Message); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeDiagnosticGroups(writer io.Writer, label string, diagnostics openapiimport.Diagnostics) error {
	groups := groupDiagnostics(diagnostics)
	if len(groups) == 0 {
		return nil
	}
	if _, err := fmt.Fprintf(writer, "%s:\n", label); err != nil {
		return err
	}
	for _, group := range groups {
		if _, err := fmt.Fprintf(writer, "  %s\t%d\n", group.code, group.count); err != nil {
			return err
		}
		for _, message := range group.messages {
			if _, err := fmt.Fprintf(writer, "    %d\t%s\n", message.count, message.message); err != nil {
				return err
			}
		}
	}
	return nil
}

func groupDiagnostics(diagnostics openapiimport.Diagnostics) []diagnosticGroup {
	type counts struct {
		total    int
		messages map[string]int
	}
	byCode := make(map[string]*counts)
	for _, diagnostic := range diagnostics {
		group := byCode[diagnostic.Code]
		if group == nil {
			group = &counts{messages: make(map[string]int)}
			byCode[diagnostic.Code] = group
		}
		group.total++
		group.messages[diagnostic.Message]++
	}
	codes := make([]string, 0, len(byCode))
	for code := range byCode {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	groups := make([]diagnosticGroup, 0, len(codes))
	for _, code := range codes {
		counts := byCode[code]
		messages := make([]string, 0, len(counts.messages))
		for message := range counts.messages {
			messages = append(messages, message)
		}
		sort.Strings(messages)
		group := diagnosticGroup{code: code, count: counts.total}
		for _, message := range messages {
			group.messages = append(group.messages, diagnosticMessageGroup{
				message: message,
				count:   counts.messages[message],
			})
		}
		groups = append(groups, group)
	}
	return groups
}

func resolveImportTarget(output string) (string, string, error) {
	target := filepath.Clean(output)
	info, err := os.Stat(target)
	switch {
	case err == nil && info.IsDir():
		target = filepath.Join(target, "design.go")
	case err == nil:
	case errors.Is(err, os.ErrNotExist):
		if hasTrailingPathSeparator(output) || filepath.Ext(target) == "" {
			target = filepath.Join(target, "design.go")
		} else if filepath.Ext(target) != ".go" {
			return "", "", fmt.Errorf("import output %q must be a .go file or directory", output)
		}
	case err != nil:
		return "", "", fmt.Errorf("inspect import output %q: %w", output, err)
	}

	absoluteTarget, err := filepath.Abs(target)
	if err != nil {
		return "", "", fmt.Errorf("resolve import output %q: %w", target, err)
	}
	packageName := filepath.Base(filepath.Dir(absoluteTarget))
	return target, packageName, nil
}

func hasTrailingPathSeparator(path string) bool {
	return strings.HasSuffix(path, string(filepath.Separator)) ||
		(filepath.Separator != '/' && strings.HasSuffix(path, "/"))
}

func installImportFile(target string, source []byte) (returnErr error) {
	if _, err := os.Lstat(target); err == nil {
		return fmt.Errorf("import output %q already exists; refusing to overwrite", target)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect import output %q: %w", target, err)
	}

	directory := filepath.Dir(target)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create import output directory %q: %w", directory, err)
	}
	temporary, err := os.CreateTemp(directory, ".loom-import-*.go")
	if err != nil {
		return fmt.Errorf("create temporary import output in %q: %w", directory, err)
	}
	temporaryPath := temporary.Name()
	defer func() {
		if removeErr := os.Remove(temporaryPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			returnErr = errors.Join(returnErr, fmt.Errorf("remove temporary import output %q: %w", temporaryPath, removeErr))
		}
	}()

	if err := writeImportTemporary(temporary, source); err != nil {
		return err
	}
	if err := os.Link(temporaryPath, target); err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("import output %q already exists; refusing to overwrite", target)
		}
		return fmt.Errorf("install import output %q: %w", target, err)
	}
	return nil
}

func writeImportTemporary(file *os.File, source []byte) error {
	if err := file.Chmod(0o644); err != nil {
		return closeImportTemporary(file, fmt.Errorf("set temporary import output permissions: %w", err))
	}
	if _, err := file.Write(source); err != nil {
		return closeImportTemporary(file, fmt.Errorf("write temporary import output: %w", err))
	}
	if err := file.Sync(); err != nil {
		return closeImportTemporary(file, fmt.Errorf("sync temporary import output: %w", err))
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close temporary import output: %w", err)
	}
	return nil
}

func closeImportTemporary(file *os.File, writeErr error) error {
	if closeErr := file.Close(); closeErr != nil {
		return errors.Join(writeErr, fmt.Errorf("close temporary import output: %w", closeErr))
	}
	return writeErr
}
