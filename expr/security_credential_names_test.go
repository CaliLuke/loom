package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestSecurityCredentialNames rejects mapped credential declarations before
// security code generation while retaining ordinary mapped attributes and
// explicit transport mappings of credentials without suffixes.
func TestSecurityCredentialNames(t *testing.T) {
	for _, kind := range []string{"username", "password", "apikey", "token", "accesstoken"} {
		for _, shape := range []string{"inline", "named", "inherited", "reference"} {
			for _, numbered := range []bool{false, true} {
				name := kind + "/" + shape
				if numbered {
					name += "/numbered"
				}
				t.Run(name, func(t *testing.T) {
					for _, explicit := range []bool{false, true} {
						err := expr.RunInvalidDSL(t, credentialNameDSL(kind, shape, numbered, explicit, "cred:wire"))
						require.ErrorContains(t, err, `security credential attribute "cred:wire" must not use a mapping suffix`)
						expr.RunDSL(t, credentialNameDSL(kind, shape, numbered, explicit, "cred"))
					}
				})
			}
		}
	}
}

func credentialNameDSL(kind, shape string, numbered, explicit bool, name string) func() {
	return func() {
		var scheme *expr.SchemeExpr
		switch kind {
		case "username", "password":
			scheme = BasicAuthSecurity("auth")
		case "apikey":
			scheme = APIKeySecurity("auth")
		case "token":
			scheme = JWTSecurity("auth")
		case "accesstoken":
			scheme = OAuth2Security("auth")
		}
		fields := func() {
			credentialNameField(kind, numbered, name)
			if kind == "username" {
				PasswordField(2, "password", String)
			}
			if kind == "password" {
				UsernameField(2, "username", String)
			}
			Field(3, "label:display", String)
		}
		var named expr.UserType
		if shape != "inline" {
			named = Type("Credentials", fields)
		}
		Service("svc", func() {
			Method("m", func() {
				Security(scheme)
				switch shape {
				case "inline":
					Payload(fields)
				case "named":
					Payload(named)
				case "reference":
					Payload(func() {
						Reference(named)
						Attribute(name)
						if kind == "username" {
							Attribute("password")
						}
						if kind == "password" {
							Attribute("username")
						}
					})
				case "inherited":
					Payload(func() {
						Extend(named)
					})
				}
				HTTP(func() {
					POST("/m")
					if explicit {
						Header("cred:X-Credential")
					}
				})
			})
		})
	}
}

func credentialNameField(kind string, numbered bool, name string) {
	if numbered {
		switch kind {
		case "username":
			UsernameField(1, name, String)
		case "password":
			PasswordField(1, name, String)
		case "apikey":
			APIKeyField(1, "auth", name, String)
		case "token":
			TokenField(1, name, String)
		case "accesstoken":
			AccessTokenField(1, name, String)
		}
		return
	}
	switch kind {
	case "username":
		Username(name, String)
	case "password":
		Password(name, String)
	case "apikey":
		APIKey("auth", name, String)
	case "token":
		Token(name, String)
	case "accesstoken":
		AccessToken(name, String)
	}
}

// TestSecurityCredentialUnusedReference keeps reference fields that are not
// selected into the payload outside the method's credential validation.
func TestSecurityCredentialUnusedReference(t *testing.T) {
	expr.RunDSL(t, func() {
		source := Type("Source", func() {
			Token("unused:wire", String)
			Attribute("label", String)
		})
		Service("svc", func() {
			Method("m", func() {
				Payload(func() {
					Reference(source)
					Attribute("label")
				})
				HTTP(func() {
					POST("/m")
				})
			})
		})
	})
}
