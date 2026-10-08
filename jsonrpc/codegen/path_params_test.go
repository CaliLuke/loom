package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/jsonrpc/codegen/testdata"
)

const pathParamsHarness = `package paths_test

import (
	"bytes"
	"context"
	json "encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	client "example.com/paths/gen/jsonrpc/paths/client"
	server "example.com/paths/gen/jsonrpc/paths/server"
	svc "example.com/paths/gen/paths"
	loomhttp "github.com/CaliLuke/loom/http"
)

type implementation struct {
	calls   int
	payload any
}

func (s *implementation) Body(_ context.Context, p *svc.BodyPayload) (string, error) {
	s.calls++
	s.payload = p
	return "ok", nil
}
func (s *implementation) Path(_ context.Context, p *svc.PathPayload) (string, error) {
	s.calls++
	s.payload = p
	return "ok", nil
}

type doer struct {
	handler http.Handler
	body    []byte
}

func (d *doer) Do(r *http.Request) (*http.Response, error) {
	if r.Body != nil {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		if err := r.Body.Close(); err != nil {
			return nil, err
		}
		d.body = data
		r.Body = io.NopCloser(bytes.NewReader(data))
	}
	w := httptest.NewRecorder()
	d.handler.ServeHTTP(w, r)
	return w.Result(), nil
}
func TestPaths(t *testing.T) {
	impl := &implementation{}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(svc.NewEndpoints(impl), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, func(_ context.Context, _ http.ResponseWriter, err error) { t.Errorf("server error: %v", err) }))
	d := &doer{handler: mux}
	c := client.NewClient("http", "example.com", d, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	id := "request"
	for _, tc := range []struct {
		name    string
		payload any
		call    func(context.Context, any) (any, error)
	}{
		{"body", &svc.BodyPayload{ID: &id, Org: 7, Region: "west", Labels: []string{"a,b", "c"}, Name: "named"}, c.Body()},
		{"path", &svc.PathPayload{Org: 8, Region: "east", Labels: []string{"x", "y"}}, c.Path()},
	} {
		before := impl.calls
		result, err := tc.call(context.Background(), tc.payload)
		if err != nil || result != "ok" {
			t.Fatalf("%s result=%v err=%v", tc.name, result, err)
		}
		if impl.calls != before+1 || !reflect.DeepEqual(impl.payload, tc.payload) {
			t.Errorf("%s payload=%#v want %#v", tc.name, impl.payload, tc.payload)
		}
		var envelope map[string]any
		if err := json.Unmarshal(d.body, &envelope); err != nil {
			t.Fatal(err)
		}
		_, hasParams := envelope["params"]
		if hasParams != (tc.name == "body") {
			t.Errorf("%s params presence=%v: %s", tc.name, hasParams, d.body)
		}
	}
	for _, tc := range []struct{ path, body string }{
		{"/orgs/not-an-int/regions/west/labels/a,b/rpc", ` + "`" + `{"jsonrpc":"2.0","id":"request","method":"path"}` + "`" + `},
		{"/orgs/7/regions/west/labels/a,b/rpc", ` + "`" + `{"jsonrpc":"2.0","id":"request","method":"body","params":{"name":17}}` + "`" + `},
	} {
		before := impl.calls
		req := httptest.NewRequest("POST", tc.path, strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		var response struct {
			Error struct {
				Code int ` + "`" + `json:"code"` + "`" + `
			} ` + "`" + `json:"error"` + "`" + `
		}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if response.Error.Code != -32602 || impl.calls != before {
			t.Errorf("invalid capture/body: status=%d body=%s calls=%d", w.Code, w.Body, impl.calls)
		}
	}
}

`

func TestPathParamsGenerated(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.PathParamsDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/paths", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "paths_test.go"), []byte(pathParamsHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "test", "./...")
}
