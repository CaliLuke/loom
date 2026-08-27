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

const vetHelpText = `Analyze an evaluated design and consuming module for incomplete Loom adoption.

Usage:
  loom vet PACKAGE [--format text|json|sarif] [--debug]

Arguments:
  PACKAGE
        Go import path to design package

Flags:
  -h, --help
        Show help for the vet command

  --format text|json|sarif
        Report format, defaults to text

  --debug
        Print debug information

`

var analyzeVetDesign = vetDesign

func runVet(args []string, stdout, stderr io.Writer) int {
	if packageCommandHelpRequested(args) {
		return writeVetHelpResult(stderr)
	}
	options, err := parseVetOptions(args, stderr)
	if errors.Is(err, flag.ErrHelp) {
		return writeVetHelpResult(stderr)
	}
	if err != nil {
		return writeCommandFailure(stderr, err)
	}
	report, err := analyzeVetDesign(options.packagePath, options.debug)
	if err != nil {
		return writeCommandFailure(stderr, err)
	}
	if err := loomvet.WriteReport(stdout, report, options.format); err != nil {
		return writeCommandFailure(stderr, err)
	}
	if report.HasDiagnostics() {
		return 1
	}
	return 0
}

func writeVetHelpResult(writer io.Writer) int {
	if _, err := fmt.Fprint(writer, vetHelpText); err != nil {
		return writeCommandFailure(writer, fmt.Errorf("write vet help: %w", err))
	}
	return 0
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
