package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CaliLuke/loom/expr"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/require"
)

func TestBodyTemplateDefaultContracts(t *testing.T) {
	root := RunHTTPDSL(t, testdata.BodyTemplateInheritanceDSL)
	service := root.API.HTTP.Service("templates")
	for _, tc := range []struct {
		name          string
		body, payload any
	}{
		{"inline", 7, 7},
		{"named", 7, 7},
		{"override", 9, 7},
		{"source_only", 9, nil},
		{"required", 9, 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			endpoint := service.Endpoint(tc.name)
			body := expr.NewMappedAttributeExpr(endpoint.Body)
			require.Equal(t, tc.body, body.GetDefault("count"))
			require.Equal(t, tc.payload, endpoint.MethodExpr.Payload.GetDefault("count"))
			require.Equal(t, tc.name == "required", body.IsRequired("count"))
		})
	}
	nested := service.Endpoint("nested")
	bodyObject := expr.NewMappedAttributeExpr(nested.Body).Find("object")
	payloadObject := nested.MethodExpr.Payload.Find("object")
	require.Equal(t, 9, bodyObject.GetDefault("count"))
	require.Equal(t, 7, payloadObject.GetDefault("count"))
	require.NotNil(t, nested.Body.GetDefault("object"), "the whole-object default is admitted")
	nullable := expr.NewMappedAttributeExpr(service.Endpoint("nullable").Body)
	require.True(t, expr.IsNullable(nullable.Find("value")))
	require.Equal(t, "body", nullable.GetDefault("value"))
}

func TestBodyTemplateInheritanceGeneratedHTTP(t *testing.T) {
	root := RunHTTPDSL(t, testdata.BodyTemplateInheritanceDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/bodytemplates", root)
	specs, err := openapiv3.Files(root)
	require.NoError(t, err)
	for _, spec := range specs {
		output, err := spec.Render(dir)
		require.NoError(t, err)
		data, err := os.ReadFile(output)
		require.NoError(t, err)
		document, err := libopenapi.NewDocument(data)
		require.NoError(t, err)
		_, err = document.BuildV3Model()
		require.NoError(t, err)
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "body_template_test.go"), []byte(bodyTemplateHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "-count=1", "./...")
	runGoCommand(t, dir, "vet", "./...")
}

const bodyTemplateHarness = `package bodytemplates_test

import (
 "context"
 "encoding/json/v2"
 "io"
 "net/http"
 "net/http/httptest"
 "reflect"
 "strings"
 "testing"

 svc "example.com/bodytemplates/gen/templates"
 server "example.com/bodytemplates/gen/http/templates/server"
 loomhttp "github.com/CaliLuke/loom/http"
 loom "github.com/CaliLuke/loom/pkg"
 "github.com/stretchr/testify/require"
)

func TestInheritedTemplateWireContract(t *testing.T) {
 calls := 0
 received := 0
 echo := loom.Endpoint(func(_ context.Context, payload any) (any, error) {
  calls++
  count := reflect.ValueOf(payload).Elem().FieldByName("Count")
  if count.Kind() == reflect.Pointer { count = count.Elem() }
  received = int(count.Int())
  return nil, nil
})
 endpoints := &svc.Endpoints{Inline: echo, Named: echo, Override: echo}
 mux := loomhttp.NewMuxer()
 server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
 for _, method := range []string{"inline", "named", "override"} {
  defaultValue := 7
  if method == "override" { defaultValue = 9 }
  for _, tc := range []struct { name, input string
 value int
 valid bool }{
   {"default", "{}", defaultValue, true},
   {"upper edge", "{\"n\":10}", 10, true},
   {"maximum", "{\"n\":11}", 0, false},
   {"local override", "{\"n\":0}", 0, method == "override"},
  } {
   t.Run(method+"/"+tc.name, func(t *testing.T) {
    before := calls
    request := httptest.NewRequest(http.MethodPost, "/"+method, strings.NewReader(tc.input))
    request.Header.Set("Content-Type", "application/json")
    recorder := httptest.NewRecorder()
    mux.ServeHTTP(recorder, request)
    response := recorder.Result()
    bytes, err := io.ReadAll(response.Body)
 require.NoError(t, err)
    require.NoError(t, response.Body.Close())
    if tc.valid {
     require.Equal(t, http.StatusNoContent, response.StatusCode, string(bytes))
     require.Equal(t, before+1, calls)
     require.Equal(t, tc.value, received)
    } else {
     require.Equal(t, http.StatusBadRequest, response.StatusCode, string(bytes))
     require.Equal(t, before, calls)
     var problem map[string]any
     require.NoError(t, json.Unmarshal(bytes, &problem))
     require.Equal(t, "invalid_range", problem["code"])
     require.Equal(t, "validation error", problem["detail"])
    }
})
  }
 }
}

func TestAdmittedTemplateDefaults(t *testing.T) {
	calls := 0
	var received any
	echo := loom.Endpoint(func(_ context.Context, payload any) (any, error) {
		calls++
		received = payload
		return nil, nil
	})
	endpoints := &svc.Endpoints{
		SourceOnly: echo,
		Nested: echo,
		NestedSame: echo,
		Required: echo,
		Nullable: echo,
	}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	for _, tc := range []struct {
		name, method, input, want, code, detail string
	}{
		{"source only", "source_only", "{}", "{\"count\":9}", "", ""},
		{"source only zero", "source_only", "{\"count\":0}", "{\"count\":0}", "", ""},
		{"source only null", "source_only", "{\"count\":null}", "", "decode_payload", "invalid request body"},
		{"whole object default", "nested", "{}", "{\"object\":{\"count\":9,\"labels\":[\"body\"]}}", "", ""},
		{"supplied object", "nested", "{\"object\":{}}", "{\"object\":{\"count\":9,\"labels\":[\"body\"]}}", "", ""},
		{"supplied zero and empty", "nested", "{\"object\":{\"count\":0,\"labels\":[]}}", "{\"object\":{\"count\":0,\"labels\":[]}}", "", ""},
		{"equal nested default", "nested_same", "{\"object\":{}}", "{\"object\":{\"count\":7}}", "", ""},
		{"absent nested object", "nested_same", "{}", "", "", ""},
		{"required absent", "required", "{}", "", "missing_field", "Missing required field: count"},
		{"required zero", "required", "{\"count\":0}", "{\"count\":0}", "", ""},
		{"nullable default", "nullable", "{}", "{\"value\":\"body\"}", "", ""},
		{"nullable null", "nullable", "{\"value\":null}", "{\"value\":null}", "", ""},
		{"nullable empty", "nullable", "{\"value\":\"\"}", "{\"value\":\"\"}", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := calls
			request := httptest.NewRequest(http.MethodPost, "/"+tc.method, strings.NewReader(tc.input))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if tc.code != "" {
				require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
				require.Equal(t, before, calls)
				var problem loomhttp.ProblemResponse
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &problem))
				require.Equal(t, tc.code, problem.Code)
				require.Equal(t, tc.detail, problem.Detail)
				return
			}
			require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())
			require.Equal(t, before+1, calls)
			if tc.name == "absent nested object" {
				value := reflect.ValueOf(received).Elem().FieldByName("Object")
				require.Equal(t, reflect.Pointer, value.Kind())
				require.True(t, value.IsNil(), "nested defaults do not invent an absent object")
				return
			}
			if tc.method == "source_only" {
				value := reflect.ValueOf(received).Elem().FieldByName("Count")
				require.Equal(t, reflect.Pointer, value.Kind(), "the payload has no default and retains pointer storage")
				require.False(t, value.IsNil(), "the admitted Body default is a supplied payload value")
			}
			wire, err := json.Marshal(received)
			require.NoError(t, err)
			require.JSONEq(t, tc.want, string(wire))
		})
	}
}
`
