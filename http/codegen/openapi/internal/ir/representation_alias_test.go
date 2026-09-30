package ir

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestByteAliasEncodingOverrideOwnsWholeChain(t *testing.T) {
	for _, outer := range []bool{false, true} {
		inner := &expr.UserTypeExpr{TypeName: "Inner", AttributeExpr: &expr.AttributeExpr{Type: expr.Bytes}}
		attr := &expr.AttributeExpr{Type: inner}
		metadata := expr.MetaExpr{"openapi:format": []string{"authored"}}
		if outer {
			attr.Meta = metadata
		} else {
			inner.Meta = metadata
		}
		context := expr.NewValueContext()
		occurrence, err := context.NewOccurrence(attr)
		require.NoError(t, err)
		plan, err := context.NewValuePlan(occurrence, expr.ValuePlanRequest{Target: attr, Codec: expr.ValueCodecJSON, Use: expr.ValuePlanSchema})
		require.NoError(t, err)
		analyzer := NewAnalyzer(nil, false)
		got := analyzer.analyzeOccurrence(attr, "override", plan.Root())
		require.Empty(t, byteProjectionIdentity(attr, plan.Root()))
		require.Empty(t, got.ContentEncoding)
		require.Equal(t, "authored", got.Format)
	}
}

func TestByteAliasBoundsIntersectBeforeProjection(t *testing.T) {
	maximum := int(^uint(0) >> 1)
	small := 2
	for _, incompatible := range []bool{false, true} {
		validation := &expr.ValidationExpr{MaxLength: &maximum}
		if incompatible {
			validation = &expr.ValidationExpr{MinLength: &maximum}
		}
		inner := &expr.UserTypeExpr{TypeName: "Inner", AttributeExpr: &expr.AttributeExpr{Type: expr.Bytes, Validation: validation}}
		attr := &expr.AttributeExpr{Type: inner, Validation: &expr.ValidationExpr{MaxLength: &small}}
		context := expr.NewValueContext()
		occurrence, err := context.NewOccurrence(attr)
		require.NoError(t, err)
		plan, err := context.NewValuePlan(occurrence, expr.ValuePlanRequest{Target: attr, Codec: expr.ValueCodecJSON, Use: expr.ValuePlanSchema})
		require.NoError(t, err)
		analyzer := NewAnalyzer(nil, false)
		got := analyzer.analyzeOccurrence(attr, "bounds", plan.Root())
		require.Nil(t, got.MaxLength, "remove the old occurrence length gate")
		require.Len(t, got.AllOf, 1, "retain the baseline occurrence overlay")
		got = got.AllOf[0]
		require.Equal(t, "base64", got.ContentEncoding)
		require.Nil(t, got.MaxLength)
		require.NotNil(t, got.Not)
		if incompatible {
			require.Equal(t, &Schema{}, got.Not)
		} else {
			require.Len(t, got.AnyOf, 3)
		}
	}
}

func TestNestedByteOverflowDiagnostic(t *testing.T) {
	maximum := int(^uint(0) >> 1)
	attr := &expr.AttributeExpr{Type: &expr.Object{{Name: "attachments", Attribute: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: &expr.Object{{Name: "blob", Attribute: &expr.AttributeExpr{Type: expr.Bytes, Validation: &expr.ValidationExpr{MaxLength: &maximum}}}}}}}}}}
	context := expr.NewValueContext()
	occurrence, err := context.NewOccurrence(attr)
	require.NoError(t, err)
	plan, err := context.NewValuePlan(occurrence, expr.ValuePlanRequest{Target: attr, Codec: expr.ValueCodecJSON, Use: expr.ValuePlanSchema})
	require.NoError(t, err)
	defer func() {
		failure := recover()
		require.NotNil(t, failure)
		message := failure.(error).Error()
		t.Log(message)
		require.Contains(t, message, "attachments")
		require.Contains(t, message, "blob")
	}()
	NewAnalyzer(nil, false).analyzeOccurrence(attr, "service/storage/method/upload/request", plan.Root())
}
