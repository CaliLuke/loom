package main

import (
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConformanceEventsRejectFalseSuccess(t *testing.T) {
	const eventRun = `{"Action":"run","Package":"github.com/CaliLuke/loom/internal/valuecontract","Test":"TestLeanConformance"}` + "\n"
	const eventPass = `{"Action":"pass","Package":"github.com/CaliLuke/loom/internal/valuecontract","Test":"TestLeanConformance"}` + "\n"
	const packagePass = `{"Action":"pass","Package":"github.com/CaliLuke/loom/internal/valuecontract"}` + "\n"
	for _, tc := range []struct {
		name   string
		events string
		valid  bool
	}{
		{name: "executed", events: eventRun + eventPass + packagePass, valid: true},
		{name: "empty"},
		{name: "package only", events: packagePass},
		{name: "missing run", events: eventPass + packagePass},
		{name: "no package completion", events: eventRun + eventPass},
		{name: "repeated invocation", events: eventRun + eventPass + eventRun + eventPass + packagePass},
		{name: "skipped", events: eventRun + strings.Replace(eventPass, `"pass"`, `"skip"`, 1) + packagePass},
		{name: "child skipped", events: eventRun + `{"Action":"skip","Package":"github.com/CaliLuke/loom/internal/valuecontract","Test":"TestLeanConformance/corpus"}` + "\n" + eventPass + packagePass},
		{name: "wrong package", events: strings.ReplaceAll(eventRun+eventPass+packagePass, "internal/valuecontract", "other")},
		{name: "malformed", events: "not JSON\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := checkEvents(strings.NewReader(tc.events))
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestConformanceReportRequiresExecutedOwners(t *testing.T) {
	valid := conformanceReport{Version: 1}
	for _, owner := range requiredOwners {
		valid.Cases = append(valid.Cases, conformanceCase{Name: owner + "/case", Owner: owner, Passed: true})
	}
	for _, tc := range []struct {
		name   string
		change func(*conformanceReport)
		valid  bool
	}{
		{name: "complete", valid: true},
		{name: "zero cases", change: func(r *conformanceReport) {
			r.Cases = nil
		}},
		{name: "missing projection", change: func(r *conformanceReport) {
			r.Cases = append(r.Cases[:2], r.Cases[3:]...)
		}},
		{name: "duplicate", change: func(r *conformanceReport) {
			r.Cases = append(r.Cases, r.Cases[0])
		}},
		{name: "failed", change: func(r *conformanceReport) {
			r.Cases[0].Passed = false
		}},
		{name: "unnamed", change: func(r *conformanceReport) {
			r.Cases[0].Name = ""
		}},
		{name: "wrong version", change: func(r *conformanceReport) {
			r.Version = 0
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			report := conformanceReport{Version: valid.Version, Cases: append([]conformanceCase(nil), valid.Cases...)}
			if tc.change != nil {
				tc.change(&report)
			}
			data, err := json.Marshal(report)
			require.NoError(t, err)
			count, err := checkReport(data)
			if tc.valid {
				require.NoError(t, err)
				require.Positive(t, count)
			} else {
				require.Error(t, err)
			}
		})
	}
}
