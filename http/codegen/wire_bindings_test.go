package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

// The owner rejects ambiguous designs; these accepted mappings must preserve values.
func TestHTTPWireBindingGenerated(t *testing.T) {
	root := RunHTTPDSL(t, testdata.EffectiveWireBindingsDSL("http"))
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/binding", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "binding_test.go"), []byte(wireBindingHarness), 0o600))
	runGoCommand(t, dir, "mod", "edit", "-require=github.com/CaliLuke/loom@v1.10.0-alpha.9.0.20261012201303-8c7d8d711b9e")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "auth_test.go"), []byte(testdata.EffectiveWireBindingsHarness("http")), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "build", "./...")
	runGoCommand(t, dir, "vet", "./...")
	t.Log(runCommand(t, dir, "go", "test", "-v", "."))
}

const wireBindingHarness = `package binding_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	client "example.com/binding/gen/http/binding/client"
	server "example.com/binding/gen/http/binding/server"
	svc "example.com/binding/gen/binding"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/require"
)

func erase[T any](decode func(*http.Request) (T, error)) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		return decode(r)
	}
}
func pointer(s string) *string {
	return &s
}
func TestWireBindings(t *testing.T) {
	mux := loomhttp.NewMuxer()
	for _, tc := range []struct {
		name string
		payload any
		encode func(*http.Request, any) error
		decode func(*http.Request) (any, error)
	}{
		{"header", &svc.HeaderPayload{A:"first", B:"second"}, client.EncodeHeaderRequest(nil), erase(server.DecodeHeaderRequest(mux,nil))},
		{"query", &svc.QueryPayload{A:"first", B:"second"}, client.EncodeQueryRequest(nil), erase(server.DecodeQueryRequest(mux,nil))},
		{"cookie", &svc.CookiePayload{A:"first", B:"second"}, client.EncodeCookieRequest(nil), erase(server.DecodeCookieRequest(mux,nil))},
		{"and-key", &svc.AndKeyPayload{Token:"token", Key:"key"}, client.EncodeAndKeyRequest(nil), erase(server.DecodeAndKeyRequest(mux,nil))},
		{"or-key-token", &svc.OrKeyPayload{Token:pointer("token")}, client.EncodeOrKeyRequest(nil), erase(server.DecodeOrKeyRequest(mux,nil))},
		{"or-key-key", &svc.OrKeyPayload{Key:pointer("key")}, client.EncodeOrKeyRequest(nil), erase(server.DecodeOrKeyRequest(mux,nil))},
		{"shared-frame", &svc.SharedFramePayload{Token:pointer("token")}, client.EncodeSharedFrameRequest(nil), erase(server.DecodeSharedFrameRequest(mux,nil))},
		{"either-basic", &svc.EitherPayload{Username:pointer("user"), Password:pointer("pass")}, client.EncodeEitherRequest(nil), erase(server.DecodeEitherRequest(mux,nil))},
		{"either-bearer", &svc.EitherPayload{Token:pointer("token")}, client.EncodeEitherRequest(nil), erase(server.DecodeEitherRequest(mux,nil))},
		{"explicit-basic", &svc.ExplicitPayload{Username:pointer("user"), Password:pointer("pass")}, client.EncodeExplicitRequest(nil), erase(server.DecodeExplicitRequest(mux,nil))},
		{"explicit-bearer", &svc.ExplicitPayload{Token:pointer("token")}, client.EncodeExplicitRequest(nil), erase(server.DecodeExplicitRequest(mux,nil))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", nil)
			require.NoError(t, tc.encode(req,tc.payload))
			decoded, err := tc.decode(req)
			t.Logf("wire header=%v query=%v; decoded=%#v error=%v", req.Header, req.URL.Query(), decoded, err)
			require.NoError(t, err)
			require.Equal(t, tc.payload, decoded)
		})
	}
}
func TestResponseWireBindings(t *testing.T) {
	value := &svc.ResponseResult{A:"first", B:"second"}
	recorder := httptest.NewRecorder()
	require.NoError(t, server.EncodeResponseResponse(loomhttp.ResponseEncoder)(context.Background(), recorder, value))
	decoded, err := client.DecodeResponseResponse(loomhttp.ResponseDecoder, false)(recorder.Result())
	t.Logf("headers=%v decoded=%#v error=%v", recorder.Header(), decoded, err)
	require.NoError(t, err)
	require.Equal(t, value, decoded)
}

`

func TestHTTPWireBindingScopeGenerated(t *testing.T) {
	root := RunHTTPDSL(t, testdata.EffectiveWireBindingScopeDSL("http"))
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/scope", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "scope_test.go"), []byte(testdata.EffectiveWireBindingScopeHarness("http")), 0o600))
	runGoCommand(t, dir, "mod", "edit", "-require=github.com/CaliLuke/loom@v1.10.0-alpha.9.0.20261012201303-8c7d8d711b9e")
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "-race", "./...")
	runGoCommand(t, dir, "vet", "./...")
}

func TestHTTPExplicitAuthorizationWireBindings(t *testing.T) {
	root := RunHTTPDSL(t, testdata.ExplicitAuthorizationWireBindingsDSL("http"))
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/binding", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "auth_test.go"), []byte(testdata.EffectiveWireBindingsHarness("http")), 0o600))
	runGoCommand(t, dir, "mod", "edit", "-require=github.com/CaliLuke/loom@v1.10.0-alpha.9.0.20261012201303-8c7d8d711b9e")
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "-race", "./...")
	runGoCommand(t, dir, "vet", "./...")
}
