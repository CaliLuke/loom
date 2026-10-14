package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestSessionPayloadIdentityGenerated(t *testing.T) {
	for _, kind := range []string{"absent", "empty", "anonymous", "named"} {
		t.Run(kind, func(t *testing.T) {
			root := RunHTTPDSL(t, func() {
				API("session-identity", func() {
				})
				jwt := JWTSecurity("jwt")
				session := SessionAuth("session", func() {
					BearerTransport(jwt, "auth")
				})
				payload := Type("ProfileInput", func() {
					Attribute("name", String)
				})
				Service("profiles", func() {
					Method("read", func() {
						SessionSecurity(session)
						switch kind {
						case "empty":
							Payload(Empty)
						case "anonymous":
							Payload(func() {
								Attribute("name", String)
							})
						case "named":
							Payload(payload)
						}
						Result(String)
						HTTP(func() {
							POST("/profile")
						})
					})
				})
			})
			require.NotNil(t, root.Services[0].Methods[0].Payload.Find("auth"))
			dir := t.TempDir()
			renderHTTPModule(t, dir, "example.com/sessionidentity", root)
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "build", "./...")
			runGoCommand(t, dir, "vet", "./...")
		})
	}
}
