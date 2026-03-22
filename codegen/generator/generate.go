package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/eval"
	"golang.org/x/tools/go/packages"
)

// Generate runs the code generation algorithms.
func Generate(dir, cmd string, debug bool) (outputs []string, err1 error) {
	startGenerate := time.Now()
	if debug {
		fmt.Fprintf(os.Stderr, "[TIMING]     [generate] Starting generator.Generate()\n")
	}

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

	if err := runPreparePlugins(cmd, genpkg, roots, debug); err != nil {
		return nil, err
	}

	genfiles, err := generateInitialFiles(genpkg, roots, genfuncs, debug)
	if err != nil {
		return nil, err
	}

	genfiles, err = runPostGenerationPlugins(cmd, genpkg, roots, genfiles, debug)
	if err != nil {
		return nil, err
	}

	// 7. Merge files that target the same path to avoid overwriting content when
	// multiple generators (or services) emit sections for the same file.
	{
		start := time.Now()
		genfiles = mergeFilesByPath(genfiles)
		if debug {
			fmt.Fprintf(os.Stderr, "[TIMING]     [generate] Stage 7: Merging files by path took %v (now %d files)\n", time.Since(start), len(genfiles))
		}
	}

	// 8. Emit loom.json version file (gen command only).
	if cmd == "gen" {
		genfiles = append(genfiles, codegen.VersionFile())
	}

	written, err := writeFiles(dir, genfiles, debug)
	if err != nil {
		return nil, err
	}

	outputs = computeOutputs(written, debug)
	sort.Strings(outputs)

	if debug {
		fmt.Fprintf(os.Stderr, "[TIMING]     [generate] Total generator.Generate() took %v\n", time.Since(startGenerate))
	}
	return outputs, nil
}

func loadRoots(debug bool) ([]eval.Root, error) {
	start := time.Now()
	roots, err := eval.Context.Roots()
	if err != nil {
		return nil, err
	}
	if debug {
		fmt.Fprintf(os.Stderr, "[TIMING]     [generate] Stage 1: Compute design roots took %v\n", time.Since(start))
	}
	return roots, nil
}

func computeGenPackage(dir string, debug bool) (string, error) {
	start := time.Now()
	base, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	path := filepath.Join(base, codegen.Gendir)
	if err := os.MkdirAll(path, 0750); err != nil {
		return "", err
	}

	dummy, err := os.CreateTemp(path, "temp.*.go")
	if err != nil {
		return "", err
	}
	dummyName := dummy.Name()
	defer func() {
		_ = os.Remove(dummyName)
	}()
	if _, err = dummy.Write([]byte("package gen")); err != nil {
		return "", err
	}
	if err = dummy.Close(); err != nil {
		return "", err
	}

	startPkgLoad := time.Now()
	pkgs, err := packages.Load(&packages.Config{Mode: packages.NeedName}, path)
	if err != nil {
		return "", err
	}
	genpkg := codegen.Gendir
	if !filepath.IsAbs(pkgs[0].PkgPath) {
		genpkg = pkgs[0].PkgPath
	}
	if debug {
		fmt.Fprintf(os.Stderr, "[TIMING]     [generate]   packages.Load took %v\n", time.Since(startPkgLoad))
		fmt.Fprintf(os.Stderr, "[TIMING]     [generate] Stage 2: Compute gen package import path took %v\n", time.Since(start))
	}
	return genpkg, nil
}

func loadGenerators(cmd string, debug bool) ([]Genfunc, error) {
	start := time.Now()
	genfuncs, err := Generators(cmd)
	if err != nil {
		return nil, err
	}
	if debug {
		fmt.Fprintf(os.Stderr, "[TIMING]     [generate] Stage 3: Retrieve Loom generators took %v (%d generators)\n", time.Since(start), len(genfuncs))
	}
	return genfuncs, nil
}

func runPreparePlugins(cmd, genpkg string, roots []eval.Root, debug bool) error {
	start := time.Now()
	if err := codegen.RunPluginsPrepare(cmd, genpkg, roots); err != nil {
		return err
	}
	if debug {
		fmt.Fprintf(os.Stderr, "[TIMING]     [generate] Stage 4: Run pre-generation plugins took %v\n", time.Since(start))
	}
	return nil
}

func generateInitialFiles(genpkg string, roots []eval.Root, genfuncs []Genfunc, debug bool) ([]*codegen.File, error) {
	start := time.Now()
	var genfiles []*codegen.File
	for i, gen := range genfuncs {
		genStart := time.Now()
		files, err := gen(genpkg, roots)
		if err != nil {
			return nil, err
		}
		genfiles = append(genfiles, files...)
		if debug {
			fmt.Fprintf(os.Stderr, "[TIMING]     [generate]   Generator %d produced %d files in %v\n", i, len(files), time.Since(genStart))
		}
	}
	if debug {
		fmt.Fprintf(os.Stderr, "[TIMING]     [generate] Stage 5: Generate initial files took %v (total %d files)\n", time.Since(start), len(genfiles))
	}
	return genfiles, nil
}

func runPostGenerationPlugins(cmd, genpkg string, roots []eval.Root, genfiles []*codegen.File, debug bool) ([]*codegen.File, error) {
	start := time.Now()
	files, err := codegen.RunPlugins(cmd, genpkg, roots, genfiles)
	if err != nil {
		return nil, err
	}
	if debug {
		fmt.Fprintf(os.Stderr, "[TIMING]     [generate] Stage 6: Run post-generation plugins took %v (now %d files)\n", time.Since(start), len(files))
	}
	return files, nil
}

type fileRenderWorkItem struct {
	index int
	file  *codegen.File
}

type fileRenderResult struct {
	index    int
	filename string
	duration time.Duration
	err      error
}

func writeFiles(dir string, genfiles []*codegen.File, debug bool) (map[string]struct{}, error) {
	start := time.Now()
	numWorkers := runtime.NumCPU()
	if debug {
		fmt.Fprintf(os.Stderr, "[TIMING]     [generate] Stage 9: Starting parallel file writing with %d workers\n", numWorkers)
	}

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
					index:    work.index,
					filename: filename,
					duration: time.Since(renderStart),
					err:      err,
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
	firstErr := collectWriteResults(written, resultChan, debug)
	if firstErr != nil {
		return nil, firstErr
	}
	if debug {
		fmt.Fprintf(os.Stderr, "[TIMING]     [generate] Stage 9: Write files took %v (%d files written)\n", time.Since(start), len(written))
	}
	return written, nil
}

func collectWriteResults(written map[string]struct{}, resultChan <-chan fileRenderResult, debug bool) error {
	var firstErr error
	slowRenders := 0
	for res := range resultChan {
		if res.err != nil && firstErr == nil {
			firstErr = res.err
		}
		if res.filename != "" {
			written[res.filename] = struct{}{}
		}
		if debug && res.duration > 100*time.Millisecond {
			fmt.Fprintf(os.Stderr, "[TIMING]     [generate]   File %d (%s) render took %v\n", res.index, res.filename, res.duration)
			slowRenders++
		}
	}
	if debug && slowRenders > 0 {
		fmt.Fprintf(os.Stderr, "[TIMING]     [generate]   Slow renders: %d\n", slowRenders)
	}
	return firstErr
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
	if debug {
		fmt.Fprintf(os.Stderr, "[TIMING]     [generate] Stage 10: Compute output filenames took %v\n", time.Since(start))
	}
	return outputs
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

	// First pass: build merged file per path
	for _, f := range files {
		if f == nil {
			continue
		}
		path := f.Path
		if existing, ok := byPath[path]; ok {
			existingSections := append([]codegen.Section{}, existing.AllSections()...)
			incomingSections := f.AllSections()

			// Merge headers (index 0) imports
			if len(existingSections) > 0 && len(incomingSections) > 0 {
				mergeHeaderImports(sectionTemplate(existingSections[0]), sectionTemplate(incomingSections[0]))
			}
			// Initialize seen section names for this path
			if namesByPath[path] == nil {
				namesByPath[path] = make(map[string]struct{})
				for _, section := range existingSections {
					namesByPath[path][section.SectionName()] = struct{}{}
				}
			}
			// Append unique sections (skip header at index 0)
			for i, section := range incomingSections {
				if i == 0 {
					continue
				}
				if _, seen := namesByPath[path][section.SectionName()]; seen {
					continue
				}
				existingSections = append(existingSections, section)
				namesByPath[path][section.SectionName()] = struct{}{}
			}
			existing.SetSections(existingSections)
			// Preserve a finalize function if destination does not have one
			if existing.FinalizeFunc == nil && f.FinalizeFunc != nil {
				existing.FinalizeFunc = f.FinalizeFunc
			}
			// Skip adding a duplicate File entry
			continue
		}

		// New path: record and initialize seen names
		byPath[path] = f
		m := make(map[string]struct{})
		for _, section := range f.AllSections() {
			m[section.SectionName()] = struct{}{}
		}
		namesByPath[path] = m
	}

	// Second pass: preserve original order by first occurrence of each path
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
