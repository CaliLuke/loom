package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// runSuite preserves the ratchet's consumer set while avoiding a second run of
// those tests for aggregate coverage. Remaining tests cannot inflate its floors.
func runSuite(configPath, testTimeout, profilePath string) (returnErr error) {
	if testTimeout != "" {
		if _, err := time.ParseDuration(testTimeout); err != nil {
			return fmt.Errorf("parse -test-timeout: %w", err)
		}
	}
	if err := os.Remove(profilePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove previous coverage profile: %w", err)
	}
	config, err := readConfig(configPath)
	if err != nil {
		return err
	}
	all, err := listPackages([]string{"./..."})
	if err != nil {
		return err
	}
	selected, err := listPackages(config.TestPackages)
	if err != nil {
		return err
	}
	remaining, err := remainingPackages(all, selected)
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "loom-suite-coverage-*")
	if err != nil {
		return fmt.Errorf("create coverage directory: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			returnErr = errors.Join(returnErr, fmt.Errorf("remove coverage directory: %w", err))
		}
	}()

	data, err := collectSelectedCoverage(config, selected, dir, testTimeout)
	if err != nil {
		return err
	}
	blocks, err := parseProfile(bytes.NewReader(data))
	if err != nil {
		return err
	}
	measurements, err := measureBoundaries(config, blocks)
	if err != nil {
		return err
	}
	data, err = appendRemainingCoverage(data, remaining, dir, testTimeout)
	if err != nil {
		return err
	}
	printMeasurements(config, measurements)
	if regressions := evaluateMeasurements(config, measurements); len(regressions) > 0 {
		return errors.New(strings.Join(regressions, "; "))
	}
	if err := os.WriteFile(profilePath, data, 0o600); err != nil {
		return fmt.Errorf("write aggregate coverage: %w", err)
	}
	return nil
}

func collectSelectedCoverage(config baselineConfig, selected []string, dir, testTimeout string) ([]byte, error) {
	config = suiteCoverageConfig(config, selected, selected)
	profile := filepath.Join(dir, "selected.out")
	if err := runSuiteCoverage(config, profile, testTimeout); err != nil {
		return nil, fmt.Errorf("collect consumer-aware coverage: %w", err)
	}
	data, err := os.ReadFile(profile)
	if err != nil {
		return nil, fmt.Errorf("read consumer-aware coverage: %w", err)
	}
	return data, nil
}

func appendRemainingCoverage(data []byte, remaining []string, dir, testTimeout string) ([]byte, error) {
	if len(remaining) == 0 {
		return data, nil
	}
	profile := filepath.Join(dir, "remaining.out")
	if err := runSuiteCoverage(baselineConfig{TestPackages: remaining}, profile, testTimeout); err != nil {
		return nil, fmt.Errorf("collect remaining coverage: %w", err)
	}
	other, err := os.ReadFile(profile)
	if err != nil {
		return nil, fmt.Errorf("read remaining coverage: %w", err)
	}
	if _, err := parseProfile(bytes.NewReader(other)); err != nil {
		return nil, err
	}
	body, ok := bytes.CutPrefix(other, []byte("mode: atomic\n"))
	if !ok {
		return nil, errors.New("remaining coverage must use mode: atomic")
	}
	return append(data, body...), nil
}

func listPackages(patterns []string) ([]string, error) {
	command := exec.Command("go", append([]string{"list", "-race"}, patterns...)...)
	command.Stderr = os.Stderr
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("list test packages: %w", err)
	}
	return strings.Fields(string(output)), nil
}

func remainingPackages(all, selected []string) ([]string, error) {
	if len(selected) == 0 {
		return nil, errors.New("no coverage test packages")
	}
	root := make(map[string]bool, len(all))
	for _, p := range all {
		root[p] = true
	}
	for _, p := range selected {
		if _, ok := root[p]; !ok {
			return nil, fmt.Errorf("coverage test package %s is outside the root suite", p)
		}
		root[p] = false
	}
	var remaining []string
	for _, p := range all {
		if root[p] {
			remaining = append(remaining, p)
		}
	}
	return remaining, nil
}
