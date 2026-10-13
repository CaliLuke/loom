package codegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/http/codegen/testdata"
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

	"github.com/gorilla/websocket"

	svc "example.com/httpcookies/gen/cookies"
	client "example.com/httpcookies/gen/http/cookies/client"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

type countedDialer struct {
	calls *atomic.Int32
}

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
					endpoint, payload = c.Ordinary(), &svc.OrdinaryPayload{Token: value}
				case "optional":
					endpoint, payload = c.Optional(), &svc.OptionalPayload{Token: &value}
				case "apikey":
					endpoint, payload = c.Apikey(), &svc.ApikeyPayload{Token: value}
				case "session":
					endpoint, payload = c.Session(), &svc.SessionPayload{Token: &value}
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
				endpoint, payload = c.Optional(), &svc.OptionalPayload{}
				want.present = false
			case "session absent":
				endpoint, payload = c.Session(), &svc.SessionPayload{}
				want.present = false
			case "explicit empty":
				empty := ""
				endpoint, payload = c.Optional(), &svc.OptionalPayload{Token: &empty}
			case "default native zero":
				endpoint, payload = c.Fallback(), &svc.FallbackPayload{}
			case "default authored value":
				endpoint, payload = c.Fallback(), &svc.FallbackPayload{Token: "fallback"}
				want.value = "fallback"
			case "native false":
				endpoint, payload = c.Native(), &svc.NativePayload{Flag: false}
				want.value = "false"
			case "native true":
				endpoint, payload = c.Native(), &svc.NativePayload{Flag: true}
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
	for _, name := range []string{"events", "socket"} {
		t.Run(name, func(t *testing.T) {
			seen := make(chan observed, 1)
			c, calls, closeHost := cookieClient(t, seen)
			defer closeHost()
			endpoint := c.Events()
			var payload any = &svc.EventsPayload{Token: "a;b"}
			if name == "socket" {
				endpoint = c.Socket()
				payload = &svc.SocketPayload{Token: "a;b"}
			}
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
		w.WriteHeader(http.StatusNoContent)
	}))
	u, err := url.Parse(host.URL)
	require.NoError(t, err)
	calls := &atomic.Int32{}
	doer := &countedDoer{client: host.Client(), calls: calls}
	return client.NewClient(u.Scheme, u.Host, doer, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false, &countedDialer{calls: calls}, nil), calls, host.Close
}

func (d *countedDialer) DialContext(context.Context, string, http.Header) (*websocket.Conn, *http.Response, error) {
	d.calls.Add(1)
	return nil, nil, errors.New("unexpected dial")
}

func (d *countedDoer) Do(req *http.Request) (*http.Response, error) {
	d.calls.Add(1)
	return d.client.Do(req)
}
`

// TestRequestCookiePublicationGenerated checks actual client publication and presence.
func TestRequestCookiePublicationGenerated(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		API("cookies", func() {
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
			for _, name := range []string{"ordinary", "optional", "fallback", "native", "apikey", "session"} {
				Method(name, func() {
					if name == "apikey" {
						Security(key)
					}
					if name == "session" {
						SessionSecurity(session)
					}
					Payload(func() {
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
					HTTP(func() {
						GET("/" + name)
						if name == "native" {
							Cookie("flag:tok")
						} else if name != "session" {
							Cookie("token:tok")
						}
						Response(StatusNoContent)
					})
				})
			}
			for _, name := range []string{"events", "socket"} {
				Method(name, func() {
					Payload(func() {
						Attribute("token", String)
						Required("token")
					})
					StreamingResult(String)
					HTTP(func() {
						GET("/" + name)
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
	renderHTTPModule(t, dir, "example.com/httpcookies", root)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "cookie_test.go"), []byte(requestCookieHarness), 0600))
	if output := os.Getenv("LOOM_COOKIE_OUTPUT"); output != "" {
		require.NoError(t, os.CopyFS(filepath.Join(output, "http"), os.DirFS(dir)))
	}
	for _, args := range [][]string{{"mod", "tidy"}, {"test", "-race", "./..."}, {"vet", "./..."}} {
		cmd := exec.Command("go", args...)
		cmd.Dir = dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, string(output))
	}
}

// TestRequestCookieUnrelatedModule keeps ordinary query output unchanged.
func TestRequestCookieUnrelatedModule(t *testing.T) {
	root := RunHTTPDSL(t, testdata.PayloadQueryStringDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/querycontrol", root)
	if output := os.Getenv("LOOM_COOKIE_OUTPUT"); output != "" {
		require.NoError(t, os.CopyFS(filepath.Join(output, "ordinary"), os.DirFS(dir)))
	}
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "build", "./...")
	runGoCommand(t, dir, "vet", "./...")
}
