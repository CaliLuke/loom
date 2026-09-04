package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestNullableNamedResponseBodyUsesPresenceValueType(t *testing.T) {
	endpoint := firstEndpointData(t, nullableNamedResponseBodyDSL)
	require.Len(t, endpoint.Result.Responses, 1)
	response := endpoint.Result.Responses[0]
	require.Len(t, response.ServerBody, 1)
	require.NotNil(t, response.ClientBody)

	const valueRef = "loom.Nullable[GetWidgetResponseBody]"
	require.Equal(t, valueRef, response.ServerBody[0].ValueRef)
	require.Equal(t, valueRef, response.ServerBody[0].Ref)
	require.Equal(t, valueRef, response.ServerBody[0].Init.ReturnTypeRef)
	require.Equal(t, valueRef, response.ClientBody.ValueRef)
	require.Equal(t, valueRef, response.ClientBody.Ref)
	require.Equal(t, "body", response.ResultInit.ClientArgs[0].Ref)
}

func TestGeneratedNullableObjectSuccessEncodesNullAndValue(t *testing.T) {
	const modulePath = "example.com/nullableresultit"

	root := RunHTTPDSL(t, nullableObjectSuccessDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, modulePath, root)

	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "nullable_object_success_test.go"),
		[]byte(nullableObjectSuccessHarness),
		0o600,
	))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

func nullableNamedResponseBodyDSL() {
	widget := Type("Widget", func() {
		Attribute("name", String)
		Required("name")
	})
	nullableWidget := Type("NullableWidget", widget, func() {
		Nullable()
	})
	Service("Widgets", func() {
		Method("GetWidget", func() {
			Result(nullableWidget, func() {
				Nullable()
			})
			HTTP(func() {
				GET("/widget")
				Response(StatusOK)
			})
		})
	})
}

func nullableObjectSuccessDSL() {
	widget := Type("Widget", func() {
		Attribute("name", String)
		Required("name")
	})
	nullableWidget := Type("NullableWidget", widget, func() {
		Nullable()
	})
	Service("NullableSuccess", func() {
		Method("Null", func() {
			Result(nullableWidget, func() {
				Nullable()
			})
			HTTP(func() {
				GET("/null")
				Response(StatusOK)
			})
		})
		Method("Object", func() {
			Result(nullableWidget, func() {
				Nullable()
			})
			HTTP(func() {
				GET("/object")
				Response(StatusOK)
			})
		})
	})
}

const nullableObjectSuccessHarness = `package nullableresultit_test

import (
	"context"
	"encoding/json/v2"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	nullablesuccess "example.com/nullableresultit/gen/nullable_success"
	nullablesuccessserver "example.com/nullableresultit/gen/http/nullable_success/server"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

func TestNullableObjectResponses(t *testing.T) {
	endpoints := &nullablesuccess.Endpoints{
		Null: func(context.Context, any) (any, error) {
			return loom.NullValue[nullablesuccess.NullableWidget](), nil
		},
		Object: func(context.Context, any) (any, error) {
			return loom.NullableValue(nullablesuccess.NullableWidget{Name: "Ada"}), nil
		},
	}
	mux := loomhttp.NewMuxer()
	server := nullablesuccessserver.New(
		endpoints,
		mux,
		loomhttp.RequestDecoder,
		loomhttp.ResponseEncoder,
		nil,
		nil,
	)
	nullablesuccessserver.Mount(mux, server)
	httpServer := httptest.NewServer(mux)
	t.Cleanup(httpServer.Close)

	assertResponse(t, httpServer.URL+"/null", nil)
	assertResponse(t, httpServer.URL+"/object", map[string]any{"name": "Ada"})
}

func assertResponse(t *testing.T, url string, wantBody any) {
	t.Helper()
	response, err := http.Get(url)
	if err != nil {
		t.Fatalf("get generated response: %v", err)
	}
	body, readErr := io.ReadAll(response.Body)
	closeErr := response.Body.Close()
	if readErr != nil {
		t.Fatalf("read generated response: %v", readErr)
	}
	if closeErr != nil {
		t.Fatalf("close generated response: %v", closeErr)
	}
	if response.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("parse content type: %v", err)
	}
	if mediaType != "application/json" {
		t.Errorf("content type = %q, want application/json", mediaType)
	}
	var gotBody any
	if err := json.Unmarshal(body, &gotBody); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if !reflect.DeepEqual(gotBody, wantBody) {
		t.Errorf("body = %#v, want %#v", gotBody, wantBody)
	}
}
`
