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

func parseOpenAPIImportArgs(args []string) (string, string, error) {
	if len(args) < 2 || args[0] != "openapi" || strings.TrimSpace(args[1]) == "" {
		return "", "", fmt.Errorf("usage: loom import openapi INPUT [-o PATH]")
	}
	if args[1] == "-" {
		return "", "", fmt.Errorf("openapi import input must be a file; stdin is not supported")
	}

	flags := flag.NewFlagSet("import openapi", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	shortOutput := flags.String("o", "", "output file or directory")
	output := flags.String("output", defaultImportOutput, "output file or directory")
	if err := flags.Parse(args[2:]); err != nil {
		return "", "", fmt.Errorf("parse import flags: %w", err)
	}
	if flags.NArg() > 0 {
		return "", "", fmt.Errorf("unexpected import arguments: %s", strings.Join(flags.Args(), " "))
	}
	if *shortOutput != "" {
		*output = *shortOutput
	}
	if strings.TrimSpace(*output) == "" {
		return "", "", fmt.Errorf("import output path must not be empty")
	}
	return args[1], *output, nil
}

func importOpenAPIDesign(input, output string) (string, error) {
	source, err := os.ReadFile(input)
	if err != nil {
		return "", fmt.Errorf("read OpenAPI input %q: %w", input, err)
	}
	document, diagnostics, err := openapiimport.Analyze(source)
	if err != nil {
		return "", fmt.Errorf("analyze OpenAPI input %q: %w", input, err)
	}
	if len(diagnostics) > 0 {
		return "", fmt.Errorf("OpenAPI import cannot preserve the input contract:\n%s", diagnostics.Error())
	}

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
