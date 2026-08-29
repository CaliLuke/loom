package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestRecursivePresenceNullabilityMatrix(t *testing.T) {
	root := RunHTTPDSL(t, recursivePresenceNullabilityDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/recursivepresence", root)

	clientCode := readGeneratedGo(t, filepath.Join(dir, "gen", "http", "recursive_presence", "client"))
	serverCode := readGeneratedGo(t, filepath.Join(dir, "gen", "http", "recursive_presence", "server"))
	require.Contains(t, serverCode, "body []loom.Nullable[string]")
	require.Contains(t, clientCode, "body map[string]loom.Nullable[[]loom.Nullable[string]]")
	require.Contains(t, clientCode, "body map[string]loom.Nullable[[]loom.Nullable[map[string]loom.Nullable[[]loom.Nullable[string]]]]")
	require.Contains(t, clientCode, `loom.InvalidNullElementError("body[key]", i)`)
	require.Contains(t, clientCode, `loom.InvalidNullElementError("body[key][*][key]", i)`)

	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "recursive_presence_matrix_test.go"),
		[]byte(recursivePresenceRuntimeHarness),
		0o600,
	))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

func recursivePresenceNullabilityDSL() {
	var PresenceRequest = Type("PresenceRequest", func() {
		Attribute("optional_batches", ArrayOf(String))
		Attribute("nullable_container", ArrayOf(String), func() {
			Nullable()
		})
		Attribute("nullable_values", MapOf(String, ArrayOf(String), func() {
			Elem(func() {
				Nullable()
			})
		}))
		Required("nullable_container", "nullable_values")
	})
	var ValidationErrors = Type("ValidationErrors", MapOf(String, ArrayOf(String)))
	var NullableValidationErrors = Type("NullableValidationErrors", MapOf(String, ArrayOf(String, func() {
		Nullable()
	})))
	var NestedValidationErrors = Type(
		"NestedValidationErrors",
		MapOf(String, ArrayOf(MapOf(String, ArrayOf(String)))),
	)

	Service("RecursivePresence", func() {
		Method("Issue324Request", func() {
			Payload(ArrayOf(String))
			HTTP(func() {
				POST("/issue-324")
			})
		})
		Method("NullableItemsRequest", func() {
			Payload(ArrayOf(String, func() {
				Nullable()
			}))
			HTTP(func() {
				POST("/nullable-items")
			})
		})
		Method("PresenceRequest", func() {
			Payload(PresenceRequest)
			HTTP(func() {
				POST("/presence")
				Body(PresenceRequest)
			})
		})
		Method("Issue327Success", func() {
			Result(ValidationErrors)
			HTTP(func() {
				GET("/issue-327")
				Response(StatusOK)
			})
		})
		Method("NullableItemsSuccess", func() {
			Result(NullableValidationErrors)
			HTTP(func() {
				GET("/nullable-items")
				Response(StatusOK)
			})
		})
		Method("NestedSuccess", func() {
			Result(NestedValidationErrors)
			HTTP(func() {
				GET("/nested")
				Response(StatusOK)
			})
		})
		Method("RejectBatches", func() {
			Error("invalid_batches", ValidationErrors)
			HTTP(func() {
				GET("/declared-error")
				Response("invalid_batches", StatusUnprocessableEntity)
			})
		})
	})
}

func readGeneratedGo(t *testing.T, dir string) string {
	t.Helper()

	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	require.NoError(t, err)
	var generated strings.Builder
	for _, file := range files {
		contents, readErr := os.ReadFile(file)
		require.NoError(t, readErr)
		generated.Write(contents)
	}
	return generated.String()
}

const recursivePresenceRuntimeHarness = `package recursivepresence_test

import (
	"context"
	json "encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	recursivepresence "example.com/recursivepresence/gen/recursive_presence"
	loomhttp "github.com/CaliLuke/loom/http"
	client "example.com/recursivepresence/gen/http/recursive_presence/client"
	server "example.com/recursivepresence/gen/http/recursive_presence/server"
)

type matrixCase struct {
	name              string
	body              string
	decode            func(string) (any, error)
	wantJSON          string
	wantErrorPath     string
	wantDeclaredError bool
}

func TestRecursivePresenceNullabilityBehavior(t *testing.T) {
	tests := []matrixCase{
		{
			name: "#324 request rejects null direct-array member",
			body: "[null]",
			decode: decodeIssue324Request,
			wantErrorPath: "body[0]",
		},
		{
			name: "#324 request retains empty direct array",
			body: "[]",
			decode: decodeIssue324Request,
			wantJSON: "[]",
		},
		{
			name: "#324 request retains concrete direct array",
			body: "[\"one\"]",
			decode: decodeIssue324Request,
			wantJSON: "[\"one\"]",
		},
		{
			name: "request accepts nullable direct-array member",
			body: "[null]",
			decode: decodeNullableItemsRequest,
			wantJSON: "[null]",
		},
		{
			name: "request accepts absent optional collection and null required container",
			body: "{\"nullable_container\":null,\"nullable_values\":{\"field\":null}}",
			decode: decodePresenceRequest,
		},
		{
			name: "request rejects null optional non-null container",
			body: "{\"optional_batches\":null,\"nullable_container\":[],\"nullable_values\":{}}",
			decode: decodePresenceRequest,
			wantErrorPath: "invalid request body",
		},
		{
			name: "request rejects absent required nullable container",
			body: "{\"nullable_values\":{}}",
			decode: decodePresenceRequest,
			wantErrorPath: "nullable_container",
		},
		{
			name: "request retains empty collections",
			body: "{\"optional_batches\":[],\"nullable_container\":[],\"nullable_values\":{}}",
			decode: decodePresenceRequest,
		},
		{
			name: "request retains concrete collections",
			body: "{\"optional_batches\":[\"one\"],\"nullable_container\":[\"two\"],\"nullable_values\":{\"field\":[\"three\"]}}",
			decode: decodePresenceRequest,
		},
		{
			name: "#327 success rejects null map-array member",
			body: "{\"field\":[null]}",
			decode: decodeIssue327Success,
			wantErrorPath: "body[key][0]",
		},
		{
			name: "#327 success retains empty map",
			body: "{}",
			decode: decodeIssue327Success,
			wantJSON: "{}",
		},
		{
			name: "#327 success retains concrete map-array",
			body: "{\"field\":[\"required\"]}",
			decode: decodeIssue327Success,
			wantJSON: "{\"field\":[\"required\"]}",
		},
		{
			name: "success accepts nullable named-alias member",
			body: "{\"field\":[null]}",
			decode: decodeNullableItemsSuccess,
			wantJSON: "{\"field\":[null]}",
		},
		{
			name: "nested success rejects null outer map value",
			body: "{\"outer\":null}",
			decode: decodeNestedSuccess,
			wantErrorPath: "body[key]",
		},
		{
			name: "nested success rejects null array map member",
			body: "{\"outer\":[null]}",
			decode: decodeNestedSuccess,
			wantErrorPath: "body[key][0]",
		},
		{
			name: "nested success rejects null inner map value",
			body: "{\"outer\":[{\"inner\":null}]}",
			decode: decodeNestedSuccess,
			wantErrorPath: "body[key][*][key]",
		},
		{
			name: "nested success rejects null inner array member",
			body: "{\"outer\":[{\"inner\":[null]}]}",
			decode: decodeNestedSuccess,
			wantErrorPath: "body[key][*][key][0]",
		},
		{
			name: "nested success retains concrete value",
			body: "{\"outer\":[{\"inner\":[\"value\"]}]}",
			decode: decodeNestedSuccess,
			wantJSON: "{\"outer\":[{\"inner\":[\"value\"]}]}",
		},
		{
			name: "declared error rejects null named-alias member",
			body: "{\"field\":[null]}",
			decode: decodeDeclaredError,
			wantErrorPath: "body[key][0]",
		},
		{
			name: "declared error accepts concrete named-alias member",
			body: "{\"field\":[\"invalid\"]}",
			decode: decodeDeclaredError,
			wantDeclaredError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.decode(test.body)
			switch {
			case test.wantErrorPath != "":
				if err == nil || !strings.Contains(err.Error(), test.wantErrorPath) {
					t.Fatalf("decode error = %v, want path %s", err, test.wantErrorPath)
				}
			case test.wantDeclaredError:
				if err == nil || strings.Contains(err.Error(), "body[key]") {
					t.Fatalf("declared error = %v", err)
				}
			default:
				if err != nil {
					t.Fatalf("decode: %v", err)
				}
				if test.wantJSON != "" {
					encoded, marshalErr := json.Marshal(result)
					if marshalErr != nil {
						t.Fatalf("marshal result: %v", marshalErr)
					}
					if got := string(encoded); got != test.wantJSON {
						t.Fatalf("decoded result = %s, want %s", got, test.wantJSON)
					}
				}
			}
			if test.wantErrorPath != "" && strings.HasPrefix(test.name, "#324 request") {
				if status := loomhttp.NewErrorResponse(context.Background(), err).StatusCode(); status != 400 {
					t.Fatalf("status = %d, want 400", status)
				}
			}
		})
	}
}

func TestIssue324GeneratedHandlerRejectsNullBeforeServiceInvocation(t *testing.T) {
	var calls atomic.Int32
	endpoints := &recursivepresence.Endpoints{
		Issue324Request: func(context.Context, any) (any, error) {
			calls.Add(1)
			return nil, nil
		},
	}
	mux := loomhttp.NewMuxer()
	generated := server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	server.Mount(mux, generated)
	httpServer := httptest.NewServer(mux)
	t.Cleanup(httpServer.Close)

	request, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		httpServer.URL+"/issue-324",
		strings.NewReader("[null]"),
	)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := httpServer.Client().Do(request)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			t.Errorf("close response body: %v", closeErr)
		}
	})
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", response.StatusCode, http.StatusBadRequest, body)
	}
	if !strings.Contains(string(body), "body[0]") {
		t.Fatalf("response body = %s, want indexed validation path", body)
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("service invocation count = %d, want 0", got)
	}
}

func decodeIssue324Request(body string) (any, error) {
	request := httptest.NewRequest(http.MethodPost, "/issue-324", strings.NewReader(body))
	return server.DecodeIssue324RequestRequest(nil, loomhttp.RequestDecoder)(request)
}

func decodeNullableItemsRequest(body string) (any, error) {
	request := httptest.NewRequest(http.MethodPost, "/nullable-items", strings.NewReader(body))
	return server.DecodeNullableItemsRequestRequest(nil, loomhttp.RequestDecoder)(request)
}

func decodePresenceRequest(body string) (any, error) {
	request := httptest.NewRequest(http.MethodPost, "/presence", strings.NewReader(body))
	return server.DecodePresenceRequestRequest(nil, loomhttp.RequestDecoder)(request)
}

func decodeIssue327Success(body string) (any, error) {
	return client.DecodeIssue327SuccessResponse(loomhttp.ResponseDecoder, false)(response(http.StatusOK, body))
}

func decodeNullableItemsSuccess(body string) (any, error) {
	return client.DecodeNullableItemsSuccessResponse(loomhttp.ResponseDecoder, false)(response(http.StatusOK, body))
}

func decodeNestedSuccess(body string) (any, error) {
	return client.DecodeNestedSuccessResponse(loomhttp.ResponseDecoder, false)(response(http.StatusOK, body))
}

func decodeDeclaredError(body string) (any, error) {
	return client.DecodeRejectBatchesResponse(loomhttp.ResponseDecoder, false)(response(http.StatusUnprocessableEntity, body))
}

func response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header: make(http.Header),
		Body: io.NopCloser(strings.NewReader(body)),
	}
}
`
