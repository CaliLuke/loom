package expr_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestHTTPStatusSelectorParse(t *testing.T) {
	for _, tc := range []struct {
		key   string
		want  expr.HTTPStatusSelector
		phase expr.HTTPResponsePhase
	}{
		{"100", expr.HTTPStatusSelector{Code: 100}, expr.PhaseInformational},
		{"101", expr.HTTPStatusSelector{Code: 101}, expr.PhaseUpgrade},
		{"199", expr.HTTPStatusSelector{Code: 199}, expr.PhaseInformational},
		{"200", expr.HTTPStatusSelector{Code: 200}, expr.PhaseFinal},
		{"404", expr.HTTPStatusSelector{Code: 404}, expr.PhaseFinal},
		{"599", expr.HTTPStatusSelector{Code: 599}, expr.PhaseFinal},
		{"1XX", expr.HTTPStatusSelector{Class: expr.Status1XX}, expr.PhaseInformational},
		{"2XX", expr.HTTPStatusSelector{Class: expr.Status2XX}, expr.PhaseFinal},
		{"3XX", expr.HTTPStatusSelector{Class: expr.Status3XX}, expr.PhaseFinal},
		{"4XX", expr.HTTPStatusSelector{Class: expr.Status4XX}, expr.PhaseFinal},
		{"5XX", expr.HTTPStatusSelector{Class: expr.Status5XX}, expr.PhaseFinal},
		{"default", expr.HTTPStatusSelector{Class: expr.StatusFallback}, expr.PhaseFinal},
	} {
		t.Run(tc.key, func(t *testing.T) {
			got, err := expr.ParseOpenAPIStatusKey(tc.key)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
			require.Equal(t, tc.key, got.OpenAPIKey())
			require.Equal(t, tc.phase, got.Phase())
			code, exact := got.Exact()
			require.Equal(t, tc.want.Code, code)
			require.Equal(t, tc.want.Code != 0, exact)
			require.Equal(t, exact, got.IsExact())
		})
	}
	for _, key := range []string{"2xx", "20X", "6XX", "0XX", "600", "099", "Default", "", "99", "0200", "+200", "200 ", " 200", "２00"} {
		t.Run(fmt.Sprintf("reject_%q", key), func(t *testing.T) {
			got, err := expr.ParseOpenAPIStatusKey(key)
			require.Error(t, err)
			require.Zero(t, got)
		})
	}
	for _, selector := range []expr.HTTPStatusSelector{{}, {Code: 99}, {Code: 600}, {Class: 6}, {Code: 200, Class: expr.Status2XX}} {
		t.Run(fmt.Sprintf("invalid_%v", selector), func(t *testing.T) {
			code, exact := selector.Exact()
			require.Zero(t, code)
			require.False(t, exact)
			require.False(t, selector.IsExact())
			require.Empty(t, selector.OpenAPIKey())
			require.Zero(t, selector.Phase())
		})
	}
}

func TestSelectFinalPrecedence(t *testing.T) {
	fallback := expr.HTTPStatusSelector{Class: expr.StatusFallback}
	okClass := expr.HTTPStatusSelector{Class: expr.Status2XX}
	errorClass := expr.HTTPStatusSelector{Class: expr.Status4XX}
	for _, tc := range []struct {
		name string
		keys []expr.HTTPStatusSelector
		code int
		want expr.HTTPStatusSelector
		ok   bool
	}{
		{"class_before_fallback", []expr.HTTPStatusSelector{fallback, okClass}, 200, okClass, true},
		{"exact_before_class", []expr.HTTPStatusSelector{okClass, {Code: 200}}, 200, expr.HTTPStatusSelector{Code: 200}, true},
		{"exact_error", []expr.HTTPStatusSelector{fallback, errorClass, {Code: 404}}, 404, expr.HTTPStatusSelector{Code: 404}, true},
		{"class_error", []expr.HTTPStatusSelector{{Code: 404}, fallback, errorClass}, 409, errorClass, true},
		{"fallback", []expr.HTTPStatusSelector{{Code: 200}, okClass, fallback}, 500, fallback, true},
		{"upper_boundary", []expr.HTTPStatusSelector{fallback}, 599, fallback, true},
		{"no_matching_key", []expr.HTTPStatusSelector{{Code: 200}, okClass}, 404, expr.HTTPStatusSelector{}, false},
		{"empty", nil, 200, expr.HTTPStatusSelector{}, false},
		{"informational", []expr.HTTPStatusSelector{{Code: 100}, {Class: expr.Status1XX}, fallback}, 100, expr.HTTPStatusSelector{}, false},
		{"upgrade", []expr.HTTPStatusSelector{{Code: 101}, fallback}, 101, expr.HTTPStatusSelector{}, false},
		{"last_informational", []expr.HTTPStatusSelector{{Code: 199}, fallback}, 199, expr.HTTPStatusSelector{}, false},
		{"out_of_range", []expr.HTTPStatusSelector{{Code: 600}, fallback}, 600, expr.HTTPStatusSelector{}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, keys := range statusSelectorOrders(tc.keys) {
				before := slices.Clone(keys)
				got, ok := expr.SelectFinal(keys, tc.code)
				require.Equal(t, tc.want, got)
				require.Equal(t, tc.ok, ok)
				require.Equal(t, before, keys)
			}
		})
	}
}

func TestSelectInformational(t *testing.T) {
	info := expr.HTTPStatusSelector{Class: expr.Status1XX}
	fallback := expr.HTTPStatusSelector{Class: expr.StatusFallback}
	for _, tc := range []struct {
		name string
		keys []expr.HTTPStatusSelector
		code int
		want expr.HTTPStatusSelector
		ok   bool
	}{
		{"automatic_exact", []expr.HTTPStatusSelector{info, {Code: 100}}, 100, expr.HTTPStatusSelector{Code: 100}, true},
		{"automatic_class", []expr.HTTPStatusSelector{info}, 100, info, true},
		{"exact", []expr.HTTPStatusSelector{info, {Code: 103}}, 103, expr.HTTPStatusSelector{Code: 103}, true},
		{"class", []expr.HTTPStatusSelector{info, {Code: 103}}, 102, info, true},
		{"last", []expr.HTTPStatusSelector{info}, 199, info, true},
		{"no_fallback", []expr.HTTPStatusSelector{fallback}, 103, expr.HTTPStatusSelector{}, false},
		{"upgrade", []expr.HTTPStatusSelector{info, {Code: 101}}, 101, expr.HTTPStatusSelector{}, false},
		{"final", []expr.HTTPStatusSelector{info, {Code: 200}}, 200, expr.HTTPStatusSelector{}, false},
		{"too_low", []expr.HTTPStatusSelector{info, {Code: 99}}, 99, expr.HTTPStatusSelector{}, false},
		{"no_match", []expr.HTTPStatusSelector{{Code: 103}}, 102, expr.HTTPStatusSelector{}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, keys := range statusSelectorOrders(tc.keys) {
				got, ok := expr.SelectInformational(keys, tc.code)
				require.Equal(t, tc.want, got)
				require.Equal(t, tc.ok, ok)
			}
		})
	}
}

func TestHTTPStatusRuntimeDomain(t *testing.T) {
	fallback := expr.HTTPStatusSelector{Class: expr.StatusFallback}
	okClass := expr.HTTPStatusSelector{Class: expr.Status2XX}
	redirect := expr.HTTPStatusSelector{Class: expr.Status3XX}
	for _, tc := range []struct {
		name     string
		keys     []expr.HTTPStatusSelector
		selector expr.HTTPStatusSelector
		body     bool
		want     []int
	}{
		{"exact", []expr.HTTPStatusSelector{{Code: 201}}, expr.HTTPStatusSelector{Code: 201}, false, []int{201}},
		{"class_minus_exact", []expr.HTTPStatusSelector{{Code: 200}, okClass}, okClass, false, statusCodeRange(201, 299)},
		{"class_body", []expr.HTTPStatusSelector{{Code: 200}, okClass}, okClass, true, statusCodeRange(201, 299, 204)},
		{"redirect_body", []expr.HTTPStatusSelector{redirect}, redirect, true, statusCodeRange(300, 399, 304)},
		{"fallback_minus_exact_and_class", []expr.HTTPStatusSelector{fallback, okClass, {Code: 404}}, fallback, false, statusCodeRange(300, 599, 404)},
		{"fallback_body", []expr.HTTPStatusSelector{fallback}, fallback, true, statusCodeRange(200, 599, 204, 304)},
		{"exact_bodiless", []expr.HTTPStatusSelector{{Code: 204}}, expr.HTTPStatusSelector{Code: 204}, false, []int{204}},
		{"exact_forbidden_body", []expr.HTTPStatusSelector{{Code: 204}}, expr.HTTPStatusSelector{Code: 204}, true, nil},
		{"exact_not_modified_body", []expr.HTTPStatusSelector{{Code: 304}}, expr.HTTPStatusSelector{Code: 304}, true, nil},
		{"undeclared", []expr.HTTPStatusSelector{fallback}, okClass, false, nil},
		{"informational", []expr.HTTPStatusSelector{{Code: 103}}, expr.HTTPStatusSelector{Code: 103}, false, nil},
		{"invalid", nil, expr.HTTPStatusSelector{}, false, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, keys := range statusSelectorOrders(tc.keys) {
				got := expr.FinalRuntimeDomain(keys, tc.selector, tc.body)
				require.Equal(t, tc.want, got)
				if len(got) > 0 {
					got[0] = -1
					require.Equal(t, tc.want, expr.FinalRuntimeDomain(keys, tc.selector, tc.body))
				}
			}
		})
	}
	info := expr.HTTPStatusSelector{Class: expr.Status1XX}
	for _, tc := range []struct {
		name     string
		keys     []expr.HTTPStatusSelector
		selector expr.HTTPStatusSelector
		want     []int
	}{
		{"emit_exact", []expr.HTTPStatusSelector{{Code: 103}}, expr.HTTPStatusSelector{Code: 103}, []int{103}},
		{"emit_class", []expr.HTTPStatusSelector{info, {Code: 103}, {Code: 199}}, info, statusCodeRange(102, 198, 103)},
		{"automatic_not_authored", []expr.HTTPStatusSelector{{Code: 100}}, expr.HTTPStatusSelector{Code: 100}, nil},
		{"upgrade_not_authored", []expr.HTTPStatusSelector{{Code: 101}}, expr.HTTPStatusSelector{Code: 101}, nil},
		{"fallback_not_interim", []expr.HTTPStatusSelector{fallback}, fallback, nil},
		{"emit_undeclared", []expr.HTTPStatusSelector{info}, expr.HTTPStatusSelector{Code: 103}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, keys := range statusSelectorOrders(tc.keys) {
				require.Equal(t, tc.want, expr.InformationalEmissionDomain(keys, tc.selector))
			}
		})
	}
}

func statusCodeRange(first, last int, excluded ...int) []int {
	var codes []int
	for code := first; code <= last; code++ {
		if !slices.Contains(excluded, code) {
			codes = append(codes, code)
		}
	}
	return codes
}

func statusSelectorOrders(keys []expr.HTTPStatusSelector) [][]expr.HTTPStatusSelector {
	orders := make([][]expr.HTTPStatusSelector, 0, 1+2*len(keys))
	orders = append(orders, slices.Clone(keys))
	for offset := range len(keys) {
		rotated := append(slices.Clone(keys[offset:]), keys[:offset]...)
		orders = append(orders, slices.Clone(rotated))
		slices.Reverse(rotated)
		orders = append(orders, rotated)
	}
	return orders
}

func TestHTTPStatusSelectorNameSuffix(t *testing.T) {
	for _, tc := range []struct {
		selector expr.HTTPStatusSelector
		want     string
	}{
		{expr.HTTPStatusSelector{Code: 200}, "OK"},
		{expr.HTTPStatusSelector{Code: 201}, "Created"},
		{expr.HTTPStatusSelector{Code: 404}, "Not Found"},
		{expr.HTTPStatusSelector{Code: 418}, "I'm a teapot"},
		{expr.HTTPStatusSelector{Code: 299}, "Status299"},
		{expr.HTTPStatusSelector{Class: expr.Status1XX}, "Status1XX"},
		{expr.HTTPStatusSelector{Class: expr.Status2XX}, "Status2XX"},
		{expr.HTTPStatusSelector{Class: expr.Status3XX}, "Status3XX"},
		{expr.HTTPStatusSelector{Class: expr.Status4XX}, "Status4XX"},
		{expr.HTTPStatusSelector{Class: expr.Status5XX}, "Status5XX"},
		{expr.HTTPStatusSelector{Class: expr.StatusFallback}, "StatusDefault"},
		{expr.HTTPStatusSelector{}, ""},
		{expr.HTTPStatusSelector{Code: 99}, ""},
		{expr.HTTPStatusSelector{Code: 200, Class: expr.Status2XX}, ""},
	} {
		t.Run(fmt.Sprintf("%v", tc.selector), func(t *testing.T) {
			require.Equal(t, tc.want, tc.selector.NameSuffix())
		})
	}
}
