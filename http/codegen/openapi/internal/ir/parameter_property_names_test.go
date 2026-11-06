package ir

import (
	"encoding/json/v2"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestParameterPropertyNames(t *testing.T) {
	var first *Document
	for _, reverse := range []bool{false, true} {
		root := codegen.RunDSL(t, testdata.ParameterPropertyNamesDSL)
		if reverse {
			slices.Reverse(root.API.HTTP.Services[0].HTTPEndpoints)
		}
		document, err := BuildDocument(root.API, root.Types, root.ResultTypes)
		require.NoError(t, err)
		if first != nil {
			require.Equal(t, first.Components.Schemas, document.Components.Schemas)
		}
		first = document
		resolve := func(schema *Schema) *Schema {
			for schema.Ref != "" {
				schema = document.Components.Schemas[strings.TrimPrefix(schema.Ref, "#/components/schemas/")]
				require.NotNil(t, schema)
			}
			return schema
		}
		exchange := document.Paths["/exchange"].Operations["POST"]
		parameter := exchange.Parameters[0]
		if parameter.Ref != "" {
			parameter = document.Components.Parameters[strings.TrimPrefix(parameter.Ref, "#/components/parameters/")]
		}
		response := exchange.Responses["200"].Value
		form := resolve(document.Paths["/form"].Operations["POST"].RequestBody.Value.Content["application/x-www-form-urlencoded"].Schema)
		for _, tc := range []struct {
			name   string
			schema *Schema
			text   bool
		}{
			{"query", parameter.Value.Schema, true},
			{"response header", response.Headers["X-Record"].Value.Schema, true},
			{"styled form", form.Properties["styled"], true},
			{"ordinary form JSON", form.Properties["ordinary"], false},
			{"request JSON", exchange.RequestBody.Value.Content["application/json"].Schema, false},
			{"response JSON", response.Content["application/json"].Schema, false},
		} {
			t.Run(tc.name, func(t *testing.T) {
				schema := resolve(tc.schema)
				var example map[string]any
				encoded, err := json.Marshal(schema.Example)
				require.NoError(t, err)
				require.NoError(t, json.Unmarshal(encoded, &example))
				if tc.text {
					require.ElementsMatch(t, []string{"count", "other"}, slices.Collect(maps.Keys(schema.Properties)))
					require.Equal(t, []string{"count"}, schema.Required)
					require.Equal(t, map[string]any{"count": float64(3), "other": float64(5)}, example)
				} else {
					require.ElementsMatch(t, []string{"wire", "alternate"}, slices.Collect(maps.Keys(schema.Properties)))
					require.Equal(t, []string{"wire"}, schema.Required)
					require.Equal(t, map[string]any{"wire": float64(3), "alternate": float64(5)}, example)
				}
			})
		}
		hidden := document.Paths["/hidden"].Operations["GET"].Parameters[0]
		if hidden.Ref != "" {
			hidden = document.Components.Parameters[strings.TrimPrefix(hidden.Ref, "#/components/parameters/")]
		}
		require.ElementsMatch(t, []string{"count", "hidden"}, slices.Collect(maps.Keys(resolve(hidden.Value.Schema).Properties)))
	}
}
