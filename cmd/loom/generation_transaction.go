package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/CaliLuke/loom/codegen"
	loom "github.com/CaliLuke/loom/pkg"
)

type (
	generationTransaction struct {
		output string
		root   string
		stage  string
		backup string
	}

	generationManifest struct {
		LoomVersion string `json:"loom_version"`
	}
)

func newGenerationTransaction(output string) (*generationTransaction, error) {
	absOutput, err := filepath.Abs(output)
	if err != nil {
		return nil, fmt.Errorf("resolve output directory %s: %w", output, err)
	}
	stageParent := filepath.Dir(absOutput)
	stageRelative := "."
	moduleRoot, err := findModuleRoot(absOutput)
	if err != nil {
		return nil, err
	}
	if moduleRoot != "" {
		stageParent = filepath.Dir(moduleRoot)
		stageRelative, err = filepath.Rel(moduleRoot, absOutput)
		if err != nil {
			return nil, fmt.Errorf("resolve output %s relative to module %s: %w", absOutput, moduleRoot, err)
		}
	}
	if err := os.MkdirAll(stageParent, 0o750); err != nil {
		return nil, fmt.Errorf("create staging parent %s: %w", stageParent, err)
	}
	stageRoot, err := os.MkdirTemp(stageParent, ".loom-gen-")
	if err != nil {
		return nil, fmt.Errorf("create generation staging directory in %s: %w", stageParent, err)
	}
	transaction := &generationTransaction{
		output: absOutput,
		root:   stageRoot,
		stage:  filepath.Join(stageRoot, stageRelative),
	}
	if err := transaction.prepareModuleContext(moduleRoot); err != nil {
		return nil, errors.Join(err, transaction.cleanup())
	}
	return transaction, nil
}

func finishGenerationTransaction(
	transaction *generationTransaction,
	outputs []string,
	debug bool,
) ([]string, error) {
	if transaction == nil {
		return outputs, nil
	}
	startValidate := time.Now()
	if err := transaction.validate(outputs); err != nil {
		return nil, wrapStageError("Validate", err)
	}
	debugStage(debug, "Validate", startValidate, "files=%d", len(outputs))
	startCommit := time.Now()
	committed, err := transaction.commit(outputs)
	if err != nil {
		return nil, wrapStageError("Commit", err)
	}
	debugStage(debug, "Commit", startCommit, "files=%d", len(committed))
	return committed, nil
}

func (t *generationTransaction) stagePath() string {
	return t.stage
}

func (t *generationTransaction) validate(outputs []string) error {
	stagedGen := filepath.Join(t.stage, codegen.Gendir)
	info, err := os.Stat(stagedGen)
	if err != nil {
		return fmt.Errorf("inspect staged generation tree %s: %w", stagedGen, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("staged generation path %s is not a directory", stagedGen)
	}
	manifestPath := filepath.Join(stagedGen, "loom.json")
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read staged generation manifest %s: %w", manifestPath, err)
	}
	var manifest generationManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return fmt.Errorf("decode staged generation manifest %s: %w", manifestPath, err)
	}
	if manifest.LoomVersion != loom.Version() {
		return fmt.Errorf(
			"staged generation manifest %s has Loom version %q, expected %q",
			manifestPath,
			manifest.LoomVersion,
			loom.Version(),
		)
	}
	for _, output := range outputs {
		rel, err := t.stagedRelativePath(output)
		if err != nil {
			return err
		}
		path := filepath.Join(t.stage, rel)
		info, err := os.Lstat(path)
		if err != nil {
			return fmt.Errorf("inspect staged output %s: %w", path, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("staged output %s is not a regular file", path)
		}
	}
	return nil
}

func (t *generationTransaction) commit(outputs []string) ([]string, error) {
	if err := os.MkdirAll(t.output, 0o750); err != nil {
		return nil, fmt.Errorf("create output directory %s: %w", t.output, err)
	}

	liveGen := filepath.Join(t.output, codegen.Gendir)
	stagedGen := filepath.Join(t.stage, codegen.Gendir)
	backup, hadLive, err := moveLiveGenerationAside(liveGen)
	if err != nil {
		return nil, err
	}
	t.backup = backup
	if err := os.Rename(stagedGen, liveGen); err != nil {
		commitErr := fmt.Errorf("replace generation tree %s: %w", liveGen, err)
		if hadLive {
			if rollbackErr := os.Rename(backup, liveGen); rollbackErr != nil {
				t.backup = ""
				return nil, errors.Join(commitErr, fmt.Errorf("restore generation tree %s: %w", liveGen, rollbackErr))
			}
			t.backup = ""
		}
		return nil, commitErr
	}
	if hadLive {
		if err := os.RemoveAll(backup); err != nil {
			return nil, fmt.Errorf("remove previous generation tree %s: %w", backup, err)
		}
		t.backup = ""
	}

	committed := make([]string, 0, len(outputs))
	for _, output := range outputs {
		rel, err := t.stagedRelativePath(output)
		if err != nil {
			return nil, err
		}
		path := filepath.Join(t.output, rel)
		if display, err := filepath.Rel(currentDirectory(), path); err == nil {
			path = display
		}
		committed = append(committed, path)
	}
	return committed, nil
}

func (t *generationTransaction) cleanup() error {
	var cleanupErr error
	if t.root != "" {
		stage := t.root
		t.root = ""
		t.stage = ""
		if err := os.RemoveAll(stage); err != nil {
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("remove generation staging directory %s: %w", stage, err))
		}
	}
	if t.backup != "" {
		backup := t.backup
		t.backup = ""
		if err := os.RemoveAll(backup); err != nil {
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("remove generation backup directory %s: %w", backup, err))
		}
	}
	return cleanupErr
}

func (t *generationTransaction) prepareModuleContext(moduleRoot string) error {
	if err := os.MkdirAll(t.stage, 0o750); err != nil {
		return fmt.Errorf("create staged output directory %s: %w", t.stage, err)
	}
	if moduleRoot == "" {
		return nil
	}
	goMod := filepath.Join(moduleRoot, "go.mod")
	contents, err := os.ReadFile(goMod)
	if err != nil {
		return fmt.Errorf("read module file %s: %w", goMod, err)
	}
	stagedGoMod := filepath.Join(t.root, "go.mod")
	if err := os.WriteFile(stagedGoMod, contents, 0o644); err != nil {
		return fmt.Errorf("write staged module file %s: %w", stagedGoMod, err)
	}
	return nil
}

func (t *generationTransaction) stagedRelativePath(output string) (string, error) {
	var absolute string
	if filepath.IsAbs(output) {
		absolute = filepath.Clean(output)
	} else {
		candidate, err := filepath.Abs(output)
		if err != nil {
			return "", fmt.Errorf("resolve generated output %s: %w", output, err)
		}
		absolute = candidate
	}
	rel, err := filepath.Rel(t.stage, absolute)
	if err != nil {
		return "", fmt.Errorf("resolve staged output %s: %w", output, err)
	}
	if pathEscapes(rel) {
		cleanOutput := filepath.Clean(output)
		if filepath.IsAbs(output) || pathEscapes(cleanOutput) || firstPathElement(cleanOutput) != codegen.Gendir {
			return "", fmt.Errorf("generated output %s is outside staging directory %s", output, t.stage)
		}
		rel = cleanOutput
	}
	if firstPathElement(rel) != codegen.Gendir {
		return "", fmt.Errorf("generated output %s is outside staged %s directory", output, codegen.Gendir)
	}
	return rel, nil
}

func moveLiveGenerationAside(liveGen string) (backup string, moved bool, err error) {
	if _, err := os.Lstat(liveGen); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("inspect generation tree %s: %w", liveGen, err)
	}
	backup, err = reserveRenamePath(filepath.Dir(liveGen))
	if err != nil {
		return "", false, err
	}
	if err := os.Rename(liveGen, backup); err != nil {
		return "", false, fmt.Errorf("preserve generation tree %s: %w", liveGen, err)
	}
	return backup, true, nil
}

func reserveRenamePath(dir string) (string, error) {
	path, err := os.MkdirTemp(dir, ".loom-gen-backup-")
	if err != nil {
		return "", fmt.Errorf("reserve generation backup path in %s: %w", dir, err)
	}
	if err := os.Remove(path); err != nil {
		return "", fmt.Errorf("prepare generation backup path %s: %w", path, err)
	}
	return path, nil
}

func currentDirectory() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	return dir
}

func pathEscapes(path string) bool {
	return path == ".." || strings.HasPrefix(path, ".."+string(filepath.Separator))
}

func firstPathElement(path string) string {
	if index := strings.IndexRune(path, filepath.Separator); index >= 0 {
		return path[:index]
	}
	return path
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
		if !errors.Is(err, os.ErrNotExist) && !errors.Is(err, syscall.ENOTDIR) {
			return "", fmt.Errorf("inspect module file %s: %w", goMod, err)
		}
		parent := filepath.Dir(path)
		if parent == path {
			return "", nil
		}
		path = parent
	}
}
