package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGRPCNullableErrorAdmission(t *testing.T) {
	for _, shape := range []string{"root", "member", "ordinary"} {
		t.Run(shape, func(t *testing.T) {
			payload := &AttributeExpr{Type: &Object{{Name: "message", Attribute: &AttributeExpr{Type: String}}}}
			switch shape {
			case "root":
				payload.Nullable = true
			case "member":
				payload.Find("message").Nullable = true
			}
			failure := &ErrorExpr{Name: "failure", AttributeExpr: payload}
			endpoint := &GRPCEndpointExpr{MethodExpr: &MethodExpr{Name: "call", Errors: []*ErrorExpr{failure}}}
			endpoint.GRPCErrors = []*GRPCErrorExpr{{Name: failure.Name, Response: &GRPCResponseExpr{Parent: endpoint}}}
			validation := newValidationErrors()
			endpoint.validateNullableTransport(validation)
			if shape == "ordinary" {
				require.Empty(t, validation.Errors)
			} else {
				require.ErrorContains(t, validation, "gRPC error response results do not support nullable attributes")
			}
		})
	}
}
