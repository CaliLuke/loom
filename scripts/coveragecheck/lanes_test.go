package main

import (
	"crypto/sha256"
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// A real miniature repository checks discovery, exactly-once execution,
// race-only packages, all lane artifacts, aggregate coverage and fail-closed
// publication together. This replaces the shell partition's command stubs.
func TestDistributedSuite(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GOWORK", "off")
	writeSuiteFile(t, "go.mod", "module github.com/CaliLuke/loom\n\ngo 1.27.0\n")
	owners := []string{"http/codegen", "grpc/codegen", "codegen/generator", "jsonrpc/codegen", "internal/openapiimport", "other"}
	for _, p := range owners {
		writeSuiteFile(t, p+"/value.go", "package fixture\nfunc A() int { return 1 }\nfunc B() int { return 2 }\n")
		writeSuiteFile(t, p+"/once_test.go", `package fixture
import (
 "os"
 "testing"
)
func TestOnce(t *testing.T) {
 f, err := os.OpenFile("ran", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
 if err != nil {
  t.Fatal(err)
 }
 if err := f.Close(); err != nil {
  t.Fatal(err)
 }
 if A() != 1 {
  t.Fatal("wrong result")
 }
}
`)
		writeSuiteFile(t, p+"/no_race_test.go", "//go:build !race\n\npackage fixture\nimport \"testing\"\nfunc TestRequiresRace(t *testing.T) { t.Fatal(\"missing race instrumentation\") }\n")
	}
	writeSuiteFile(t, "raceonly/only_test.go", "//go:build race\n\npackage raceonly\nimport (\"testing\"; \"os\")\nfunc TestRaceOnly(t *testing.T) { if err := os.WriteFile(\"ran\", nil, 0600); err != nil { t.Fatal(err) } }\n")
	// An excluded consumer covers B. It must not lift the protected HTTP floor.
	writeSuiteFile(t, "other/consumer_test.go", "package fixture\nimport (\"testing\"; svc \"github.com/CaliLuke/loom/http/codegen\")\nfunc TestOtherConsumer(t *testing.T) { if svc.B() != 2 { t.Fatal(\"bad result\") } }\n")
	config := baselineConfig{
		CoverPackages: []string{"github.com/CaliLuke/loom/http/codegen"},
		TestPackages:  []string{"./http/codegen", "./grpc/codegen", "./codegen/generator", "./jsonrpc/codegen"},
		Boundaries:    []boundary{{Name: "http", Packages: []string{"github.com/CaliLuke/loom/http/codegen"}, MinimumBasisPoints: 5000, Rationale: "selected consumers only"}},
	}
	require.NoError(t, writeConfig("baseline.json", config))
	for _, lane := range suiteLanes {
		require.NoError(t, runLane("baseline.json", "1m", "parts", "source-a", lane))
	}
	require.NoError(t, mergeSuite("baseline.json", "parts", "source-a", "cover.out"))
	for _, p := range append(owners, "raceonly") {
		require.FileExists(t, p+"/ran")
	}
	data, err := os.ReadFile("cover.out")
	require.NoError(t, err)
	require.Equal(t, 1, strings.Count(string(data), "mode: atomic"))
	blocks, err := parseProfile(strings.NewReader(string(data)))
	require.NoError(t, err)
	for _, p := range owners {
		measurement, err := measureBoundary(blocks, boundary{Packages: []string{"github.com/CaliLuke/loom/" + p}})
		require.NoError(t, err)
		require.Positive(t, measurement.CoveredStatements)
	}
	// Mutate completed artifacts rather than rerunning the six child suites
	// for each rejection case. Restore each input before the next assertion.
	marker := filepath.Join("parts", "coverage-http", "result.json")
	original, err := os.ReadFile(marker)
	require.NoError(t, err)
	for _, tc := range []struct {
		name string
		edit func(*laneResult)
	}{
		{"source", func(r *laneResult) {
			r.Source = "another-source"
		}},
		{"config", func(r *laneResult) {
			r.ConfigHash = "another-config"
		}},
		{"missing package", func(r *laneResult) {
			r.Packages = nil
		}},
		{"duplicate package", func(r *laneResult) {
			r.Packages = append(r.Packages, r.Packages...)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var result laneResult
			require.NoError(t, json.Unmarshal(original, &result))
			tc.edit(&result)
			data, err := json.Marshal(result)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(marker, data, 0o600))
			require.ErrorContains(t, mergeSuite("baseline.json", "parts", "source-a", "cover.out"), "mismatch")
			require.NoFileExists(t, "cover.out")
			require.NoError(t, os.WriteFile(marker, original, 0o600))
		})
	}
	require.NoError(t, os.Remove(marker))
	require.ErrorContains(t, mergeSuite("baseline.json", "parts", "source-a", "cover.out"), "incomplete lane")
	require.NoError(t, os.WriteFile(marker, original, 0o600))
	profile := filepath.Join("parts", "coverage-http", "selected.out")
	profileData, err := os.ReadFile(profile)
	require.NoError(t, err)
	for _, bad := range []string{"mode: count\n", "mode: atomic\ninvalid block\n"} {
		require.NoError(t, os.WriteFile(profile, []byte(bad), 0o600))
		require.Error(t, mergeSuite("baseline.json", "parts", "source-a", "cover.out"))
		require.NoFileExists(t, "cover.out")
	}
	require.NoError(t, os.WriteFile(profile, profileData, 0o600))
	// Raise the floor and align all manifests with that configuration. The
	// excluded consumer executed B, but selected coverage still reaches only A.
	config.Boundaries[0].MinimumBasisPoints = 10000
	require.NoError(t, writeConfig("baseline.json", config))
	configData, err := os.ReadFile("baseline.json")
	require.NoError(t, err)
	for _, lane := range suiteLanes {
		path := filepath.Join("parts", "coverage-"+lane, "result.json")
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		var result laneResult
		require.NoError(t, json.Unmarshal(data, &result))
		result.ConfigHash = fmt.Sprintf("%x", sha256.Sum256(configData))
		data, err = json.Marshal(result)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path, data, 0o600))
	}
	require.ErrorContains(t, mergeSuite("baseline.json", "parts", "source-a", "cover.out"), "below checked-in baseline")
	require.NoFileExists(t, "cover.out")
	// Existing markers make the second execution fail. Its completion marker
	// must be removed, preventing the previous success from masking this failure.
	require.Error(t, runLane("baseline.json", "1m", "parts", "source-a", "http"))
	require.NoFileExists(t, marker)
}

func TestSuiteTestArgs(t *testing.T) {
	args := suiteTestArgs(baselineConfig{TestPackages: []string{"./one", "./two"}}, "profile.out", "41m")
	for _, flag := range []string{"-race", "-shuffle=on", "-count=1", "-covermode=atomic", "-timeout=41m"} {
		require.Contains(t, args, flag)
	}
	require.Equal(t, []string{"./one", "./two"}, args[len(args)-2:])
	require.False(t, slices.Contains(args, "-covermode=count"))
}
