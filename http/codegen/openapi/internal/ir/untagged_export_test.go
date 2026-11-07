package ir

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestUntaggedUnionExportClosure(t *testing.T) {
	for _, named := range []bool{false, true} {
		for _, nullable := range []bool{false, true} {
			for _, closed := range []bool{false, true} {
				union := &expr.Union{TypeName: "Choice", Untagged: true, Values: []*expr.NamedAttributeExpr{
					{Name: "Anything", Attribute: &expr.AttributeExpr{Type: expr.Any}},
					{Name: "Text", Attribute: &expr.AttributeExpr{Type: expr.String}},
				}}
				var typ expr.DataType = union
				if named {
					typ = &expr.UserTypeExpr{TypeName: "Choice", AttributeExpr: &expr.AttributeExpr{Type: union}}
				}
				attribute := &expr.AttributeExpr{Type: typ, Nullable: nullable}
				if closed {
					attribute.Meta = expr.MetaExpr{"openapi:unevaluatedProperties": {"false"}}
				}
				analyzer := NewAnalyzer(expr.NewRandom("union-closure"), false)
				schema := analyzer.AnalyzeSchema(attribute)
				require.NoError(t, analyzer.Err())
				require.NotNil(t, schema)
				if nullable {
					require.Nil(t, schema.UnevaluatedProperties)
					require.Len(t, schema.AnyOf, 2)
					schema = schema.AnyOf[0]
				}
				closure := schema.UnevaluatedProperties
				if schema.Ref != "" {
					schema = analyzer.Components()[strings.TrimPrefix(schema.Ref, "#/components/schemas/")]
					require.NotNil(t, schema)
				}
				if closure == nil {
					closure = schema.UnevaluatedProperties
				}
				if closed {
					require.NotNil(t, closure, "named=%t nullable=%t closed=%t schema=%#v", named, nullable, closed, schema)
					require.NotNil(t, closure.Bool)
					require.False(t, *closure.Bool)
				} else {
					require.Nil(t, closure)
				}
				require.Len(t, schema.OneOf, 2)
				for _, branch := range schema.OneOf {
					require.Nil(t, branch.UnevaluatedProperties)
				}
				require.False(t, union.Closed())
			}
		}
	}
}

func TestUntaggedUnionExportShapes(t *testing.T) {
	minimum := 2
	record := &expr.UserTypeExpr{TypeName: "Record", AttributeExpr: &expr.AttributeExpr{
		Type:       &expr.Object{{Name: "value", Attribute: &expr.AttributeExpr{Type: expr.Boolean}}},
		Validation: &expr.ValidationExpr{Required: []string{"value"}},
	}}
	attribute := &expr.AttributeExpr{Type: &expr.Union{TypeName: "Choice", Untagged: true, Values: []*expr.NamedAttributeExpr{
		{Name: "Inline", Attribute: &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{MinLength: &minimum}}},
		{Name: "Named", Attribute: &expr.AttributeExpr{Type: record}},
		{Name: "Nullable", Attribute: &expr.AttributeExpr{Type: expr.Int, Nullable: true}},
	}}}
	analyzer := NewAnalyzer(expr.NewRandom("union-shapes"), false)
	schema := analyzer.AnalyzeSchema(attribute)
	require.NoError(t, analyzer.Err())
	require.Len(t, schema.OneOf, 3)
	require.Empty(t, schema.OneOf[0].Ref)
	require.Equal(t, "string", schema.OneOf[0].Type)
	require.Equal(t, &minimum, schema.OneOf[0].MinLength)
	require.Equal(t, "#/components/schemas/Record", schema.OneOf[1].Ref)
	require.Len(t, schema.OneOf[2].AnyOf, 2)
	require.Equal(t, "integer", schema.OneOf[2].AnyOf[0].Type)
	require.Equal(t, "null", schema.OneOf[2].AnyOf[1].Type)
}

func TestUntaggedUnionExportLocations(t *testing.T) {
	root := codegen.RunDSL(t, testdata.UntaggedLocationsDSL)
	document, err := BuildDocument(root.API, root.Types, root.ResultTypes)
	require.NoError(t, err)
	for _, tc := range []struct {
		path, location, style string
		explode               *bool
	}{
		{"/query", "query", "", nil},
		{"/QueryList", "query", "", boolPtr(false)},
		{"/QueryDeep", "query", "deepObject", nil},
		{"/HeaderScalar", "header", "", nil},
		{"/CookieScalar", "cookie", "cookie", nil},
		{"/PathList/{value}", "path", "label", boolPtr(true)},
	} {
		t.Run(tc.path, func(t *testing.T) {
			parameter := document.Paths[tc.path].Operations["GET"].Parameters[0]
			if parameter.Ref != "" {
				parameter = document.Components.Parameters[strings.TrimPrefix(parameter.Ref, "#/components/parameters/")]
			}
			require.Equal(t, tc.location, parameter.Value.In)
			require.Equal(t, tc.style, parameter.Value.Style)
			require.Equal(t, tc.explode, parameter.Value.Explode)
			schema := parameter.Value.Schema
			if schema.Ref != "" {
				schema = document.Components.Schemas[strings.TrimPrefix(schema.Ref, "#/components/schemas/")]
			}
			require.Len(t, schema.OneOf, 2)
		})
	}
	header := document.Paths["/ResponseUnion"].Operations["GET"].Responses["204"].Value.Headers["X-Value"]
	if header.Ref != "" {
		header = document.Components.Headers[strings.TrimPrefix(header.Ref, "#/components/headers/")]
	}
	schema := header.Value.Schema
	if schema.Ref != "" {
		schema = document.Components.Schemas[strings.TrimPrefix(schema.Ref, "#/components/schemas/")]
	}
	require.Len(t, schema.OneOf, 2)
}

func TestUntaggedUnionClosureKeepsAnonymousBodiesDistinct(t *testing.T) {
	for _, closedFirst := range []bool{false, true} {
		t.Run(map[bool]string{false: "open first", true: "closed first"}[closedFirst], func(t *testing.T) {
			root := codegen.RunDSL(t, testdata.UntaggedRootsDSL)
			if closedFirst {
				slices.Reverse(root.API.HTTP.Services[0].HTTPEndpoints)
			}
			document, err := BuildDocument(root.API, root.Types, root.ResultTypes)
			require.NoError(t, err)
			body := func(path string) *Schema {
				ref := document.Paths[path].Operations["POST"].RequestBody
				if ref.Ref != "" {
					ref = document.Components.RequestBodies[strings.TrimPrefix(ref.Ref, "#/components/requestBodies/")]
				}
				return ref.Value.Content["application/json"].Schema
			}
			resolve := func(schema *Schema) *Schema {
				if schema.Ref != "" {
					return document.Components.Schemas[strings.TrimPrefix(schema.Ref, "#/components/schemas/")]
				}
				return schema
			}
			openBody, closedBody := body("/openbody"), body("/closedbody")
			require.NotEqual(t, openBody.Ref, closedBody.Ref)
			openValue := resolve(resolve(openBody).Properties["value"])
			closedValue := resolve(resolve(closedBody).Properties["value"])
			require.Nil(t, openValue.UnevaluatedProperties)
			require.NotNil(t, closedValue.UnevaluatedProperties)
			require.NotNil(t, closedValue.UnevaluatedProperties.Bool)
			require.False(t, *closedValue.UnevaluatedProperties.Bool)
		})
	}
}
