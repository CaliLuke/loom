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
 "context"
 "encoding/json/v2"
 "github.com/stretchr/testify/require"
 "github.com/stretchr/testify/assert"
 "github.com/CaliLuke/loom/security"
 "net/http"
 "net/http/httptest"
 "testing"
 svc "example.com/credentials/gen/user"
 client "example.com/credentials/gen/http/user/client"
 server "example.com/credentials/gen/http/user/server"
 loomhttp "github.com/CaliLuke/loom/http"
)
type basicImplementation struct {
 svc.Service
 authCalls, serviceCalls int
 optional *svc.OptionalPayload
}
func (s *basicImplementation) BasicAuth(ctx context.Context, _, _ string, _ *security.BasicScheme) (context.Context, error) {
 s.authCalls++
 return ctx, nil
}
func (s *basicImplementation) Required(context.Context, *svc.RequiredPayload) error {
 s.serviceCalls++
 return nil
}
func (s *basicImplementation) Optional(_ context.Context, p *svc.OptionalPayload) error {
 s.serviceCalls++
 s.optional = p
 return nil
}
func TestBasicConstraintWire(t *testing.T) {
 for _, method := range []string{"required", "optional"} {
  for _, tc := range []struct {name, user, pass, code string; absent bool}{
   {name:"valid", user:"alice", pass:"secret"},
   {name:"short username", user:"x", pass:"secret", code:"invalid_length"},
   {name:"short password", user:"alice", pass:"x", code:"invalid_length"},
   {name:"username pattern", user:"UPPER", pass:"secret", code:"invalid_pattern"},
   {name:"password pattern", user:"alice", pass:"UPPER", code:"invalid_pattern"},
   {name:"supplied empty", user:"alice", code:"invalid_pattern"},
   {name:"absent", absent:true},
  } {
   t.Run(method+"/"+tc.name, func(t *testing.T) {
    impl := &basicImplementation{}
    mux := loomhttp.NewMuxer()
    server.Mount(mux, server.New(&svc.Endpoints{Required: svc.NewRequiredEndpoint(impl, impl.BasicAuth), Optional: svc.NewOptionalEndpoint(impl, impl.BasicAuth)}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
    req := httptest.NewRequest(http.MethodGet, "/"+method, nil)
    if !tc.absent {
     req.SetBasicAuth(tc.user, tc.pass)
    }
    rec := httptest.NewRecorder()
    mux.ServeHTTP(rec, req)
    code := tc.code
    if tc.absent && method == "required" {
     code = "missing_field"
    }
    if code != "" {
     assert.Zero(t, impl.authCalls)
     assert.Zero(t, impl.serviceCalls)
     require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
     var problem map[string]any
     require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &problem))
     if tc.name != "supplied empty" {
      require.Equal(t, code, problem["code"])
     }
     require.NotEmpty(t, problem["detail"])
    } else {
     require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
     require.Equal(t, 1, impl.authCalls)
     require.Equal(t, 1, impl.serviceCalls)
     if method == "optional" {
      if tc.absent {
       require.Nil(t, impl.optional.User)
       require.Nil(t, impl.optional.Pass)
      } else {
       require.Equal(t, svc.Credential(tc.user), *impl.optional.User)
       require.Equal(t, svc.Credential(tc.pass), *impl.optional.Pass)
      }
     }
    }
   })
  }
 }
}
func TestConstrainedCredentials(t *testing.T) {
 for _, input := range []string{"valid", "x", "UPPER"} {
  _, err := client.BuildRequiredPayload(input, input)
  if (err == nil) != (input == "valid") {
t.Errorf("required %q: %v", input, err)
}
  _, err = client.BuildOptionalPayload(input, input)
  if (err == nil) != (input == "valid") {
t.Errorf("optional %q: %v", input, err)
}
 }
 for _, tc := range []struct {name string; payload any; encode func(*http.Request,any) error}{
  {"jwt", &svc.JWTPayload{Token: svc.TokenValue("token")}, client.EncodeJWTRequest(nil)},
  {"oauth", &svc.OauthPayload{Token: svc.TokenValue("token")}, client.EncodeOauthRequest(nil)},
 } {
  req := httptest.NewRequest("GET", "/", nil)
  if err := tc.encode(req,tc.payload); err != nil {
t.Fatal(err)
}
  if got := req.Header.Get("Authorization"); got != "Bearer token" {
t.Errorf("%s header = %q",tc.name,got)
}
  if tc.name == "jwt" {
   p,err := server.DecodeJWTRequest(loomhttp.NewMuxer(), nil)(req)
   if err != nil || p.Token != "token" {
t.Errorf("JWT decoded = %#v, %v",p,err)
}
  } else {
   p,err := server.DecodeOauthRequest(loomhttp.NewMuxer(), nil)(req)
   if err != nil || p.Token != "token" {
t.Errorf("OAuth decoded = %#v, %v",p,err)
}
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

func TestBasicConstraintsParticipateInRequestPlan(t *testing.T) {
	data := CreateHTTPServices(RunHTTPDSL(t, servicedata.ConstrainedCredentialsDSL)).Get("user")
	for _, name := range []string{"required", "optional"} {
		t.Run(name, func(t *testing.T) {
			request := data.Endpoint(name).Payload.Request
			require.True(t, request.DecodePlan.MustValidate, "Basic attributes must participate in the shared validation barrier")
		})
	}
}
