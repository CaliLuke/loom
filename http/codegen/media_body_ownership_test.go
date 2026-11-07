package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

func TestNameUnionBodiesKeepsMediaBranchOwnership(t *testing.T) {
	for _, response := range []bool{false, true} {
		name := "request"
		if response {
			name = "response"
		}
		t.Run(name, func(t *testing.T) {
			branch := &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: &expr.AttributeExpr{Type: expr.String}}}}
			body := &expr.AttributeExpr{Type: &expr.Union{TypeName: "Source", Values: []*expr.NamedAttributeExpr{{Name: "Form", Attribute: branch}}}}
			variant := &transportir.MediaVariant{Branch: "Form", Body: branch}
			endpoint := &transportir.Endpoint{Name: "submit", Request: &transportir.Request{}, Response: &transportir.Response{}}
			if response {
				endpoint.Response.Responses = []*transportir.ResponseStatus{{Body: body, MediaVariants: []*transportir.MediaVariant{variant}}}
			} else {
				endpoint.Request.Body, endpoint.Request.MediaVariants = body, []*transportir.MediaVariant{variant}
			}
			nameUnionBodies([]*transportir.Endpoint{endpoint})
			emitted := endpoint.Request.Body
			if response {
				emitted = endpoint.Response.Responses[0].Body
			}
			require.NotSame(t, body, emitted)
			require.Same(t, expr.HTTPMediaBranch(emitted, "Form").Attribute, variant.Body, "field projection must modify the emitted media branch")
			require.Equal(t, "Source", body.Type.Name())
		})
	}
}
