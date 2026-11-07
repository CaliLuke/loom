package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormFieldNullAdmissionFollowsCodec(t *testing.T) {
	for _, tc := range []struct {
		name, media string
		accept      bool
	}{
		{"JSON", "application/json", true},
		{"text", "text/plain", false},
		{"implicit text", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := &AttributeExpr{Type: &Object{{Name: "value", Attribute: &AttributeExpr{Type: &Union{Untagged: true, Values: []*NamedAttributeExpr{{Name: "text", Attribute: &AttributeExpr{Type: String}}, {Name: "integer", Attribute: &AttributeExpr{Type: Int}}}}, Nullable: true}}}, Meta: MetaExpr{}}
			if tc.media != "" {
				body.Meta["openapi:encoding:value:contentType"] = []string{tc.media}
			}
			err := validateMediaFields(body, "application/x-www-form-urlencoded")
			if tc.accept {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
