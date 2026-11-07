package ir

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

func TestFormEncodingMetadataUsesCapturedNames(t *testing.T) {
	body := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "first", Attribute: &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:tag:form": {"second"}}}},
		{Name: "second", Attribute: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}, Meta: expr.MetaExpr{"struct:tag:form": {"first"}}}},
		{Name: "skip", Attribute: &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:tag:form": {"-"}}}},
	}, Meta: expr.MetaExpr{
		"openapi:encoding:first:contentType": {"application/json"},
		"openapi:encoding:second:style":      {"form"},
		"openapi:encoding:second:explode":    {"false"},
		"openapi:encoding:skip:contentType":  {"application/json"},
	}}
	form, err := expr.HTTPFormBodyEncoding(body)
	require.NoError(t, err)
	media := &MediaType{Metadata: body.Meta.Dup()}
	require.NoError(t, applyMediaFieldEncoding(&transportir.Request{Body: body, FormEncoded: true, Form: form}, media))
	require.Equal(t, map[string][]string{
		"openapi:encoding:second:contentType": {"application/json"},
		"openapi:encoding:first:style":        {"form"},
		"openapi:encoding:first:explode":      {"false"},
	}, media.Metadata)
}

func TestFormUnionEncodingRemainsExplicit(t *testing.T) {
	body := &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: encodingUnion("", &expr.AttributeExpr{Type: expr.String}, &expr.AttributeExpr{Type: expr.Int})}}, Meta: expr.MetaExpr{"openapi:encoding:value:style": {"form"}}}
	form, err := expr.HTTPFormBodyEncoding(body)
	require.NoError(t, err)
	media := &MediaType{}
	require.NoError(t, applyMediaFieldEncoding(&transportir.Request{Body: body, FormEncoded: true, Form: form}, media))
	require.Equal(t, []string{"form"}, media.Metadata["openapi:encoding:value:style"])
}
