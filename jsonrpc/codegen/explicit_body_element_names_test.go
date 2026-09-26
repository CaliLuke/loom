package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

// TestJSONRPCMappedExplicitBodyGeneratedModule compiles and vets a JSON-RPC
// service whose explicit params bodies list payload attributes with an
// element name suffix, such as Attribute("name:n") in Body, round-trips
// payloads through the generated client and server, and sends raw requests to
// the generated server, checking that the params use the suffixes as the JSON
// names of the fields, that the body attributes have the types, validations
// and requiredness of the payload attributes, and that errors name the
// attributes.
func TestJSONRPCMappedExplicitBodyGeneratedModule(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcMappedExplicitBodyDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/jsonrpcmappedbody", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "mapped_body_test.go"), []byte(jsonRPCMappedExplicitBodyHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=1", "./...")
}

func jsonrpcMappedExplicitBodyDSL() {
	API("mappedbody", func() {
		JSONRPC(func() {})
	})
	var Account = Type("Account", func() {
		Attribute("name", String, func() {
			MinLength(2)
		})
		Attribute("age", Int)
		Required("name")
	})
	Service("mappedbody", func() {
		JSONRPC(func() {
			POST("/rpc")
		})
		Method("create", func() {
			Payload(Account)
			Result(Account)
			JSONRPC(func() {
				Body(func() {
					Attribute("name:n")
					Attribute("age:ag")
				})
			})
		})
		Method("pair", func() {
			Payload(func() {
				Attribute("a", Int)
				Attribute("b", Int)
				Required("a")
			})
			Result(Int)
			JSONRPC(func() {
				Body(func() {
					Attribute("a:x")
					Attribute("b:y")
					Required("a:x")
				})
			})
		})
	})
}

const jsonRPCMappedExplicitBodyHarness = `package jsonrpcmappedbody_test

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

	client "example.com/jsonrpcmappedbody/gen/jsonrpc/mappedbody/client"
	server "example.com/jsonrpcmappedbody/gen/jsonrpc/mappedbody/server"
	mappedbody "example.com/jsonrpcmappedbody/gen/mappedbody"
	loomhttp "github.com/CaliLuke/loom/http"
)

type service struct {
	mu       sync.Mutex
	accounts []*mappedbody.Account
	pairs    []*mappedbody.PairPayload
}

func (s *service) Create(_ context.Context, p *mappedbody.Account) (*mappedbody.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accounts = append(s.accounts, p)
	return p, nil
}

func (s *service) Pair(_ context.Context, p *mappedbody.PairPayload) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pairs = append(s.pairs, p)
	return p.A, nil
}

func (s *service) take() ([]*mappedbody.Account, []*mappedbody.PairPayload) {
	s.mu.Lock()
	defer s.mu.Unlock()
	accounts, pairs := s.accounts, s.pairs
	s.accounts, s.pairs = nil, nil
	return accounts, pairs
}

func ptr[T any](v T) *T { return &v }

func TestMappedExplicitBody(t *testing.T) {
	svc := &service{}
	mux := loomhttp.NewMuxer()
	errhandler := func(context.Context, http.ResponseWriter, error) {}
	server.Mount(mux, server.New(mappedbody.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errhandler))
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := client.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)

	for _, payload := range []*mappedbody.Account{{Name: "nm"}, {Name: "name", Age: ptr(40)}} {
		res, err := c.Create()(context.Background(), payload)
		if err != nil {
			t.Errorf("create %+v: %v", payload, err)
			continue
		}
		if accounts, _ := svc.take(); len(accounts) != 1 || !reflect.DeepEqual(accounts[0], payload) {
			t.Errorf("create: server received %+v, want %+v", accounts, payload)
		}
		if !reflect.DeepEqual(res, payload) {
			t.Errorf("create: client received %+v, want %+v", res, payload)
		}
	}
	for _, payload := range []*mappedbody.PairPayload{{A: 1}, {A: 2, B: ptr(3)}} {
		res, err := c.Pair()(context.Background(), payload)
		if err != nil {
			t.Errorf("pair %+v: %v", payload, err)
			continue
		}
		if _, pairs := svc.take(); len(pairs) != 1 || !reflect.DeepEqual(pairs[0], payload) {
			t.Errorf("pair: server received %+v, want %+v", pairs, payload)
		}
		if res != payload.A {
			t.Errorf("pair: client received %v, want %d", res, payload.A)
		}
	}

	for _, tc := range []struct {
		name    string
		method  string
		params  string
		code    int
		message string
		account *mappedbody.Account
		pair    *mappedbody.PairPayload
	}{
		{"element names", "create", ` + "`" + `{"n":"nm","ag":3}` + "`" + `, 0, "", &mappedbody.Account{Name: "nm", Age: ptr(3)}, nil},
		{"attribute names", "create", ` + "`" + `{"name":"nm","age":3}` + "`" + `, -32602, "Missing required field: name", nil, nil},
		{"too short", "create", ` + "`" + `{"n":"x"}` + "`" + `, -32602, "", nil, nil},
		{"pair", "pair", ` + "`" + `{"x":1,"y":2}` + "`" + `, 0, "", nil, &mappedbody.PairPayload{A: 1, B: ptr(2)}},
		{"pair missing required", "pair", ` + "`" + `{"y":2}` + "`" + `, -32602, "Missing required field: a", nil, nil},
		{"pair attribute names", "pair", ` + "`" + `{"a":1,"b":2}` + "`" + `, -32602, "Missing required field: a", nil, nil},
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
		accounts, pairs := svc.take()
		if tc.code != 0 {
			if envelope.Error == nil || envelope.Error.Code != tc.code || (tc.message != "" && envelope.Error.Message != tc.message) {
				t.Errorf("%s: response %s, want error (%d, %q)", tc.name, raw, tc.code, tc.message)
			}
			if len(accounts)+len(pairs) != 0 {
				t.Errorf("%s: service invoked with %+v %+v", tc.name, accounts, pairs)
			}
			continue
		}
		if envelope.Error != nil {
			t.Errorf("%s: unexpected error %s", tc.name, raw)
			continue
		}
		if tc.account != nil && (len(accounts) != 1 || !reflect.DeepEqual(accounts[0], tc.account)) {
			t.Errorf("%s: server received %+v, want %+v", tc.name, accounts, tc.account)
		}
		if tc.pair != nil && (len(pairs) != 1 || !reflect.DeepEqual(pairs[0], tc.pair)) {
			t.Errorf("%s: server received %+v, want %+v", tc.name, pairs, tc.pair)
		}
	}
}
`
