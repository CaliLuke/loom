package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGRPCStreamingRequestPreservesValidation(t *testing.T) {
	minimum, maximum := 2, 4
	for _, test := range []struct {
		name string
		typ  DataType
	}{
		{"array", &Array{ElemType: &AttributeExpr{Type: String}}},
		{"map", &Map{KeyType: &AttributeExpr{Type: String}, ElemType: &AttributeExpr{Type: String}}},
		{"string", String},
		{"bytes", Bytes},
	} {
		for _, named := range []bool{false, true} {
			name := test.name
			if named {
				name += " named"
			}
			t.Run(name, func(t *testing.T) {
				validation := &ValidationExpr{MinLength: &minimum, MaxLength: &maximum}
				owner := &AttributeExpr{Type: test.typ, Validation: validation}
				payload := owner
				if named {
					payload = &AttributeExpr{Type: &UserTypeExpr{TypeName: "Values", AttributeExpr: owner}}
				}
				endpoint := &GRPCEndpointExpr{
					MethodExpr: &MethodExpr{StreamingPayload: payload},
					Service:    &GRPCServiceExpr{ServiceExpr: &ServiceExpr{}},
				}
				endpoint.Prepare()
				endpoint.finalizeStreamingRequest()
				require.Equal(t, validation, endpoint.StreamingRequest.Validation)
				require.NotSame(t, validation, endpoint.StreamingRequest.Validation)
				endpoint.StreamingRequest.Validation.MinLength = nil
				require.Equal(t, &minimum, owner.Validation.MinLength)
			})
		}
	}
}

func TestGRPCStreamingRequestPreservesObjectRequirements(t *testing.T) {
	owner := &AttributeExpr{
		Type:       &Object{{Name: "value", Attribute: &AttributeExpr{Type: String}}},
		Validation: &ValidationExpr{Required: []string{"value"}},
	}
	endpoint := &GRPCEndpointExpr{
		MethodExpr: &MethodExpr{StreamingPayload: &AttributeExpr{
			Type: &UserTypeExpr{TypeName: "Item", AttributeExpr: owner},
		}},
		Service: &GRPCServiceExpr{ServiceExpr: &ServiceExpr{}},
	}
	endpoint.Prepare()
	endpoint.finalizeStreamingRequest()
	require.Equal(t, []string{"value"}, endpoint.StreamingRequest.Validation.Required)
	endpoint.StreamingRequest.Validation.Required[0] = "other"
	require.Equal(t, []string{"value"}, owner.Validation.Required)
}

func TestGRPCStreamingRequestPreservesOtherConstraints(t *testing.T) {
	minimum, maximum := 1.0, 9.0
	for _, test := range []struct {
		name       string
		typ        DataType
		validation *ValidationExpr
	}{
		{"numeric", Int, &ValidationExpr{Minimum: &minimum, ExclusiveMaximum: &maximum}},
		{"pattern", String, &ValidationExpr{Pattern: "^[a-z]+$"}},
		{"enum", String, &ValidationExpr{Values: []any{"yes", "no"}}},
		{"unconstrained", String, nil},
		{"empty", Empty, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			endpoint := &GRPCEndpointExpr{
				MethodExpr: &MethodExpr{StreamingPayload: &AttributeExpr{
					Type: test.typ, Validation: test.validation,
				}},
				Service: &GRPCServiceExpr{ServiceExpr: &ServiceExpr{}},
			}
			endpoint.Prepare()
			endpoint.finalizeStreamingRequest()
			want := test.validation
			if want == nil {
				want = &ValidationExpr{}
			}
			require.Equal(t, want, endpoint.StreamingRequest.Validation)
			require.NotSame(t, want, endpoint.StreamingRequest.Validation)
		})
	}
}
