package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGRPCPresenceAdmissionBoundaries(t *testing.T) {
	for _, test := range []struct {
		name     string
		source   *AttributeExpr
		local    *AttributeExpr
		metadata bool
		want     string
	}{
		{"source nullable", &AttributeExpr{Type: String, Nullable: true}, nil, false, "request payloads"},
		{"local nullable", &AttributeExpr{Type: String}, &AttributeExpr{Type: String, Nullable: true}, false, "request messages"},
		{"native", &AttributeExpr{Type: String}, nil, false, ""},
		{"nullable Any source", &AttributeExpr{Type: Any, Nullable: true}, nil, false, ""},
		{"nullable Any local", &AttributeExpr{Type: Any}, &AttributeExpr{Type: Any, Nullable: true}, false, ""},
		{"metadata source nullable", &AttributeExpr{Type: String, Nullable: true}, nil, true, "request metadata"},
		{"metadata source Any", &AttributeExpr{Type: Any}, nil, true, "request metadata"},
		{"metadata native", &AttributeExpr{Type: String}, nil, true, ""},
		{"metadata uses source type", &AttributeExpr{Type: String}, &AttributeExpr{Type: Any}, true, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			endpoint := &GRPCEndpointExpr{MethodExpr: &MethodExpr{Name: "call"}}
			endpoint.MethodExpr.Payload = test.source
			endpoint.Request = test.local
			if test.metadata {
				endpoint.MethodExpr.Payload = &AttributeExpr{Type: &Object{{Name: "value:field", Attribute: test.source}}}
				local := test.local
				if local == nil {
					local = &AttributeExpr{Type: String}
				}
				endpoint.Request = nil
				endpoint.Metadata = NewMappedAttributeExpr(&AttributeExpr{Type: &Object{{Name: "value:x-value", Attribute: local}}})
			}
			verr := newValidationErrors()
			endpoint.validateNullableTransport(verr)
			if test.want == "" {
				require.Empty(t, verr.Errors)
			} else {
				require.ErrorContains(t, verr, "gRPC "+test.want+" do not support nullable attributes")
			}
		})
	}
}

func TestGRPCPresenceAdmissionNamedAndRecursiveGraphs(t *testing.T) {
	for _, nullable := range []bool{false, true} {
		node := &UserTypeExpr{TypeName: "Node", AttributeExpr: &AttributeExpr{}}
		node.Type = &Object{
			{Name: "next", Attribute: &AttributeExpr{Type: node}},
			{Name: "value", Attribute: &AttributeExpr{Type: String, Nullable: nullable}},
		}
		alias := &UserTypeExpr{TypeName: "Alias", AttributeExpr: &AttributeExpr{Type: node}}
		endpoint := &GRPCEndpointExpr{MethodExpr: &MethodExpr{
			Name: "call", Payload: &AttributeExpr{Type: alias},
		}}
		verr := newValidationErrors()
		endpoint.validateNullableTransport(verr)
		if nullable {
			require.ErrorContains(t, verr, "request payloads do not support nullable attributes")
		} else {
			require.Empty(t, verr.Errors)
		}
	}
}

func TestGRPCPresenceAdmissionRetainsSourceAndLocalGraphs(t *testing.T) {
	local := &AttributeExpr{Type: String, Nullable: true}
	source := &AttributeExpr{Type: Any}
	endpoint := &GRPCEndpointExpr{
		MethodExpr: &MethodExpr{Name: "call", Payload: &AttributeExpr{Type: &Object{{Name: "value", Attribute: source}}}},
		Request:    &AttributeExpr{Type: &Object{{Name: "value", Attribute: local}}},
	}
	verr := newValidationErrors()
	endpoint.validateNullableTransport(verr)
	require.Empty(t, verr.Errors)
	require.Equal(t, String, local.Type)
	require.True(t, local.Nullable)
	require.Equal(t, Any, source.Type)
	require.False(t, source.Nullable)
}

func TestGRPCPresenceAdmissionRejectsNestedNullableAny(t *testing.T) {
	for _, typ := range []DataType{
		&Array{ElemType: &AttributeExpr{Type: Any, Nullable: true}},
		&Map{KeyType: &AttributeExpr{Type: String}, ElemType: &AttributeExpr{Type: Any, Nullable: true}},
	} {
		endpoint := &GRPCEndpointExpr{MethodExpr: &MethodExpr{
			Name: "call", Payload: &AttributeExpr{Type: typ},
		}}
		verr := newValidationErrors()
		endpoint.validateNullableTransport(verr)
		require.ErrorContains(t, verr, "request payloads do not support nullable attributes")
	}
}

func TestGRPCPresenceAdmissionTraversesRecursiveLocalGraphs(t *testing.T) {
	for _, nullable := range []bool{false, true} {
		local := &UserTypeExpr{TypeName: "Local", AttributeExpr: &AttributeExpr{}}
		source := &UserTypeExpr{TypeName: "Source", AttributeExpr: &AttributeExpr{}}
		local.Type = &Object{
			{Name: "next", Attribute: &AttributeExpr{Type: local}},
			{Name: "value", Attribute: &AttributeExpr{Type: String, Nullable: nullable}},
		}
		source.Type = &Object{
			{Name: "next", Attribute: &AttributeExpr{Type: source}},
			{Name: "value", Attribute: &AttributeExpr{Type: String}},
		}
		endpoint := &GRPCEndpointExpr{
			MethodExpr: &MethodExpr{Name: "call", Payload: &AttributeExpr{Type: source}},
			Request:    &AttributeExpr{Type: local},
		}
		verr := newValidationErrors()
		endpoint.validateNullableTransport(verr)
		if nullable {
			require.ErrorContains(t, verr, "request messages do not support nullable attributes")
		} else {
			require.Empty(t, verr.Errors)
		}
		require.Equal(t, nullable, AsObject(local.Type).Attribute("value").Nullable)
		require.False(t, AsObject(source.Type).Attribute("value").Nullable)
	}
}

func TestGRPCPresenceAdmissionRetainsLocalUnionPosition(t *testing.T) {
	local := &AttributeExpr{Type: &Union{TypeName: "Local", Values: []*NamedAttributeExpr{
		{Name: "value", Attribute: &AttributeExpr{Type: String, Nullable: true}},
	}}}
	source := &AttributeExpr{Type: &Union{TypeName: "Source", Values: []*NamedAttributeExpr{
		{Name: "value", Attribute: &AttributeExpr{Type: Any}},
	}}}
	endpoint := &GRPCEndpointExpr{
		MethodExpr: &MethodExpr{Name: "call", Payload: source},
		Request:    local,
	}
	verr := newValidationErrors()
	endpoint.validateNullableTransport(verr)
	require.ErrorContains(t, verr, "request messages do not support nullable attributes")
}
