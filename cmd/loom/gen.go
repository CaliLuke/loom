package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/CaliLuke/loom/codegen"
	"golang.org/x/tools/go/packages"
)

var getwd = os.Getwd

// Generator is the code generation management data structure.
type Generator struct {
	// Command is the name of the command to run.
	Command string

	// DesignPath is the Go import path to the design package.
	DesignPath string

	// Output is the absolute path to the output directory.
	Output string

	// DesignVersion is the major component of the Loom version used by the design DSL.
	DesignVersion int

	// bin is the filename of the generated generator.
	bin string

	// tmpDir is the temporary directory used to compile the generator.
	tmpDir string

	// hasVendorDirectory is a flag to indicate whether the project uses vendoring
	hasVendorDirectory bool
}

// NewGenerator creates a Generator.
func NewGenerator(cmd, path, output string, debug bool) *Generator {
	bin := "loom"
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}

	var version int
	var hasVendorDirectory bool
	{
		version = codegen.DesignVersion
		matched := false
		startPkgLoad := time.Now()
		pkgs, _ := packages.Load(&packages.Config{Mode: packages.NeedFiles | packages.NeedModule}, path)
		debugStage(debug, "design-package-load", startPkgLoad, "path=%s packages=%d", path, len(pkgs))
		fset := token.NewFileSet()
		p := regexp.MustCompile(`github.com/CaliLuke/loom(?:/v(\d+))?/dsl`)
		for _, pkg := range pkgs {
			// Nil check in case packages.Load can't get module info
			if pkg.Module != nil {
				if _, err := os.Stat(filepath.Join(pkg.Module.Dir, "vendor")); !os.IsNotExist(err) {
					hasVendorDirectory = true
				}
			}
			for _, gof := range pkg.GoFiles {
				if bs, err := os.ReadFile(gof); err == nil {
					if f, err := parser.ParseFile(fset, "", string(bs), parser.ImportsOnly); err == nil {
						for _, s := range f.Imports {
							matches := p.FindStringSubmatch(s.Path.Value)
							if len(matches) == 2 {
								matched = true
								if matches[1] != "" {
									if parsedVersion, err := strconv.Atoi(matches[1]); err == nil {
										version = parsedVersion
									}
								}
							}
						}
					}
				}
				if matched {
					break
				}
			}
			if matched {
				break
			}
		}
	}

	return &Generator{
		Command:            cmd,
		DesignPath:         path,
		Output:             output,
		DesignVersion:      version,
		hasVendorDirectory: hasVendorDirectory,
		bin:                bin,
	}
}

// Write writes the main file.
func (g *Generator) Write(_ bool) error {
	var tmpDir string
	{
		wd := "."
		if cwd, err := getwd(); err == nil {
			wd = cwd
		}
		tmp, err := os.MkdirTemp(wd, "loom")
		if err != nil {
			return err
		}
		tmpDir = tmp
	}
	g.tmpDir = tmpDir

	var sections []codegen.Section
	{
		data := map[string]any{
			"Command":       g.Command,
			"CleanupDirs":   cleanupDirs(g.Command, g.Output),
			"DesignVersion": g.DesignVersion,
		}
		imports := []*codegen.ImportSpec{
			codegen.SimpleImport("flag"),
			codegen.SimpleImport("fmt"),
			codegen.SimpleImport("os"),
			codegen.SimpleImport("path/filepath"),
			codegen.SimpleImport("sort"),
			codegen.SimpleImport("strconv"),
			codegen.SimpleImport("strings"),
			codegen.SimpleImport("time"),
			codegen.SimpleImport("github.com/CaliLuke/loom/codegen"),
			codegen.SimpleImport("github.com/CaliLuke/loom/codegen/generator"),
			codegen.SimpleImport("github.com/CaliLuke/loom/eval"),
			codegen.SimpleImport("github.com/CaliLuke/loom/expr"),
			codegen.NewImport("loom", "github.com/CaliLuke/loom/pkg"),
			codegen.NewImport("_", g.DesignPath),
		}
		sections = []codegen.Section{
			codegen.Header("Code Generator", "main", imports),
			codegen.NewTextTemplateSection("main", mainT, nil, data),
		}
	}

	f := &codegen.File{Path: "main.go", Sections: sections}
	_, err := f.Render(tmpDir)
	return err
}

// Compile compiles the generator.
func (g *Generator) Compile(debug bool) error {
	// We first need to go get the generated package to make sure that all
	// dependencies are added to go.sum prior to compiling.
	startLoad := time.Now()
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedName,
		Env: append(
			os.Environ(),
			"GO111MODULE=on",
			"GOWORK=off",
			"GOFLAGS=-mod=mod",
		),
	}, fmt.Sprintf(".%c%s", filepath.Separator, g.tmpDir))
	if err != nil {
		return err
	}
	if len(pkgs) != 1 {
		return fmt.Errorf("expected to find one package in %s", g.tmpDir)
	}
	debugStage(debug, "temp-package-load", startLoad, "tmpDir=%s packages=%d", g.tmpDir, len(pkgs))

	startBuild := time.Now()
	err = g.runGoCmd("build", "-o", g.bin)
	debugStage(debug, "go-build", startBuild, "binary=%s", g.bin)

	// If we're in vendor context we check the error string to see if it's an issue of unsatisfied dependencies
	if err != nil && g.hasVendorDirectory {
		if strings.Contains(err.Error(), "cannot find package") && strings.Contains(err.Error(), "/github.com/CaliLuke/loom/codegen/generator") {
			return errors.New("generated code expected `github.com/CaliLuke/loom/codegen/generator` to be present in the vendor directory, see documentation for more details")
		}
	}

	return err
}

// Run runs the compiled binary and return the output lines.
func (g *Generator) Run(debug bool) ([]string, error) {
	var cmdl string
	{
		args := make([]string, len(os.Args)-1)
		gopaths := filepath.SplitList(os.Getenv("GOPATH"))
		if len(gopaths) == 0 {
			gopaths = []string{build.Default.GOPATH}
		}
		for i, a := range os.Args[1:] {
			for _, p := range gopaths {
				if strings.HasPrefix(a, p) {
					args[i] = strings.Replace(a, p, "$(GOPATH)", 1)
					break
				}
			}
			if args[i] == "" {
				args[i] = a
			}
		}
		cmdl = " " + strings.Join(args, " ")
		rawcmd := filepath.Base(os.Args[0])
		// Remove .exe suffix to avoid different output on Windows.
		rawcmd = strings.TrimSuffix(rawcmd, ".exe")

		cmdl = fmt.Sprintf("$ %s%s", rawcmd, cmdl)
	}

	args := []string{"--version=" + strconv.Itoa(g.DesignVersion), "--output=" + g.Output, "--cmd=" + cmdl, "--debug=" + strconv.FormatBool(debug)}
	cmd := exec.Command(filepath.Join(g.tmpDir, g.bin), args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)
	err := cmd.Run()
	if err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("%w\n%s", err, strings.TrimRight(stderr.String(), "\n"))
		}
		return nil, err
	}
	res := strings.Split(stdout.String(), "\n")
	for (len(res) > 0) && (res[len(res)-1] == "") {
		res = res[:len(res)-1]
	}
	return res, nil
}

// Remove deletes the package files.
func (g *Generator) Remove() error {
	if g.tmpDir != "" {
		if err := os.RemoveAll(g.tmpDir); err != nil {
			return fmt.Errorf("remove temporary generator directory %s: %w", g.tmpDir, err)
		}
		g.tmpDir = ""
	}
	return nil
}

func (g *Generator) runGoCmd(args ...string) error {
	gobin, err := exec.LookPath("go")
	if err != nil {
		return fmt.Errorf(`failed to find a go compiler, looked in "%s"`, os.Getenv("PATH"))
	}
	c := exec.Cmd{
		Path: gobin,
		Args: append([]string{gobin}, args...),
		Dir:  g.tmpDir,
		Env: append(
			os.Environ(),
			"GO111MODULE=on",
			"GOWORK=off",
			"GOFLAGS=-mod=mod",
		),
	}
	out, err := c.CombinedOutput()
	if err != nil {
		if len(out) > 0 {
			return fmt.Errorf("%s", out)
		}
		return fmt.Errorf("failed to compile generator: %w", err)
	}
	return nil
}

// cleanupDirs returns the paths of the subdirectories under gendir to delete
// before generating code.
func cleanupDirs(cmd, output string) []string {
	if cmd == "gen" {
		gendirPath := filepath.Join(output, codegen.Gendir)
		gendir, err := os.Open(gendirPath)
		if err != nil {
			return nil
		}
		defer func() {
			err := gendir.Close()
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to close gendir: %s", err)
			}
		}()
		finfos, err := gendir.Readdir(-1)
		if err != nil {
			return []string{gendirPath}
		}
		var dirs []string

		for _, fi := range finfos {
			if fi.IsDir() {
				dirs = append(dirs, filepath.Join(gendirPath, fi.Name()))
			}
		}
		return dirs
	}
	return nil
}

// mainT is the template for the generator main.
const mainT = `func main() {
	var (
		out     = flag.String("output", "", "")
		version = flag.String("version", "", "")
		cmdl    = flag.String("cmd", "", "")
		debug   = flag.Bool("debug", false, "")
		ver int
	)
	{
		flag.Parse()
		if *out == "" {
			fail("missing output flag")
		}
		if *version == "" {
			fail("missing version flag")
		}
		if *cmdl == "" {
			fail("missing cmd flag")
		}
		v, err := strconv.Atoi(*version)
		if err != nil {
			fail("invalid version %s", *version)
		}
		ver = v
	}

	startBinary := time.Now()
	if *debug {
		debugStage(*debug, "binary-startup", startBinary, "cmd=%s output=%s", *cmdl, *out)
	}

	if ver > loom.Major {
		fail("cannot run loom %s on design using loom v%s\n", loom.Version(), *version)
	}

	startCheckErrors := time.Now()
	if err := eval.Context.Errors; err != nil {
		failStage("eval.Context.Errors", err)
	}
	debugStage(*debug, "eval.Context.Errors", startCheckErrors, "status=ok")

	startRunDSL := time.Now()
	if err := expr.RegisterDefaultRoots(); err != nil {
		failStage("eval.RunDSL", err)
	}
	if err := eval.RunDSL(); err != nil {
		failStage("eval.RunDSL", err)
	}
	debugStage(*debug, "eval.RunDSL", startRunDSL, "status=ok")

{{- range .CleanupDirs }}
	if err := os.RemoveAll({{ printf "%q" . }}); err != nil {
		fail(err.Error())
	}
{{- end }}
{{- if gt .DesignVersion 2 }}
	codegen.DesignVersion = ver
{{- end }}

	startGenerate := time.Now()
	outputs, err := generator.Generate(*out, {{ printf "%q" .Command }}, *debug)
	if err != nil {
		failStage("generator.Generate", err)
	}
	debugStage(*debug, "generator.Generate", startGenerate, "outputs=%d", len(outputs))
	debugStage(*debug, "total", startBinary, "outputs=%d", len(outputs))
	fmt.Println(strings.Join(outputs, "\n"))
}

func fail(msg string, vals ...any) {
	fmt.Fprintf(os.Stderr, msg, vals...)
	os.Exit(1)
}

func failStage(stage string, err error) {
	fail("stage %s: %s", stage, err)
}

func debugStage(debug bool, stage string, start time.Time, format string, vals ...any) {
	if !debug {
		return
	}
	msg := ""
	if format != "" {
		msg = " " + fmt.Sprintf(format, vals...)
	}
	fmt.Fprintf(os.Stderr, "[loom-debug] stage=%s duration=%s%s\n", stage, time.Since(start), msg)
}
`
