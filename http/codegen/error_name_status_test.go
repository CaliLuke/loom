package codegen

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestErrorNameFromStatusPlan(t *testing.T) {
	root := RunHTTPDSL(t, testdata.OpenAPIStatusErrorNameDSL)
	services := CreateHTTPServices(root)
	service := services.Get("sharedErrorResponses")
	for _, endpoint := range root.API.HTTP.Services[0].HTTPEndpoints {
		statusSource := strings.HasPrefix(endpoint.Name(), "status_")
		ir := transportir.BuildEndpoint(endpoint)
		for _, response := range ir.Response.ErrorResponses {
			if statusSource {
				require.Equal(t, "loomError", response.StatusErrorName)
			} else {
				require.Empty(t, response.StatusErrorName)
			}
		}
	}
	for _, endpoint := range service.Endpoints {
		statusSource := strings.HasPrefix(endpoint.Method.Name, "status_")
		for _, group := range endpoint.Errors {
			for _, failure := range group.Errors {
				response := failure.Response
				require.Equal(t, statusSource, response.ErrorNameFromStatus)
				if !statusSource {
					continue
				}
				require.Empty(t, response.Headers)
				args := response.ResultInit.ClientArgs
				require.NotEmpty(t, args)
				identity := args[len(args)-1]
				require.Equal(t, "Kind", identity.FieldName)
				require.Equal(t, "string", identity.TypeRef)
				require.Equal(t, strconv.Quote(failure.Name), identity.Ref)
				require.Equal(t, endpoint.Method.Name != "status_empty", response.EncodePlan.NeedsErrorSource)
			}
		}
	}
}

func TestErrorNameFromStatusGeneratedHTTP(t *testing.T) {
	root := RunHTTPDSL(t, testdata.OpenAPIStatusErrorNameDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/statusidentity", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "identity_test.go"), []byte(errorNameStatusHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

const errorNameStatusHarness = `package statusidentity_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	service "example.com/statusidentity/gen/shared_error_responses"
	client "example.com/statusidentity/gen/http/shared_error_responses/client"
	server "example.com/statusidentity/gen/http/shared_error_responses/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func TestDecodeIdentity(t *testing.T) {
	for _, mode := range []struct {
		name string
		wire string
		decode func(*http.Response) (any, error)
	}{
		{"body", "{\"message\":\"failed\"}", client.DecodeStatusBodyResponse(loomhttp.ResponseDecoder, false)},
		{"selected", "\"failed\"", client.DecodeStatusSelectedResponse(loomhttp.ResponseDecoder, false)},
		{"empty", "", client.DecodeStatusEmptyResponse(loomhttp.ResponseDecoder, false)},
	} {
		for _, failure := range []struct {status int; name string}{
			{400, "bad_request"},
			{404, "not_found"},
		} {
			for _, header := range []string{"", "contradictory"} {
				t.Run(mode.name+"/"+failure.name+"/"+header, func(t *testing.T) {
					response := &http.Response{
						StatusCode: failure.status,
						Header: http.Header{"Content-Type": {"application/json"}},
						Body: io.NopCloser(strings.NewReader(mode.wire)),
					}
					if header != "" {
						response.Header.Set("Loom-Error", header)
					}
					result, err := mode.decode(response)
					require.Nil(t, result)
					var typed *service.ExceptionResponse
					require.ErrorAs(t, err, &typed)
					require.Equal(t, failure.name, typed.Kind)
					if mode.name != "empty" {
						require.Equal(t, "failed", *typed.Message)
					}
				})
			}
		}
	}
}

func TestDecodeInvalid(t *testing.T) {
	for _, test := range []struct {
		name, body string
		status int
		decode func(*http.Response) (any, error)
	}{
		{"malformed", "{", 400, client.DecodeStatusBodyResponse(loomhttp.ResponseDecoder, false)},
		{"constraint", "{\"message\":\"\"}", 400, client.DecodeStatusBodyResponse(loomhttp.ResponseDecoder, false)},
		{"unknown status", "{}", 418, client.DecodeStatusBodyResponse(loomhttp.ResponseDecoder, false)},
		{"required header", "{}", 401, client.DecodeFirstResponse(loomhttp.ResponseDecoder, false)},
		{"shared status header", "{}", 400, client.DecodeFirstResponse(loomhttp.ResponseDecoder, false)},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.decode(&http.Response{
				StatusCode: test.status,
				Header: http.Header{"Content-Type": {"application/json"}},
				Body: io.NopCloser(strings.NewReader(test.body)),
			})
			require.Error(t, err)
			var typed *service.ExceptionResponse
			require.NotErrorAs(t, err, &typed)
			if test.name == "required header" {
				require.ErrorContains(t, err, "loomError")
				require.ErrorContains(t, err, "header")
			}
		})
	}
}

func TestServerIdentity(t *testing.T) {
	name := "bad_request"
	message := "failed"
	calls := 0
	endpoint := func(context.Context, any) (any, error) {
		calls++
		return nil, &service.ExceptionResponse{Kind: name, Message: &message}
	}
	endpoints := &service.Endpoints{
		First: endpoint, Second: endpoint,
		StatusBody: endpoint, StatusSelected: endpoint, StatusEmpty: endpoint,
	}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	for _, failure := range []struct {name string; status int}{
		{"bad_request", 400},
		{"not_found", 404},
	} {
		name = failure.name
		for _, mode := range []string{"body", "selected", "empty"} {
			t.Run(mode+"/"+name, func(t *testing.T) {
				before := calls
				recorder := httptest.NewRecorder()
				mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/status/"+mode, nil))
				require.Equal(t, before+1, calls)
				require.Equal(t, failure.status, recorder.Code)
				require.Empty(t, recorder.Header().Get("Loom-Error"))
				require.NotContains(t, recorder.Body.String(), "loomError")
				require.NotContains(t, recorder.Body.String(), name)
				if mode == "empty" {
					require.Empty(t, recorder.Body.String())
				} else {
					require.Contains(t, recorder.Body.String(), "failed")
				}
				contracts := map[string][]loomhttp.ResponseContractCase{
					"body": server.StatusBodyResponseContractCases(),
					"selected": server.StatusSelectedResponseContractCases(),
					"empty": server.StatusEmptyResponseContractCases(),
				}
				found := false
				for _, contract := range contracts[mode] {
					if contract.StatusCode != failure.status {
						continue
					}
					found = true
					require.Equal(t, name, contract.ErrorName)
					require.True(t, contract.ErrorNameFromStatus)
					response := recorder.Result()
					require.NoError(t, loomhttp.ValidateResponseContract(response, contract))
					response.Header.Set("Loom-Error", name)
					require.ErrorContains(t, loomhttp.ValidateResponseContract(response, contract), "want empty")
				}
				require.True(t, found)
			})
		}
	}
	name = "unauthorized"
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/first", nil))
	require.Equal(t, 401, recorder.Code)
	require.Equal(t, name, recorder.Header().Get("Loom-Error"))
	_, err := client.DecodeFirstResponse(loomhttp.ResponseDecoder, false)(recorder.Result())
	var typed *service.ExceptionResponse
	require.ErrorAs(t, err, &typed)
	require.Equal(t, name, typed.Kind)
}
`
