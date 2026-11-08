package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestHTTPMetadataSemanticCapture(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		hints := dsl.Type("Hints", func() {
			dsl.Attribute("link", dsl.String)
		})
		dsl.Service("values", func() {
			dsl.Method("read", func() {
				dsl.HTTP(func() {
					dsl.GET("/")
					dsl.Informational("early", dsl.StatusEarlyHints, hints, func() {
						dsl.Header("link:Link")
					})
					dsl.Informational("progress", 102, hints, func() {
						dsl.Header("link:Link")
					})
				})
			})
		})
	})
	method := root.Services[0].Methods[0]
	infos := root.API.HTTP.Services[0].HTTPEndpoints[0].Informational
	for i, info := range infos {
		info.Value.UserExamples = []*expr.ExampleExpr{{Value: map[string]any{"link": []string{"early", "progress"}[i]}}}
	}
	owner := NewSemanticData(root)
	captured := owner.Method(method)
	require.Len(t, captured.HTTPMetadataValues, 2)
	for _, info := range infos {
		value := captured.HTTPMetadataValues[info.HTTPResponseExpr]
		require.NotNil(t, value)
		require.Zero(t, value.Example.Outcome(), "semantic capture must not select examples")
	}
	require.NotEqual(t, captured.HTTPMetadataValues[infos[0].HTTPResponseExpr].Occurrence.ID(), captured.HTTPMetadataValues[infos[1].HTTPResponseExpr].Occurrence.ID())
	infos[0].Value.UserExamples[0].Value = map[string]any{"link": "mutated"}
	backend := owner.Services()
	selected := backend.Get("values").Method("read")
	for i, info := range infos {
		value := selected.HTTPMetadataValues[info.HTTPResponseExpr]
		require.NotNil(t, value)
		actual, ok := value.Example.LegacyValue()
		require.True(t, ok)
		require.Equal(t, map[string]any{"link": []string{"early", "progress"}[i]}, actual)
		require.Equal(t, captured.HTTPMetadataValues[info.HTTPResponseExpr].Occurrence.ID(), value.Occurrence.ID())
	}
	require.Same(t, selected.HTTPMetadataValues[infos[0].HTTPResponseExpr], backend.Get("values").Method("read").HTTPMetadataValues[infos[0].HTTPResponseExpr])
	delete(captured.HTTPMetadataValues, infos[0].HTTPResponseExpr)
	require.Len(t, owner.Method(method).HTTPMetadataValues, 2)
}
