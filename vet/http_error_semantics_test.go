package vet

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestAnalyzeDocumentedConcurrencyOverload(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.API("admission_repro", func() {})
		dsl.Service("work", func() {
			dsl.Method("run", func() {
				dsl.Error("overloaded", func() {
					dsl.Temporary()
				})
				dsl.HTTP(func() {
					dsl.POST("/work")
					dsl.Response(dsl.StatusNoContent)
					dsl.Response("overloaded", dsl.StatusServiceUnavailable)
				})
			})
		})
	})

	var report Report
	analyzeHTTPErrorSemantics(root, &report)

	require.Empty(t, report.Diagnostics)
}

func TestAnalyzeHTTPErrorTraitsIndependentOfStatus(t *testing.T) {
	for _, test := range []struct {
		name      string
		status    int
		meta      expr.MetaExpr
		remedy    *expr.ErrorRemedyExpr
		wantRetry bool
	}{
		{name: "unclassified internal error", status: 500},
		{name: "unclassified not implemented", status: 501},
		{name: "unclassified version unsupported", status: 505},
		{name: "unclassified custom server error", status: 599},
		{name: "declared fault", status: 500, meta: expr.MetaExpr{"loom:error:fault": nil}},
		{name: "temporary internal error", status: 500, meta: expr.MetaExpr{"loom:error:temporary": nil}},
		{name: "gateway timeout", status: 504, meta: expr.MetaExpr{"loom:error:timeout": nil}},
		{name: "temporary overload", status: 503, meta: expr.MetaExpr{"loom:error:temporary": nil}},
		{name: "temporary bad gateway", status: 502, meta: expr.MetaExpr{"loom:error:temporary": nil}},
		{
			name:   "overload retry guidance",
			status: 503,
			remedy: &expr.ErrorRemedyExpr{RetryHint: "Retry after capacity recovers."},
		},
		{
			name:   "quota retry guidance",
			status: 429,
			remedy: &expr.ErrorRemedyExpr{RetryHint: "Retry after quota resets."},
		},
		{
			name:      "fault does not imply retry",
			status:    503,
			meta:      expr.MetaExpr{"loom:error:fault": nil},
			wantRetry: true,
		},
		{
			name:      "timeout does not imply retry",
			status:    503,
			meta:      expr.MetaExpr{"loom:error:timeout": nil},
			wantRetry: true,
		},
		{name: "unclassified overload", status: 503, wantRetry: true},
		{name: "unclassified bad gateway", status: 502, wantRetry: true},
		{name: "unclassified quota", status: 429, wantRetry: true},
		{name: "empty retry guidance", status: 503, remedy: &expr.ErrorRemedyExpr{}, wantRetry: true},
		{
			name:      "blank retry guidance",
			status:    503,
			remedy:    &expr.ErrorRemedyExpr{RetryHint: " \n\t"},
			wantRetry: true,
		},
		{
			name:   "suppressed retry guidance",
			status: 503,
			meta:   expr.MetaExpr{SuppressionMeta: {RuleRetryableError}},
		},
		{name: "ordinary client error", status: 400},
	} {
		t.Run(test.name, func(t *testing.T) {
			var rules []string
			analyzeHTTPErrorMapping(httpError("failure", test.status, test.meta, test.remedy),
				func(_ *expr.HTTPErrorExpr, _ int, rule, _ string) {
					rules = append(rules, rule)
				})
			if test.wantRetry {
				require.Equal(t, []string{RuleRetryableError}, rules)
			} else {
				require.Empty(t, rules)
			}
		})
	}
}
