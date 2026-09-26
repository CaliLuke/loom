package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
)

// TestJSONRPCMappedPayloadReferencesGeneratedModule compiles and vets a
// JSON-RPC service whose Body selects payload attributes declared with an
// element name suffix by their attribute names, with an attribute name
// argument and in a Body function, round-trips payloads through the generated
// client and server, and sends raw requests to the generated server, checking
// the params, the decoded payload and the errors.
func TestJSONRPCMappedPayloadReferencesGeneratedModule(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcMappedPayloadReferencesDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/jsonrpcmappedrefs", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "mapped_refs_test.go"), []byte(jsonRPCMappedPayloadReferencesHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=1", "./...")
}

// TestJSONRPCMappedOptionalValueParamsCode checks that the JSON-RPC code
// generated when Body("v") selects optional primitive, array, map, bytes and
// alias payload attributes and a defaulted string declared with the key
// "v:x" is the code generated for the same attributes declared as "v".
func TestJSONRPCMappedOptionalValueParamsCode(t *testing.T) {
	render := func(key string) string {
		services := CreateJSONRPCServices(RunJSONRPCDSL(t, jsonrpcOptionalValueParamsModuleDSL(key)))
		var code string
		require.NotPanics(t, func() {
			for _, files := range [][]*codegen.File{
				ServerTypeFiles("gen", services), ClientTypeFiles("gen", services),
				ServerFiles("gen", services), ClientFiles("gen", services), ClientCLIFiles("gen", services),
			} {
				for _, file := range files {
					code += renderCodegenFile(t, file)
				}
			}
		})
		return code
	}
	assert.Equal(t, render("v"), render("v:x"))
}

// TestJSONRPCMappedOptionalValueParamsGeneratedModule runs the generated module
// test of optional values selected with Body (see
// TestJSONRPCOptionalValueParamsGeneratedModule) on payload attributes
// declared with the key "v:x", including an optional alias and a defaulted
// string.
func TestJSONRPCMappedOptionalValueParamsGeneratedModule(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcOptionalValueParamsModuleDSL("v:x"))
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/jsonrpcoptvalue", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "optional_value_params_test.go"), []byte(jsonRPCOptionalValueParamsHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=1", "./...")
}

func jsonrpcMappedPayloadReferencesDSL() {
	API("mappedrefs", func() {
		JSONRPC(func() {})
	})
	Service("mappedrefs", func() {
		JSONRPC(func() {
			POST("/rpc")
		})
		Method("put", func() {
			Payload(func() {
				Attribute("data:d", MapOf(String, Int))
				Required("data:d")
			})
			Result(Int)
			JSONRPC(func() {
				Body("data")
			})
		})
		Method("rename", func() {
			Payload(func() {
				Attribute("name:nm", String, func() {
					MinLength(2)
				})
				Attribute("note:nt", String)
				Required("name:nm")
			})
			Result(String)
			JSONRPC(func() {
				Body(func() {
					Attribute("name")
					Attribute("note")
					Required("name")
				})
			})
		})
	})
}

const jsonRPCMappedPayloadReferencesHarness = `package jsonrpcmappedrefs_test

import (
	"context"
	json "encoding/json/v2"
	"encoding/json/jsontext"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	client "example.com/jsonrpcmappedrefs/gen/jsonrpc/mappedrefs/client"
	server "example.com/jsonrpcmappedrefs/gen/jsonrpc/mappedrefs/server"
	mappedrefs "example.com/jsonrpcmappedrefs/gen/mappedrefs"
	loomhttp "github.com/CaliLuke/loom/http"
)

type service struct {
	mu   sync.Mutex
	seen []any
}

func (s *service) record(p any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen = append(s.seen, p)
}

func (s *service) take() []any {
	s.mu.Lock()
	defer s.mu.Unlock()
	seen := s.seen
	s.seen = nil
	return seen
}

func (s *service) Put(_ context.Context, p *mappedrefs.PutPayload) (int, error) {
	s.record(p)
	return len(p.Data), nil
}

func (s *service) Rename(_ context.Context, p *mappedrefs.RenamePayload) (string, error) {
	s.record(p)
	return p.Name, nil
}

func ptr[T any](v T) *T { return &v }

func TestMappedPayloadReferences(t *testing.T) {
	svc := &service{}
	mux := loomhttp.NewMuxer()
	errhandler := func(context.Context, http.ResponseWriter, error) {}
	server.Mount(mux, server.New(mappedrefs.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errhandler))
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := client.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)

	put := &mappedrefs.PutPayload{Data: map[string]int{"a": 1, "b": 2}}
	res, err := c.Put()(context.Background(), put)
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if seen := svc.take(); len(seen) != 1 || !reflect.DeepEqual(seen[0], put) {
		t.Errorf("put: server received %+v, want %+v", seen, put)
	}
	if res != 2 {
		t.Errorf("put: client received %v, want 2", res)
	}
	for _, rename := range []*mappedrefs.RenamePayload{{Name: "ab"}, {Name: "cd", Note: ptr("n")}} {
		res, err := c.Rename()(context.Background(), rename)
		if err != nil {
			t.Errorf("rename %+v: %v", rename, err)
			continue
		}
		if seen := svc.take(); len(seen) != 1 || !reflect.DeepEqual(seen[0], rename) {
			t.Errorf("rename: server received %+v, want %+v", seen, rename)
		}
		if res != rename.Name {
			t.Errorf("rename: client received %v, want %s", res, rename.Name)
		}
	}

	for _, tc := range []struct {
		name    string
		method  string
		params  string
		code    int
		message string
		want    any
	}{
		{"map body", "put", ` + "`" + `{"a":1}` + "`" + `, 0, "", &mappedrefs.PutPayload{Data: map[string]int{"a": 1}}},
		{"attribute names", "rename", ` + "`" + `{"name":"ab","note":"n"}` + "`" + `, 0, "", &mappedrefs.RenamePayload{Name: "ab", Note: ptr("n")}},
		{"payload element names", "rename", ` + "`" + `{"nm":"ab","nt":"n"}` + "`" + `, -32602, "Missing required field: name", nil},
		{"too short", "rename", ` + "`" + `{"name":"a"}` + "`" + `, -32602, "", nil},
	} {
		body := ` + "`" + `{"jsonrpc":"2.0","id":1,"method":"` + "`" + ` + tc.method + ` + "`" + `","params":` + "`" + ` + tc.params + "}"
		resp, err := hs.Client().Post(hs.URL+"/rpc", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		raw, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("%s: read: %v", tc.name, err)
		}
		if err := resp.Body.Close(); err != nil {
			t.Fatalf("%s: close: %v", tc.name, err)
		}
		var envelope struct {
			Result jsontext.Value ` + "`" + `json:"result"` + "`" + `
			Error  *struct {
				Code    int    ` + "`" + `json:"code"` + "`" + `
				Message string ` + "`" + `json:"message"` + "`" + `
			} ` + "`" + `json:"error"` + "`" + `
		}
		if err := json.Unmarshal(raw, &envelope); err != nil {
			t.Errorf("%s: decode response %q: %v", tc.name, raw, err)
			continue
		}
		seen := svc.take()
		if tc.code != 0 {
			if envelope.Error == nil || envelope.Error.Code != tc.code || (tc.message != "" && envelope.Error.Message != tc.message) {
				t.Errorf("%s: response %s, want error (%d, %q)", tc.name, raw, tc.code, tc.message)
			}
			if len(seen) != 0 {
				t.Errorf("%s: service invoked with %+v", tc.name, seen)
			}
			continue
		}
		if envelope.Error != nil {
			t.Errorf("%s: unexpected error %s", tc.name, raw)
			continue
		}
		if len(seen) != 1 || !reflect.DeepEqual(seen[0], tc.want) {
			t.Errorf("%s: server received %+v, want %+v", tc.name, seen, tc.want)
		}
	}
}
`
