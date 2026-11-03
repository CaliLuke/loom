package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

type laneResult struct {
	// Source identifies the immutable revision tested by this lane.
	Source string `json:"source"`
	// ConfigHash binds the result to the coverage consumer and floor configuration.
	ConfigHash string `json:"config_hash"`
	// SelectedHash authenticates the protected-consumer profile bytes.
	SelectedHash string `json:"selected_hash"`
	// RemainingHash authenticates the ordinary package profile bytes.
	RemainingHash string `json:"remaining_hash"`
	// Packages records the complete ordered package inventory executed by the lane.
	Packages []string `json:"packages"`
}

var suiteLanes = []string{"http", "grpc", "generator", "jsonrpc", "importer", "remaining"}

// packageLane preserves whole-package shuffled execution. New packages belong
// to remaining automatically, including packages selected by race build tags.
func packageLane(pkg string) string {
	switch pkg {
	case "github.com/CaliLuke/loom/http/codegen":
		return "http"
	case "github.com/CaliLuke/loom/grpc/codegen":
		return "grpc"
	case "github.com/CaliLuke/loom/codegen/generator":
		return "generator"
	case "github.com/CaliLuke/loom/jsonrpc/codegen":
		return "jsonrpc"
	case "github.com/CaliLuke/loom/internal/openapiimport":
		return "importer"
	default:
		return "remaining"
	}
}

func lanePackages(packages []string, lane string) []string {
	var result []string
	for _, pkg := range packages {
		if packageLane(pkg) == lane {
			result = append(result, pkg)
		}
	}
	return result
}

// suiteCoverageConfig instruments the same consumer universe in every lane;
// only the owned packages execute. Remaining consumers use ordinary coverage.
func suiteCoverageConfig(config baselineConfig, consumers, owned []string) baselineConfig {
	config.TestPackages = owned
	config.CoverPackages = slices.Clone(config.CoverPackages)
	for _, pkg := range consumers {
		if !slices.Contains(config.CoverPackages, pkg) {
			config.CoverPackages = append(config.CoverPackages, pkg)
		}
	}
	return config
}

func suiteTestArgs(config baselineConfig, profile, timeout string) []string {
	args := coverageTestArgs(config, profile, timeout)
	for i, arg := range args {
		if arg == "-covermode=count" {
			args[i] = "-covermode=atomic"
		}
	}
	return slices.Insert(args, 1, "-race", "-shuffle=on")
}

func runSuiteCoverage(config baselineConfig, profile, timeout string) error {
	if len(config.TestPackages) == 0 {
		return os.WriteFile(profile, []byte("mode: atomic\n"), 0o600)
	}
	command := exec.Command("go", suiteTestArgs(config, profile, timeout)...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}

func suiteInventory(config baselineConfig) (all, selected, remaining []string, err error) {
	all, err = listPackages([]string{"./..."})
	if err != nil {
		return
	}
	selected, err = listPackages(config.TestPackages)
	if err != nil {
		return
	}
	remaining, err = remainingPackages(all, selected)
	return
}

func runLane(configPath, timeout, dir, source, lane string) error {
	if dir == "" || source == "" || !slices.Contains(suiteLanes, lane) {
		return errors.New("lane requires a known lane, parts directory and source revision")
	}
	config, err := readConfig(configPath)
	if err != nil {
		return err
	}
	all, selected, remaining, err := suiteInventory(config)
	if err != nil {
		return err
	}
	packages := lanePackages(all, lane)
	if len(packages) == 0 {
		return fmt.Errorf("empty suite lane %s", lane)
	}
	partDir := filepath.Join(dir, "coverage-"+lane)
	if err := os.MkdirAll(partDir, 0o700); err != nil {
		return err
	}
	// Remove the completion marker before executing anything. A failed retry
	// must not leave an earlier successful result eligible for aggregation.
	marker := filepath.Join(partDir, "result.json")
	if err := os.Remove(marker); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	config = suiteCoverageConfig(config, selected, lanePackages(selected, lane))
	if err := runSuiteCoverage(config, filepath.Join(partDir, "selected.out"), timeout); err != nil {
		return fmt.Errorf("selected coverage: %w", err)
	}
	if err := runSuiteCoverage(baselineConfig{TestPackages: lanePackages(remaining, lane)}, filepath.Join(partDir, "remaining.out"), timeout); err != nil {
		return fmt.Errorf("remaining coverage: %w", err)
	}
	configHash, err := fileDigest(configPath)
	if err != nil {
		return err
	}
	selectedHash, err := fileDigest(filepath.Join(partDir, "selected.out"))
	if err != nil {
		return err
	}
	remainingHash, err := fileDigest(filepath.Join(partDir, "remaining.out"))
	if err != nil {
		return err
	}
	result, err := json.Marshal(laneResult{Source: source, ConfigHash: configHash, SelectedHash: selectedHash, RemainingHash: remainingHash, Packages: packages})
	if err != nil {
		return err
	}
	return os.WriteFile(marker, result, 0o600)
}

func mergeSuite(configPath, dir, source, profile string) error {
	if err := os.Remove(profile); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if dir == "" || source == "" {
		return errors.New("merge requires a parts directory and source revision")
	}
	config, err := readConfig(configPath)
	if err != nil {
		return err
	}
	all, err := listPackages([]string{"./..."})
	if err != nil {
		return err
	}
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	configHash := fmt.Sprintf("%x", sha256.Sum256(configData))
	selected, remaining := []byte("mode: atomic\n"), []byte("mode: atomic\n")
	for _, lane := range suiteLanes {
		partDir := filepath.Join(dir, "coverage-"+lane)
		protected, ordinary, err := readLane(partDir, source, configHash, lanePackages(all, lane))
		if err != nil {
			return fmt.Errorf("lane %s: %w", lane, err)
		}
		selected = append(selected, protected...)
		remaining = append(remaining, ordinary...)
	}
	blocks, err := parseProfile(bytes.NewReader(selected))
	if err != nil {
		return err
	}
	measurements, err := measureBoundaries(config, blocks)
	if err != nil {
		return err
	}
	printMeasurements(config, measurements)
	if regressions := evaluateMeasurements(config, measurements); len(regressions) > 0 {
		return errors.New(strings.Join(regressions, "; "))
	}
	return os.WriteFile(profile, append(selected, remaining[len("mode: atomic\n"):]...), 0o600)
}

func fileDigest(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(data)), nil
}

// readLane admits only complete results from this source and package partition.
func readLane(dir, source, configHash string, packages []string) ([]byte, []byte, error) {
	data, err := os.ReadFile(filepath.Join(dir, "result.json"))
	if err != nil {
		return nil, nil, fmt.Errorf("incomplete lane: %w", err)
	}
	var result laneResult
	if err := json.Unmarshal(data, &result, json.RejectUnknownMembers(true)); err != nil {
		return nil, nil, err
	}
	if result.Source != source || result.ConfigHash != configHash || !slices.Equal(result.Packages, packages) {
		return nil, nil, errors.New("source, config or package inventory mismatch")
	}
	selected, err := readLaneProfile(filepath.Join(dir, "selected.out"), result.SelectedHash)
	if err != nil {
		return nil, nil, err
	}
	remaining, err := readLaneProfile(filepath.Join(dir, "remaining.out"), result.RemainingHash)
	return selected, remaining, err
}

func readLaneProfile(path, hash string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if fmt.Sprintf("%x", sha256.Sum256(data)) != hash {
		return nil, fmt.Errorf("profile %s digest mismatch", path)
	}
	body, ok := bytes.CutPrefix(data, []byte("mode: atomic\n"))
	if !ok {
		return nil, fmt.Errorf("profile %s has invalid coverage mode", path)
	}
	if _, err := parseProfile(bytes.NewReader(data)); err != nil {
		return nil, err
	}
	return body, nil
}
