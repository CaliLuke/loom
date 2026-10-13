package codegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

const requestCookieHarness = `package cookies_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	svc "example.com/rpccookies/gen/cookies"
	client "example.com/rpccookies/gen/jsonrpc/cookies/client"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

type countedDoer struct {
	client *http.Client
	calls  *atomic.Int32
}

type observed struct {
	value   string
	present bool
}

// TestRequestCookiePublication checks valid and rejected values before network publication.
func TestRequestCookiePublication(t *testing.T) {
	for _, method := range []string{"ordinary", "optional", "apikey", "session"} {
		for _, value := range []string{"plain", "a,b", "a b", "", "a;b", "a\"b", "café"} {
			t.Run(fmt.Sprintf("%s/%q", method, value), func(t *testing.T) {
				valid := (&http.Cookie{Name: "tok", Value: value}).Valid() == nil
				response := httptest.NewRecorder()
				responseErr := loomhttp.SetResponseCookie(context.Background(), response, &http.Cookie{Name: "tok", Value: value})
				require.Equal(t, valid, responseErr == nil, "existing response contract")
				if !valid {
					require.Empty(t, response.Header().Values("Set-Cookie"))
				}
				seen := make(chan observed, 1)
				c, calls, closeHost := cookieClient(t, seen)
				defer closeHost()
				var endpoint loom.Endpoint
				var payload any
				switch method {
				case "ordinary":
					endpoint, payload = c.Ordinary(), &svc.OrdinaryPayload{ID: "request", Token: value}
				case "optional":
					endpoint, payload = c.Optional(), &svc.OptionalPayload{ID: "request", Token: &value}
				case "apikey":
					endpoint, payload = c.Apikey(), &svc.ApikeyPayload{ID: "request", Token: value}
				case "session":
					endpoint, payload = c.Session(), &svc.SessionPayload{ID: "request", Token: &value}
				}
				_, err := endpoint(context.Background(), payload)
				if valid {
					require.Equal(t, int32(1), calls.Load())
					require.NoError(t, err)
					select {
					case got := <-seen:
						require.True(t, got.present)
						require.Equal(t, value, got.value)
					default:
						t.Error("valid request was not published")
					}
				} else {
					require.Zero(t, calls.Load(), "invalid cookie must fail before Do")
					if err == nil {
						t.Error("invalid cookie value accepted; expected encoding error before send")
					} else {
						var clientErr *loomhttp.ClientError
						require.ErrorAs(t, err, &clientErr)
						require.Equal(t, "encoding_error", clientErr.Name)
						require.Equal(t, "cookies", clientErr.Service)
						require.Equal(t, method, clientErr.Method)
					}
					select {
					case got := <-seen:
						t.Errorf("invalid request published: authored=%q wire=%q", value, got.value)
					default:
					}
				}
			})
		}
	}
}

// TestRequestCookiePresenceControls retains absence, empty and native-zero values.
func TestRequestCookiePresenceControls(t *testing.T) {
	for _, name := range []string{"absent", "session absent", "explicit empty", "default native zero", "default authored value", "native false", "native true"} {
		t.Run(name, func(t *testing.T) {
			seen := make(chan observed, 1)
			c, calls, closeHost := cookieClient(t, seen)
			defer closeHost()
			var endpoint loom.Endpoint
			var payload any
			want := observed{present: true}
			switch name {
			case "absent":
				endpoint, payload = c.Optional(), &svc.OptionalPayload{ID: "request"}
				want.present = false
			case "session absent":
				endpoint, payload = c.Session(), &svc.SessionPayload{ID: "request"}
				want.present = false
			case "explicit empty":
				empty := ""
				endpoint, payload = c.Optional(), &svc.OptionalPayload{ID: "request", Token: &empty}
			case "default native zero":
				endpoint, payload = c.Fallback(), &svc.FallbackPayload{ID: "request"}
			case "default authored value":
				endpoint, payload = c.Fallback(), &svc.FallbackPayload{ID: "request", Token: "fallback"}
				want.value = "fallback"
			case "native false":
				endpoint, payload = c.Native(), &svc.NativePayload{ID: "request", Flag: false}
				want.value = "false"
			case "native true":
				endpoint, payload = c.Native(), &svc.NativePayload{ID: "request", Flag: true}
				want.value = "true"
			}
			_, err := endpoint(context.Background(), payload)
			require.NoError(t, err)
			require.Equal(t, int32(1), calls.Load())
			select {
			case got := <-seen:
				require.Equal(t, want, got)
			default:
				t.Error("control request was not published")
			}
		})
	}
}

// TestInvalidStreamCookieNeverPublishes checks stream requests before Do or Dial.
func TestInvalidStreamCookieNeverPublishes(t *testing.T) {
	for _, name := range []string{"events"} {
		t.Run(name, func(t *testing.T) {
			seen := make(chan observed, 1)
			c, calls, closeHost := cookieClient(t, seen)
			defer closeHost()
			endpoint := c.Events()
			var payload any = &svc.EventsPayload{ID: "request", Token: "a;b"}

			_, err := endpoint(context.Background(), payload)
			var clientErr *loomhttp.ClientError
			require.ErrorAs(t, err, &clientErr)
			require.Equal(t, "encoding_error", clientErr.Name)
			require.Equal(t, "cookies", clientErr.Service)
			require.Equal(t, name, clientErr.Method)
			require.Zero(t, calls.Load(), "invalid cookies must fail before Do or Dial")
		})
	}
}

func cookieClient(t *testing.T, seen chan<- observed) (*client.Client, *atomic.Int32, func()) {
	t.Helper()
	host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("tok")
		if err != nil && !errors.Is(err, http.ErrNoCookie) {
			t.Error(err)
		}
		got := observed{}
		if cookie != nil {
			got.present = true
			got.value = cookie.Value
		}
		seen <- got
		w.Header().Set("Content-Type", "application/json")
		if _, err := fmt.Fprint(w, ` + "`" + `{"jsonrpc":"2.0","id":"request","result":null}` + "`" + `); err != nil {
			t.Error(err)
		}
	}))
	u, err := url.Parse(host.URL)
	require.NoError(t, err)
	calls := &atomic.Int32{}
	doer := &countedDoer{client: host.Client(), calls: calls}
	return client.NewClient(u.Scheme, u.Host, doer, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false), calls, host.Close
}

func (d *countedDoer) Do(req *http.Request) (*http.Response, error) {
	d.calls.Add(1)
	return d.client.Do(req)
}
`

// TestRequestCookiePublicationGenerated checks actual client publication and presence.
func TestRequestCookiePublicationGenerated(t *testing.T) {
	root := RunJSONRPCDSL(t, func() {
		API("cookies", func() {
			JSONRPC(func() {
			})
		})
		key := APIKeySecurity("key")
		jwt := JWTSecurity("jwt")
		session := SessionAuth("session", func() {
			BearerTransport(jwt, "auth")
			CookieTransport(key, "token", func() {
				CookieName("tok")
			})
		})
		Service("cookies", func() {
			JSONRPC(func() {
				POST("/rpc")
			})
			for _, name := range []string{"ordinary", "optional", "fallback", "native", "apikey", "session"} {
				Method(name, func() {
					if name == "apikey" {
						Security(key)
					}
					if name == "session" {
						SessionSecurity(session)
					}
					Payload(func() {
						ID("id", String)
						Required("id")
						switch name {
						case "native":
							Attribute("flag", Boolean, func() {
								Default(true)
							})
							Required("flag")
						case "apikey":
							APIKey("key", "token", String)
							Required("token")
						case "session":
							Attribute("message", String)
						default:
							Attribute("token", String, func() {
								if name == "fallback" {
									Default("fallback")
								}
							})
							if name == "ordinary" {
								Required("token")
							}
						}
					})
					Result(Empty)
					JSONRPC(func() {
						if name == "native" {
							Cookie("flag:tok")
						} else if name != "session" {
							Cookie("token:tok")
						}
					})
				})
			}
			for _, name := range []string{"events"} {
				Method(name, func() {
					Payload(func() {
						ID("id", String)
						Required("id")
						Attribute("token", String)
						Required("token")
					})
					StreamingResult(String)
					JSONRPC(func() {
						Cookie("token:tok")
						if name == "events" {
							ServerSentEvents()
						}
					})
				})
			}
		})
	})
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/rpccookies", root)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "cookie_test.go"), []byte(requestCookieHarness), 0600))
	if output := os.Getenv("LOOM_COOKIE_OUTPUT"); output != "" {
		require.NoError(t, os.CopyFS(filepath.Join(output, "rpc"), os.DirFS(dir)))
	}
	for _, args := range [][]string{{"mod", "tidy"}, {"test", "-race", "./..."}, {"vet", "./..."}} {
		cmd := exec.Command("go", args...)
		cmd.Dir = dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, string(output))
	}
}
