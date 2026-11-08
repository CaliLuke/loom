package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestGeneratedSSEHandshakeContract(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		testdata.ConstructorOwnership("metadata", false, false)
	})
	services := CreateHTTPServices(root)
	data := services.Get("constructors")
	client := codegen.SectionCode(t, clientEndpointSection(data.Endpoint("watch")))
	require.NotContains(t, client, "resp.StatusCode != http.StatusOK")
	require.Contains(t, client, "mime.ParseMediaType")
	var handlers strings.Builder
	for _, file := range ServerFiles("example.com/handshake/gen", services) {
		for _, section := range file.Section("server-handler-init") {
			handlers.WriteString(codegen.SectionCode(t, section))
		}
	}
	require.Contains(t, handlers.String(), "WithSSEHandshake")
	require.Contains(t, handlers.String(), "WithSSEExchange")
	require.Contains(t, handlers.String(), "FinalizeSuccess")
}

func TestGeneratedImplicitHandshakeAndDynamicView(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		testdata.MixedMetadataDSL()
		testdata.ImplicitHandshakeDSL()
	})
	var code strings.Builder
	for _, file := range ServerFiles("example.com/handshake/gen", CreateHTTPServices(root)) {
		for _, section := range file.Sections {
			if name := section.SectionName(); name == "response-encoder" || name == "server-handler-init" {
				code.WriteString(codegen.SectionCode(t, section))
			}
		}
	}
	server := code.String()
	_, dynamic, found := strings.Cut(server, "func EncodeDynamicResponse")
	require.True(t, found)
	dynamic, _, _ = strings.Cut(dynamic, "\n}\n")
	require.Contains(t, dynamic, `w.Header().Set("loom-view", res.View)`)
	_, viewed, found := strings.Cut(server, "func EncodeViewedResponse")
	require.True(t, found)
	viewed, _, _ = strings.Cut(viewed, "\n}\n")
	require.NotContains(t, viewed, "loom-view", "fixed views keep no wire view header")
	require.Equal(t, 2, strings.Count(server, `value := &implicit.ImplicitOpening{Trace: "abc", Count: 7, Session: "sess", Token: func() custom.Token {`), server)
	require.Equal(t, 2, strings.Count(server, `loom.ParseStringValue[custom.Token]("bad:default", "default.token")`), server)
	require.NotContains(t, server, "new(implicit.ImplicitOpening)")
	require.Contains(t, server, "return loomhttp.SSEHandshake{}, &loomhttp.StatusViolationError{}", "required metadata still rejects absence")
}

func TestGeneratedFinalSelectorAdmission(t *testing.T) {
	for _, selector := range []any{Status2XX, StatusFallback} {
		t.Run(fmt.Sprint(selector), func(t *testing.T) {
			root := RunHTTPDSL(t, func() {
				opening := Type("Opening", func() {
					Attribute("status", Int)
					Required("status")
				})
				Service("selector", func() {
					Method("watch", func() {
						StreamingResult(String)
						HTTP(func() {
							GET("/")
							ServerSentEvents()
							Response(selector, opening, func() {
								StatusField("status")
							})
						})
					})
				})
			})
			require.NotPanics(t, func() {
				CreateHTTPServices(root).Get("selector")
			})
		})
	}
}

func TestSSEHandshakeGenerated(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		testdata.ConstructorOwnership("metadata-handshake", false, false)
		testdata.ResponseStatusSelectionDSL()
		testdata.MixedMetadataDSL()
		testdata.ImplicitHandshakeDSL()
	})
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/handshake", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "handshake_test.go"), []byte(sseHandshakeHarness), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "status_test.go"), []byte(finalStatusHarness), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "mixed_test.go"), []byte(mixedHandshakeHarness), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "implicit_test.go"), []byte(implicitHandshakeHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

const sseHandshakeHarness = `package handshake_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	svc "example.com/handshake/gen/constructors"
	client "example.com/handshake/gen/http/constructors/client"
	server "example.com/handshake/gen/http/constructors/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

type cookieContextKey struct{}

type implementation struct {
	events      bool
	cookieValue string
}

func (s *implementation) M(context.Context) (string, error) {
	return "ordinary", nil
}

func (s *implementation) Watch(ctx context.Context, stream svc.WatchServerStream) error {
	setter, ok := any(stream).(interface {
		SetResponse(*svc.Opening) error
	})
	if !ok {
		return errors.New("generated stream lacks typed SetResponse")
	}
	if err := setter.SetResponse(&svc.Opening{Status: 201, Session: "session"}); err != nil {
		return err
	}
	if s.events {
		return stream.SendWithContext(ctx, "event")
	}
	return nil
}

func (s *implementation) Cookie(ctx context.Context, stream svc.CookieServerStream) error {
	value := s.cookieValue
	if value == "" {
		value = "session"
	}
	if err := stream.SetResponse(&svc.Opening{Status: 201, Session: value}); err != nil {
		return err
	}
	if s.events {
		return stream.SendWithContext(ctx, "cookie-event")
	}
	return nil
}

func TestCookiePolicyPreparation(t *testing.T) {
	for _, tc := range []struct {
		name        string
		value       string
		policyError bool
		status      int
	}{
		{name: "prepared once", status: 201},
		{name: "policy failure", policyError: true, status: 500},
		{name: "invalid cookie", value: "bad\nvalue", status: 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			policy := loomhttp.ResponseCookiePolicy(func(ctx context.Context, cookie *http.Cookie) error {
				calls.Add(1)
				if ctx.Value(cookieContextKey{}) != "retained" {
					t.Error("cookie preparation lost request context")
				}
				if tc.policyError {
					return errors.New("policy refused")
				}
				cookie.Secure = true
				return nil
			})
			mux := loomhttp.NewMuxer()
			transport := server.New(svc.NewEndpoints(&implementation{cookieValue: tc.value}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
			server.Mount(mux, transport)
			handler := policy.Handler(mux)
			hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handler.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), cookieContextKey{}, "retained")))
			}))
			defer hs.Close()
			response, err := hs.Client().Get(hs.URL + "/cookie")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := io.ReadAll(response.Body); err != nil {
				t.Error(err)
			}
			if err := response.Body.Close(); err != nil {
				t.Error(err)
			}
			if response.StatusCode != tc.status || calls.Load() != 1 {
				t.Errorf("status=%d cookie policy calls=%d", response.StatusCode, calls.Load())
			}
			if tc.status == 201 {
				cookies := response.Cookies()
				if len(cookies) != 1 || cookies[0].Value != "session" || !cookies[0].Secure {
					t.Errorf("prepared cookies=%v", cookies)
				}
			} else if len(response.Header.Values("Set-Cookie")) != 0 || response.Header.Get("Content-Type") == "text/event-stream" {
				t.Errorf("invalid metadata leaked: %v", response.Header)
			}
		})
	}
}

func TestTypedCookieHandshake(t *testing.T) {
	mux := loomhttp.NewMuxer()
	transport := server.New(svc.NewEndpoints(&implementation{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	server.Mount(mux, transport)
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := client.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	result, err := c.Cookie()(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	stream := result.(svc.CookieClientStream)
	defer func() {
		if err := stream.Close(); err != nil {
			t.Error(err)
		}
	}()
	if got := stream.Response(); got == nil || got.Status != 201 || got.Session != "session" {
		t.Errorf("cookie metadata=%#v", got)
	}
	if _, err := stream.Recv(); !errors.Is(err, io.EOF) {
		t.Errorf("zero-event cookie success=%v", err)
	}
}

func TestTypedHandshake(t *testing.T) {
	for _, events := range []bool{false, true} {
		t.Run(map[bool]string{false: "zero", true: "event"}[events], func(t *testing.T) {
			mux := loomhttp.NewMuxer()
			errhandler := func(_ context.Context, _ http.ResponseWriter, err error) {
				t.Errorf("server failure: %v", err)
			}
			transport := server.New(svc.NewEndpoints(&implementation{events: events}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errhandler, nil)
			server.Mount(mux, transport)
			hs := httptest.NewServer(mux)
			defer hs.Close()
			var status int
			decoder := func(resp *http.Response) loomhttp.Decoder {
				status = resp.StatusCode
				if resp.Header.Get("Content-Type") != "application/json" || resp.Request != nil {
					t.Errorf("event response must use detached JSON context: %#v", resp)
				}
				return loomhttp.ResponseDecoder(resp)
			}
			c := client.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, decoder, false)
			result, err := c.Watch()(context.Background(), nil)
			if err != nil {
				t.Fatal(err)
			}
			stream := result.(svc.WatchClientStream)
			defer func() {
				if err := stream.Close(); err != nil {
					t.Error(err)
				}
			}()
			opening, ok := any(stream).(interface {
				Response() *svc.Opening
			})
			if !ok {
				t.Fatal("generated client lacks typed Response")
			}
			if got := opening.Response(); got == nil || got.Status != 201 || got.Session != "session" {
				t.Fatalf("handshake: %#v", got)
			}
			if events {
				got, err := stream.Recv()
				if err != nil || got != "event" || status != 201 {
					t.Fatalf("event/status: %q, %d, %v", got, status, err)
				}
			}
			if _, err := stream.Recv(); !errors.Is(err, io.EOF) {
				t.Fatalf("end: %v", err)
			}
		})
	}
}
`
