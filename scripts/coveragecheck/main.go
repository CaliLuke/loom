// Command coveragecheck enforces consumer-aware statement coverage baselines.
package main

import (
	"bufio"
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"sort"
	"strconv"
	"strings"
)

type (
	baselineConfig struct {
		CoverPackages []string   `json:"cover_packages"`
		TestPackages  []string   `json:"test_packages"`
		Boundaries    []boundary `json:"boundaries"`
	}

	boundary struct {
		Name               string   `json:"name"`
		Packages           []string `json:"packages"`
		MinimumBasisPoints int      `json:"minimum_basis_points"`
		Rationale          string   `json:"rationale"`
	}

	profileBlock struct {
		Package    string
		Statements int
		Covered    bool
	}

	measurement struct {
		CoveredStatements int
		TotalStatements   int
		BasisPoints       int
	}
)

func main() {
	configPath := flag.String("config", "coverage/baselines.json", "path to the checked-in coverage baseline")
	update := flag.Bool("update", false, "replace each baseline with the current measured coverage")
	flag.Parse()

	if err := run(*configPath, *update); err != nil {
		fmt.Fprintln(os.Stderr, "coverage ratchet:", err)
		os.Exit(1)
	}
}

func run(configPath string, update bool) error {
	config, err := readConfig(configPath)
	if err != nil {
		return err
	}
	blocks, err := collectCoverage(config)
	if err != nil {
		return err
	}
	measurements, err := measureBoundaries(config, blocks)
	if err != nil {
		return err
	}
	printMeasurements(config, measurements)
	if update {
		for i := range config.Boundaries {
			config.Boundaries[i].MinimumBasisPoints = measurements[config.Boundaries[i].Name].BasisPoints
		}
		return writeConfig(configPath, config)
	}
	if regressions := evaluateMeasurements(config, measurements); len(regressions) > 0 {
		return errors.New(strings.Join(regressions, "; "))
	}
	return nil
}

func collectCoverage(config baselineConfig) (blocks map[string]profileBlock, returnErr error) {
	profile, err := os.CreateTemp("", "loom-coverage-*.out")
	if err != nil {
		return nil, fmt.Errorf("create temporary coverage profile: %w", err)
	}
	profilePath := profile.Name()
	if err := profile.Close(); err != nil {
		return nil, fmt.Errorf("close temporary coverage profile: %w", err)
	}
	defer func() {
		if removeErr := os.Remove(profilePath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			returnErr = errors.Join(returnErr, fmt.Errorf("remove temporary coverage profile: %w", removeErr))
		}
	}()

	args := make([]string, 0, 5+len(config.TestPackages))
	args = append(args,
		"test",
		"-count=1",
		"-covermode=count",
		"-coverpkg="+strings.Join(config.CoverPackages, ","),
		"-coverprofile="+profilePath,
	)
	args = append(args, config.TestPackages...)
	command := exec.Command("go", args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("collect consumer-aware coverage: %w", err)
	}

	profileData, err := os.ReadFile(profilePath)
	if err != nil {
		return nil, fmt.Errorf("read coverage profile: %w", err)
	}
	return parseProfile(bytes.NewReader(profileData))
}

func measureBoundaries(config baselineConfig, blocks map[string]profileBlock) (map[string]measurement, error) {
	measurements := make(map[string]measurement, len(config.Boundaries))
	for _, boundary := range config.Boundaries {
		measured, err := measureBoundary(blocks, boundary)
		if err != nil {
			return nil, err
		}
		measurements[boundary.Name] = measured
	}
	return measurements, nil
}

func printMeasurements(config baselineConfig, measurements map[string]measurement) {
	for _, boundary := range config.Boundaries {
		measured := measurements[boundary.Name]
		fmt.Printf(
			"coverage %-20s %6.2f%% (%d/%d statements; baseline %.2f%%)\n",
			boundary.Name,
			float64(measured.BasisPoints)/100,
			measured.CoveredStatements,
			measured.TotalStatements,
			float64(boundary.MinimumBasisPoints)/100,
		)
	}
}

func readConfig(configPath string) (baselineConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return baselineConfig{}, fmt.Errorf("read baseline %s: %w", configPath, err)
	}
	var config baselineConfig
	if err := json.Unmarshal(data, &config, json.RejectUnknownMembers(true)); err != nil {
		return baselineConfig{}, fmt.Errorf("decode baseline %s: %w", configPath, err)
	}
	if err := validateConfig(config); err != nil {
		return baselineConfig{}, fmt.Errorf("validate baseline %s: %w", configPath, err)
	}
	return config, nil
}

func writeConfig(configPath string, config baselineConfig) error {
	data, err := json.Marshal(config, json.Deterministic(true), jsontext.WithIndent("  "))
	if err != nil {
		return fmt.Errorf("encode baseline %s: %w", configPath, err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		return fmt.Errorf("write baseline %s: %w", configPath, err)
	}
	return nil
}

func validateConfig(config baselineConfig) error {
	if len(config.CoverPackages) == 0 {
		return errors.New("cover_packages must not be empty")
	}
	if len(config.TestPackages) == 0 {
		return errors.New("test_packages must not be empty")
	}
	if len(config.Boundaries) == 0 {
		return errors.New("boundaries must not be empty")
	}
	names := make(map[string]struct{}, len(config.Boundaries))
	for _, boundary := range config.Boundaries {
		if boundary.Name == "" {
			return errors.New("boundary name must not be empty")
		}
		if _, exists := names[boundary.Name]; exists {
			return fmt.Errorf("duplicate boundary %q", boundary.Name)
		}
		names[boundary.Name] = struct{}{}
		if len(boundary.Packages) == 0 {
			return fmt.Errorf("boundary %q has no packages", boundary.Name)
		}
		if boundary.MinimumBasisPoints < 0 || boundary.MinimumBasisPoints > 10000 {
			return fmt.Errorf("boundary %q minimum_basis_points must be between 0 and 10000", boundary.Name)
		}
		if boundary.Rationale == "" {
			return fmt.Errorf("boundary %q rationale must not be empty", boundary.Name)
		}
	}
	return nil
}

func parseProfile(reader io.Reader) (map[string]profileBlock, error) {
	scanner := bufio.NewScanner(reader)
	if !scanner.Scan() {
		return nil, errors.New("coverage profile is empty")
	}
	if !strings.HasPrefix(scanner.Text(), "mode: ") {
		return nil, fmt.Errorf("coverage profile has invalid mode line %q", scanner.Text())
	}
	blocks := make(map[string]profileBlock)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 3 {
			return nil, fmt.Errorf("coverage profile has invalid line %q", scanner.Text())
		}
		location := fields[0]
		separator := strings.LastIndex(location, ":")
		if separator < 1 {
			return nil, fmt.Errorf("coverage profile has invalid location %q", location)
		}
		statements, err := strconv.Atoi(fields[1])
		if err != nil || statements < 0 {
			return nil, fmt.Errorf("coverage profile has invalid statement count %q", fields[1])
		}
		count, err := strconv.ParseUint(fields[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("coverage profile has invalid execution count %q", fields[2])
		}
		file := location[:separator]
		block, exists := blocks[location]
		if exists && block.Statements != statements {
			return nil, fmt.Errorf("coverage block %q has inconsistent statement counts", location)
		}
		block.Package = path.Dir(file)
		block.Statements = statements
		block.Covered = block.Covered || count > 0
		blocks[location] = block
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read coverage profile: %w", err)
	}
	return blocks, nil
}

func measureBoundary(blocks map[string]profileBlock, boundary boundary) (measurement, error) {
	packageSet := make(map[string]struct{}, len(boundary.Packages))
	for _, packagePath := range boundary.Packages {
		packageSet[packagePath] = struct{}{}
	}
	seenPackages := make(map[string]struct{}, len(boundary.Packages))
	var result measurement
	for _, block := range blocks {
		if _, included := packageSet[block.Package]; !included {
			continue
		}
		seenPackages[block.Package] = struct{}{}
		result.TotalStatements += block.Statements
		if block.Covered {
			result.CoveredStatements += block.Statements
		}
	}
	missing := make([]string, 0)
	for _, packagePath := range boundary.Packages {
		if _, seen := seenPackages[packagePath]; !seen {
			missing = append(missing, packagePath)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return measurement{}, fmt.Errorf("boundary %q has no statements for %s", boundary.Name, strings.Join(missing, ", "))
	}
	if result.TotalStatements == 0 {
		return measurement{}, fmt.Errorf("boundary %q has no statements", boundary.Name)
	}
	result.BasisPoints = result.CoveredStatements * 10000 / result.TotalStatements
	return result, nil
}

func evaluateMeasurements(config baselineConfig, measurements map[string]measurement) []string {
	var regressions []string
	for _, boundary := range config.Boundaries {
		measured := measurements[boundary.Name]
		if measured.BasisPoints < boundary.MinimumBasisPoints {
			regressions = append(regressions, fmt.Sprintf(
				"%s coverage is %.2f%%, below checked-in baseline %.2f%%",
				boundary.Name,
				float64(measured.BasisPoints)/100,
				float64(boundary.MinimumBasisPoints)/100,
			))
		}
	}
	return regressions
}
