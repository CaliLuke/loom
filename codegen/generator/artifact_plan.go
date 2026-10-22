package generator

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/CaliLuke/loom/codegen"
)

type (
	artifactPlan struct {
		files    []*codegen.File
		removals []string
	}
	fileRenderResult struct {
		index         int
		filename      string
		skipExist     bool
		requestedPath string
		err           error
	}
)

// planArtifacts admits every lexical destination before any section or finalizer
// runs. It owns its pending slice, but never clears caller-owned file sections.
func planArtifacts(dir, cmd, designPackage, designDigest string, files []*codegen.File, debug bool) (*artifactPlan, error) {
	if err := normalizeFilePolicies(dir, files); err != nil {
		return nil, wrapStageError("file-policies", "", err)
	}
	removals := collectRemovePaths(files)
	for _, path := range removals {
		if _, err := generatedRemovalPath(dir, path); err != nil {
			return nil, wrapStageError("remove-files", path, err)
		}
	}
	var manifest *codegen.File
	if cmd == "gen" {
		manifest = codegen.ManifestFile(designPackage, designDigest)
		for _, file := range files {
			if file != nil && strings.EqualFold(file.Path, manifest.Path) {
				return nil, wrapStageError("file-policies", file.Path, fmt.Errorf("artifact path is reserved for the framework manifest"))
			}
		}
	}
	start := time.Now()
	files = mergeFilesByPath(files)
	files = slices.Clone(files)
	if manifest != nil {
		files = append(files, manifest)
	}
	if err := validateArtifactDestinations(files); err != nil {
		return nil, wrapStageError("file-policies", "", err)
	}
	debugStage(debug, "merge-files", start, "files=%d", len(files))
	if err := emitWarnings(files); err != nil {
		return nil, wrapStageError("emit-warnings", "", err)
	}
	return &artifactPlan{files: files, removals: removals}, nil
}

func validateArtifactDestinations(files []*codegen.File) error {
	paths := make(map[string]string, len(files))
	for _, file := range files {
		key := artifactPathKey(file.Path)
		if prior, exists := paths[key]; exists {
			return fmt.Errorf("artifact destinations %s and %s collide", prior, file.Path)
		}
		paths[key] = file.Path
	}
	for _, file := range files {
		for parent := filepath.Dir(artifactPathKey(file.Path)); parent != "."; parent = filepath.Dir(parent) {
			if prior, exists := paths[parent]; exists {
				return fmt.Errorf("artifact destination %s is a file ancestor of %s", prior, file.Path)
			}
		}
	}
	return nil
}

// artifactPathKey canonicalizes Unicode simple-fold classes, including sigma
// and long-s variants that lowercasing alone does not equate.
func artifactPathKey(path string) string {
	return strings.Map(func(r rune) rune {
		smallest := r
		for next := unicode.SimpleFold(r); next != r; next = unicode.SimpleFold(next) {
			smallest = min(smallest, next)
		}
		return smallest
	}, path)
}

func (p *artifactPlan) render(dir string, workers int, debug bool) (map[string]codegen.Artifact, error) {
	if workers < 1 {
		return nil, fmt.Errorf("render workers must be positive")
	}
	start := time.Now()
	workers = min(workers, len(p.files))
	// The bounded result queue contains only outcome metadata, never render data.
	results := make(chan fileRenderResult, workers)
	var running sync.WaitGroup
	next, active := 0, 0
	dispatch := func() {
		index, file := next, p.files[next]
		p.files[next] = nil
		next++
		active++
		running.Go(func() {
			filename, err := file.Render(dir)
			results <- fileRenderResult{index: index, filename: filename, skipExist: file.SkipExist, requestedPath: file.Path, err: err}
		})
	}
	for range workers {
		dispatch()
	}
	written := make(map[string]codegen.Artifact)
	firstIndex := len(p.files)
	var firstErr error
	for active > 0 {
		result := <-results
		active--
		if result.err != nil && firstErr == nil {
			clear(p.files[next:])
		}
		if result.err != nil && result.index < firstIndex {
			path := result.requestedPath
			if result.filename != "" {
				path = result.filename
			}
			firstIndex, firstErr = result.index, wrapStageError("write-files", path, result.err)
		}
		if result.filename != "" {
			written[result.filename] = codegen.Artifact{Path: result.filename, SkipExist: result.skipExist}
		}
		if firstErr == nil && next < len(p.files) {
			dispatch()
		}
	}
	running.Wait()
	clear(p.files)
	p.files = nil
	if firstErr != nil {
		return nil, firstErr
	}
	debugStage(debug, "write-files", start, "files=%d workers=%d", len(written), workers)
	return written, nil
}
