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

func TestNamedCredentialConversions(t *testing.T) {
	root := codegen.RunDSL(t, testdata.NamedCredentialFieldsDSL)
	data := endpointData(NewServicesData(root).Get("credentials"))
	for _, method := range data.Methods {
		if method.Name == "ordinary" {
			continue
		}
		for _, scheme := range method.Schemes {
			if scheme.Type == "Basic" {
				require.Equal(t, "Credential", scheme.UsernameType.Name())
				require.Equal(t, "Credential", scheme.PasswordType.Name())
				require.Equal(t, scheme.UsernameType, scheme.Dup().UsernameType)
			} else {
				require.Equal(t, "Credential", scheme.CredType.Name())
				require.Equal(t, scheme.CredType, scheme.Dup().CredType)
			}
		}
		rendered := codegen.SectionCode(t, endpointMethodSection(method))
		for _, field := range []string{"Login", "Secret", "AccessKey", "JWTValue", "OAuthValue"} {
			value := "p." + field
			if method.Name == "optional" {
				value = "*" + value
			}
			require.Contains(t, rendered, "string("+value+")")
		}
	}
}
