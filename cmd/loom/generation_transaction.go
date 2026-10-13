package main

import (
	json "encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/internal/artifactpolicy"
	"github.com/CaliLuke/loom/internal/designfingerprint"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
)

type (
	generationTransaction struct {
		output             string
		root               string
		stage              string
		backup             string
		externalBackups    []string
		externalCandidates []string
	}

	externalGenerationOutput struct {
		rel         string
		staged      string
		destination string
		backup      string
		installed   bool
		skipExist   bool
		skipped     bool
		identity    os.FileInfo
	}
)

func newGenerationTransaction(output string) (*generationTransaction, error) {
	absOutput, err := filepath.Abs(output)
	if err != nil {
		return nil, fmt.Errorf("resolve output directory %s: %w", output, err)
	}
	stageRelative := "."
	moduleRoot, err := findModuleRoot(absOutput)
	if err != nil {
		return nil, err
	}
	if moduleRoot != "" {
		stageRelative, err = filepath.Rel(moduleRoot, absOutput)
		if err != nil {
			return nil, fmt.Errorf("resolve output %s relative to module %s: %w", absOutput, moduleRoot, err)
		}
	}
	if err := os.MkdirAll(absOutput, 0o750); err != nil {
		return nil, fmt.Errorf("create output directory %s: %w", absOutput, err)
	}
	stageRoot, err := os.MkdirTemp(absOutput, ".loom-gen-")
	if err != nil {
		return nil, fmt.Errorf("create generation staging directory in %s: %w", absOutput, err)
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
	outputs []codegen.Artifact,
	debug bool,
) ([]string, error) {
	if transaction == nil {
		paths := make([]string, len(outputs))
		for i, output := range outputs {
			paths[i] = output.Path
		}
		return paths, nil
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

func (t *generationTransaction) validate(outputs []codegen.Artifact) error {
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
	var manifest designfingerprint.Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return fmt.Errorf("decode staged generation manifest %s: %w", manifestPath, err)
	}
	// The generator is built from the consuming module's Loom dependency, which
	// may differ from the launcher. Its manifest records that dependency's version.
	if manifest.LoomVersion != "(devel)" && !semver.IsValid(manifest.LoomVersion) {
		return fmt.Errorf("staged generation manifest %s has Loom version %q: expected a module version or (devel)", manifestPath, manifest.LoomVersion)
	}
	if err := manifest.ValidateProvenance(); err != nil {
		return fmt.Errorf("staged generation manifest %s: %w", manifestPath, err)
	}
	if err := t.validateWritePolicies(outputs); err != nil {
		return err
	}
	for _, output := range outputs {
		rel, err := t.stagedRelativePath(output.Path)
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

func (t *generationTransaction) commit(outputs []codegen.Artifact) ([]string, error) {
	external, err := t.prepareExternalOutputs(outputs)
	if err != nil {
		return nil, err
	}
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
	if err := t.installExternalOutputs(external); err != nil {
		return nil, errors.Join(err, t.rollbackInstalledOutputs(external, liveGen, hadLive))
	}
	if hadLive {
		if err := os.RemoveAll(backup); err != nil {
			return nil, fmt.Errorf("remove previous generation tree %s: %w", backup, err)
		}
		t.backup = ""
	}
	if err := t.removeExternalBackups(external); err != nil {
		return nil, err
	}

	installed := make(map[string]bool, len(external))
	for _, output := range external {
		installed[output.rel] = output.installed
	}
	committed := make([]string, 0, len(outputs))
	for _, output := range outputs {
		rel, err := t.stagedRelativePath(output.Path)
		if err != nil {
			return nil, err
		}
		if firstPathElement(rel) != codegen.Gendir && !installed[rel] {
			continue
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
	for _, backup := range t.externalBackups {
		if err := os.Remove(backup); err != nil && !errors.Is(err, os.ErrNotExist) {
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("remove generated output backup %s: %w", backup, err))
		}
	}
	t.externalBackups = nil
	for _, candidate := range t.externalCandidates {
		if err := os.Remove(candidate); err != nil && !errors.Is(err, os.ErrNotExist) {
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("remove generated output candidate %s: %w", candidate, err))
		}
	}
	t.externalCandidates = nil
	return cleanupErr
}

func (t *generationTransaction) prepareExternalOutputs(outputs []codegen.Artifact) ([]*externalGenerationOutput, error) {
	if err := t.validateWritePolicies(outputs); err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(outputs))
	external := make([]*externalGenerationOutput, 0, len(outputs))
	for _, output := range outputs {
		rel, err := t.stagedRelativePath(output.Path)
		if err != nil {
			return nil, err
		}
		if firstPathElement(rel) == codegen.Gendir {
			continue
		}
		if _, ok := seen[rel]; ok {
			continue
		}
		seen[rel] = struct{}{}
		destination := filepath.Join(t.output, rel)
		if err := os.MkdirAll(filepath.Dir(destination), 0o750); err != nil {
			return nil, fmt.Errorf("create generated output directory %s: %w", filepath.Dir(destination), err)
		}
		skipped := false
		if info, err := os.Lstat(destination); err == nil {
			skipped = output.SkipExist
			if !output.SkipExist && !info.Mode().IsRegular() {
				return nil, fmt.Errorf("generated output destination %s is not a regular file", destination)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("inspect generated output destination %s: %w", destination, err)
		}
		candidate := ""
		if !skipped {
			candidate, err = t.preparePublicationFile(filepath.Join(t.stage, rel), filepath.Dir(destination))
			if err != nil {
				return nil, err
			}
		}
		external = append(external, &externalGenerationOutput{
			rel:         rel,
			staged:      candidate,
			destination: destination,
			skipExist:   output.SkipExist,
			skipped:     skipped,
		})
	}
	return external, nil
}

func (t *generationTransaction) installExternalOutputs(outputs []*externalGenerationOutput) error {
	for _, output := range outputs {
		if output.skipped {
			continue
		}
		if output.skipExist {
			// Link publishes the fully rendered file only if the live name is absent.
			// Unlike Stat followed by Rename, it cannot replace a concurrent creator.
			info, err := os.Lstat(output.staged)
			if err != nil {
				return fmt.Errorf("inspect generated output %s: %w", output.staged, err)
			}
			if err := os.Link(output.staged, output.destination); errors.Is(err, os.ErrExist) {
				continue
			} else if err != nil {
				return fmt.Errorf("create generated output %s: %w", output.destination, err)
			}
			output.identity = info
			output.installed = true
			continue
		}
		if _, err := os.Lstat(output.destination); err == nil {
			backup, err := reserveRenamePath(filepath.Dir(output.destination))
			if err != nil {
				return err
			}
			if err := os.Rename(output.destination, backup); err != nil {
				return fmt.Errorf("preserve generated output %s: %w", output.destination, err)
			}
			output.backup = backup
			t.externalBackups = append(t.externalBackups, backup)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect generated output %s: %w", output.destination, err)
		}
		if err := os.Rename(output.staged, output.destination); err != nil {
			return fmt.Errorf("replace generated output %s: %w", output.destination, err)
		}
		output.installed = true
	}
	return nil
}

func (t *generationTransaction) removeExternalBackups(outputs []*externalGenerationOutput) error {
	for _, output := range outputs {
		if output.backup == "" {
			continue
		}
		if err := os.Remove(output.backup); err != nil {
			return fmt.Errorf("remove previous generated output %s: %w", output.backup, err)
		}
		t.forgetExternalBackup(output.backup)
		output.backup = ""
	}
	return nil
}

func (t *generationTransaction) rollbackInstalledOutputs(
	outputs []*externalGenerationOutput,
	liveGen string,
	hadLive bool,
) error {
	var rollbackErr error
	for _, output := range slices.Backward(outputs) {
		if output.installed {
			if output.skipExist {
				info, err := os.Lstat(output.destination)
				if err == nil && !os.SameFile(output.identity, info) {
					// Another owner replaced our entry. Rollback must not remove theirs.
					continue
				}
				if err != nil && !errors.Is(err, os.ErrNotExist) {
					rollbackErr = errors.Join(rollbackErr, fmt.Errorf("inspect generated output during rollback %s: %w", output.destination, err))
					continue
				}
			}
			if err := os.Remove(output.destination); err != nil && !errors.Is(err, os.ErrNotExist) {
				rollbackErr = errors.Join(rollbackErr, fmt.Errorf("remove generated output %s: %w", output.destination, err))
			}
		}
		if output.backup != "" {
			if err := os.Rename(output.backup, output.destination); err != nil {
				rollbackErr = errors.Join(rollbackErr, fmt.Errorf("restore generated output %s: %w", output.destination, err))
			} else {
				t.forgetExternalBackup(output.backup)
				output.backup = ""
			}
		}
	}
	if err := os.RemoveAll(liveGen); err != nil {
		rollbackErr = errors.Join(rollbackErr, fmt.Errorf("remove failed generation tree %s: %w", liveGen, err))
	}
	if hadLive {
		if err := os.Rename(t.backup, liveGen); err != nil {
			rollbackErr = errors.Join(rollbackErr, fmt.Errorf("restore generation tree %s: %w", liveGen, err))
		} else {
			t.backup = ""
		}
	}
	return rollbackErr
}

func (t *generationTransaction) forgetExternalBackup(path string) {
	for index, backup := range t.externalBackups {
		if backup == path {
			t.externalBackups = append(t.externalBackups[:index], t.externalBackups[index+1:]...)
			return
		}
	}
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
	module, err := modfile.Parse(goMod, contents, nil)
	if err != nil {
		return fmt.Errorf("parse module file %s: %w", goMod, err)
	}
	for _, replacement := range module.Replace {
		if replacement.New.Version != "" || filepath.IsAbs(replacement.New.Path) {
			continue
		}
		target := filepath.Join(moduleRoot, replacement.New.Path)
		if err := module.AddReplace(replacement.Old.Path, replacement.Old.Version, target, ""); err != nil {
			return fmt.Errorf("resolve staged module replacement %s: %w", replacement.Old.Path, err)
		}
	}
	contents, err = module.Format()
	if err != nil {
		return fmt.Errorf("format staged module file %s: %w", goMod, err)
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
		if filepath.IsAbs(output) || pathEscapes(cleanOutput) {
			return "", fmt.Errorf("generated output %s is outside staging directory %s", output, t.stage)
		}
		rel = cleanOutput
	}
	if rel == "." || rel == "" {
		return "", fmt.Errorf("generated output %s does not name a file", output)
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

// validateWritePolicies admits the complete batch against the live filesystem
// before preparation creates directories or commit swaps the generated tree.
func (t *generationTransaction) validateWritePolicies(outputs []codegen.Artifact) error {
	for _, output := range outputs {
		relative, err := t.stagedRelativePath(output.Path)
		if err != nil {
			return err
		}
		if err := artifactpolicy.Validate(t.output, relative, output.SkipExist); err != nil {
			return err
		}
	}
	return nil
}
