package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

// TestParameterDefaultFaultsGenerated compiles and exercises authored defaults
// that cannot be materialized. Server scalar text defaults fail with a 500
// fault without calling the service, only when the value is absent, and client
// response decoders keep their validation envelope with the fault as cause. A
// text union whose candidates meet ordinary supplied-value mismatches still
// selects its unique match; untagged branches cannot declare fallible defaults.
func TestParameterDefaultFaultsGenerated(t *testing.T) {
	root := RunHTTPDSL(t, parameterDefaultFaultsGeneratedDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/faults", root)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "label"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "label", "label.go"), []byte(jsonQueryLabelSource), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "faults_test.go"), []byte(parameterDefaultFaultsHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	t.Log(runCommand(t, dir, "go", "test", "-count=1", "-v", "./..."))
}

func parameterDefaultFaultsGeneratedDSL() {
	label := func() {
		Meta("struct:field:type", "label.Label", "example.com/faults/label")
	}
	faulty := func() {
		label()
		Default("oops")
	}
	long := Type("Long", func() {
		Attribute("name", String, func() {
			MinLength(5)
		})
		Attribute("tag", String, func() {
			Default("t")
		})
		Required("name")
	})
	short := Type("Short", func() {
		Attribute("name", String, func() {
			MaxLength(4)
		})
		Required("name")
	})
	pick := Type("Pick", OneOf(func() {
		Untagged()
		Attribute("long", long)
		Attribute("short", short)
	}))
	Service("faults", func() {
		Method("scalars", func() {
			Payload(func() {
				Attribute("q", String, faulty)
				Attribute("h", String, faulty)
				Attribute("c", String, faulty)
				Attribute("p", String, label)
			})
			HTTP(func() {
				GET("/scalars")
				Param("q")
				Param("p")
				Header("h:X-H")
				Cookie("c")
				Response(StatusNoContent)
			})
		})
		Method("pick", func() {
			Payload(func() {
				Attribute("pick", pick)
			})
			HTTP(func() {
				GET("/pick")
				Param("pick")
				Response(StatusNoContent)
			})
		})
		Method("response", func() {
			Result(func() {
				Attribute("h", String, faulty)
				Attribute("c", String, faulty)
			})
			HTTP(func() {
				GET("/response")
				Response(StatusOK, func() {
					Header("h:X-H")
					Cookie("c")
				})
			})
		})
	})
}

const parameterDefaultFaultsHarness = `package faults_test

import (
	"context"
	"encoding/json/v2"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	service "example.com/faults/gen/faults"
	client "example.com/faults/gen/http/faults/client"
	server "example.com/faults/gen/http/faults/server"
	"example.com/faults/label"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type call struct {
	target string
	header string
	cookie string
}

func serve(t *testing.T, c call) (*httptest.ResponseRecorder, []any) {
	t.Helper()
	var received []any
	endpoint := func(_ context.Context, payload any) (any, error) {
		received = append(received, payload)
		return nil, nil
	}
	mux := loomhttp.NewMuxer()
	endpoints := &service.Endpoints{Scalars: endpoint, Pick: endpoint, Response: endpoint}
	server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	request := httptest.NewRequest("GET", c.target, nil)
	if c.header != "" {
		request.Header.Set("X-H", c.header)
	}
	if c.cookie != "" {
		request.AddCookie(&http.Cookie{Name: "c", Value: c.cookie})
	}
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	return response, received
}

func requireProblem(t *testing.T, response *httptest.ResponseRecorder, status int, code string, fault bool) {
	t.Helper()
	require.Equal(t, status, response.Code, response.Body.String())
	var problem loomhttp.ProblemResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &problem))
	assert.Equal(t, code, problem.Code, response.Body.String())
	assert.Equal(t, fault, problem.Fault, response.Body.String())
}

func TestScalarDefaultsFaultOnlyWhenAbsent(t *testing.T) {
	supplied := call{target: "/scalars?q=label:a", header: "label:b", cookie: "label:c"}
	response, received := serve(t, supplied)
	require.Equal(t, 204, response.Code, response.Body.String())
	require.Len(t, received, 1)
	payload := received[0].(*service.ScalarsPayload)
	assert.Equal(t, label.Label{Text: "a"}, payload.Q)
	assert.Equal(t, label.Label{Text: "b"}, payload.H)
	assert.Equal(t, label.Label{Text: "c"}, payload.C)
	assert.Nil(t, payload.P, "an optional custom text value without a default stays absent")

	response, received = serve(t, call{target: "/scalars?q=label:a&p=label:p", header: "label:b", cookie: "label:c"})
	require.Equal(t, 204, response.Code, response.Body.String())
	require.Equal(t, &label.Label{Text: "p"}, received[0].(*service.ScalarsPayload).P)

	for _, test := range []struct {
		name string
		call call
	}{
		{"absent query", call{target: "/scalars", header: "label:b", cookie: "label:c"}},
		{"absent header", call{target: "/scalars?q=label:a", cookie: "label:c"}},
		{"absent cookie", call{target: "/scalars?q=label:a", header: "label:b"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			response, received := serve(t, test.call)
			requireProblem(t, response, 500, "fault", true)
			require.Empty(t, received)
		})
	}

	response, received = serve(t, call{target: "/scalars?q=", header: "label:b", cookie: "label:c"})
	requireProblem(t, response, 400, "invalid_field_type", false)
	require.Empty(t, received, "a supplied empty value reaches its codec and is never defaulted")
}

func TestTextUnionOrdinaryMismatches(t *testing.T) {
	for _, test := range []struct {
		name   string
		target string
		status int
	}{
		{"long candidate mismatch selects short", "/pick?name=abc", 204},
		{"short candidate mismatch selects long with its absent default", "/pick?name=abcdef", 204},
		{"supplied member selects long", "/pick?name=abcdef&tag=x", 204},
		{"mismatches in every candidate are a client error", "/pick?tag=x", 400},
	} {
		t.Run(test.name, func(t *testing.T) {
			response, received := serve(t, call{target: test.target})
			if test.status == 204 {
				require.Equal(t, 204, response.Code, response.Body.String())
				require.Len(t, received, 1)
				return
			}
			requireProblem(t, response, 400, "decode_payload", false)
			require.Empty(t, received)
		})
	}
}

func TestResponseDefaultFaultCause(t *testing.T) {
	decode := client.DecodeResponseResponse(loomhttp.ResponseDecoder, false)

	recorder := httptest.NewRecorder()
	recorder.Header().Set("X-H", "label:h")
	http.SetCookie(recorder, &http.Cookie{Name: "c", Value: "label:c"})
	recorder.WriteHeader(200)
	result, err := decode(recorder.Result())
	require.NoError(t, err)
	assert.Equal(t, &service.ResponseResult{H: label.Label{Text: "h"}, C: label.Label{Text: "c"}}, result)

	recorder = httptest.NewRecorder()
	recorder.WriteHeader(200)
	_, err = decode(recorder.Result())
	require.Error(t, err)
	var clientErr *loomhttp.ClientError
	require.True(t, errors.As(err, &clientErr), "the response keeps its client envelope: %v", err)
	assert.Equal(t, "validation_error", clientErr.Name)
	var fault *loom.ServiceError
	require.True(t, errors.As(err, &fault), "the fault is the envelope's cause: %v", err)
	assert.True(t, fault.Fault)
}
`
