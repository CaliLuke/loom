package valuecontract

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type (
	artifactDifference struct {
		// Path identifies a generated artifact relative to the temporary module.
		Path string `json:"path"`
		// Kind classifies an added, removed, or changed artifact.
		Kind string `json:"kind"`
		// Before is the baseline artifact hash, empty when the path is absent.
		Before string `json:"before,omitempty"`
		// After is the candidate artifact hash, empty when the path is absent.
		After string `json:"after,omitempty"`
	}
	allowedDifference struct {
		// Probe identifies the design that owns the intended difference.
		Probe string `json:"probe"`
		// Path identifies a generated artifact relative to the temporary module.
		Path string `json:"path"`
		// Before is the baseline artifact hash, empty when the path is absent.
		Before string `json:"before,omitempty"`
		// After is the candidate artifact hash, empty when the path is absent.
		After string `json:"after,omitempty"`
		// Reason explains why the exact artifact change is intentional.
		Reason string `json:"reason"`
	}
	expectation struct {
		// Phase names the generation, compilation, vetting, or decoder stage.
		Phase string `json:"phase,omitempty"`
		// Contains is the required diagnostic substring for a characterized failure.
		Contains string `json:"contains,omitempty"`
		// Issue links the Loom ticket tracking the characterized behavior.
		Issue string `json:"issue,omitempty"`
	}
	phaseResult struct {
		// Phase names the generation, compilation, vetting, or decoder stage.
		Phase string `json:"phase"`
		// Exit is zero on success and nonzero on command failure.
		Exit int `json:"exit"`
		// Output retains the command output and execution error.
		Output string `json:"output,omitempty"`
		// Infrastructure distinguishes process/tool/resource failures from contract defects.
		Infrastructure bool `json:"infrastructure,omitempty"`
	}
)

func digest(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func snapshotArtifacts(source, destination string, inputs map[string]string) (map[string]string, error) {
	artifacts := make(map[string]string)
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if relative == "design" || relative == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if relative == "go.mod" || relative == "go.sum" || relative == "contract_test.go" || relative == "observation.json" {
			return nil
		}
		if _, supplied := inputs[filepath.ToSlash(relative)]; supplied {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("non-regular artifact %s", relative)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0600); err != nil {
			return err
		}
		artifacts[filepath.ToSlash(relative)] = digest(data)
		return nil
	})
	return artifacts, err
}

func compareArtifacts(before, after map[string]string) []artifactDifference {
	paths := make(map[string]bool)
	for path := range before {
		paths[path] = true
	}
	for path := range after {
		paths[path] = true
	}
	ordered := make([]string, 0, len(paths))
	for path := range paths {
		ordered = append(ordered, path)
	}
	slices.Sort(ordered)
	var differences []artifactDifference
	for _, path := range ordered {
		old, oldOK := before[path]
		next, nextOK := after[path]
		if old == next && oldOK == nextOK {
			continue
		}
		kind := "changed"
		if !oldOK {
			kind = "added"
		} else if !nextOK {
			kind = "removed"
		}
		differences = append(differences, artifactDifference{Path: path, Kind: kind, Before: old, After: next})
	}
	return differences
}

func checkDifferences(probe string, differences []artifactDifference, allowed []allowedDifference) error {
	expected := make(map[string]allowedDifference)
	for _, entry := range allowed {
		if entry.Probe == probe {
			if _, ok := expected[entry.Path]; ok {
				return fmt.Errorf("duplicate intended difference %s/%s", probe, entry.Path)
			}
			expected[entry.Path] = entry
		}
	}
	for _, difference := range differences {
		want, ok := expected[difference.Path]
		if !ok || want.Reason == "" || want.Before != difference.Before || want.After != difference.After {
			return fmt.Errorf("unlisted artifact difference %s/%s: %s -> %s", probe, difference.Path, difference.Before, difference.After)
		}
		delete(expected, difference.Path)
	}
	if len(expected) != 0 {
		return fmt.Errorf("stale intended differences for %s: %v", probe, expected)
	}
	return nil
}

func checkOutcome(want expectation, got phaseResult) error {
	if got.Infrastructure || strings.Contains(got.Output, "no space left on device") {
		return fmt.Errorf("infrastructure failure during %s: %s", got.Phase, got.Output)
	}
	if want.Phase == "" {
		if got.Exit != 0 {
			return fmt.Errorf("unexpected %s failure: %s", got.Phase, got.Output)
		}
		return nil
	}
	if got.Exit == 0 {
		return errors.New("unexpected pass: remove the stale legacy expectation")
	}
	if want.Phase != got.Phase || want.Contains == "" || !strings.Contains(got.Output, want.Contains) {
		return fmt.Errorf("expected %s containing %q, got %s: %s", want.Phase, want.Contains, got.Phase, got.Output)
	}
	return nil
}
