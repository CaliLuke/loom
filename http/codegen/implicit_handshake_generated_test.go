package codegen

const implicitHandshakeHarness = `package handshake_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	implicitclient "example.com/handshake/gen/http/implicit/client"
	implicitserver "example.com/handshake/gen/http/implicit/server"
	implicitsvc "example.com/handshake/gen/implicit"
	"example.com/handshake/scalar"
	loomhttp "github.com/CaliLuke/loom/http"
)

type implicitImplementation struct{}

func (implicitImplementation) Watch(_ context.Context, payload *implicitsvc.WatchPayload, stream implicitsvc.WatchServerStream) error {
	if payload.Explicit != nil && *payload.Explicit {
		if err := stream.SetResponse(&implicitsvc.ImplicitOpening{}); err != nil {
			return err
		}
	}
	return stream.SendWithContext(context.Background(), "event")
}

func (implicitImplementation) Mixed(context.Context, implicitsvc.MixedServerStream) (string, error) {
	return "whole", nil
}

func (implicitImplementation) Invalid(_ context.Context, stream implicitsvc.InvalidServerStream) error {
	return stream.SendWithContext(context.Background(), "event")
}

func (implicitImplementation) InvalidMixed(context.Context, implicitsvc.InvalidMixedServerStream) (string, error) {
	return "whole", nil
}

// withoutHint drops the unmapped member, which has no wire reconstruction.
func withoutHint(opening implicitsvc.ImplicitOpening) implicitsvc.ImplicitOpening {
	opening.Hint = ""
	return opening
}

func implicitServer(t *testing.T, refuse bool) (*httptest.Server, *atomic.Int32) {
	calls := new(atomic.Int32)
	policy := loomhttp.ResponseCookiePolicy(func(ctx context.Context, cookie *http.Cookie) error {
		calls.Add(1)
		if ctx.Value(cookieContextKey{}) != "retained" {
			t.Error("implicit cookie preparation lost request context")
		}
		if refuse {
			return errors.New("policy refused")
		}
		cookie.Secure = true
		return nil
	})
	mux := loomhttp.NewMuxer()
	transport := implicitserver.New(implicitsvc.NewEndpoints(implicitImplementation{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	implicitserver.Mount(mux, transport)
	handler := policy.Handler(mux)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), cookieContextKey{}, "retained")))
	})), calls
}

func TestImplicitHandshakeWire(t *testing.T) {
	for _, tc := range []struct {
		path    string
		accept  string
		refuse  bool
		status  int
		calls   int32
		trace   []string
		count   string
		token   string
		cookie  string
		content string
	}{
		{path: "/implicit", status: 200, calls: 1, trace: []string{"abc"}, count: "7", token: "token:default", cookie: "sess", content: "text/event-stream"},
		{path: "/implicit?explicit=true", status: 200, calls: 1, trace: []string{""}, count: "0", token: "", cookie: "", content: "text/event-stream"},
		{path: "/implicit/mixed", accept: "application/json", status: 200, calls: 1, trace: []string{"abc"}, count: "7", token: "token:default", cookie: "sess", content: "application/json"},
		{path: "/implicit", refuse: true, status: 500, calls: 1},
		{path: "/implicit/mixed", accept: "application/json", refuse: true, status: 500, calls: 1},
		{path: "/implicit/invalid", status: 500},
		{path: "/implicit/invalid/mixed", accept: "application/json", status: 500},
	} {
		t.Run(tc.path+"/"+tc.accept, func(t *testing.T) {
			hs, calls := implicitServer(t, tc.refuse)
			defer hs.Close()
			request, err := http.NewRequest(http.MethodGet, hs.URL+tc.path, nil)
			if err != nil {
				t.Fatal(err)
			}
			if tc.accept != "" {
				request.Header.Set("Accept", tc.accept)
			}
			response, err := hs.Client().Do(request)
			if err != nil {
				t.Fatal(err)
			}
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Error(err)
			}
			if err := response.Body.Close(); err != nil {
				t.Error(err)
			}
			if response.StatusCode != tc.status || calls.Load() != tc.calls {
				t.Fatalf("status=%d cookie policy calls=%d body=%s", response.StatusCode, calls.Load(), body)
			}
			if tc.status != 200 {
				if len(response.Header.Values("Set-Cookie")) != 0 || response.Header.Get("X-Trace") != "" || len(response.Header.Values("X-Token")) != 0 || strings.Contains(response.Header.Get("Content-Type"), "text/event-stream") || strings.Contains(string(body), "event") {
					t.Errorf("refused policy published metadata: %v %q", response.Header, body)
				}
				return
			}
			if got := response.Header.Values("X-Trace"); len(got) != len(tc.trace) || got[0] != tc.trace[0] {
				t.Errorf("X-Trace=%q", got)
			}
			if got := response.Header.Values("X-Token"); len(got) != 1 || got[0] != tc.token {
				t.Errorf("X-Token=%q", got)
			}
			if response.Header.Get("X-Count") != tc.count || len(response.Header.Values("X-Note")) != 0 {
				t.Errorf("headers=%v", response.Header)
			}
			cookies := response.Cookies()
			if len(cookies) != 1 || cookies[0].Name != "sid" || cookies[0].Value != tc.cookie || !cookies[0].Secure {
				t.Errorf("cookies=%v", cookies)
			}
			if !strings.HasPrefix(response.Header.Get("Content-Type"), tc.content) {
				t.Errorf("content type=%q", response.Header.Get("Content-Type"))
			}
		})
	}
}

func TestImplicitHandshakeClient(t *testing.T) {
	hs, _ := implicitServer(t, false)
	defer hs.Close()
	c := implicitclient.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	service := implicitsvc.NewClient(c.Watch(), c.Mixed(), c.MixedStream(), c.Invalid(), c.InvalidMixed(), c.InvalidMixedStream())
	defaults := implicitsvc.ImplicitOpening{Trace: "abc", Count: 7, Session: "sess", Token: scalar.Token("token:default"), Hint: scalar.Token("token:hint")}
	for explicit, want := range map[bool]implicitsvc.ImplicitOpening{
		false: defaults,
		true:  {},
	} {
		stream, err := service.Watch(context.Background(), &implicitsvc.WatchPayload{Explicit: &explicit})
		if err != nil {
			t.Fatal(err)
		}
		if got := stream.Response(); got == nil || withoutHint(*got) != withoutHint(want) {
			t.Errorf("explicit=%v metadata=%#v", explicit, got)
		}
		if event, err := stream.Recv(); err != nil || event != "event" {
			t.Errorf("event=%q err=%v", event, err)
		}
		if err := stream.Close(); err != nil {
			t.Error(err)
		}
	}
	result, metadata, err := service.Mixed(context.Background())
	if err != nil || result != "whole" || metadata == nil || withoutHint(*metadata) != withoutHint(defaults) {
		t.Errorf("mixed implicit tuple: %q %#v %v", result, metadata, err)
	}
}
`

const implicitTokenRepresentation = `package scalar

import (
	"errors"
	"strings"
)

// Token is a module-local custom String representation whose parser rejects
// the bad prefix, so an authored default can fail only at runtime.
type Token string

// MarshalText returns the token text.
func (t Token) MarshalText() ([]byte, error) {
	return []byte(t), nil
}

// UnmarshalText admits any text without the bad prefix.
func (t *Token) UnmarshalText(text []byte) error {
	if strings.HasPrefix(string(text), "bad") {
		return errors.New("rejected token")
	}
	*t = Token(text)
	return nil
}
`
