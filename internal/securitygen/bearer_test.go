package securitygen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/service"
)

func TestBearerMapping(t *testing.T) {
	for _, tc := range []struct {
		name, attribute, kind string
		want                  bool
	}{
		{"Authorization", "token", "JWT", true},
		{"authorization", "token", "OAuth2", true},
		{"Authorization", "token", "APIKey", false},
		{"X-Token", "token", "JWT", false},
		{"Authorization", "unrelated", "JWT", false},
	} {
		t.Run(tc.name+tc.kind+tc.attribute, func(t *testing.T) {
			require.Equal(t, tc.want, IsBearer(tc.name, tc.attribute, service.SchemesData{
				&service.SchemeData{KeyAttr: "token", Type: tc.kind},
			}))
		})
	}
}
