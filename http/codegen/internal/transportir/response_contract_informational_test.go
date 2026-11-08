package transportir_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	testcodegen "github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

func TestAnalyzeResponseContractCasesIncludesInformational(t *testing.T) {
	root := testcodegen.RunDSL(t, informationalContractDSL)
	endpoint := transportir.BuildEndpoint(root.API.HTTP.Services[0].HTTPEndpoints[0])

	analysis := transportir.AnalyzeResponseContractCases(endpoint)
	require.True(t, analysis.Supported())
	require.Len(t, analysis.Cases, 3)
	require.Equal(t, transportir.ResponseContractSuccess, analysis.Cases[0].Kind)

	hints, progress := analysis.Cases[1], analysis.Cases[2]
	require.Equal(t, "widgets.list.informational.103", hints.ID)
	require.Equal(t, transportir.ResponseContractInformational, hints.Kind)
	require.Equal(t, transportir.ResponseContractHTTP, hints.Transport)
	require.Equal(t, expr.HTTPStatusSelector{Code: dsl.StatusEarlyHints}, hints.Selector)
	require.Equal(t, []int{dsl.StatusEarlyHints}, hints.Domain)
	require.False(t, hints.HasBody)
	require.Empty(t, hints.ContentTypes)
	require.Equal(t, []transportir.ResponseContractHeader{{Name: "link", HTTPName: "Link", Required: true}}, hints.Headers)

	require.Equal(t, "widgets.list.informational.1XX", progress.ID)
	require.Equal(t, expr.HTTPStatusSelector{Class: expr.Status1XX}, progress.Selector)
	require.Equal(t, endpoint.Response.Informational[1].Response.Domain, progress.Domain)
	require.NotContains(t, progress.Domain, dsl.StatusEarlyHints, "the exact key shadows the class")
	require.False(t, progress.HasBody)

	progress.Domain[0] = 0
	require.NotEqual(t, 0, endpoint.Response.Informational[1].Response.Domain[0], "case domains are detached")
}

func informationalContractDSL() {
	hints := dsl.Type("Hints", func() {
		dsl.Attribute("link", dsl.String)
		dsl.Required("link")
	})
	progress := dsl.Type("Progress", func() {
		dsl.Attribute("code", dsl.Int)
		dsl.Required("code")
	})
	dsl.Service("widgets", func() {
		dsl.Method("list", func() {
			dsl.Result(dsl.String)
			dsl.HTTP(func() {
				dsl.GET("/widgets")
				dsl.Informational("hints", dsl.StatusEarlyHints, hints, func() {
					dsl.Header("link:Link")
				})
				dsl.Informational("progress", dsl.Status1XX, progress, func() {
					dsl.StatusField("code")
				})
				dsl.Response(dsl.StatusOK)
			})
		})
	})
}
