package testdata

import "strings"

// CredentialAlternativesHTTPHarness checks HTTP and JSON-RPC POST requests
// against the same credential, callback and service contract.
func CredentialAlternativesHTTPHarness(transport string) string {
	code := strings.ReplaceAll(credentialAlternativesHTTPHarness, "TRANSPORT", transport)
	decodeArgs := ""
	extraImports := ""
	setup := `server.Mount(mux, server.New(svc.NewEndpoints(impl), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))`
	request := `request := httptest.NewRequest("GET", "/"+tc.method, nil)`
	response := `if tc.decodeError != "" {
 require.Equal(t, 400, recorder.Code, recorder.Body.String())
 } else if tc.success {
 require.Equal(t, 200, recorder.Code, recorder.Body.String())
 } else {
 require.Equal(t, 500, recorder.Code, recorder.Body.String())
 }`
	if transport == "jsonrpc" {
		decodeArgs = `, &jsonrpc.RawRequest{Params: []byte("{}")}`
		extraImports = `"strings"
 "github.com/CaliLuke/loom/jsonrpc"`
		setup = `server.Mount(mux, server.New(svc.NewEndpoints(impl), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))`
		request = `request := httptest.NewRequest("POST", "/", strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\""+tc.method+"\",\"params\":{}}"))
 request.Header.Set("Content-Type", "application/json")`
		response = `require.Equal(t, 200, recorder.Code, recorder.Body.String())
 if tc.success {
 require.Contains(t, recorder.Body.String(), "\"result\":\"ok\"")
 } else {
 if tc.decodeError != "" {
 require.Contains(t, recorder.Body.String(), "\"code\":-32602")
 } else {
 require.Contains(t, recorder.Body.String(), "\"code\":-32603")
 }
 }`
	}
	return strings.NewReplacer("IMPLEMENTATION", credentialAlternativesImplementation, "EXTRA_IMPORTS", extraImports, "DECODE_ARGS", decodeArgs, "SETUP", setup, "REQUEST", request, "RESPONSE", response).Replace(code)
}

const credentialAlternativesHTTPHarness = `package alternatives_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	server "example.com/alternatives/gen/TRANSPORT/alternatives/server"
	svc "example.com/alternatives/gen/alternatives"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
	"github.com/stretchr/testify/require"
	EXTRA_IMPORTS
)

IMPLEMENTATION

func TestCredentialAlternatives(t *testing.T) {
	mux := loomhttp.NewMuxer()
	decoders := map[string]func(*http.Request) (any, error){
		"either": func(r *http.Request) (any, error) {
			return server.DecodeEitherRequest(mux, nil)(r DECODE_ARGS)
		},
		"reverse": func(r *http.Request) (any, error) {
			return server.DecodeReverseRequest(mux, nil)(r DECODE_ARGS)
		},
		"key":    func(r *http.Request) (any, error) {
			return server.DecodeKeyRequest(mux, nil)(r DECODE_ARGS)
		},
		"and":    func(r *http.Request) (any, error) {
			return server.DecodeAndRequest(mux, nil)(r DECODE_ARGS)
		},
		"single": func(r *http.Request) (any, error) {
			return server.DecodeSingleRequest(mux, nil)(r DECODE_ARGS)
		},
		"required": func(r *http.Request) (any, error) {
			return server.DecodeRequiredRequest(mux, nil)(r DECODE_ARGS)
		},
		"required_token": func(r *http.Request) (any, error) {
			return server.DecodeRequiredTokenRequest(mux, nil)(r DECODE_ARGS)
		},
		"anonymous": func(r *http.Request) (any, error) {
			return server.DecodeAnonymousRequest(mux, nil)(r DECODE_ARGS)
		},
	}
	for _, tc := range []struct {
		method, header, key, decodeError, auth string
		context                                any
		success, tokenPresent                  bool
	}{
		{"either", "Basic dXNlcjpwYXNz", "", "", "B", "B", true, false},
		{"reverse", "Basic dXNlcjpwYXNz", "", "", "JB", "B", true, false},
		{"either", "bAsIc dXNlcjpwYXNz", "", "", "B", "B", true, false},
		{"either", "Bearer good", "", "", "BJ", "J", true, true},
		{"key", "Basic dXNlcjpwYXNz", "key", "", "JK", "K", true, false},
		{"key", "Digest challenge", "key", "", "JK", "K", true, false},
		{"key", "Bearer wrong", "key", "", "JK", "K", true, true},
		{"and", "Bearer good", "key", "", "JK", "JK", true, true},
		{"and", "Bearer wrong", "key", "", "J", nil, false, true},
		{"and", "Basic dXNlcjpwYXNz", "key", "decode_payload", "", nil, false, false},
		{"single", "Basic dXNlcjpwYXNz", "", "decode_payload", "", nil, false, false},
		{"required", "Basic dXNlcjpwYXNz", "", "", "B", "B", true, false},
		{"anonymous", "Bearer ", "", "decode_payload", "", nil, false, false},
		{"required", "Bearer good", "", "missing_field", "", nil, false, false},
		{"required_token", "Basic dXNlcjpwYXNz", "key", "decode_payload", "", nil, false, false},
		{"anonymous", "Basic dXNlcjpwYXNz", "", "", "J", nil, true, false},
		{"either", "", "", "", "BJ", nil, false, false},
		{"key", "Bearer UPPER", "key", "invalid_pattern", "", nil, false, false},
		{"required_token", "", "key", "missing_field", "", nil, false, false},
		{"key", "<empty>", "key", "decode_payload", "", nil, false, false},
		{"key", "raw-token", "key", "decode_payload", "", nil, false, false},
		{"key", "Bearer", "key", "decode_payload", "", nil, false, false},
		{"key", "Bearer ", "key", "decode_payload", "", nil, false, false},
		{"key", "Bearer a b", "key", "decode_payload", "", nil, false, false},
	} {
		t.Run(tc.method+"/"+tc.header, func(t *testing.T) {
			REQUEST
			if tc.header != "" {
				if tc.header == "<empty>" {
					request.Header.Set("Authorization", "")
				} else {
					request.Header.Set("Authorization", tc.header)
				}
			}
			if tc.key != "" {
				request.Header.Set("X-Key", tc.key)
			}
			payload, err := decoders[tc.method](request.Clone(context.Background()))
			if tc.decodeError != "" {
				var detail *loom.ServiceError
				require.ErrorAs(t, err, &detail)
				require.Equal(t, tc.decodeError, detail.Name)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.tokenPresent, !reflect.ValueOf(payload).Elem().FieldByName("Token").IsNil())
			}
			impl := &implementation{}
			mux := loomhttp.NewMuxer()
			SETUP
			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, request)
			RESPONSE
			require.Equal(t, tc.success, impl.calls == 1, recorder.Body.String())
			require.Equal(t, tc.auth, join(impl.auth))
			require.Equal(t, tc.context, impl.context)
			if tc.success {
				require.Equal(t, payload, impl.payload)
			}
			for i, previous := range impl.seen {
				if tc.method == "and" && i > 0 {
					require.Equal(t, "J", previous)
				} else {
					require.Nil(t, previous, "failed alternative context leaked")
				}
			}
		})
	}
}
func join(values []string) string {
	result := ""
	for _, value := range values {
		result += value
	}
	return result
}
`

const credentialAlternativesImplementation = `type authKey struct{}
type implementation struct {
	svc.Service
	calls   int
	auth    []string
	seen    []any
	context any
	payload any
}

func (s *implementation) authenticate(ctx context.Context, kind string, valid bool) (context.Context, error) {
	s.auth = append(s.auth, kind)
	s.seen = append(s.seen, ctx.Value(authKey{}))
	if !valid {
		return context.WithValue(ctx, authKey{}, "poison"), errors.New("denied")
	}
	previous, _ := ctx.Value(authKey{}).(string)
	return context.WithValue(ctx, authKey{}, previous+kind), nil
}
func (s *implementation) BasicAuth(ctx context.Context, user, pass string, _ *security.BasicScheme) (context.Context, error) {
	return s.authenticate(ctx, "B", user == "user" && pass == "pass")
}
func (s *implementation) JWTAuth(ctx context.Context, token string, _ *security.JWTScheme) (context.Context, error) {
	return s.authenticate(ctx, "J", token == "good")
}
func (s *implementation) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	return s.authenticate(ctx, "K", key == "key")
}
func (s *implementation) result(ctx context.Context, payload any) (string, error) {
	s.calls++
	s.context = ctx.Value(authKey{})
	s.payload = payload
	return "ok", nil
}
func (s *implementation) Either(ctx context.Context, p *svc.EitherPayload) (string, error) {
	return s.result(ctx, p)
}
func (s *implementation) Reverse(ctx context.Context, p *svc.ReversePayload) (string, error) {
	return s.result(ctx, p)
}
func (s *implementation) Key(ctx context.Context, p *svc.KeyPayload) (string, error) {
	return s.result(ctx, p)
}
func (s *implementation) And(ctx context.Context, p *svc.AndPayload) (string, error) {
	return s.result(ctx, p)
}
func (s *implementation) Single(ctx context.Context, p *svc.SinglePayload) (string, error) {
	return s.result(ctx, p)
}
func (s *implementation) Required(ctx context.Context, p *svc.RequiredPayload) (string, error) {
	return s.result(ctx, p)
}
func (s *implementation) RequiredToken(ctx context.Context, p *svc.RequiredTokenPayload) (string, error) {
	return s.result(ctx, p)
}
func (s *implementation) Anonymous(ctx context.Context, p *svc.AnonymousPayload) (string, error) {
	return s.result(ctx, p)
}
`
