package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPartitionPackages(t *testing.T) {
	for _, tc := range []struct {
		name      string
		selected  []string
		remaining []string
		wantError string
	}{
		{name: "partial", selected: []string{"b"}, remaining: []string{"a", "c"}},
		{name: "complete", selected: []string{"a", "b", "c"}},
		{name: "overlapping selectors", selected: []string{"b", "b"}, remaining: []string{"a", "c"}},
		{name: "outside root", selected: []string{"outside"}, wantError: "outside the root suite"},
		{name: "empty", wantError: "no coverage test packages"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			remaining, err := remainingPackages([]string{"a", "b", "c"}, tc.selected)
			if tc.wantError != "" {
				require.ErrorContains(t, err, tc.wantError)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.remaining, remaining)
		})
	}
}

// Exercise real Go discovery, test execution and coverage with a tiny independent
// module. The exclusive marker fails if either test package runs a second time.
func TestSuitePreservesTestsAndConsumerCoverage(t *testing.T) {
	for _, tc := range []struct {
		name      string
		floor     int
		fail      string
		wantError string
	}{
		{name: "success", floor: 5000},
		{name: "remaining consumer cannot inflate baseline", floor: 10000, wantError: "below checked-in baseline"},
		{name: "selected failure", floor: 5000, fail: "consumer", wantError: "collect consumer-aware coverage"},
		{name: "remaining failure", floor: 5000, fail: "other", wantError: "collect remaining coverage"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			t.Setenv("GOWORK", "off")
			t.Setenv("LOOM_SUITE_FAIL", tc.fail)
			writeSuiteFile(t, "go.mod", "module example.com/suite\n\ngo 1.27.0\n")
			writeSuiteFile(t, "service/service.go", "package service\nfunc A() int { return 1 }\nfunc B() int { return 2 }\n")
			writeSuiteFile(t, "consumer/consumer.go", "package consumer\nfunc Value() int { return 3 }\n")
			writeSuiteFile(t, "other/other.go", "package other\nfunc Value() int { return 4 }\n")
			for _, p := range []struct{ name, call string }{{"consumer", "A"}, {"other", "B"}} {
				writeSuiteFile(t, p.name+"/once_test.go", strings.ReplaceAll(strings.ReplaceAll(`package PKG
import (
 "os"
 "testing"
 "example.com/suite/service"
)
func TestOnce(t *testing.T) {
 f, err := os.OpenFile("ran", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
 if err != nil { t.Fatal(err) }
 if err := f.Close(); err != nil { t.Fatal(err) }
 if os.Getenv("LOOM_SUITE_FAIL") == "PKG" { t.Fatal("requested failure") }
 if service.CALL() == 0 || Value() == 0 { t.Fatal("missing value") }
}
`, "PKG", p.name), "CALL", p.call))
			}
			config := baselineConfig{
				CoverPackages: []string{"example.com/suite/service"},
				TestPackages:  []string{"./service", "./consumer/...", "./consumer"},
				Boundaries: []boundary{{
					Name: "service", Packages: []string{"example.com/suite/service"},
					MinimumBasisPoints: tc.floor, Rationale: "Consumer coverage must remain scoped.",
				}},
			}
			require.NoError(t, writeConfig("baseline.json", config))
			writeSuiteFile(t, "cover.out", "stale profile")
			err := runSuite("baseline.json", "1m", "cover.out")
			if tc.wantError != "" {
				require.ErrorContains(t, err, tc.wantError)
				require.NoFileExists(t, "cover.out", "a failed suite must not publish stale or partial coverage")
				return
			}
			require.NoError(t, err)
			require.FileExists(t, "consumer/ran")
			require.FileExists(t, "other/ran")
			data, err := os.ReadFile("cover.out")
			require.NoError(t, err)
			require.Equal(t, 1, strings.Count(string(data), "mode: count"))
			blocks, err := parseProfile(strings.NewReader(string(data)))
			require.NoError(t, err)
			for _, p := range []string{"service", "consumer", "other"} {
				measured, err := measureBoundary(blocks, boundary{Name: p, Packages: []string{"example.com/suite/" + p}})
				require.NoError(t, err)
				require.Positive(t, measured.CoveredStatements)
			}
		})
	}
}

func writeSuiteFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}
