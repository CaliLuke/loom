package ir

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	commontestdata "github.com/CaliLuke/loom/codegen/testdata"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestComponentAnnotationAuthoredByteAliasBindings(t *testing.T) {
	// These scalar annotations were independently captured from the parent
	// public renderer with the same DSL and both component-analysis orders.
	for _, test := range []struct {
		name           string
		neutral        bool
		entries, items int
	}{
		{"neutral-first", true, 5641850088042733796, 5298921495019751303},
		{"planned-first", false, 5641850088042733796, 5641850088042733796},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := codegen.RunDSL(t, commontestdata.DeclarationAuthorityBytesDSL)
			calls := 0
			analyzer := NewAnalyzer(root.API.ExampleGenerator, false, WithExampleValue(func(attribute *expr.AttributeExpr, raw any) (any, bool) {
				calls++
				return OpenAPIExampleValue(attribute, raw)
			}))
			if test.neutral {
				analyzeComponentTypes(analyzer, root.Types, root.ResultTypes)
			}
			prepared, err := representation.PrepareService(root.API.HTTP.Services[0], nil)
			require.NoError(t, err)
			for _, endpoint := range prepared.Endpoints {
				analyzeEndpointBodies(analyzer, endpoint)
			}
			for _, endpoint := range prepared.Endpoints {
				analyzeAsyncSchemas(analyzer, endpoint)
			}
			firstCalls := calls
			for _, endpoint := range prepared.Endpoints {
				analyzeEndpointBodies(analyzer, endpoint)
			}
			for _, endpoint := range prepared.Endpoints {
				analyzeAsyncSchemas(analyzer, endpoint)
			}
			require.Equal(t, firstCalls, calls, "revisiting the same roots must not sample a second time")
			analyzer.finalizeRepresentations()
			envelope := annotationObject(t, analyzer.schemas["AuthorityEnvelope"], analyzer.schemas)
			entries := annotationObject(t, envelope.Properties["entries"].AdditionalProperties.Schema, analyzer.schemas)
			items := annotationObject(t, envelope.Properties["items"].Items, analyzer.schemas)
			require.Equal(t, test.entries, entries.Properties["count"].Example, "map alias preserves its original annotation binding")
			require.Equal(t, test.items, items.Properties["count"].Example, "array alias preserves its original annotation binding")
			for _, schema := range []*Schema{entries, items} {
				require.Equal(t, "base64", schema.Properties["blob"].ContentEncoding)
				require.NotNil(t, schema.Properties["blob"].Not)
			}
		})
	}
}

func annotationObject(t *testing.T, schema *Schema, components map[string]*Schema) *Schema {
	t.Helper()
	seen := make(map[*Schema]bool)
	for schema != nil && !seen[schema] {
		seen[schema] = true
		if len(schema.Properties) > 0 {
			return schema
		}
		if schema.Ref != "" {
			name, ok := schemaComponentName(schema.Ref)
			require.True(t, ok)
			schema = components[name]
			continue
		}
		if len(schema.AllOf) > 0 {
			schema = schema.AllOf[0]
			continue
		}
		break
	}
	require.FailNow(t, "schema has no owned object shape")
	return nil
}

func TestExplicitBodyKeepsTargetDeclarationAuthority(t *testing.T) {
	root := codegen.RunDSL(t, testdata.OpenAPIExplicitBodyWrapperExamplesDSL)
	body := root.API.HTTP.Services[0].HTTPEndpoints[0].Body
	context := expr.NewValueContext()
	occurrence, err := context.NewOccurrence(body)
	require.NoError(t, err)
	plan, err := context.NewValuePlan(occurrence, expr.ValuePlanRequest{Target: body, Codec: expr.ValueCodecJSON, Use: expr.ValuePlanSchema})
	require.NoError(t, err)
	require.Equal(t, "SearchFiltersRequestBody", plan.Root().TargetDeclarationID(), "an authored target declaration is distinct from its semantic payload mapping")
}
