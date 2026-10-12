package expr

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/eval"
)

func TestHTTPWireBindingRequirementMembership(t *testing.T) {
	for _, together := range []bool{false, true} {
		t.Run(map[bool]string{false: "alternatives", true: "conjunction"}[together], func(t *testing.T) {
			SetupTestDSL(t)
			basic := &SchemeExpr{Kind: BasicAuthKind, SchemeName: "basic"}
			jwt := &SchemeExpr{Kind: JWTKind, SchemeName: "jwt"}
			payload := &AttributeExpr{Type: &Object{
				{Name: "user", Attribute: &AttributeExpr{Type: String, Meta: MetaExpr{"security:username": {""}}}},
				{Name: "pass", Attribute: &AttributeExpr{Type: String, Meta: MetaExpr{"security:password": {""}}}},
				{Name: "token", Attribute: &AttributeExpr{Type: String, Meta: MetaExpr{"security:token": {""}}}},
			}}
			requirements := []*SecurityExpr{{Schemes: []*SchemeExpr{basic}}, {Schemes: []*SchemeExpr{jwt}}}
			if together {
				requirements = []*SecurityExpr{{Schemes: []*SchemeExpr{basic, jwt}}, {Schemes: []*SchemeExpr{basic}}}
			}
			method := &MethodExpr{Name: "read", Payload: payload, Requirements: requirements}
			endpoint := &HTTPEndpointExpr{MethodExpr: method}
			plan := endpoint.ensureWireBindings()
			require.Len(t, plan.bindings, 2)
			verr := new(eval.ValidationErrors)
			plan.validate(endpoint, verr)
			if together {
				require.Len(t, verr.Errors, 1, "an OR branch must not hide the common AND requirement")
			} else {
				require.Empty(t, verr.Errors)
			}
			require.Empty(t, jwt.Name, "resolving the HTTP occurrence must not alter the selected declaration")
			require.Empty(t, basic.Name)
			require.Same(t, plan, endpoint.ensureWireBindings())
		})
	}
}

func TestHTTPWireBindingErrorProjectionOwnership(t *testing.T) {
	for _, body := range []string{"full", "empty", "selected"} {
		t.Run(body, func(t *testing.T) {
			SetupTestDSL(t)
			response := &HTTPResponseExpr{Headers: NewEmptyMappedAttributeExpr()}
			switch body {
			case "empty":
				response.Body = &AttributeExpr{Type: Empty}
			case "selected":
				response.Body = &AttributeExpr{Type: String, Meta: MetaExpr{"origin:attribute": {"detail"}}}
			}
			authored := response.Headers
			response.prepareErrorWireBindings(&AttributeExpr{Type: ErrorResult})
			verr := new(eval.ValidationErrors)
			response.validateWireBindings(verr)
			require.Empty(t, verr.Errors)
			require.True(t, authored.IsEmpty())
			plan := response.wireBindings
			mapping := response.Headers
			response.prepareErrorWireBindings(&AttributeExpr{Type: ErrorResult})
			response.validateWireBindings(verr)
			require.Same(t, plan, response.wireBindings)
			require.Same(t, mapping, response.Headers)
			if body == "full" {
				require.Empty(t, plan.bindings)
			} else {
				require.NotEmpty(t, plan.bindings)
				if body == "selected" {
					require.Nil(t, response.Headers.Find("detail"))
				}
			}
		})
	}
}
