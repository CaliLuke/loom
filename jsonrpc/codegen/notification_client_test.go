package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

// TestJSONRPCNotificationClient asserts that the unary endpoint of the
// generated JSON-RPC HTTP client checks the response to a notification, a
// request whose payload ID is empty, with jsonrpc.DecodeNotificationResponse
// and returns the zero result instead of decoding a response. A method
// without an ID attribute always sends an ID and keeps the plain decoding.
func TestJSONRPCNotificationClient(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcNotificationDSL)
	services := CreateJSONRPCServices(root)
	client := jsonrpcSectionsSource(requireFileWithSection(t, ClientFiles("", services), "jsonrpc-client-endpoint-init"), "jsonrpc-client-endpoint-init")
	cases := []struct {
		Method string
		Want   string
	}{
		{
			Method: "Fire",
			Want: "\t\tif p := v.(*notify.FirePayload); p.ID == nil || *p.ID == \"\" {\n" +
				"\t\t\treturn nil, jsonrpc.DecodeNotificationResponse(\"Notify\", \"fire\", resp)\n" +
				"\t\t}\n" +
				"\t\treturn decodeResponse(resp)\n",
		},
		{
			Method: "Req",
			Want: "\t\tif p := v.(*notify.ReqPayload); p.ID == \"\" {\n" +
				"\t\t\tvar res string\n" +
				"\t\t\treturn res, jsonrpc.DecodeNotificationResponse(\"Notify\", \"req\", resp)\n" +
				"\t\t}\n",
		},
		{
			Method: "Res",
			Want: "\t\tif p := v.(*notify.ResPayload); p.ID == nil || *p.ID == \"\" {\n" +
				"\t\t\tvar res *notify.ResResult\n" +
				"\t\t\treturn res, jsonrpc.DecodeNotificationResponse(\"Notify\", \"res\", resp)\n" +
				"\t\t}\n",
		},
	}
	for _, c := range cases {
		t.Run(c.Method, func(t *testing.T) {
			assert.Contains(t, client, c.Want)
		})
	}
	assert.NotContains(t, client, "v.(*notify.NoidPayload)")
}

// TestJSONRPCNotificationClientGeneratedModule compiles and vets a JSON-RPC
// service with methods whose payload has an optional or a required ID, with
// and without results, and calls them through the generated HTTP client with
// and without an ID. A notification succeeds with the zero result, even when
// the service fails or, for a required ID, the server rejects it without an
// answer, and a request with an ID decodes its response as usual.
func TestJSONRPCNotificationClientGeneratedModule(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcNotificationDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/jsonrpcnotify", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "notification_test.go"), []byte(jsonRPCNotificationHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=1", "./...")
}

// jsonrpcNotificationDSL is a JSON-RPC design whose methods take a payload
// with an optional ID (Fire, Res), a required ID (Req) or no ID (Noid).
func jsonrpcNotificationDSL() {
	dsl.API("notify", func() {
		dsl.JSONRPC(func() {})
	})
	dsl.Service("Notify", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("fire", func() {
			dsl.Payload(func() {
				dsl.ID("id", dsl.String)
				dsl.Attribute("v", dsl.String)
			})
			dsl.Error("failed")
			dsl.JSONRPC(func() {
				dsl.Response("failed", func() {
					dsl.Code(-32001)
				})
			})
		})
		dsl.Method("req", func() {
			dsl.Payload(func() {
				dsl.ID("id", dsl.String)
				dsl.Attribute("v", dsl.String)
				dsl.Required("id")
			})
			dsl.Result(dsl.String)
			dsl.JSONRPC(func() {})
		})
		dsl.Method("res", func() {
			dsl.Payload(func() {
				dsl.ID("id", dsl.String)
				dsl.Attribute("v", dsl.String)
			})
			dsl.Result(func() {
				dsl.Attribute("n", dsl.Int)
			})
			dsl.JSONRPC(func() {})
		})
		dsl.Method("noid", func() {
			dsl.Payload(func() {
				dsl.Attribute("v", dsl.String)
			})
			dsl.Result(dsl.String)
			dsl.JSONRPC(func() {})
		})
	})
}

const jsonRPCNotificationHarness = `package jsonrpcnotify_test

import (
	"bytes"
	"context"
	json "encoding/json/v2"
	"encoding/json/jsontext"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	client "example.com/jsonrpcnotify/gen/jsonrpc/notify/client"
	server "example.com/jsonrpcnotify/gen/jsonrpc/notify/server"
	notify "example.com/jsonrpcnotify/gen/notify"
	loomhttp "github.com/CaliLuke/loom/http"
)

// service records the value of every call. Fire fails when the value is
// "fail".
type service struct {
	mu   sync.Mutex
	seen []string
}

func (s *service) Fire(_ context.Context, p *notify.FirePayload) error {
	s.record(p.V)
	if p.V != nil && *p.V == "fail" {
		return notify.MakeFailed(errors.New("failed"))
	}
	return nil
}

func (s *service) Req(_ context.Context, p *notify.ReqPayload) (string, error) {
	s.record(p.V)
	return "req", nil
}

func (s *service) Res(_ context.Context, p *notify.ResPayload) (*notify.ResResult, error) {
	s.record(p.V)
	n := 7
	return &notify.ResResult{N: &n}, nil
}

func (s *service) Noid(_ context.Context, p *notify.NoidPayload) (string, error) {
	s.record(p.V)
	return "noid", nil
}

func (s *service) record(v *string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v == nil {
		s.seen = append(s.seen, "")
		return
	}
	s.seen = append(s.seen, *v)
}

func (s *service) take() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	seen := s.seen
	s.seen = nil
	return seen
}

func ptr[T any](v T) *T { return &v }

func TestNotifications(t *testing.T) {
	svc := &service{}
	mux := loomhttp.NewMuxer()
	// The server reports the failures of notifications, which it does not
	// answer, to the error handler.
	errhandler := func(context.Context, http.ResponseWriter, error) {}
	server.Mount(mux, server.New(notify.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errhandler))
	var (
		mu    sync.Mutex
		ids   []bool
		empty []bool
	)
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var envelope map[string]jsontext.Value
		if err := json.Unmarshal(body, &envelope); err != nil {
			t.Errorf("decode request %q: %v", body, err)
		}
		_, hasID := envelope["id"]
		r.Body = io.NopCloser(bytes.NewReader(body))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, r)
		mu.Lock()
		ids = append(ids, hasID)
		empty = append(empty, rec.Body.Len() == 0)
		mu.Unlock()
		for k, v := range rec.Header() {
			w.Header()[k] = v
		}
		w.WriteHeader(rec.Code)
		if _, err := w.Write(rec.Body.Bytes()); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer hs.Close()
	c := client.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	ctx := context.Background()
	cases := []struct {
		name         string
		call         func() (any, error)
		want         any
		wantErr      bool
		notification bool
		// calls is the number of service invocations.
		calls int
	}{
		{"fire notification", func() (any, error) { return c.Fire()(ctx, &notify.FirePayload{V: ptr("a")}) }, nil, false, true, 1},
		{"fire empty id notification", func() (any, error) { return c.Fire()(ctx, &notify.FirePayload{ID: ptr(""), V: ptr("b")}) }, nil, false, true, 1},
		{"fire failing notification", func() (any, error) { return c.Fire()(ctx, &notify.FirePayload{V: ptr("fail")}) }, nil, false, true, 1},
		{"fire request", func() (any, error) { return c.Fire()(ctx, &notify.FirePayload{ID: ptr("1"), V: ptr("c")}) }, nil, false, false, 1},
		{"fire failing request", func() (any, error) { return c.Fire()(ctx, &notify.FirePayload{ID: ptr("2"), V: ptr("fail")}) }, nil, true, false, 1},
		{"req notification", func() (any, error) { return c.Req()(ctx, &notify.ReqPayload{V: ptr("d")}) }, "", false, true, 0},
		{"req request", func() (any, error) { return c.Req()(ctx, &notify.ReqPayload{ID: "3", V: ptr("e")}) }, "req", false, false, 1},
		{"res notification", func() (any, error) { return c.Res()(ctx, &notify.ResPayload{V: ptr("f")}) }, (*notify.ResResult)(nil), false, true, 1},
		{"res request", func() (any, error) { return c.Res()(ctx, &notify.ResPayload{ID: ptr("4"), V: ptr("g")}) }, 7, false, false, 1},
		{"noid request", func() (any, error) { return c.Noid()(ctx, &notify.NoidPayload{V: ptr("h")}) }, "noid", false, false, 1},
	}
	for _, tc := range cases {
		got, err := tc.call()
		if tc.wantErr != (err != nil) {
			t.Errorf("%s: error %v, want error %t", tc.name, err, tc.wantErr)
		}
		if res, ok := got.(*notify.ResResult); ok && res != nil && res.N != nil {
			got = *res.N
		}
		if !tc.wantErr && got != tc.want {
			t.Errorf("%s: result %#v, want %#v", tc.name, got, tc.want)
		}
		if seen := svc.take(); len(seen) != tc.calls {
			t.Errorf("%s: service invoked %d times, want %d", tc.name, len(seen), tc.calls)
		}
		mu.Lock()
		if len(ids) != 1 || ids[0] == tc.notification || empty[0] != tc.notification {
			t.Errorf("%s: request has id %v and response empty %v, want notification %t", tc.name, ids, empty, tc.notification)
		}
		ids, empty = nil, nil
		mu.Unlock()
	}
}
`
