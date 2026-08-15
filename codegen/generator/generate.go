package generator

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
	"golang.org/x/tools/go/packages"
)

// Generate runs the code generation algorithms.
func Generate(dir, cmd string, debug bool) (outputs []string, err1 error) {
	startGenerate := time.Now()
	registry := codegen.DefaultRegistrySnapshot()

	roots, err := loadRoots(debug)
	if err != nil {
		return nil, err
	}

	genpkg, err := computeGenPackage(dir, debug)
	if err != nil {
		return nil, err
	}

	genfuncs, err := loadGenerators(cmd, debug)
	if err != nil {
		return nil, err
	}
	if err := runPreparePlugins(registry, cmd, genpkg, roots, debug); err != nil {
		return nil, err
	}

	genfiles, err := generateInitialFiles(genpkg, roots, genfuncs, debug)
	if err != nil {
		return nil, err
	}
	genfiles, err = runPostGenerationPlugins(registry, cmd, genpkg, roots, genfiles, debug)
	if err != nil {
		return nil, err
	}
	removePaths := collectRemovePaths(genfiles)

	// 7. Merge files that target the same path to avoid overwriting content when
	// multiple generators (or services) emit sections for the same file.
	{
		start := time.Now()
		genfiles = mergeFilesByPath(genfiles)
		debugStage(debug, "merge-files", start, "files=%d", len(genfiles))
	}
	if err := emitWarnings(genfiles); err != nil {
		return nil, wrapStageError("emit-warnings", "", err)
	}

	// 8. Emit loom.json version file (gen command only).
	if cmd == "gen" {
		genfiles = append(genfiles, codegen.VersionFile())
	}

	written, err := writeFiles(dir, genfiles, debug)
	if err != nil {
		return nil, err
	}
	if err := removeGeneratedPaths(dir, removePaths, written); err != nil {
		return nil, wrapStageError("remove-files", "", err)
	}

	outputs = computeOutputs(written, debug)
	sort.Strings(outputs)

	debugStage(debug, "total", startGenerate, "outputs=%d", len(outputs))
	return outputs, nil
}

func loadRoots(debug bool) ([]eval.Root, error) {
	start := time.Now()
	if err := expr.RegisterDefaultRoots(); err != nil {
		return nil, wrapStageError("load-roots", "", err)
	}
	roots, err := eval.Context.Roots()
	if err != nil {
		return nil, wrapStageError("load-roots", "", err)
	}
	debugStage(debug, "load-roots", start, "roots=%d", len(roots))
	return roots, nil
}

func computeGenPackage(dir string, debug bool) (genpkg string, err error) {
	start := time.Now()
	base, err := filepath.Abs(dir)
	if err != nil {
		return "", wrapStageError("compute-gen-package", dir, err)
	}
	path := filepath.Join(base, codegen.Gendir)
	if err := os.MkdirAll(path, 0750); err != nil {
		return "", wrapStageError("compute-gen-package", path, err)
	}

	dummy, err := os.CreateTemp(path, "temp.*.go")
	if err != nil {
		return "", wrapStageError("compute-gen-package", path, err)
	}
	dummyName := dummy.Name()
	defer func() {
		if removeErr := os.Remove(dummyName); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			err = errors.Join(err, wrapStageError("compute-gen-package", dummyName, removeErr))
		}
	}()
	if _, err = dummy.Write([]byte("package gen")); err != nil {
		return "", wrapStageError("compute-gen-package", dummyName, err)
	}
	if err = dummy.Close(); err != nil {
		return "", wrapStageError("compute-gen-package", dummyName, err)
	}

	moduleRoot, err := findModuleRoot(path)
	if err != nil {
		return "", wrapStageError("compute-gen-package", path, err)
	}
	if moduleRoot == "" {
		debugStage(debug, "compute-gen-package", start, "path=%s genpkg=%s", path, codegen.Gendir)
		return codegen.Gendir, nil
	}

	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedName,
		Env:  append(os.Environ(), "GOWORK=off"),
		Dir:  path,
	}, ".")
	if err != nil {
		return "", wrapStageError("compute-gen-package", path, err)
	}
	if len(pkgs) == 0 {
		return "", wrapStageError("compute-gen-package", path, errors.New("packages.Load returned no packages"))
	}
	genpkg = codegen.Gendir
	if !filepath.IsAbs(pkgs[0].PkgPath) {
		genpkg = pkgs[0].PkgPath
	}
	debugStage(debug, "compute-gen-package", start, "path=%s genpkg=%s", path, genpkg)
	return genpkg, nil
}

func findModuleRoot(path string) (string, error) {
	for {
		goMod := filepath.Join(path, "go.mod")
		info, err := os.Stat(goMod)
		if err == nil {
			if info.Mode().IsRegular() {
				return path, nil
			}
			return "", fmt.Errorf("module file %s is not a regular file", goMod)
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("inspect module file %s: %w", goMod, err)
		}
		parent := filepath.Dir(path)
		if parent == path {
			return "", nil
		}
		path = parent
	}
}

func loadGenerators(cmd string, debug bool) ([]genfunc, error) {
	start := time.Now()
	genfuncs, err := generatorLoader(cmd)
	if err != nil {
		return nil, wrapStageError("load-generators", cmd, err)
	}
	debugStage(debug, "load-generators", start, "command=%s generators=%d", cmd, len(genfuncs))
	return genfuncs, nil
}

func runPreparePlugins(registry *codegen.Registry, cmd, genpkg string, roots []eval.Root, debug bool) error {
	start := time.Now()
	if err := registry.RunPluginsPrepare(cmd, genpkg, roots); err != nil {
		return wrapStageError("prepare-plugins", genpkg, err)
	}
	debugStage(debug, "prepare-plugins", start, "genpkg=%s roots=%d", genpkg, len(roots))
	return nil
}

func generateInitialFiles(genpkg string, roots []eval.Root, genfuncs []genfunc, debug bool) ([]*codegen.File, error) {
	start := time.Now()
	var genfiles []*codegen.File
	for i, gen := range genfuncs {
		files, err := gen(genpkg, roots)
		if err != nil {
			return nil, wrapStageError("generate-initial-files", fmt.Sprintf("generator[%d]", i), err)
		}
		genfiles = append(genfiles, files...)
	}
	debugStage(debug, "generate-initial-files", start, "files=%d generators=%d", len(genfiles), len(genfuncs))
	return genfiles, nil
}

func runPostGenerationPlugins(
	registry *codegen.Registry,
	cmd, genpkg string,
	roots []eval.Root,
	genfiles []*codegen.File,
	debug bool,
) ([]*codegen.File, error) {
	start := time.Now()
	files, err := registry.RunPlugins(cmd, genpkg, roots, genfiles)
	if err != nil {
		return nil, wrapStageError("post-generation-plugins", genpkg, err)
	}
	debugStage(debug, "post-generation-plugins", start, "files=%d", len(files))
	return files, nil
}

type fileRenderWorkItem struct {
	index int
	file  *codegen.File
}

type fileRenderResult struct {
	index         int
	filename      string
	requestedPath string
	duration      time.Duration
	err           error
}

func writeFiles(dir string, genfiles []*codegen.File, debug bool) (map[string]struct{}, error) {
	start := time.Now()
	numWorkers := runtime.NumCPU()

	workChan := make(chan fileRenderWorkItem, len(genfiles))
	resultChan := make(chan fileRenderResult, len(genfiles))

	var wg sync.WaitGroup
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for work := range workChan {
				renderStart := time.Now()
				filename, err := work.file.Render(dir)
				resultChan <- fileRenderResult{
					index:         work.index,
					filename:      filename,
					requestedPath: work.file.Path,
					duration:      time.Since(renderStart),
					err:           err,
				}
			}
		}()
	}

	for i, file := range genfiles {
		workChan <- fileRenderWorkItem{index: i, file: file}
	}
	close(workChan)

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	written := make(map[string]struct{})
	firstErr := collectWriteResults(written, resultChan)
	if firstErr != nil {
		return nil, firstErr
	}
	debugStage(debug, "write-files", start, "files=%d workers=%d", len(written), numWorkers)
	return written, nil
}

func collectWriteResults(written map[string]struct{}, resultChan <-chan fileRenderResult) error {
	var firstErr error
	for res := range resultChan {
		if res.err != nil && firstErr == nil {
			path := res.requestedPath
			if res.filename != "" {
				path = res.filename
			}
			firstErr = wrapStageError("write-files", path, res.err)
		}
		if res.filename != "" {
			written[res.filename] = struct{}{}
		}
	}
	return firstErr
}

func collectRemovePaths(files []*codegen.File) []string {
	paths := make(map[string]struct{})
	for _, file := range files {
		if file == nil {
			continue
		}
		for _, path := range file.RemovePaths {
			paths[path] = struct{}{}
		}
	}
	return slices.Sorted(maps.Keys(paths))
}

func removeGeneratedPaths(dir string, paths []string, written map[string]struct{}) (err error) {
	if len(paths) == 0 {
		return nil
	}
	base, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(base)
	if err != nil {
		return fmt.Errorf("open output directory %s: %w", base, err)
	}
	defer func() {
		if closeErr := root.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close output directory %s: %w", base, closeErr))
		}
	}()
	for _, path := range paths {
		target, err := generatedRemovalPath(base, path)
		if err != nil {
			return err
		}
		info, err := root.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			delete(written, target)
			continue
		}
		if err != nil {
			return fmt.Errorf("inspect generated removal path %s: %w", target, err)
		}
		if info.IsDir() {
			return fmt.Errorf("generated removal path %s is a directory", target)
		}
		if err := root.Remove(path); err != nil {
			return fmt.Errorf("remove generated file %s: %w", target, err)
		}
		delete(written, target)
	}
	return nil
}

func generatedRemovalPath(base, path string) (string, error) {
	if !filepath.IsLocal(path) || filepath.Clean(path) == "." {
		return "", fmt.Errorf("generated removal path %s is outside output directory", path)
	}
	return filepath.Join(base, filepath.Clean(path)), nil
}

func computeOutputs(written map[string]struct{}, debug bool) []string {
	start := time.Now()
	outputs := make([]string, 0, len(written))
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	for output := range written {
		rel, err := filepath.Rel(cwd, output)
		if err != nil {
			rel = output
		}
		outputs = append(outputs, rel)
	}
	debugStage(debug, "compute-outputs", start, "outputs=%d", len(outputs))
	return outputs
}

func emitWarnings(files []*codegen.File) error {
	warnings := make(map[string]struct{})
	for _, file := range files {
		if file == nil {
			continue
		}
		for _, warning := range file.Warnings {
			if warning != "" {
				warnings[warning] = struct{}{}
			}
		}
	}
	ordered := slices.Sorted(maps.Keys(warnings))
	for _, warning := range ordered {
		if _, err := fmt.Fprintf(os.Stderr, "[loom-warning] %s\n", warning); err != nil {
			return fmt.Errorf("write warning: %w", err)
		}
	}
	return nil
}

// mergeFilesByPath coalesces files that share the same output path by
// concatenating their non-header sections and merging header imports. This
// prevents later renders from truncating earlier content when multiple
// services contribute sections to the same file (e.g., shared user types with
// union value methods).
func mergeFilesByPath(files []*codegen.File) []*codegen.File {
	if len(files) <= 1 {
		return files
	}

	byPath := make(map[string]*codegen.File)
	namesByPath := make(map[string]map[string]struct{})
	for _, f := range files {
		if f == nil {
			continue
		}
		if mergeFileByPath(byPath, namesByPath, f) {
			continue
		}
		recordFileByPath(byPath, namesByPath, f)
	}
	merged := make([]*codegen.File, 0, len(byPath))
	seenPaths := make(map[string]struct{})
	for _, f := range files {
		if f == nil {
			continue
		}
		if _, ok := seenPaths[f.Path]; ok {
			continue
		}
		if mf, ok := byPath[f.Path]; ok {
			merged = append(merged, mf)
			seenPaths[f.Path] = struct{}{}
		}
	}
	return merged
}

func mergeFileByPath(byPath map[string]*codegen.File, namesByPath map[string]map[string]struct{}, f *codegen.File) bool {
	existing, ok := byPath[f.Path]
	if !ok {
		return false
	}
	existingSections := append([]codegen.Section{}, existing.AllSections()...)
	incomingSections := f.AllSections()
	mergeHeaderSections(existingSections, incomingSections)
	initSectionNames(namesByPath, f.Path, existingSections)
	existing.SetSections(appendUniqueSections(existingSections, incomingSections, namesByPath[f.Path]))
	existing.Warnings = append(existing.Warnings, f.Warnings...)
	if existing.FinalizeFunc == nil && f.FinalizeFunc != nil {
		existing.FinalizeFunc = f.FinalizeFunc
	}
	return true
}

func mergeHeaderSections(existingSections, incomingSections []codegen.Section) {
	if len(existingSections) == 0 || len(incomingSections) == 0 {
		return
	}
	mergeHeaderImports(sectionTemplate(existingSections[0]), sectionTemplate(incomingSections[0]))
}

func initSectionNames(namesByPath map[string]map[string]struct{}, path string, sections []codegen.Section) {
	if namesByPath[path] != nil {
		return
	}
	namesByPath[path] = make(map[string]struct{}, len(sections))
	for _, section := range sections {
		namesByPath[path][section.SectionName()] = struct{}{}
	}
}

func appendUniqueSections(existingSections, incomingSections []codegen.Section, seen map[string]struct{}) []codegen.Section {
	for i, section := range incomingSections {
		if i == 0 {
			continue
		}
		if _, ok := seen[section.SectionName()]; ok {
			continue
		}
		existingSections = append(existingSections, section)
		seen[section.SectionName()] = struct{}{}
	}
	return existingSections
}

func recordFileByPath(byPath map[string]*codegen.File, namesByPath map[string]map[string]struct{}, f *codegen.File) {
	byPath[f.Path] = f
	initSectionNames(namesByPath, f.Path, f.AllSections())
}

func sectionTemplate(section codegen.Section) *codegen.SectionTemplate {
	if section == nil {
		return nil
	}
	template, _ := section.(*codegen.SectionTemplate)
	return template
}

// mergeHeaderImports merges the import specs from src header into dst header,
// deduplicating by (Name, Path). If either section is not a header produced by
// codegen.Header, this function is a no-op.
func mergeHeaderImports(dst, src *codegen.SectionTemplate) {
	if dst == nil || src == nil {
		return
	}
	dmap, dok := dst.Data.(map[string]any)
	smap, sok := src.Data.(map[string]any)
	if !dok || !sok {
		return
	}
	dlist, _ := dmap["Imports"].([]*codegen.ImportSpec)
	slist, _ := smap["Imports"].([]*codegen.ImportSpec)
	if len(slist) == 0 {
		return
	}
	seen := make(map[string]struct{}, len(dlist))
	for _, imp := range dlist {
		if imp == nil {
			continue
		}
		seen[imp.Name+"|"+imp.Path] = struct{}{}
	}
	for _, imp := range slist {
		if imp == nil {
			continue
		}
		key := imp.Name + "|" + imp.Path
		if _, ok := seen[key]; ok {
			continue
		}
		dlist = append(dlist, imp)
		seen[key] = struct{}{}
	}
	dmap["Imports"] = dlist
}

func wrapStageError(stage, path string, err error) error {
	if path == "" {
		return fmt.Errorf("stage %s: %w", stage, err)
	}
	return fmt.Errorf("stage %s path %s: %w", stage, path, err)
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
