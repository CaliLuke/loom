package ir

import (
	"testing"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/stretchr/testify/require"
)

func TestAsyncRetainedCutKeepsOrdinaryViewPolicy(t *testing.T) {
	root := codegen.RunDSL(t, testdata.StreamingPayloadResultCollectionWithExplicitViewDSL)
	attribute := root.Services[0].Methods[0].Result
	endpoint := inlineAsyncTestEndpoint(t, attribute)
	target := representation.PrepareStreamSchema(endpoint, attribute, false)
	for _, retained := range []bool{false, true} {
		name := "ordinary"
		if retained {
			name = "retained"
		}
		t.Run(name, func(t *testing.T) {
			analyzer := NewAnalyzer(nil, false)
			var prepared *asyncSchema
			if retained {
				// A named sampler position is a retained cut, unlike the inline
				// sampler produced by asyncSamplerAttribute.
				prepared = analyzer.acquireAsyncBaseline(attribute, representationRoot(attribute, target), expr.DupAtt(attribute), "view-cut")
			} else {
				prepared = &asyncSchema{schema: analyzer.analyzeSchema(attribute, "view-cut", true)}
			}
			analyzer.finalizeRepresentations()
			result := materializeAsyncSchema(prepared, analyzer.schemas).schema
			if result.Ref != "" {
				component, local := schemaComponentName(result.Ref)
				require.True(t, local)
				result = analyzer.schemas[component]
			}
			require.NotNil(t, result.Items)
			item := result.Items
			if item.Ref != "" {
				component, local := schemaComponentName(item.Ref)
				require.True(t, local)
				item = analyzer.schemas[component]
			}
			require.Len(t, item.Properties, 1)
			require.Contains(t, item.Properties, "a", "retained and ordinary owners still select the tiny view")
		})
	}
}
