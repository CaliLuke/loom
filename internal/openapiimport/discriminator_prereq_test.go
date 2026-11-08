package openapiimport

import (
	"testing"

	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/httpencoding"
	"github.com/stretchr/testify/require"
)

func TestDiscriminatorPrerequisites(t *testing.T) {
	t.Run("captured field codecs", func(t *testing.T) {
		root := expr.RunDSL(t, func() {
			flat := dsl.Type("Flat", func() {
				dsl.Attribute("type", dsl.String)
				dsl.Attribute("code", dsl.String)
				dsl.Required("type", "code")
			})
			dsl.Type("Labels", dsl.MapOf(dsl.String, dsl.String))
			nested := dsl.Type("Nested", func() {
				dsl.Attribute("child", flat)
			})
			dsl.Type("Tagged", dsl.OneOf(flat, nested))
			dsl.Type("FullObject", dsl.OneOf(func() {
				dsl.Untagged()
				dsl.Attribute("flat", flat)
				dsl.Attribute("labels", dsl.MapOf(dsl.String, dsl.String))
			}))
		})
		for _, tc := range []struct {
			name, typ, style, media string
			shape                   httpencoding.Shape
			codec                   httpencoding.Style
			literals, prefixes      []string
			reject                  bool
		}{
			{name: "object deepObject", typ: "Flat", style: "deepObject", shape: httpencoding.Object, codec: httpencoding.DeepObject, literals: []string{"Values[type]", "Values[code]"}},
			{name: "map deepObject", typ: "Labels", style: "deepObject", shape: httpencoding.ScalarMap, codec: httpencoding.DeepObject, prefixes: []string{"Values"}},
			{name: "tagged default JSON including nested payload", typ: "Tagged", shape: httpencoding.Nested, codec: httpencoding.JSON, literals: []string{"Values"}},
			{name: "tagged explicit style rejected", typ: "Tagged", style: "deepObject", shape: httpencoding.Nested, reject: true},
			{name: "nested explicit style rejected", typ: "Nested", style: "deepObject", shape: httpencoding.Nested, reject: true},
			{name: "nested JSON does not override style", typ: "Nested", style: "deepObject", media: "application/json", shape: httpencoding.Nested, reject: true},
			{name: "FullObject deepObject", typ: "FullObject", style: "deepObject", shape: httpencoding.Union, codec: httpencoding.DeepObject, literals: []string{"Values[type]", "Values[code]"}, prefixes: []string{"Values"}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				shape := expr.HTTPValueShape(root.UserType(tc.typ).Attribute())
				require.Equal(t, tc.shape, shape.Shape)
				codec, keys, err := httpencoding.Resolve(httpencoding.FormField, "Values", shape, httpencoding.Serialization{Style: tc.style, MediaType: tc.media})
				if tc.reject {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				require.Equal(t, tc.codec, codec.Style)
				require.ElementsMatch(t, tc.literals, keys.Literals)
				require.ElementsMatch(t, tc.prefixes, keys.Prefixes)
			})
		}
	})
	t.Run("mapping occurrence and canonical target", func(t *testing.T) {
		entry := []byte(`openapi: 3.2.0
info: {title: Mapping, version: '1'}
paths: {}
components:
  schemas:
    Cat: {type: string}
    Choice:
      oneOf:
        - {$ref: '#/components/schemas/Cat'}
        - {$ref: './pets.yaml#/components/schemas/Dog'}
      discriminator:
        propertyName: kind
        mapping:
          cat: Cat
          dog: ./pets.yaml#/components/schemas/Dog
        defaultMapping: https://E.TEST:443/model#/Other
`)
		pets := "components:\n  schemas:\n    Dog: {type: string}\n"
		source := resolverFiles(t, entry, map[string]string{"pets.yaml": pets})
		resolved, diagnostics, err := resolveDescription(source)
		require.NoError(t, err)
		require.Empty(t, diagnostics)
		_, err = bundleResolution(resolved)
		require.NoError(t, err)
		outcomes := make(map[string]mappingOutcome)
		for _, outcome := range resolved.discriminatorEdges() {
			outcomes[outcome.source.pointer] = outcome
		}
		base := "/components/schemas/Choice/discriminator/"
		require.Equal(t, "Cat", outcomes[base+"mapping/cat"].entry)
		require.Equal(t, "#/components/schemas/Dog", outcomes[base+"mapping/dog"].bundled)
		require.Equal(t, "https://e.test/model#/Other", outcomes[base+"defaultMapping"].external)
		var branchTarget, mappingTarget *targetKey
		for _, edge := range resolved.edges {
			if edge.at.pointer == "/components/schemas/Choice/oneOf/1/$ref" {
				branchTarget = edge.target
			}
			if edge.at.pointer == base+"mapping/dog" {
				mappingTarget = edge.target
			}
		}
		require.NotNil(t, branchTarget)
		require.Equal(t, branchTarget, mappingTarget)
		require.NotEqual(t, outcomes[base+"mapping/dog"].source, *mappingTarget)

		withoutBase, _, err := resolveDescription(Source{Bytes: entry})
		require.NoError(t, err)
		for _, outcome := range withoutBase.discriminatorEdges() {
			if outcome.source.pointer == base+"mapping/dog" {
				require.NotNil(t, outcome.failure, "document mappings do not invent an entry retrieval base")
				return
			}
		}
		t.Fatal("missing relative mapping occurrence")
	})
}
