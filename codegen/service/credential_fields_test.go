package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service/testdata"
)

func TestCredentialFieldNames(t *testing.T) {
	root := codegen.RunDSL(t, testdata.CredentialFieldNamesDSL)
	services := NewServicesData(root)
	for _, name := range []string{"required", "optional", "ordinary"} {
		t.Run(name, func(t *testing.T) {
			method := services.Get("credentials").Method(name)
			require.NotNil(t, method)
			fields := []string{"Login", "Secret", "AccessKey", "JWTValue", "OAuthValue"}
			if name == "ordinary" {
				fields = []string{"User", "Pass", "Key", "Token", "Access"}
			}
			basic := method.Requirements.Scheme("credential_basic")
			require.Equal(t, fields[0], basic.UsernameField)
			require.Equal(t, fields[1], basic.PasswordField)
			require.Equal(t, name == "optional", basic.UsernamePointer)
			require.Equal(t, name == "optional", basic.PasswordPointer)
			for i, scheme := range []string{"key", "credential_jwt", "oauth"} {
				data := method.Requirements.Scheme(scheme)
				require.Equal(t, fields[i+2], data.CredField)
				require.Equal(t, name == "optional", data.CredPointer)
				require.Equal(t, name != "optional", data.CredRequired)
			}
		})
	}
}
