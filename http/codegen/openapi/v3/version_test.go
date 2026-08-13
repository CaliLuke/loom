package openapiv3

import (
	"encoding/json"
	"testing"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/openapi"
	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/require"
)

func TestConstructForVersion(t *testing.T) {
	tests := []struct {
		name         string
		target       openAPIVersion
		constructors []versionedConstructor[string]
		want         string
		wantOK       bool
	}{
		{
			name:   "additive feature omitted before minimum version",
			target: openAPIVersion31,
			constructors: []versionedConstructor[string]{
				{
					versions: versionRange{from: openAPIVersion32},
					construct: func() string {
						return "3.2 field"
					},
				},
			},
		},
		{
			name:   "additive feature emitted at minimum version",
			target: openAPIVersion32,
			constructors: []versionedConstructor[string]{
				{
					versions: versionRange{from: openAPIVersion32},
					construct: func() string {
						return "3.2 field"
					},
				},
			},
			want:   "3.2 field",
			wantOK: true,
		},
		{
			name:   "incompatible representation routes by version",
			target: openAPIVersion32,
			constructors: []versionedConstructor[string]{
				{
					versions: versionRange{from: openAPIVersion31, through: openAPIVersion31},
					construct: func() string {
						return "3.1 shape"
					},
				},
				{
					versions: versionRange{from: openAPIVersion32},
					construct: func() string {
						return "3.2 shape"
					},
				},
			},
			want:   "3.2 shape",
			wantOK: true,
		},
		{
			name:   "newer constructor overrides an open ended older constructor",
			target: openAPIVersion32 + 1,
			constructors: []versionedConstructor[string]{
				{
					versions: versionRange{from: openAPIVersion32},
					construct: func() string {
						return "3.2 shape"
					},
				},
				{
					versions: versionRange{from: openAPIVersion32 + 1},
					construct: func() string {
						return "future shape"
					},
				},
			},
			want:   "future shape",
			wantOK: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := versionRouter{target: test.target}
			got, ok := router.construct(test.constructors...)
			require.Equal(t, test.wantOK, ok)
			require.Equal(t, test.want, got)
		})
	}
}

func TestFilterOpenAPI31PreservesContentSchema(t *testing.T) {
	contentSchema := &openapi.Schema{Type: openapi.Object}
	schema := &openapi.Schema{
		Type:          openapi.String,
		ContentSchema: contentSchema,
	}

	filterSchemaCompatibility(schema, make(map[*openapi.Schema]struct{}))

	require.Same(t, contentSchema, schema.ContentSchema)
}

func TestNewRejectsInvalidOpenAPIVersion(t *testing.T) {
	root := &expr.RootExpr{
		API: &expr.APIExpr{
			Meta: expr.MetaExpr{"openapi:version": {"3.3"}},
			HTTP: &expr.HTTPExpr{Services: []*expr.HTTPServiceExpr{{}}},
		},
	}

	require.Nil(t, New(root))
}

func TestFilterOpenAPI31RecursesPropertyCarrierSchemas(t *testing.T) {
	additional := compatibilityOnlySchema()
	unevaluated := compatibilityOnlySchema()
	schema := &openapi.Schema{
		AdditionalProperties:  additional,
		UnevaluatedProperties: unevaluated,
	}
	additional.Properties = map[string]*openapi.Schema{"root": schema}
	unevaluated.Items = unevaluated

	filterSchemaCompatibility(schema, make(map[*openapi.Schema]struct{}))

	for _, child := range []*openapi.Schema{additional, unevaluated} {
		require.Empty(t, child.Discriminator.DefaultMapping)
		require.False(t, child.Discriminator.Optional)
		require.Contains(t, child.Required, "kind")
		require.Empty(t, child.XML.NodeType)
	}

	booleanCarriers := &openapi.Schema{
		AdditionalProperties:  false,
		UnevaluatedProperties: true,
	}
	filterSchemaCompatibility(booleanCarriers, make(map[*openapi.Schema]struct{}))
	require.Equal(t, false, booleanCarriers.AdditionalProperties)
	require.Equal(t, true, booleanCarriers.UnevaluatedProperties)
}

func TestFilterOpenAPI31RendersCompatiblePropertyCarrierSchemas(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: OpenAPICompatibilityVersion,
		Info: &Info{
			Title:   "compatibility",
			Version: "1.0.0",
		},
		Paths: map[string]*PathItem{},
		Components: &Components{Schemas: map[string]*openapi.Schema{
			"Envelope": {
				Type:                  openapi.Object,
				AdditionalProperties:  compatibilityOnlySchema(),
				UnevaluatedProperties: compatibilityOnlySchema(),
			},
		}},
	}

	filterOpenAPI31(spec)
	source, err := toJSON(nil, spec)
	require.NoError(t, err)
	parsed, err := libopenapi.NewDocument([]byte(source))
	require.NoError(t, err)
	_, err = parsed.BuildV3Model()
	require.NoError(t, err)

	var rendered map[string]any
	require.NoError(t, json.Unmarshal([]byte(source), &rendered))
	components := rendered["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)
	envelope := schemas["Envelope"].(map[string]any)
	for _, field := range []string{"additionalProperties", "unevaluatedProperties"} {
		child := envelope[field].(map[string]any)
		require.NotContains(t, child["discriminator"].(map[string]any), "defaultMapping")
		require.Contains(t, child["required"].([]any), "kind")
		require.NotContains(t, child["xml"].(map[string]any), "nodeType")
	}
}

func TestFilterOpenAPI31WarnsAboutDroppedPathOperations(t *testing.T) {
	get := new(Operation)
	spec := &OpenAPI{Paths: map[string]*PathItem{
		"/compatible": {
			Get: new(Operation),
		},
		"/mixed": {
			Get:   get,
			Query: new(Operation),
		},
		"/query": {
			Query: new(Operation),
		},
		"/tunnel": {
			Connect: new(Operation),
			AdditionalOperations: map[string]*Operation{
				"COPY": new(Operation),
			},
		},
	}}

	warnings := filterOpenAPI31(spec)

	require.Equal(t, []string{
		`OpenAPI 3.1 omits unsupported method QUERY from path "/mixed"`,
		`OpenAPI 3.1 omits unsupported method QUERY from path "/query" and removes the path because no compatible operations remain`,
		`OpenAPI 3.1 omits unsupported methods CONNECT, COPY from path "/tunnel" and removes the path because no compatible operations remain`,
	}, warnings)
	require.Same(t, get, spec.Paths["/mixed"].Get)
	require.Nil(t, spec.Paths["/mixed"].Query)
	require.NotNil(t, spec.Paths["/compatible"].Get)
	require.NotContains(t, spec.Paths, "/query")
	require.NotContains(t, spec.Paths, "/tunnel")
}

func compatibilityOnlySchema() *openapi.Schema {
	return &openapi.Schema{
		Type: openapi.Object,
		Discriminator: &openapi.Discriminator{
			PropertyName:   "kind",
			DefaultMapping: "#/components/schemas/Fallback",
			Optional:       true,
		},
		XML: &openapi.XML{NodeType: "element"},
	}
}
