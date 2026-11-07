package expr

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/eval"
)

func TestTextResponseCodecUsesResponseRawDirection(t *testing.T) {
	union := &Union{TypeName: "Documented", Untagged: true, Values: []*NamedAttributeExpr{
		{Name: "String", Attribute: &AttributeExpr{Type: String}},
		{Name: "Int", Attribute: &AttributeExpr{Type: Int}},
	}}
	for _, tc := range []struct {
		name                    string
		requestRaw, responseRaw bool
		valid                   bool
	}{
		{"raw response documentation", false, true, true},
		{"raw request typed response", true, false, false},
		{"typed response", false, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			endpoint := &HTTPEndpointExpr{SkipRequestBodyEncodeDecode: tc.requestRaw, SkipResponseBodyEncodeDecode: tc.responseRaw}
			response := &HTTPResponseExpr{ContentType: "text/plain", OpenAPIBody: &AttributeExpr{Type: union}}
			errors := new(eval.ValidationErrors)
			response.validateTextContentType(endpoint, errors)
			require.Equal(t, tc.valid, len(errors.Errors) == 0)
		})
	}
}
