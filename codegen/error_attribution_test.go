package codegen

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

func TestNewErrorRetainsUsableAttribution(t *testing.T) {
	authored := &expr.AttributeExpr{}
	authored.SetDSLSource(func() {
	})
	earlier := &expr.AttributeExpr{}
	earlier.SetDSLSource(func() {
	})
	synthetic := &expr.AttributeExpr{}
	cases := []struct {
		name     string
		existing eval.Expression
		fallback eval.Expression
		want     eval.Expression
	}{
		{"nil owner accepts authored fallback", nil, authored, authored},
		{"synthetic owner accepts authored fallback", synthetic, authored, authored},
		{"located owner retains attribution", earlier, authored, earlier},
		{"synthetic fallback cannot replace owner", earlier, synthetic, earlier},
		{"unlocated owners retain first identity", synthetic, &expr.AttributeExpr{}, synthetic},
		{"nil fallback retains owner", synthetic, nil, synthetic},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cause := &DefaultValueError{Attribute: authored, Err: errors.New("materialization")}
			inner := &Error{Expr: tc.existing, Service: "original", Method: "read", Err: cause}
			wrapped := fmt.Errorf("analysis: %w", inner)
			got := NewError(NewSilentContext().WithService(&expr.ServiceExpr{Name: "other"}).WithMethod(&expr.MethodExpr{Name: "other"}), tc.fallback, wrapped)
			require.Same(t, wrapped, got)
			require.Same(t, tc.want, inner.Expr)
			require.Equal(t, "original", inner.Service)
			require.Equal(t, "read", inner.Method)
			var marker *DefaultValueError
			require.ErrorAs(t, got, &marker)
			require.Same(t, cause, marker)
			require.ErrorIs(t, got, cause.Err)
		})
	}
}
