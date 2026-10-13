package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEffectiveStreamingResult(t *testing.T) {
	result := &AttributeExpr{Type: &Object{{Name: "summary", Attribute: &AttributeExpr{Type: String}}}}
	streaming := &AttributeExpr{Type: &Object{{Name: "chunk", Attribute: &AttributeExpr{Type: Int}}}}
	for _, tc := range []struct {
		name      string
		result    *AttributeExpr
		streaming *AttributeExpr
		want      *AttributeExpr
	}{
		{"distinct", result, streaming, streaming},
		{"prepared ordinary", result, result, result},
		{"unprepared final", result, nil, result},
		{"empty explicit fallback", result, &AttributeExpr{Type: Empty}, result},
		{"no result", nil, nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := &MethodExpr{Result: tc.result, StreamingResult: tc.streaming}
			require.True(t, m.EffectiveStreamingResult() == tc.want, "retain the actual owning attribute")
		})
	}
}

func TestMissingStreamingResultAccess(t *testing.T) {
	i := &InterceptorExpr{Name: "audit", ReadStreamingResult: &AttributeExpr{Type: &Object{{Name: "chunk", Attribute: &AttributeExpr{Type: Int}}}}}
	m := &MethodExpr{Name: "watch", Service: &ServiceExpr{Name: "events"}, Stream: ServerStreamKind}
	require.ErrorContains(t, i.validate(m), "result is not an object")
}
