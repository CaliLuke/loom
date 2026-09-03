package codegen

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
)

func TestGeneratedHTTPAnyUnionPreservesNumberSpelling(t *testing.T) {
	const modulePath = "example.com/unionany"
	root := RunHTTPDSL(t, unionAnyDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, modulePath, root)

	if err := os.WriteFile(filepath.Join(dir, "union_any_test.go"), []byte(unionAnyHarness), 0o600); err != nil {
		t.Fatalf("write Any union harness: %v", err)
	}

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

func unionAnyDSL() {
	object := Type("Object", func() {
		Attribute("name", String)
		Required("name")
	})
	result := Type("Result", func() {
		Attribute("choice", OneOf(Any, object))
		Required("choice")
	})

	Service("UnionAny", func() {
		Method("Show", func() {
			Result(result)
			HTTP(func() {
				GET("/value")
				Response(StatusOK)
			})
		})
	})
}

const unionAnyHarness = `package unionany_test

import (
	"context"
	json "encoding/json/v2"
	"net/http/httptest"
	"net/url"
	"testing"

	unionany "example.com/unionany/gen/union_any"
	client "example.com/unionany/gen/http/union_any/client"
	server "example.com/unionany/gen/http/union_any/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

const unionAnyJSON = "{\"choice\":{\"type\":\"Any\",\"value\":18446744073709551615}}"

func TestAnyUnionRoundTrip(t *testing.T) {
	var direct unionany.Result
	if err := json.Unmarshal([]byte(unionAnyJSON), &direct); err != nil {
		t.Fatalf("unmarshal direct result: %v", err)
	}
	directJSON, err := json.Marshal(&direct, json.Deterministic(true))
	if err != nil {
		t.Fatalf("marshal direct result: %v", err)
	}
	if string(directJSON) != unionAnyJSON {
		t.Fatalf("direct = %#v; direct round-trip JSON = %s, want %s", direct, directJSON, unionAnyJSON)
	}

	endpoints := &unionany.Endpoints{
		Show: func(context.Context, any) (any, error) {
			var result unionany.Result
			if err := json.Unmarshal([]byte(unionAnyJSON), &result); err != nil {
				return nil, err
			}
			return &result, nil
		},
	}
	mux := loomhttp.NewMuxer()
	generated := server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	server.Mount(mux, generated)
	httpServer := httptest.NewServer(mux)
	t.Cleanup(httpServer.Close)

	u, err := url.Parse(httpServer.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	generatedClient := client.NewClient(u.Scheme, u.Host, httpServer.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	response, err := generatedClient.Show()(t.Context(), nil)
	if err != nil {
		t.Fatalf("call generated client: %v", err)
	}
	result, ok := response.(*unionany.Result)
	if !ok {
		t.Fatalf("result type = %T, want *unionany.Result", response)
	}
	encoded, err := json.Marshal(result, json.Deterministic(true))
	if err != nil {
		t.Fatalf("marshal round-trip result: %v", err)
	}
	if string(encoded) != unionAnyJSON {
		t.Errorf("round-trip JSON = %s, want %s", encoded, unionAnyJSON)
	}
}
`
