package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestSecurityCredentialTypes(t *testing.T) {
	for _, kind := range []string{"username", "password", "key", "jwt", "oauth"} {
		for _, shape := range []string{"plain", "named", "nested_named", "integer", "bytes", "nullable", "custom", "custom_named", "nullable_named"} {
			t.Run(kind+"/"+shape, func(t *testing.T) {
				design := func() {
					var typ expr.DataType = String
					switch shape {
					case "named":
						typ = Type("Credential", String)
					case "nested_named":
						typ = Type("Credential", Type("CredentialBase", String))
					case "custom_named":
						typ = Type("Credential", String, func() {
							Meta("struct:field:type", "custom.Credential", "example.com/custom")
						})
					case "nullable_named":
						typ = Type("Credential", String, func() {
							Nullable()
						})
					case "integer":
						typ = Int
					case "bytes":
						typ = Bytes
					}
					customization := func() {
						if shape == "nullable" {
							Nullable()
						}
						if shape == "custom" {
							Meta("struct:field:type", "custom.Credential", "example.com/custom")
						}
					}
					var scheme *expr.SchemeExpr
					switch kind {
					case "username", "password":
						scheme = BasicAuthSecurity("auth")
					case "key":
						scheme = APIKeySecurity("auth")
					case "jwt":
						scheme = JWTSecurity("auth")
					case "oauth":
						scheme = OAuth2Security("auth")
					}
					Service("credentials", func() {
						Method("call", func() {
							Security(scheme)
							Payload(func() {
								switch kind {
								case "username":
									Username("credential", typ, customization)
									Password("password", String)
								case "password":
									Username("username", String)
									Password("credential", typ, customization)
								case "key":
									APIKey("auth", "credential", typ, customization)
								case "jwt":
									Token("credential", typ, customization)
								case "oauth":
									AccessToken("credential", typ, customization)
								}
							})
						})
					})
				}
				if shape == "plain" || shape == "named" || shape == "nested_named" {
					expr.RunDSL(t, design)
				} else {
					err := expr.RunInvalidDSL(t, design)
					require.ErrorContains(t, err, `security credential attribute "credential" must use String or a named string type without nullability or a custom Go type`)
				}
			})
		}
	}
}
