package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	servicedata "github.com/CaliLuke/loom/codegen/service/testdata"
)

const credentialConstraintsHarness = `package credentials_test
import (
 "net/http"
 "net/http/httptest"
 "testing"
 svc "example.com/credentials/gen/user"
 client "example.com/credentials/gen/http/user/client"
 server "example.com/credentials/gen/http/user/server"
 loomhttp "github.com/CaliLuke/loom/http"
)
func TestConstrainedCredentials(t *testing.T) {
 for _, input := range []string{"valid", "x", "UPPER"} {
  _, err := client.BuildRequiredPayload(input, input)
  if (err == nil) != (input == "valid") { t.Errorf("required %q: %v", input, err) }
  _, err = client.BuildOptionalPayload(input, input)
  if (err == nil) != (input == "valid") { t.Errorf("optional %q: %v", input, err) }
 }
 for _, tc := range []struct {name string; payload any; encode func(*http.Request,any) error}{
  {"jwt", &svc.JWTPayload{Token: svc.TokenValue("token")}, client.EncodeJWTRequest(nil)},
  {"oauth", &svc.OauthPayload{Token: svc.TokenValue("token")}, client.EncodeOauthRequest(nil)},
 } {
  req := httptest.NewRequest("GET", "/", nil)
  if err := tc.encode(req,tc.payload); err != nil {t.Fatal(err)}
  if got := req.Header.Get("Authorization"); got != "Bearer token" { t.Errorf("%s header = %q",tc.name,got) }
  if tc.name == "jwt" {
   p,err := server.DecodeJWTRequest(loomhttp.NewMuxer(), nil)(req)
   if err != nil || p.Token != "token" {t.Errorf("JWT decoded = %#v, %v",p,err)}
  } else {
   p,err := server.DecodeOauthRequest(loomhttp.NewMuxer(), nil)(req)
   if err != nil || p.Token != "token" {t.Errorf("OAuth decoded = %#v, %v",p,err)}
  }
 }
}
`

func TestConstrainedCredentialsGenerated(t *testing.T) {
	root := RunHTTPDSL(t, servicedata.ConstrainedCredentialsDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/credentials", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "credentials_test.go"), []byte(credentialConstraintsHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}
