package codegen

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
)

func TestGeneratedHTTPUnionArraysCompileAndRoundTrip(t *testing.T) {
	const modulePath = "example.com/unionarray"
	root := RunHTTPDSL(t, unionArrayDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, modulePath, root)

	if err := os.WriteFile(filepath.Join(dir, "union_array_test.go"), []byte(unionArrayHarness), 0o600); err != nil {
		t.Fatalf("write union array harness: %v", err)
	}

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

func unionArrayDSL() {
	a := Type("A", func() {
		Attribute("a", String)
		Required("a")
	})
	b := Type("B", func() {
		Attribute("b", String)
		Required("b")
	})
	item := OneOf(a, b)
	nestedItems := ArrayOf(item)
	nullableNestedItems := ArrayOf(item, func() {
		Nullable()
	})
	result := Type("Result", func() {
		Attribute("required_items", ArrayOf(item))
		Attribute("optional_items", ArrayOf(item))
		Attribute("nullable_items", ArrayOf(item, func() {
			Nullable()
		}))
		Attribute("empty_items", ArrayOf(item))
		Attribute("nested_items", ArrayOf(nestedItems))
		Attribute("nullable_nested_items", ArrayOf(nullableNestedItems))
		Required("required_items", "nullable_items", "nested_items", "nullable_nested_items")
	})

	Service("UnionCollection", func() {
		Method("Show", func() {
			Result(result)
			HTTP(func() {
				GET("/items")
				Response(StatusOK)
			})
		})
	})
}

const unionArrayHarness = `package unionarray_test

import (
	"context"
	json "encoding/json/v2"
	"net/http/httptest"
	"net/url"
	"testing"

	unioncollection "example.com/unionarray/gen/union_collection"
	client "example.com/unionarray/gen/http/union_collection/client"
	server "example.com/unionarray/gen/http/union_collection/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

const unionArrayJSON = "{\"required_items\":[{\"type\":\"A\",\"value\":{\"a\":\"alpha\"}},{\"type\":\"B\",\"value\":{\"b\":\"beta\"}}],\"optional_items\":[{\"type\":\"B\",\"value\":{\"b\":\"optional\"}}],\"nullable_items\":[null,{\"type\":\"A\",\"value\":{\"a\":\"nullable\"}}],\"empty_items\":[],\"nested_items\":[[{\"type\":\"A\",\"value\":{\"a\":\"nested-a\"}}],[{\"type\":\"B\",\"value\":{\"b\":\"nested-b\"}}]],\"nullable_nested_items\":[[null,{\"type\":\"A\",\"value\":{\"a\":\"nested-nullable\"}}],[]]}"

func TestUnionArrayRoundTrip(t *testing.T) {
	endpoints := &unioncollection.Endpoints{
		Show: func(context.Context, any) (any, error) {
			var result unioncollection.Result
			if err := json.Unmarshal([]byte(unionArrayJSON), &result); err != nil {
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
	result, ok := response.(*unioncollection.Result)
	if !ok {
		t.Fatalf("result type = %T, want *unioncollection.Result", response)
	}
	if len(result.RequiredItems) != 2 {
		t.Errorf("required item count = %d, want 2", len(result.RequiredItems))
	}
	if len(result.OptionalItems) != 1 {
		t.Errorf("optional item count = %d, want 1", len(result.OptionalItems))
	}
	if len(result.NullableItems) != 2 || !result.NullableItems[0].IsNull() {
		t.Errorf("nullable items = %#v, want leading null and one value", result.NullableItems)
	}
	if len(result.EmptyItems) != 0 {
		t.Errorf("empty item count = %d, want 0", len(result.EmptyItems))
	}
	if len(result.NestedItems) != 2 || len(result.NestedItems[0]) != 1 || len(result.NestedItems[1]) != 1 {
		t.Errorf("nested items = %#v, want two single-item arrays", result.NestedItems)
	}
	if len(result.NullableNestedItems) != 2 || len(result.NullableNestedItems[0]) != 2 || !result.NullableNestedItems[0][0].IsNull() || len(result.NullableNestedItems[1]) != 0 {
		t.Errorf("nullable nested items = %#v, want [null, value] and empty arrays", result.NullableNestedItems)
	}
	encoded, err := json.Marshal(result, json.Deterministic(true))
	if err != nil {
		t.Fatalf("marshal round-trip result: %v", err)
	}
	const want = "{\"required_items\":[{\"type\":\"A\",\"value\":{\"a\":\"alpha\"}},{\"type\":\"B\",\"value\":{\"b\":\"beta\"}}],\"optional_items\":[{\"type\":\"B\",\"value\":{\"b\":\"optional\"}}],\"nullable_items\":[null,{\"type\":\"A\",\"value\":{\"a\":\"nullable\"}}],\"nested_items\":[[{\"type\":\"A\",\"value\":{\"a\":\"nested-a\"}}],[{\"type\":\"B\",\"value\":{\"b\":\"nested-b\"}}]],\"nullable_nested_items\":[[null,{\"type\":\"A\",\"value\":{\"a\":\"nested-nullable\"}}],[]]}"
	if string(encoded) != want {
		t.Errorf("round-trip JSON = %s, want %s", encoded, want)
	}
}
`
