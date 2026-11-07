package representation

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func TestFormJSONShapeUsesCapturedNames(t *testing.T) {
	inner := &expr.UserTypeExpr{TypeName: "Inner", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{
		{Name: "member", Attribute: &expr.AttributeExpr{Type: expr.Int64, Meta: expr.MetaExpr{"struct:tag:json": {"jsonMember"}}}},
	}}}
	body := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "value", Attribute: &expr.AttributeExpr{Type: inner, Meta: expr.MetaExpr{"struct:tag:form": {"wire"}, "struct:tag:json": {"wrong"}}}},
		{Name: "skip", Attribute: &expr.AttributeExpr{Type: expr.Int, Meta: expr.MetaExpr{"struct:tag:form": {"-"}}}},
	}, Validation: &expr.ValidationExpr{Required: []string{"value"}}}
	contract, err := expr.HTTPFormBodyEncoding(body)
	require.NoError(t, err)
	context := codegen.NewAttributeContext(true, false, false, "", codegen.NewNameScope())
	shape, err := FormJSONShape(body, contract, context)
	require.NoError(t, err)
	for _, tc := range []struct {
		wire string
		want bool
	}{
		{`{"wire":{"jsonMember":999999999999999999999999}}`, true},
		{`{"wrong":{"jsonMember":1}}`, false},
		{`{"wire":{"jsonMember":"bad"}}`, false},
	} {
		matched, err := shape.Match([]byte(tc.wire))
		require.NoError(t, err)
		require.Equal(t, tc.want, matched, tc.wire)
	}
	evaluated, err := shape.Evaluated([]byte(`{"wire":{"jsonMember":1}}`))
	require.NoError(t, err)
	require.Equal(t, []string{"wire"}, evaluated)
}

func TestFormJSONShapeNormalizesOnlyTextUnionBranches(t *testing.T) {
	record := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "x", Attribute: &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:tag:json": {"jsonX"}}}},
	}, Validation: &expr.ValidationExpr{Required: []string{"x"}}}
	choice := &expr.AttributeExpr{Type: &expr.Union{TypeName: "Choice", Untagged: true, Values: []*expr.NamedAttributeExpr{
		{Name: "Record", Attribute: record}, {Name: "Flag", Attribute: &expr.AttributeExpr{Type: expr.Boolean}},
	}}}
	body := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "text", Attribute: &expr.AttributeExpr{Type: choice.Type}}, {Name: "json", Attribute: &expr.AttributeExpr{Type: choice.Type}},
	}, Meta: expr.MetaExpr{
		"openapi:encoding:text:style": {"form"}, "openapi:encoding:text:explode": {"false"},
		"openapi:encoding:json:contentType": {"application/json"},
	}, Validation: &expr.ValidationExpr{Required: []string{"text", "json"}}}
	contract, err := expr.HTTPFormBodyEncoding(body)
	require.NoError(t, err)
	context := codegen.NewAttributeContext(true, false, false, "", codegen.NewNameScope())
	shape, err := FormJSONShape(body, contract, context)
	require.NoError(t, err)
	for _, tc := range []struct {
		raw  string
		want bool
	}{
		{`{"text":{"x":"ok"},"json":{"jsonX":"ok"}}`, true},
		{`{"text":{"jsonX":"ok"},"json":{"jsonX":"ok"}}`, false},
		{`{"text":{"x":"ok"},"json":{"x":"ok"}}`, false},
	} {
		match, err := shape.Match([]byte(tc.raw))
		require.NoError(t, err)
		require.Equal(t, tc.want, match, tc.raw)
	}
	require.Equal(t, []string{"jsonX"}, record.Type.(*expr.Object).Attribute("x").Meta["struct:tag:json"])
}
