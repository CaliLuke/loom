package vet

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestAnalyzeRetryAfterResponseGuidance(t *testing.T) {
	for _, test := range []struct {
		name      string
		status    int
		header    string
		scope     string
		required  bool
		wantRetry bool
	}{
		{name: "optional mapped delay", status: 429, header: "delay:Retry-After"},
		{name: "lowercase wire name", status: 502, header: "delay:retry-after"},
		{name: "uppercase wire name", status: 503, header: "delay:RETRY-AFTER"},
		{name: "unmapped wire name", status: 429, header: "Retry-After"},
		{name: "required delay", status: 429, header: "delay:Retry-After", required: true},
		{name: "service inheritance", status: 429, header: "delay:Retry-After", scope: "service"},
		{name: "API inheritance", status: 503, header: "delay:Retry-After", scope: "api"},
		{name: "body field only", status: 429, wantRetry: true},
		{name: "unrelated header", status: 429, header: "delay:X-Retry-After", wantRetry: true},
		{
			name:      "source name is not wire name",
			status:    429,
			header:    "Retry-After:X-Wait",
			wantRetry: true,
		},
		{
			name:      "request header",
			status:    429,
			header:    "delay:Retry-After",
			scope:     "request",
			wantRetry: true,
		},
		{
			name:      "success header",
			status:    429,
			header:    "delay:Retry-After",
			scope:     "success",
			wantRetry: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				field := expr.AttributeName(test.header)
				if field == "" {
					field = "retry_after"
				}
				denied := dsl.Type("Denied", func() {
					dsl.Attribute("message", dsl.String)
					dsl.Attribute(field, dsl.Int64)
				})
				mapping := func() {
					dsl.Response("limited", test.status, func() {
						if test.header != "" && test.scope != "request" && test.scope != "success" {
							dsl.Headers(func() {
								dsl.Header(test.header)
								if test.required {
									dsl.Required("delay")
								}
							})
						}
					})
				}
				dsl.API("retry", func() {
					if test.scope == "api" {
						dsl.Error("limited", denied)
						dsl.HTTP(mapping)
					}
				})
				dsl.Service("work", func() {
					dsl.Error("limited", denied)
					if test.scope == "service" {
						dsl.HTTP(mapping)
					}
					dsl.Method("run", func() {
						dsl.Payload(denied)
						dsl.Result(denied)
						dsl.HTTP(func() {
							dsl.POST("/work")
							if test.scope == "request" {
								dsl.Header(test.header)
							}
							dsl.Response(dsl.StatusOK, func() {
								if test.scope == "success" {
									dsl.Header(test.header)
								}
							})
							if test.scope != "api" && test.scope != "service" {
								mapping()
							}
						})
					})
				})
			})

			var report Report
			analyzeHTTPErrorSemantics(root, &report)

			if test.wantRetry {
				require.Equal(t, []string{RuleRetryableError + ":service.work.error.limited"}, diagnosticKeys(report.Diagnostics))
			} else {
				require.Empty(t, report.Diagnostics)
			}
			require.False(t, hasMeta(root.Services[0].Errors[0].Meta, "loom:error:temporary"))
		})
	}
}

func TestAnalyzeRetryAfterGuidanceIsResponseLocal(t *testing.T) {
	shared := httpError("limited", 429, nil, nil)
	withHeader := shared.Dup()
	withHeader.Response.Headers = expr.NewMappedAttributeExpr(&expr.AttributeExpr{Type: &expr.Object{
		{Name: "delay:Retry-After", Attribute: &expr.AttributeExpr{Type: expr.Int64}},
	}})
	for _, response := range []*expr.HTTPErrorExpr{withHeader, shared, withHeader} {
		var rules []string
		analyzeHTTPErrorMapping(response, func(_ *expr.HTTPErrorExpr, _ int, rule, _ string) {
			rules = append(rules, rule)
		})
		if response == shared {
			require.Equal(t, []string{RuleRetryableError}, rules)
		} else {
			require.Empty(t, rules)
		}
	}
}
