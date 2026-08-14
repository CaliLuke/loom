package main

import (
	"errors"
	"fmt"
	"go/build"
	"os"
	"strings"
	"time"

	"flag"

	"github.com/CaliLuke/loom/internal/openapiimport"
	loom "github.com/CaliLuke/loom/pkg"
)

type generatorRunner interface {
	Write(bool) error
	Compile(bool) error
	Run(bool) ([]string, error)
	Remove() error
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "import" {
		if exitCode := runOpenAPIImport(os.Args[2:]); exitCode != 0 {
			os.Exit(exitCode)
		}
		return
	}

	var (
		cmd    string
		path   string
		offset int
	)
	if len(os.Args) == 1 {
		usage()
		return
	}

	switch os.Args[1] {
	case "version":
		fmt.Println("Loom version " + loom.Version())
		os.Exit(0)
	case "gen", "example", "test-scaffold":
		if len(os.Args) == 2 {
			usage()
			return
		}
		cmd = os.Args[1]
		path = os.Args[2]
		offset = 2
	default:
		usage()
		return
	}

	var (
		output = "."
		debug  bool
	)
	if len(os.Args) > offset+1 {
		var (
			fset = flag.NewFlagSet("default", flag.ExitOnError)
			o    = fset.String("o", "", "output `directory`")
			out  = fset.String("output", output, "output `directory`")
		)
		fset.BoolVar(&debug, "debug", false, "Print debug information")

		fset.Usage = usage
		if err := fset.Parse(os.Args[offset+1:]); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}

		output = *o
		if output == "" {
			output = *out
		}
	}

	if err := gen(cmd, path, output, debug); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func runOpenAPIImport(args []string) int {
	arguments, err := parseOpenAPIImportArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	switch {
	case arguments.listTags:
		return runOpenAPITagList(arguments)
	case arguments.report:
		return runOpenAPIReport(arguments)
	case arguments.skipUnrenderable:
		return runOpenAPIPartialImport(arguments)
	case arguments.selection.Active():
		return runOpenAPISelectedImport(arguments)
	default:
		return runOpenAPIStrictImport(arguments)
	}
}

func runOpenAPITagList(arguments openAPIImportArgs) int {
	tags, err := listOpenAPITags(arguments.input)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	printOpenAPITags(tags)
	return 0
}

func runOpenAPIReport(arguments openAPIImportArgs) int {
	analysis, _, err := analyzePartialOpenAPI(
		arguments.input,
		arguments.allowLossy,
		arguments.selection,
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	if err := writePartialReport(os.Stdout, analysis); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	return partialImportExitCode(analysis)
}

func runOpenAPIPartialImport(arguments openAPIImportArgs) int {
	target, analysis, report, err := importPartialOpenAPI(
		arguments.input,
		arguments.output,
		arguments.allowLossy,
		arguments.selection,
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	if err := writePartialReport(os.Stderr, analysis); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	printUnclaimedPaths(report.UnclaimedPaths)
	if target != "" {
		fmt.Println(target)
	}
	return partialImportExitCode(analysis)
}

func runOpenAPISelectedImport(arguments openAPIImportArgs) int {
	target, warnings, report, err := importSelectedOpenAPI(
		arguments.input,
		arguments.output,
		arguments.allowLossy,
		arguments.selection,
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	printImportWarnings(warnings)
	printUnclaimedPaths(report.UnclaimedPaths)
	fmt.Println(target)
	return 0
}

func runOpenAPIStrictImport(arguments openAPIImportArgs) int {
	target, warnings, err := importOpenAPI(arguments.input, arguments.output, arguments.allowLossy)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	printImportWarnings(warnings)
	fmt.Println(target)
	return 0
}

func partialImportExitCode(analysis *openapiimport.PartialAnalysis) int {
	if analysis == nil || analysis.Document == nil || len(analysis.Document.Operations) == 0 {
		return 2
	}
	if len(analysis.Document.Operations) < analysis.TotalOperations {
		return 3
	}
	return 0
}

func printOpenAPITags(tags []openapiimport.TagSummary) {
	fmt.Println("TAG\tOPERATIONS\tPATHS")
	for _, tag := range tags {
		fmt.Printf("%s\t%d\t%d\n", tag.Name, tag.Operations, tag.Paths)
	}
}

func printUnclaimedPaths(paths []string) {
	for _, path := range paths {
		fmt.Fprintf(os.Stderr, "unclaimed path: %s\n", path)
	}
}

func printImportWarnings(warnings openapiimport.Diagnostics) {
	for _, warning := range warnings {
		fmt.Fprintf(os.Stderr, "warning: %s: %s (%s)\n", warning.Path, warning.Message, warning.Code)
	}
}

// help with tests
var (
	usage                 = help
	gen                   = generate
	importOpenAPI         = importOpenAPIDesign
	importSelectedOpenAPI = importOpenAPIDesignSelected
	importPartialOpenAPI  = importOpenAPIPartial
	analyzePartialOpenAPI = analyzeOpenAPIPartial
	listOpenAPITags       = inspectOpenAPITags
	newGenerator          = func(cmd, path, output string, debug bool) generatorRunner {
		return NewGenerator(cmd, path, output, debug)
	}
)

func generate(cmd, path, output string, debug bool) (returnErr error) {
	var (
		files                                                                    []string
		err                                                                      error
		tmp                                                                      generatorRunner
		transaction                                                              *generationTransaction
		startTotal, startImport, startNewGen, startWrite, startCompile, startRun time.Time
	)
	requestedOutput := output

	startTotal = time.Now()

	startImport = time.Now()
	if _, err = build.Import(path, ".", 0); err != nil {
		return cleanupGenerator(tmp, debug, wrapStageError("build.Import", err))
	}
	debugStage(debug, "build.Import", startImport, "path=%s", path)
	if cmd == "gen" {
		transaction, err = newGenerationTransaction(output)
		if err != nil {
			return wrapStageError("Prepare", err)
		}
		defer func() {
			if cleanupErr := transaction.cleanup(); cleanupErr != nil {
				returnErr = errors.Join(returnErr, cleanupErr)
			}
		}()
		output = transaction.stagePath()
	}

	startNewGen = time.Now()
	tmp = newGenerator(cmd, path, output, debug)
	debugStage(debug, "NewGenerator", startNewGen, "command=%s output=%s", cmd, output)

	startWrite = time.Now()
	if err = tmp.Write(debug); err != nil {
		return cleanupGenerator(tmp, debug, wrapStageError("Write", err))
	}
	debugStage(debug, "Write", startWrite, "command=%s", cmd)

	startCompile = time.Now()
	if err = tmp.Compile(debug); err != nil {
		return cleanupGenerator(tmp, debug, wrapStageError("Compile", err))
	}
	debugStage(debug, "Compile", startCompile, "command=%s", cmd)

	startRun = time.Now()
	if files, err = tmp.Run(debug); err != nil {
		return cleanupGenerator(tmp, debug, wrapStageError("Run", err))
	}
	debugStage(debug, "Run", startRun, "files=%d", len(files))
	if files, err = finishGenerationTransaction(transaction, files, debug); err != nil {
		return cleanupGenerator(tmp, debug, err)
	}
	debugStage(debug, "total", startTotal, "command=%s output=%s", cmd, requestedOutput)
	fmt.Println(strings.Join(files, "\n"))
	if !debug {
		if err := tmp.Remove(); err != nil {
			return err
		}
	}
	return nil
}

func help() {
	fmt.Fprint(os.Stderr, `loom is the code generation tool for the Loom framework.
Learn more at https://github.com/CaliLuke/loom.

Usage:
  loom import openapi INPUT [-o FILE-OR-DIRECTORY] [--allow-lossy] [FILTERS]
  loom gen PACKAGE [--output DIRECTORY] [--debug]
  loom example PACKAGE [--output DIRECTORY] [--debug]
  loom test-scaffold PACKAGE [--output DIRECTORY] [--debug]
  loom version

Commands:
  import openapi
        Create a Loom design from a supported OpenAPI 3.1 or 3.2 contract.
  gen
        Generate service interfaces, endpoints, transport code and OpenAPI spec.
  example
        Generate example server and client tool.
  test-scaffold
        Generate consumer-owned HTTP response contract tests.
  version
        Print version information.

Args:
  INPUT
        OpenAPI JSON or YAML file

  PACKAGE
        Go import path to design package

Flags:
  -o, --output FILE-OR-DIRECTORY (import)
        import output, defaults to design/design.go

  --allow-lossy (import)
        allow explicitly lossy metadata omissions and report them as warnings

  --tag TAG (import, repeatable)
        select operations with this tag

  --path-prefix PREFIX (import, repeatable)
        select operations below this OpenAPI path prefix

  --path PATTERN (import, repeatable)
        select operations that match this path pattern

  --list-tags (import)
        list operation and path counts by tag without writing a design

  --report (import)
        report grouped import blockers without writing a design

  --skip-unrenderable (import)
        write all renderable operations and report skipped operations

  -o, --output DIRECTORY (gen, example, test-scaffold)
        output directory, defaults to the current working directory

  --debug
        Print debug information (mainly intended for Loom developers)

Example:

  loom import openapi openapi.yaml -o design
  loom gen github.com/CaliLuke/loom-examples/cellar/design -o gendir

`)
}

func cleanupGenerator(tmp generatorRunner, debug bool, err error) error {
	if !debug && tmp != nil {
		if removeErr := tmp.Remove(); removeErr != nil {
			return errors.Join(err, removeErr)
		}
	}
	return err
}

func wrapStageError(stage string, err error) error {
	return fmt.Errorf("stage %s: %w", stage, err)
}

func debugStage(debug bool, stage string, start time.Time, format string, args ...any) {
	if !debug {
		return
	}
	msg := ""
	if format != "" {
		msg = " " + fmt.Sprintf(format, args...)
	}
	fmt.Fprintf(os.Stderr, "[loom-debug] stage=%s duration=%s%s\n", stage, time.Since(start), msg)
}
