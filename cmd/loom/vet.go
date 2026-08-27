package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/build"
	"io"
	"strings"

	loomvet "github.com/CaliLuke/loom/vet"
)

type vetOptions struct {
	packagePath string
	format      loomvet.Format
	debug       bool
}

var analyzeVetDesign = vetDesign

func runVet(args []string, stdout, stderr io.Writer) int {
	options, err := parseVetOptions(args, stderr)
	if err != nil {
		return writeVetFailure(stderr, err)
	}
	report, err := analyzeVetDesign(options.packagePath, options.debug)
	if err != nil {
		return writeVetFailure(stderr, err)
	}
	if err := loomvet.WriteReport(stdout, report, options.format); err != nil {
		return writeVetFailure(stderr, err)
	}
	if report.HasDiagnostics() {
		return 1
	}
	return 0
}

func writeVetFailure(stderr io.Writer, err error) int {
	if _, writeErr := fmt.Fprintln(stderr, err.Error()); writeErr != nil {
		return 1
	}
	return 1
}

func parseVetOptions(args []string, stderr io.Writer) (vetOptions, error) {
	if len(args) == 0 {
		return vetOptions{}, fmt.Errorf("usage: loom vet PACKAGE [--format text|json|sarif] [--debug]")
	}
	options := vetOptions{packagePath: args[0], format: loomvet.FormatText}
	flags := flag.NewFlagSet("vet", flag.ContinueOnError)
	flags.SetOutput(stderr)
	format := flags.String("format", string(loomvet.FormatText), "report format: text, json, or sarif")
	flags.BoolVar(&options.debug, "debug", false, "print debug information")
	if err := flags.Parse(args[1:]); err != nil {
		return vetOptions{}, err
	}
	if flags.NArg() != 0 {
		return vetOptions{}, fmt.Errorf("unexpected vet arguments: %s", strings.Join(flags.Args(), " "))
	}
	parsedFormat, err := loomvet.ParseFormat(*format)
	if err != nil {
		return vetOptions{}, err
	}
	options.format = parsedFormat
	return options, nil
}

func vetDesign(path string, debug bool) (report loomvet.Report, returnErr error) {
	if _, err := build.Import(path, ".", 0); err != nil {
		return report, wrapStageError("build.Import", err)
	}
	generator := NewGenerator("vet", path, ".", debug)
	defer func() {
		if debug && returnErr == nil {
			return
		}
		if err := generator.Remove(); err != nil {
			returnErr = errors.Join(returnErr, err)
		}
	}()
	if err := generator.Write(debug); err != nil {
		return report, wrapStageError("Write", err)
	}
	if err := generator.Compile(debug); err != nil {
		return report, wrapStageError("Compile", err)
	}
	lines, err := generator.Run(debug)
	if err != nil {
		return report, wrapStageError("Run", err)
	}
	if err := json.Unmarshal([]byte(strings.Join(lines, "\n")), &report); err != nil {
		return report, fmt.Errorf("decode vet report: %w", err)
	}
	return report, nil
}
