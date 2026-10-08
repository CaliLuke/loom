package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	servicedata "github.com/CaliLuke/loom/codegen/service/testdata"
)

const credentialFieldsHarness = `package credentials_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	svc "example.com/credentials/gen/credentials"
	client "example.com/credentials/gen/http/credentials/client"
	server "example.com/credentials/gen/http/credentials/server"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

type contextKey struct{}
type implementation struct{ calls *int }

func (s implementation) result(ctx context.Context) (string, error) {
	*s.calls++
	if ctx.Value(contextKey{}) != 4 {
		return "", errors.New("lost authentication context")
	}
	return "ok", nil
}
func (s implementation) Required(ctx context.Context, p *svc.RequiredPayload) (string, error) {
	return s.result(ctx)
}
func (s implementation) Optional(ctx context.Context, p *svc.OptionalPayload) (string, error) {
	return s.result(ctx)
}
func (s implementation) Ordinary(ctx context.Context, p *svc.OrdinaryPayload) (string, error) {
	return s.result(ctx)
}

func eraseDecoder[T any](decode func(*http.Request) (T, error)) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		return decode(r)
	}
}

func TestCredentials(t *testing.T) {
	for _, reject := range []bool{false, true} {
		calls, authCalls := 0, 0
		failure := errors.New("authentication rejected")
		expected := []string{"user", "pass", "key", "token", "access"}
		next := func(ctx context.Context, value string, index int, scopes []string) (context.Context, error) {
			if value != expected[index] {
				t.Errorf("credential %d = %q", index, value)
			}
			if !reflect.DeepEqual(scopes, []string{"read"}) {
				t.Errorf("scopes = %v", scopes)
			}
			authCalls++
			previous, _ := ctx.Value(contextKey{}).(int)
			if previous != authCalls-1 {
				t.Errorf("context = %v, call = %d", previous, authCalls)
			}
			if reject {
				return ctx, failure
			}
			return context.WithValue(ctx, contextKey{}, authCalls), nil
		}
		basic := func(ctx context.Context, user, pass string, scheme *security.BasicScheme) (context.Context, error) {
			if pass != expected[1] {
				t.Errorf("password = %q", pass)
			}
			return next(ctx, user, 0, scheme.RequiredScopes)
		}
		key := func(ctx context.Context, value string, scheme *security.APIKeyScheme) (context.Context, error) {
			return next(ctx, value, 2, scheme.RequiredScopes)
		}
		jwt := func(ctx context.Context, value string, scheme *security.JWTScheme) (context.Context, error) {
			return next(ctx, value, 3, scheme.RequiredScopes)
		}
		oauth := func(ctx context.Context, value string, scheme *security.OAuth2Scheme) (context.Context, error) {
			return next(ctx, value, 4, scheme.RequiredScopes)
		}
		impl := implementation{&calls}
		endpoints := &svc.Endpoints{
			Required: svc.NewRequiredEndpoint(impl, basic, key, jwt, oauth),
			Optional: svc.NewOptionalEndpoint(impl, basic, key, jwt, oauth),
			Ordinary: svc.NewOrdinaryEndpoint(impl, basic, key, jwt, oauth),
		}
		user, pass, k, token, access := "user", "pass", "key", "token", "access"
		mux := loomhttp.NewMuxer()
		for _, tc := range []struct {
			name     string
			payload  any
			encode   func(*http.Request, any) error
			decode   func(*http.Request) (any, error)
			endpoint loom.Endpoint
		}{
			{"required", &svc.RequiredPayload{Login: user, Secret: pass, AccessKey: k, JWTValue: token, OAuthValue: access}, client.EncodeRequiredRequest(loomhttp.RequestEncoder), eraseDecoder(server.DecodeRequiredRequest(mux, loomhttp.RequestDecoder)), endpoints.Required},
			{"optional", &svc.OptionalPayload{Login: &user, Secret: &pass, AccessKey: &k, JWTValue: &token, OAuthValue: &access}, client.EncodeOptionalRequest(loomhttp.RequestEncoder), eraseDecoder(server.DecodeOptionalRequest(mux, loomhttp.RequestDecoder)), endpoints.Optional},
			{"ordinary", &svc.OrdinaryPayload{User: user, Pass: pass, Key: k, Token: token, Access: access}, client.EncodeOrdinaryRequest(loomhttp.RequestEncoder), eraseDecoder(server.DecodeOrdinaryRequest(mux, loomhttp.RequestDecoder)), endpoints.Ordinary},
		} {
			t.Run(tc.name, func(t *testing.T) {
				authCalls = 0
				before := calls
				req := httptest.NewRequest("GET", "/"+tc.name, nil)
				if err := tc.encode(req, tc.payload); err != nil {
					t.Fatal(err)
				}
				if req.Header.Get("X-Key") != "key" {
					t.Errorf("key header = %q", req.Header.Get("X-Key"))
				}
				decoded, err := tc.decode(req)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(decoded, tc.payload) {
					t.Errorf("payload = %#v, want %#v", decoded, tc.payload)
				}
				result, err := tc.endpoint(context.Background(), decoded)
				if reject {
					if !errors.Is(err, failure) || calls != before {
						t.Errorf("error = %v, calls = %d", err, calls)
					}
				} else if err != nil || result != "ok" || calls != before+1 || authCalls != 4 {
					t.Errorf("result = %v, error = %v, calls = %d, auth calls = %d", result, err, calls, authCalls)
				}
			})
		}
	}
}

func TestCredentialCLI(t *testing.T) {
	required, err := client.BuildRequiredPayload("key", "token", "access", "user", "pass")
	if err != nil {
		t.Fatal(err)
	}
	if string(required.Login) != "user" || string(required.Secret) != "pass" || string(required.AccessKey) != "key" || string(required.JWTValue) != "token" || string(required.OAuthValue) != "access" {
		t.Errorf("required CLI payload = %#v", required)
	}
	optional, err := client.BuildOptionalPayload("", "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if optional.Login != nil || optional.Secret != nil || optional.AccessKey != nil || optional.JWTValue != nil || optional.OAuthValue != nil {
		t.Errorf("omitted CLI values became present: %#v", optional)
	}
}
`

func TestCredentialFieldNamesGenerated(t *testing.T) {
	root := RunHTTPDSL(t, servicedata.CredentialFieldNamesDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/credentials", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "credentials_test.go"), []byte(credentialFieldsHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "-count=1", "./...")
}

func TestNamedCredentialFieldsGenerated(t *testing.T) {
	root := RunHTTPDSL(t, servicedata.NamedCredentialFieldsDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/credentials", root)
	harness := strings.Replace(credentialFieldsHarness,
		`user, pass, k, token, access := "user", "pass", "key", "token", "access"`,
		`user, pass, k, token, access := svc.Credential("user"), svc.Credential("pass"), svc.Credential("key"), svc.Credential("token"), svc.Credential("access")`, 1)
	harness = strings.Replace(harness,
		`&svc.OrdinaryPayload{User: user, Pass: pass, Key: k, Token: token, Access: access}`,
		`&svc.OrdinaryPayload{User: string(user), Pass: string(pass), Key: string(k), Token: string(token), Access: string(access)}`, 1)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "credentials_test.go"), []byte(harness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "-count=1", "./...")
}
