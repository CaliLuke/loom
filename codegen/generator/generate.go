package generator

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"time"

	"golang.org/x/tools/go/packages"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/artifactpolicy"
	"github.com/CaliLuke/loom/internal/designfingerprint"
)

// Generate renders the command output and returns each written artifact with
// its write intent. designPackage is the canonical import path of the evaluated
// design that owns the output. Callers installing staged output must apply
// SkipExist to the live destination rather than treating every rendered path
// as replaceable. Options configure per-run resource use; invalid options fail
// before design loading or generation effects.
func Generate(dir, cmd, designPackage string, debug bool, options ...GenerateOption) (outputs []codegen.Artifact, err1 error) {
	config, err := generationConfiguration(options)
	if err != nil {
		return nil, wrapStageError("options", "", err)
	}
	startGenerate := time.Now()
	registry := codegen.DefaultRegistrySnapshot()

	roots, err := loadRoots(debug)
	if err != nil {
		return nil, err
	}

	design := designfingerprint.Design{Package: designPackage, Roots: roots}
	if _, err := design.Validate(); err != nil {
		return nil, wrapStageError("design-provenance", "", err)
	}

	genpkg, err := computeGenPackage(dir, debug)
	if err != nil {
		return nil, err
	}
	var designDigest string
	if cmd == "gen" {
		designDigest, err = designfingerprint.Digest(design, cmd, genpkg, codegen.DesignVersion)
		if err != nil {
			return nil, wrapStageError("design-digest", "", err)
		}
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
	outputs, err = renderGeneratedFiles(dir, cmd, designPackage, designDigest, genfiles, debug, config.workers)
	if err != nil {
		return nil, err
	}

	debugStage(debug, "total", startGenerate, "outputs=%d", len(outputs))
	return outputs, nil
}

// renderGeneratedFiles admits and merges output before installing files and
// applying removals. Admission must precede all rendering side effects.
func renderGeneratedFiles(dir, cmd, designPackage, designDigest string, genfiles []*codegen.File, debug bool, workers int) ([]codegen.Artifact, error) {
	plan, err := planArtifacts(dir, cmd, designPackage, designDigest, genfiles, debug)
	if err != nil {
		return nil, err
	}
	written, err := plan.render(dir, workers, debug)
	if err != nil {
		return nil, err
	}
	if err := removeGeneratedPaths(dir, plan.removals, written); err != nil {
		return nil, wrapStageError("remove-files", "", err)
	}

	outputs := computeOutputs(written, debug)
	slices.SortFunc(outputs, func(a, b codegen.Artifact) int {
		return cmp.Compare(a.Path, b.Path)
	})

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

func removeGeneratedPaths(dir string, paths []string, written map[string]codegen.Artifact) (err error) {
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
			err = root.RemoveAll(path)
		} else {
			err = root.Remove(path)
		}
		if err != nil {
			return fmt.Errorf("remove generated file %s: %w", target, err)
		}
		removeWrittenPaths(written, target, info.IsDir())
	}
	return nil
}

func removeWrittenPaths(written map[string]codegen.Artifact, target string, recursive bool) {
	for path := range written {
		if path == target {
			delete(written, path)
			continue
		}
		if !recursive {
			continue
		}
		relative, err := filepath.Rel(target, path)
		if err == nil && filepath.IsLocal(relative) && relative != "." {
			delete(written, path)
		}
	}
}

func generatedRemovalPath(base, path string) (string, error) {
	if !filepath.IsLocal(path) || filepath.Clean(path) == "." {
		return "", fmt.Errorf("generated removal path %s is outside output directory", path)
	}
	return filepath.Join(base, filepath.Clean(path)), nil
}

func computeOutputs(written map[string]codegen.Artifact, debug bool) []codegen.Artifact {
	start := time.Now()
	outputs := make([]codegen.Artifact, 0, len(written))
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	for output, artifact := range written {
		rel, err := filepath.Rel(cwd, output)
		if err != nil {
			rel = output
		}
		artifact.Path = rel
		outputs = append(outputs, artifact)
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
	if len(files) == 1 && files[0] != nil {
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
	if filepath.Ext(f.Path) == ".go" {
		if len(existingSections) == 1 {
			if merged, ok := existingSections[0].(*mergedGoSection); ok {
				existingSections = merged.sections
			}
		}
		existing.SetSections([]codegen.Section{&mergedGoSection{
			path: f.Path, sections: append(existingSections, incomingSections...),
		}})
		existing.Warnings = append(existing.Warnings, f.Warnings...)
		if existing.FinalizeFunc == nil {
			existing.FinalizeFunc = f.FinalizeFunc
		}
		return true
	}
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
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	fmt.Fprintf(os.Stderr, "[loom-debug] stage=%s duration=%s total_alloc=%d heap_alloc=%d heap_sys=%d num_gc=%d%s\n", stage, time.Since(start), memory.TotalAlloc, memory.HeapAlloc, memory.HeapSys, memory.NumGC, msg)
}

// normalizeFilePolicies resolves aliases before parallel rendering and refuses
// competing ownership declarations instead of depending on generator order.
func normalizeFilePolicies(dir string, files []*codegen.File) error {
	policies := make(map[string]bool)
	for _, file := range files {
		if file == nil {
			continue
		}
		file.Path = filepath.Clean(file.Path)
		if err := artifactpolicy.Validate(dir, file.Path, file.SkipExist); err != nil {
			return err
		}
		if policy, ok := policies[file.Path]; ok && policy != file.SkipExist {
			return fmt.Errorf("conflicting write policies for %s", file.Path)
		}
		policies[file.Path] = file.SkipExist
	}
	return nil
}
