package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

const basicPresenceHarness = `package presence_test

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	svc "example.com/presence/gen/presence"
	client "example.com/presence/gen/http/presence/client"
	server "example.com/presence/gen/http/presence/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func erase[T any](decode func(*http.Request) (T, error)) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) { return decode(r) }
}
func TestBasicPresence(t *testing.T) {
	mux := loomhttp.NewMuxer()
	for _, tc := range []struct {
		name                       string
		payload                    any
		encode                     func(*http.Request, any) error
		decode                     func(*http.Request) (any, error)
		userRequired, passRequired bool
	}{
		{"both", &svc.BothPayload{}, client.EncodeBothRequest(nil), erase(server.DecodeBothRequest(mux, nil)), true, true},
		{"user", &svc.UserPayload{}, client.EncodeUserRequest(nil), erase(server.DecodeUserRequest(mux, nil)), true, false},
		{"pass", &svc.PassPayload{}, client.EncodePassRequest(nil), erase(server.DecodePassRequest(mux, nil)), false, true},
		{"neither", &svc.NeitherPayload{}, client.EncodeNeitherRequest(nil), erase(server.DecodeNeitherRequest(mux, nil)), false, false},
	} {
		for _, u := range []string{"absent", "", "alice"} {
			for _, p := range []string{"absent", "", "secret"} {
				if tc.userRequired && u == "absent" || tc.passRequired && p == "absent" {
					continue
				}
				t.Run(tc.name+"/"+u+"/"+p, func(t *testing.T) {
					payload := reflect.New(reflect.TypeOf(tc.payload).Elem())
					for name, value := range map[string]string{"User": u, "Pass": p} {
						field := payload.Elem().FieldByName(name)
						if value == "absent" {
							continue
						}
						if field.Kind() == reflect.Pointer {
							field.Set(reflect.New(field.Type().Elem()))
							field = field.Elem()
						}
						field.SetString(value)
					}
					req := httptest.NewRequest("GET", "/", nil)
					if err := tc.encode(req, payload.Interface()); err != nil {
						t.Fatal(err)
					}
					gotUser, gotPass, present := req.BasicAuth()
					wantPresence := tc.userRequired || tc.passRequired || u != "absent" || p != "absent"
					expectedUser, expectedPass := u, p
					if u == "absent" {
						expectedUser = ""
					}
					if p == "absent" {
						expectedPass = ""
					}
					if present != wantPresence || gotUser != expectedUser || gotPass != expectedPass {
						t.Fatalf("wire=%q %q %v", gotUser, gotPass, present)
					}
					decoded, err := tc.decode(req)
					if err != nil {
						t.Fatal(err)
					}
					for name, value := range map[string]string{"User": expectedUser, "Pass": expectedPass} {
						field := reflect.ValueOf(decoded).Elem().FieldByName(name)
						if field.Kind() == reflect.Pointer {
							if field.IsNil() {
								if present {
									t.Errorf("%s missing with present header", name)
								}
								continue
							}
							if !present {
								t.Errorf("%s fabricated presence", name)
							}
							field = field.Elem()
						}
						if field.String() != value {
							t.Errorf("%s=%q want %q", name, field.String(), value)
						}
					}
				})
			}
		}
		for _, header := range []string{"", "Basic invalid", "Bearer token"} {
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("Authorization", header)
			decoded, err := tc.decode(req)
			if tc.userRequired || tc.passRequired {
				if err == nil {
					t.Errorf("%s accepted %q", tc.name, header)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				p := decoded.(*svc.NeitherPayload)
				if p.User != nil || p.Pass != nil {
					t.Errorf("invalid header fabricated credentials: %#v", p)
				}
			}
		}
	}
}
`

func TestBasicPresenceGenerated(t *testing.T) {
	root := RunHTTPDSL(t, testdata.BasicPresenceDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/presence", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "presence_test.go"), []byte(basicPresenceHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

func TestBasicPresencePlan(t *testing.T) {
	data := CreateHTTPServices(RunHTTPDSL(t, testdata.BasicPresenceDSL)).Get("presence")
	for _, tc := range []struct {
		name           string
		headerRequired bool
	}{
		{"both", true}, {"user", true}, {"pass", true}, {"neither", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan := data.Endpoint(tc.name).BasicScheme
			require.Equal(t, tc.headerRequired, plan.HeaderRequired)
			require.Equal(t, tc.headerRequired, plan.AlwaysSend)
		})
	}
}
