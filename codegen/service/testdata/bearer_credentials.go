package testdata

import (
	"fmt"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// BearerCredentialsDSL covers protocol extraction independently of credential
// representation, presence, scheme kind and wire mapping.
var BearerCredentialsDSL = func() {
	named := Type("Token", String, func() {
		Pattern("^[a-z]+$")
		MinLength(2)
	})
	jwt := JWTSecurity("jwt")
	oauth := OAuth2Security("oauth", func() {
		AuthorizationCodeFlow("https://example.com/auth", "https://example.com/token", "")
	})
	key := APIKeySecurity("key")
	Service("credentials", func() {
		Meta("openapi:generate", "false")
		for i := range 16 {
			Method(fmt.Sprintf("token%d", i), func() {
				scheme := jwt
				if i&1 != 0 {
					scheme = oauth
				}
				Security(scheme)
				attribute := []string{"token", "value", "vals", "security"}[i/4]
				Payload(func() {
					var typ expr.DataType = String
					if i&2 != 0 {
						typ = named
					}
					validation := func() {
						Pattern("^[a-z]+$")
						MinLength(2)
						Meta("struct:field:name", "Token")
					}
					if scheme == jwt {
						Token(attribute, typ, validation)
					} else {
						AccessToken(attribute, typ, validation)
					}
					if i&4 == 0 {
						Required(attribute)
					}
				})
				Result(String)
				header := "Authorization"
				if i&8 != 0 {
					header = "X-Token"
				}
				HTTP(func() {
					GET(fmt.Sprintf("/token%d", i))
					Header(attribute + ":" + header)
				})
				GRPC(func() {
					Metadata(func() {
						Attribute(attribute + ":" + header)
					})
				})
			})
		}
		Method("key", func() {
			Security(key)
			Payload(func() {
				APIKey("key", "token", String)
				Required("token")
			})
			Result(String)
			HTTP(func() {
				GET("/key")
				Header("token:Authorization")
			})
			GRPC(func() {
				Metadata(func() {
					Attribute("token:Authorization")
				})
			})
		})
	})
}
