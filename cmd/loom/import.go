package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/CaliLuke/loom/internal/openapiimport"
)

const defaultImportOutput = "design"

type openAPIImportArgs struct {
	input      string
	output     string
	allowLossy bool
	selection  openapiimport.Selection
	listTags   bool
}

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
	if *listTags && selection.Active() {
		return openAPIImportArgs{}, fmt.Errorf("--list-tags cannot be combined with --tag, --path-prefix, or --path")
	}
	return openAPIImportArgs{
		input:      args[1],
		output:     *output,
		allowLossy: *allowLossy,
		selection:  selection,
		listTags:   *listTags,
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

	target, packageName, err := resolveImportTarget(output)
	if err != nil {
		return "", nil, report, err
	}
	rendered, err := openapiimport.Render(document, openapiimport.Options{PackageName: packageName})
	if err != nil {
		return "", nil, report, err
	}
	if err := installImportFile(target, rendered); err != nil {
		return "", nil, report, err
	}
	return target, warnings, report, nil
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
