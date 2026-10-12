package testdata

import "strings"

// SecurityScopeEndpointTest returns a compiled endpoint regression for scope
// selection. Transport tests compile the same endpoint owner with their adapters.
func SecurityScopeEndpointTest(session bool) string {
	cases := `{"valid", &svc.ReadPayload{User: &user, Password: &pass}, true},
{"invalid", &svc.ReadPayload{}, false},`
	variables := `user, pass := "user", "pass"`
	callbacks := `impl.basic == 2 && impl.jwt == 0 && impl.key == 0`
	if session {
		variables = `token := "session-token"`
		cases = `{"bearer", &svc.ReadPayload{Auth: &token}, true},
{"cookie", &svc.ReadPayload{Cookie: &token}, true},
{"invalid", &svc.ReadPayload{}, false},`
		callbacks = `impl.basic == 0 && impl.jwt == 3 && impl.key == 2`
	}
	return strings.NewReplacer("CASES", cases, "CALLBACKS", callbacks, "VARIABLES", variables).Replace(securityScopeEndpointTest)
}

const securityScopeEndpointTest = `package scope_test

import (
 "context"
 "testing"
 svc "example.com/scope/gen/scope"
 loom "github.com/CaliLuke/loom/pkg"
 "github.com/CaliLuke/loom/security"
)
type endpointImplementation struct {
 svc.Service
 basic, jwt, key, calls int
}
func (s *endpointImplementation) BasicAuth(ctx context.Context, user, pass string, _ *security.BasicScheme) (context.Context, error) {
 s.basic++
 if user == "user" && pass == "pass" {
  return ctx, nil
 }
 return ctx, loom.PermanentError("unauthorized", "invalid")
}
func (s *endpointImplementation) JWTAuth(ctx context.Context, token string, _ *security.JWTScheme) (context.Context, error) {
 s.jwt++
 if token == "session-token" {
  return ctx, nil
 }
 return ctx, loom.PermanentError("unauthorized", "invalid")
}
func (s *endpointImplementation) APIKeyAuth(ctx context.Context, token string, _ *security.APIKeyScheme) (context.Context, error) {
 s.key++
 if token == "session-token" {
  return ctx, nil
 }
 return ctx, loom.PermanentError("unauthorized", "invalid")
}
func (s *endpointImplementation) Read(context.Context, *svc.ReadPayload) error {
 s.calls++
 return nil
}
func TestSelectedEndpoint(t *testing.T) {
 impl := &endpointImplementation{}
 endpoint := svc.NewEndpoints(impl).Read
 VARIABLES
 for _, tc := range []struct { name string; payload *svc.ReadPayload; want bool }{
 CASES
 } {
  t.Run(tc.name, func(t *testing.T) {
   before := impl.calls
   _, err := endpoint(context.Background(), tc.payload)
   if (err == nil) != tc.want || (impl.calls > before) != tc.want {
    t.Errorf("err %v, service called %v", err, impl.calls > before)
   }
  })
 }
 if !(CALLBACKS) {
  t.Errorf("callbacks basic=%d jwt=%d key=%d", impl.basic, impl.jwt, impl.key)
 }
}
`
