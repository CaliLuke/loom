package http_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

func TestStatusSelectorMatchesExpr(t *testing.T) {
	t.Run("selection", testStatusSelectorSelection)
	t.Run("domain", testStatusSelectorDomain)
	t.Run("unnamed-violation", testStatusSelectorViolation)
}

func testStatusSelectorSelection(t *testing.T) {
	for _, tc := range []struct {
		name          string
		keys          []loomhttp.StatusSelector
		code          int
		final         loomhttp.StatusSelector
		finalOK       bool
		informational loomhttp.StatusSelector
		infoOK        bool
	}{
		{
			name: "exact-before-class-and-fallback", code: 404,
			keys:  []loomhttp.StatusSelector{{Class: -1}, {Class: 4}, {Code: 404}},
			final: loomhttp.StatusSelector{Code: 404}, finalOK: true,
		},
		{
			name: "class-before-fallback", code: 200,
			keys:  []loomhttp.StatusSelector{{Class: -1}, {Class: 2}},
			final: loomhttp.StatusSelector{Class: 2}, finalOK: true,
		},
		{
			name: "fallback-outside-covered-class", code: 599,
			keys:  []loomhttp.StatusSelector{{Code: 200}, {Class: 2}, {Class: -1}},
			final: loomhttp.StatusSelector{Class: -1}, finalOK: true,
		},
		{
			name: "uncovered-final", code: 200,
			keys: []loomhttp.StatusSelector{{Class: 4}},
		},
		{
			name: "observed-continue", code: 100,
			keys:          []loomhttp.StatusSelector{{Class: -1}, {Class: 1}, {Code: 100}},
			informational: loomhttp.StatusSelector{Code: 100}, infoOK: true,
		},
		{
			name: "informational-class", code: 199,
			keys:          []loomhttp.StatusSelector{{Class: -1}, {Class: 1}, {Code: 103}},
			informational: loomhttp.StatusSelector{Class: 1}, infoOK: true,
		},
		{
			name: "informational-exact", code: 103,
			keys:          []loomhttp.StatusSelector{{Class: 1}, {Code: 103}},
			informational: loomhttp.StatusSelector{Code: 103}, infoOK: true,
		},
		{
			name: "upgrade-is-neither-phase", code: 101,
			keys: []loomhttp.StatusSelector{{Code: 101}, {Class: 1}, {Class: -1}},
		},
		{
			name: "fallback-is-not-informational", code: 199,
			keys: []loomhttp.StatusSelector{{Class: -1}},
		},
		{
			name: "below-http-range", code: 99,
			keys: []loomhttp.StatusSelector{{Class: 1}, {Class: -1}},
		},
		{
			name: "above-http-range", code: 600,
			keys: []loomhttp.StatusSelector{{Class: 5}, {Class: -1}},
		},
		{name: "undeclared", code: 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			keys := slices.Clone(tc.keys)
			for range 2 {
				exprKeys := statusSelectorsForExpr(keys)
				final, finalOK := loomhttp.SelectFinal(keys, tc.code)
				require.Equal(t, tc.finalOK, finalOK)
				require.Equal(t, tc.final, final)
				exprFinal, exprFinalOK := expr.SelectFinal(exprKeys, tc.code)
				require.Equal(t, exprFinalOK, finalOK)
				require.Equal(t, expr.HTTPStatusSelector{Code: final.Code, Class: expr.HTTPStatusClass(final.Class)}, exprFinal)

				info, infoOK := loomhttp.SelectInformational(keys, tc.code)
				require.Equal(t, tc.infoOK, infoOK)
				require.Equal(t, tc.informational, info)
				exprInfo, exprInfoOK := expr.SelectInformational(exprKeys, tc.code)
				require.Equal(t, exprInfoOK, infoOK)
				require.Equal(t, expr.HTTPStatusSelector{Code: info.Code, Class: expr.HTTPStatusClass(info.Class)}, exprInfo)
				slices.Reverse(keys)
			}
			require.Equal(t, tc.keys, keys, "selection must not mutate declaration order")
		})
	}
}

func testStatusSelectorDomain(t *testing.T) {
	for _, tc := range []struct {
		name     string
		keys     []loomhttp.StatusSelector
		selector loomhttp.StatusSelector
		hasBody  bool
		want     []int
	}{
		{
			name: "exact", keys: []loomhttp.StatusSelector{{Class: 2}, {Code: 200}},
			selector: loomhttp.StatusSelector{Code: 200}, hasBody: true, want: []int{200},
		},
		{
			name: "class-minus-exact", keys: []loomhttp.StatusSelector{{Class: 2}, {Code: 200}},
			selector: loomhttp.StatusSelector{Class: 2}, want: selectorTestCodes(201, 299),
		},
		{
			name: "body-class-minus-exact-and-no-content", keys: []loomhttp.StatusSelector{{Class: 2}, {Code: 200}},
			selector: loomhttp.StatusSelector{Class: 2}, hasBody: true, want: selectorTestCodes(201, 299, 204),
		},
		{
			name: "body-class-minus-not-modified", keys: []loomhttp.StatusSelector{{Class: 3}},
			selector: loomhttp.StatusSelector{Class: 3}, hasBody: true, want: selectorTestCodes(300, 399, 304),
		},
		{
			name:     "fallback-minus-exact-classes-and-bodiless",
			keys:     []loomhttp.StatusSelector{{Class: -1}, {Class: 2}, {Class: 4}, {Code: 301}},
			selector: loomhttp.StatusSelector{Class: -1}, hasBody: true,
			want: append(selectorTestCodes(300, 399, 301, 304), selectorTestCodes(500, 599)...),
		},
		{
			name: "exact-no-content-without-body", keys: []loomhttp.StatusSelector{{Code: 204}},
			selector: loomhttp.StatusSelector{Code: 204}, want: []int{204},
		},
		{
			name: "exact-no-content-with-body", keys: []loomhttp.StatusSelector{{Code: 204}},
			selector: loomhttp.StatusSelector{Code: 204}, hasBody: true,
		},
		{
			name: "exact-not-modified-without-body", keys: []loomhttp.StatusSelector{{Code: 304}},
			selector: loomhttp.StatusSelector{Code: 304}, want: []int{304},
		},
		{
			name: "exact-not-modified-with-body", keys: []loomhttp.StatusSelector{{Code: 304}},
			selector: loomhttp.StatusSelector{Code: 304}, hasBody: true,
		},
		{
			name: "informational-is-not-final", keys: []loomhttp.StatusSelector{{Class: 1}},
			selector: loomhttp.StatusSelector{Class: 1},
		},
		{
			name: "selector-must-be-declared", keys: []loomhttp.StatusSelector{{Class: -1}},
			selector: loomhttp.StatusSelector{Code: 404},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := slices.Clone(tc.keys)
			got := loomhttp.FinalStatusDomain(tc.keys, tc.selector, tc.hasBody)
			require.True(t, slices.Equal(tc.want, got), "domain: got %v, want %v", got, tc.want)
			exprSelector := expr.HTTPStatusSelector{Code: tc.selector.Code, Class: expr.HTTPStatusClass(tc.selector.Class)}
			exprDomain := expr.FinalRuntimeDomain(statusSelectorsForExpr(tc.keys), exprSelector, tc.hasBody)
			require.True(t, slices.Equal(exprDomain, got), "expr/runtime domains differ")
			for code := 99; code <= 600; code++ {
				err := loomhttp.CheckFinalStatus(tc.keys, tc.selector, code, tc.hasBody)
				if slices.Contains(tc.want, code) {
					require.NoError(t, err, "status %d", code)
				} else {
					var violation *loomhttp.StatusViolationError
					require.ErrorAs(t, err, &violation, "status %d", code)
					require.Equal(t, code, violation.Code)
					require.Equal(t, tc.selector, violation.Selector)
				}
			}
			require.Equal(t, before, tc.keys)
		})
	}
}

func testStatusSelectorViolation(t *testing.T) {
	selector := loomhttp.StatusSelector{Code: 200}
	err := loomhttp.CheckFinalStatus([]loomhttp.StatusSelector{selector}, selector, 404, false)
	var violation *loomhttp.StatusViolationError
	require.ErrorAs(t, fmt.Errorf("encode response: %w", err), &violation)
	require.Equal(t, 404, violation.Code)
	require.Equal(t, selector, violation.Selector)
	require.Contains(t, err.Error(), "404")
	require.Empty(t, loom.ResolveErrorResponse(err).Name, "framework violation must not reselect a designed error")
}

func statusSelectorsForExpr(keys []loomhttp.StatusSelector) []expr.HTTPStatusSelector {
	converted := make([]expr.HTTPStatusSelector, len(keys))
	for i, key := range keys {
		converted[i] = expr.HTTPStatusSelector{Code: key.Code, Class: expr.HTTPStatusClass(key.Class)}
	}
	return converted
}

func selectorTestCodes(first, last int, excluded ...int) []int {
	codes := make([]int, 0, last-first+1)
	for code := first; code <= last; code++ {
		if !slices.Contains(excluded, code) {
			codes = append(codes, code)
		}
	}
	return codes
}
