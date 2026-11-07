package ir

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

func encodingArray(item expr.DataType) *expr.AttributeExpr {
	return &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: item}}}
}

func encodingUnion(source string, branches ...*expr.AttributeExpr) *expr.AttributeExpr {
	values := make([]*expr.NamedAttributeExpr, len(branches))
	for i, branch := range branches {
		values[i] = &expr.NamedAttributeExpr{Name: string(rune('a' + i)), Attribute: branch}
	}
	result := &expr.AttributeExpr{Type: &expr.Union{Untagged: true, Values: values}}
	if source != "" {
		result.Meta = expr.MetaExpr{"http:encoding:sourceType": {source}}
	}
	return result
}

func TestEncodingSourceTypeSchema(t *testing.T) {
	for _, tc := range []struct {
		name    string
		value   *expr.AttributeExpr
		typ     string
		wrapper bool
	}{
		{"source array union", encodingUnion("array", encodingArray(expr.String), encodingArray(expr.Int)), "array", false},
		{"typeless singleton", &expr.AttributeExpr{Type: encodingArray(expr.String).Type, Meta: expr.MetaExpr{"http:encoding:sourceType": {"absent"}}}, "", true},
		{"native array", encodingArray(expr.String), "array", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			analyzer := NewAnalyzer(expr.NewRandom("source-type"), false)
			schema := analyzer.AnalyzeSchema(tc.value)
			require.NoError(t, analyzer.err)
			require.Equal(t, tc.typ, schema.Type)
			if tc.wrapper {
				require.Len(t, schema.OneOf, 1)
				require.Equal(t, "array", schema.OneOf[0].Type)
			}
		})
	}
}

func TestEncodingBranchDefaultsRemainImplicit(t *testing.T) {
	field := encodingUnion("array", encodingArray(expr.String), encodingArray(&expr.Object{{Name: "x", Attribute: &expr.AttributeExpr{Type: expr.String}}}))
	body := &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: field}}}
	form, err := expr.HTTPFormBodyEncoding(body)
	require.NoError(t, err)
	media := &MediaType{}
	require.NoError(t, applyMediaFieldEncoding(&transportir.Request{Body: body, FormEncoded: true, Form: form}, media))
	require.Empty(t, media.Metadata, "heterogeneous item defaults must remain local to their schemas")
}

func TestEncodingNestedRootAuthoredMedia(t *testing.T) {
	body := &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: encodingUnion("", &expr.AttributeExpr{Type: expr.String}, &expr.AttributeExpr{Type: expr.Int})}}, Meta: expr.MetaExpr{"openapi:encoding:value:contentType": {"application/example+json; charset=utf-8"}}}
	root := encodingUnion("", encodingUnion("", body))
	form, err := expr.HTTPFormBodyEncoding(root)
	require.NoError(t, err)
	media := &MediaType{}
	require.NoError(t, applyMediaFieldEncoding(&transportir.Request{Body: root, FormEncoded: true, Form: form}, media))
	require.Equal(t, []string{"application/example+json; charset=utf-8"}, media.Metadata["openapi:encoding:value:contentType"])
}

func TestEncodingRootAlternativesReconcile(t *testing.T) {
	for _, multipart := range []bool{false, true} {
		for _, tc := range []struct {
			name, left, right string
			conflict          bool
		}{
			{"same explicit", "application/json", "application/json", false},
			{"incompatible scalar grammars", "application/json", "text/plain", true},
			{"explicit and implicit scalar", "text/plain", "", false},
		} {
			t.Run(tc.name+map[bool]string{false: "/form", true: "/multipart"}[multipart], func(t *testing.T) {
				branch := func(label string) *expr.AttributeExpr {
					value := &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: &expr.AttributeExpr{Type: expr.String}}}, Meta: expr.MetaExpr{}}
					if label != "" {
						value.Meta["openapi:encoding:value:contentType"] = []string{label}
					}
					return value
				}
				body := encodingUnion("", branch(tc.left), branch(tc.right))
				request := &transportir.Request{Body: body, FormEncoded: !multipart, Multipart: multipart}
				var err error
				if multipart {
					request.MultipartEncoding, err = expr.HTTPMultipartBodyEncoding(body)
				} else {
					request.Form, err = expr.HTTPFormBodyEncoding(body)
				}
				require.NoError(t, err)
				media := &MediaType{}
				err = applyMediaFieldEncoding(request, media)
				if tc.conflict {
					require.ErrorContains(t, err, "incompatible Encoding Objects")
				} else {
					require.NoError(t, err)
					require.Equal(t, []string{tc.left}, media.Metadata["openapi:encoding:value:contentType"])
				}
			})
		}
	}
}

func TestEncodingFlagsSuppressMedia(t *testing.T) {
	for _, multipart := range []bool{false, true} {
		body := &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: encodingArray(expr.Int)}}, Meta: expr.MetaExpr{"openapi:encoding:value:contentType": {"application/json"}, "openapi:encoding:value:allowReserved": {"false"}}}
		request := &transportir.Request{Body: body, FormEncoded: !multipart, Multipart: multipart}
		var err error
		if multipart {
			request.MultipartEncoding, err = expr.HTTPMultipartBodyEncoding(body)
		} else {
			request.Form, err = expr.HTTPFormBodyEncoding(body)
		}
		require.NoError(t, err)
		media := &MediaType{Metadata: body.Meta.Dup()}
		require.NoError(t, applyMediaFieldEncoding(request, media))
		require.NotContains(t, media.Metadata, "openapi:encoding:value:contentType")
		require.Equal(t, []string{"form"}, media.Metadata["openapi:encoding:value:style"])
	}
}

func TestEncodingSourceTypeAliasIsolation(t *testing.T) {
	array := &expr.UserTypeExpr{TypeName: "Strings", AttributeExpr: encodingArray(expr.String)}
	source := &expr.AttributeExpr{Type: array, Meta: expr.MetaExpr{"http:encoding:sourceType": {"absent"}}}
	ordinary := &expr.AttributeExpr{Type: array}
	analyzer := NewAnalyzer(expr.NewRandom("source-isolation"), false)
	require.NotEqual(t, analyzer.FingerprintAttribute(ordinary), analyzer.FingerprintAttribute(source))
	resolve := func(schema *Schema) *Schema {
		if schema.Ref != "" {
			return analyzer.schemas[strings.TrimPrefix(schema.Ref, "#/components/schemas/")]
		}
		return schema
	}
	require.Equal(t, "array", resolve(analyzer.AnalyzeSchema(ordinary)).Type)
	marked := resolve(analyzer.AnalyzeSchema(source))
	require.Empty(t, marked.Type)
	require.Len(t, marked.OneOf, 1)
	require.Equal(t, "array", resolve(marked.OneOf[0]).Type)
	require.Empty(t, array.Meta)
	require.Equal(t, "array", resolve(analyzer.AnalyzeSchema(ordinary)).Type)
}

func TestEncodingSourceTypePreservesNullMultiplicity(t *testing.T) {
	for _, tc := range []struct {
		name          string
		first, second bool
		nullable      bool
	}{
		{"one null branch", true, false, true},
		{"two null branches", true, true, false},
		{"no null branches", false, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value := encodingUnion("string", &expr.AttributeExpr{Type: expr.String, Nullable: tc.first}, &expr.AttributeExpr{Type: expr.String, Nullable: tc.second})
			analyzer := NewAnalyzer(expr.NewRandom("source-null"), false)
			schema := analyzer.AnalyzeSchema(value)
			require.NoError(t, analyzer.err)
			require.Equal(t, tc.nullable, expr.AllowsNull(value))
			if tc.nullable {
				require.Len(t, schema.AnyOf, 2)
				require.Equal(t, "null", schema.AnyOf[1].Type)
				schema = schema.AnyOf[0]
			} else {
				require.Empty(t, schema.AnyOf)
			}
			require.Equal(t, "string", schema.Type)
			require.Len(t, schema.OneOf, 2, "the original branch count remains unchanged")
		})
	}
}

func TestEncodingRootMetadataOverridesBranchDeclaration(t *testing.T) {
	branch := &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: &expr.AttributeExpr{Type: expr.String}}}, Meta: expr.MetaExpr{"openapi:encoding:value:contentType": {"text/plain"}}}
	root := encodingUnion("", encodingUnion("", branch))
	root.Meta = expr.MetaExpr{"openapi:encoding:value:contentType": {"application/example+json"}}
	for _, multipart := range []bool{false, true} {
		request := &transportir.Request{Body: root, FormEncoded: !multipart, Multipart: multipart}
		var err error
		if multipart {
			request.MultipartEncoding, err = expr.HTTPMultipartBodyEncoding(root)
		} else {
			request.Form, err = expr.HTTPFormBodyEncoding(root)
		}
		require.NoError(t, err)
		media := &MediaType{Metadata: root.Meta.Dup()}
		require.NoError(t, applyMediaFieldEncoding(request, media))
		require.Equal(t, []string{"application/example+json"}, media.Metadata["openapi:encoding:value:contentType"])
		require.Equal(t, []string{"text/plain"}, branch.Meta["openapi:encoding:value:contentType"])
	}
}

func TestEncodingSourceTypeCanonicalAliasOccurrence(t *testing.T) {
	for _, nullable := range []bool{false, true} {
		array := &expr.UserTypeExpr{TypeName: "Strings", AttributeExpr: encodingArray(expr.String)}
		array.Meta = expr.MetaExpr{"openapi:typename:canonical": {"true"}}
		alias := &expr.UserTypeExpr{TypeName: "Alias", AttributeExpr: &expr.AttributeExpr{Type: array, Nullable: nullable, Meta: expr.MetaExpr{"openapi:typename:canonical": {"true"}, "http:encoding:sourceType": {"absent"}}}}
		analyzer := NewAnalyzer(expr.NewRandom("canonical-source"), false)
		analyzer.AnalyzeSchema(&expr.AttributeExpr{Type: array})
		analyzer.AnalyzeSchema(&expr.AttributeExpr{Type: alias})
		require.NoError(t, analyzer.err)
		require.Equal(t, "array", analyzer.Components()["Strings"].Type)
		shape := analyzer.Components()["Alias"]
		if nullable {
			require.Len(t, shape.AnyOf, 2)
			require.Equal(t, "null", shape.AnyOf[1].Type)
			shape = shape.AnyOf[0]
		}
		require.Empty(t, shape.Type)
		require.Len(t, shape.OneOf, 1)
		require.Equal(t, "#/components/schemas/Strings", shape.OneOf[0].Ref)
	}
}

func TestEncodingSourceTypeWrapperAnnotations(t *testing.T) {
	for _, nullable := range []bool{false, true} {
		value := &expr.AttributeExpr{Type: expr.String, Nullable: nullable, Title: "Value", Description: "Outer annotation", Meta: expr.MetaExpr{"http:encoding:sourceType": {"absent"}, "openapi:readOnly": {"true"}, "openapi:writeOnly": {"true"}, "openapi:deprecated": {"true"}}}
		analyzer := NewAnalyzer(expr.NewRandom("source-annotations"), false)
		schema := analyzer.AnalyzeSchema(value)
		require.Equal(t, "Value", schema.Title)
		require.Equal(t, "Outer annotation", schema.Description)
		require.True(t, schema.ReadOnly)
		require.True(t, schema.WriteOnly)
		require.True(t, schema.Deprecated)
		if nullable {
			require.Len(t, schema.AnyOf, 2)
			schema = schema.AnyOf[0]
		}
		require.Len(t, schema.OneOf, 1)
		require.Empty(t, schema.OneOf[0].Title)
		require.False(t, schema.OneOf[0].ReadOnly)
	}
}

func TestEncodingSourceTypeOverlayPreservesInheritedNull(t *testing.T) {
	array := &expr.UserTypeExpr{TypeName: "NullableStrings", AttributeExpr: encodingArray(expr.String)}
	array.Nullable = true
	array.Meta = expr.MetaExpr{"openapi:typename:canonical": {"true"}}
	value := &expr.AttributeExpr{Type: array, Meta: expr.MetaExpr{"http:encoding:sourceType": {"array"}}}
	analyzer := NewAnalyzer(expr.NewRandom("source-null-overlay"), false)
	schema := analyzer.AnalyzeSchema(value)
	require.True(t, expr.AllowsNull(value))
	require.Len(t, schema.AnyOf, 2)
	require.Equal(t, "null", schema.AnyOf[1].Type)
	require.Equal(t, "array", schema.AnyOf[0].Type)
	require.Equal(t, "#/components/schemas/NullableStrings", schema.AnyOf[0].Ref)
	require.Len(t, analyzer.Components()["NullableStrings"].AnyOf, 2)
}
