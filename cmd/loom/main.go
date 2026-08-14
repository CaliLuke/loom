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
		runOpenAPIImport(os.Args[2:])
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

func runOpenAPIImport(args []string) {
	arguments, err := parseOpenAPIImportArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	target, warnings, err := importOpenAPI(arguments.input, arguments.output, arguments.allowLossy)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	printImportWarnings(warnings)
	fmt.Println(target)
}

func printImportWarnings(warnings openapiimport.Diagnostics) {
	for _, warning := range warnings {
		fmt.Fprintf(os.Stderr, "warning: %s: %s (%s)\n", warning.Path, warning.Message, warning.Code)
	}
}

// help with tests
var (
	usage         = help
	gen           = generate
	importOpenAPI = importOpenAPIDesign
	newGenerator  = func(cmd, path, output string, debug bool) generatorRunner {
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
  loom import openapi INPUT [-o FILE-OR-DIRECTORY] [--allow-lossy]
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

  -o, -output DIRECTORY (gen, example, test-scaffold)
    output directory, defaults to the current working directory

  -debug
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
