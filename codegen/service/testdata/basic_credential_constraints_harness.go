package testdata

import "strings"

// BasicCredentialConstraintsHarness exercises the generated transport's Basic
// credential constraints and records calls at authentication/service boundaries.
func BasicCredentialConstraintsHarness(transport string) string {
	source := basicCredentialConstraintsHarness
	if transport == "http" {
		source = strings.ReplaceAll(source, "TRANSPORT", "http")
		source = strings.ReplaceAll(source, "IS_RPC", "false")
		source = strings.ReplaceAll(source, "EXTRA_METHODS", `methods = append(methods, "multipart", "custom", "bound")`)
		source = strings.ReplaceAll(source, "EXTRA_ENDPOINTS", `Bound: svc.NewBoundEndpoint(impl, auth), Multipart: svc.NewMultipartEndpoint(impl, auth), Custom: svc.NewCustomEndpoint(impl, auth),`)
		source = strings.ReplaceAll(source, "EXTRA_SERVER_ARGS", `, nil, func(_ *multipart.Reader, p **svc.CustomPayload) error {
 *p = &svc.CustomPayload{}
 return nil
}`)
		source = strings.ReplaceAll(source, "EXTRA_IMPLEMENTATIONS", `func (s *implementation) Bound(_ context.Context, p *svc.BoundPayload) error {
return s.record(p)
}
func (s *implementation) Multipart(_ context.Context, p *svc.MultipartPayload) error {
return s.record(p)
}
func (s *implementation) Custom(_ context.Context, p *svc.CustomPayload) error {
return s.record(p)
}`)
	} else {
		source = strings.ReplaceAll(source, "TRANSPORT", "jsonrpc")
		source = strings.ReplaceAll(source, "IS_RPC", "true")
		for _, marker := range []string{"EXTRA_METHODS", "EXTRA_ENDPOINTS", "EXTRA_SERVER_ARGS", "EXTRA_IMPLEMENTATIONS"} {
			source = strings.ReplaceAll(source, marker, "")
		}
	}
	return source
}

const basicCredentialConstraintsHarness = `package basic_test

import (
 "bytes"
 "context"
 "encoding/json/v2"
 "fmt"
 "mime/multipart"
 "net/http"
 "net/http/httptest"
 "reflect"
 "strings"
 "testing"

 svc "example.com/basic/gen/basic"
 server "example.com/basic/gen/TRANSPORT/basic/server"
 loomhttp "github.com/CaliLuke/loom/http"
 "github.com/CaliLuke/loom/security"
 "github.com/stretchr/testify/assert"
 "github.com/stretchr/testify/require"
)

type implementation struct {
 svc.Service
 calls int
 payload any
}
func (s *implementation) record(p any) error {
s.calls++
 s.payload = p
 return nil
}
func (s *implementation) User(_ context.Context, p *svc.UserPayload) error {
return s.record(p)
}
func (s *implementation) Pass(_ context.Context, p *svc.PassPayload) error {
return s.record(p)
}
func (s *implementation) Optional(_ context.Context, p *svc.OptionalPayload) error {
return s.record(p)
}
func (s *implementation) Defaulted(_ context.Context, p *svc.DefaultedPayload) error {
return s.record(p)
}
func (s *implementation) Refined(_ context.Context, p *svc.RefinedPayload) error {
return s.record(p)
}
EXTRA_IMPLEMENTATIONS

func TestBasicCredentialConstraints(t *testing.T) {
 methods := []string{"user", "pass", "optional", "defaulted", "refined"}
 EXTRA_METHODS
 for _, method := range methods {
  for _, tc := range []struct {name, user, pass, code string; absent bool}{
   {name:"valid", user:"alice", pass:"secret@example.com"},
   {name:"unicode", user:"éé", pass:"秘密"},
   {name:"short user", user:"x", pass:"secret@example.com", code:"invalid_length"},
   {name:"short password", user:"alice", pass:"x", code:"invalid_length"},
   {name:"empty password", user:"alice", code:"invalid_length"},
   {name:"absent", absent:true},
   {name:"refined enum", user:"carol", pass:"secret@example.com"},
   {name:"refined max", user:"charlie", pass:"secret@example.com"},
   {name:"refined format", user:"alice", pass:"secret"},
  } {
   t.Run(method+"/"+tc.name, func(t *testing.T) {
    impl, authCalls := &implementation{}, 0
    auth := func(ctx context.Context, _, _ string, _ *security.BasicScheme) (context.Context, error) {
 authCalls++
 return ctx, nil
}
    endpoints := &svc.Endpoints{
     User: svc.NewUserEndpoint(impl, auth), Pass: svc.NewPassEndpoint(impl, auth),
     Optional: svc.NewOptionalEndpoint(impl, auth), Defaulted: svc.NewDefaultedEndpoint(impl, auth),
     Refined: svc.NewRefinedEndpoint(impl, auth), EXTRA_ENDPOINTS
    }
    mux := loomhttp.NewMuxer()
    server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil EXTRA_SERVER_ARGS))
    req := httptest.NewRequest(http.MethodPost, "/"+method, strings.NewReader("{}"))
    req.Header.Set("Content-Type", "application/json")
    if IS_RPC {
     req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"method\":%q,\"params\":{},\"id\":1}",method)))
     req.Header.Set("Content-Type", "application/json")
    }
    if method == "multipart" || method == "custom" {
     var body bytes.Buffer
     writer := multipart.NewWriter(&body)
     require.NoError(t, writer.WriteField("unused", "value"))
     require.NoError(t, writer.Close())
     req = httptest.NewRequest(http.MethodPost, "/"+method, &body)
     req.Header.Set("Content-Type", writer.FormDataContentType())
    }
    if method == "bound" {
     req.URL.RawQuery = "count=1"
     if tc.absent {
req.URL.RawQuery = "count=0"
}
    }
    if !tc.absent {
req.SetBasicAuth(tc.user,tc.pass)
}
    rec := httptest.NewRecorder()
    mux.ServeHTTP(rec,req)
    invalid := tc.code != ""
    if tc.absent && (method == "user" || method == "pass" || method == "bound") {
invalid = true
}
    if method == "refined" && (tc.name == "unicode" || strings.HasPrefix(tc.name,"refined")) {
invalid = true
}
    if invalid {
     assert.Zero(t,authCalls)
     assert.Zero(t,impl.calls)
     var problem map[string]any
     require.NoError(t,json.Unmarshal(rec.Body.Bytes(), &problem), rec.Body.String())
     if IS_RPC {
      require.Equal(t,http.StatusOK,rec.Code)
      responseError, ok := problem["error"].(map[string]any)
      require.True(t,ok,rec.Body.String())
      require.Equal(t,float64(-32602),responseError["code"])
      require.NotEmpty(t,responseError["message"])
     } else {
      require.Equal(t,http.StatusBadRequest,rec.Code)
      if method == "refined" && (tc.name == "short password" || tc.name == "empty password") {
       require.Equal(t,"invalid_format",problem["code"])
      } else if tc.code != "" && !(method == "refined" && tc.name == "short user") {
       require.Equal(t,tc.code,problem["code"])
      } else if tc.absent {
       if method == "bound" {
require.Equal(t,"invalid_range",problem["code"])
} else {
require.Equal(t,"missing_field",problem["code"])
}
      } else if method == "refined" && (tc.name == "unicode" || tc.name == "refined enum" || tc.name == "refined max" || tc.name == "short user") {
       require.Equal(t,"invalid_enum_value",problem["code"])
      } else {
       require.Equal(t,"invalid_format",problem["code"])
      }
      require.NotEmpty(t,problem["detail"])
     }
    } else {
     require.Equal(t,1,authCalls,rec.Body.String())
     require.Equal(t,1,impl.calls,rec.Body.String())
     if IS_RPC {
require.Equal(t,http.StatusOK,rec.Code)
} else {
require.Equal(t,http.StatusNoContent,rec.Code)
}
     fields := reflect.ValueOf(impl.payload).Elem()
     for _, item := range []struct {field, want string}{{"AuthUser",tc.user},{"AuthPassword",tc.pass}} {
      value := fields.FieldByName(item.field)
      if value.Kind() == reflect.Pointer {
       if tc.absent {
require.True(t,value.IsNil())
 continue
}
       require.False(t,value.IsNil())
       value = value.Elem()
      }
      require.Equal(t,item.want,value.String())
     }
    }
   })
  }
 }
}
`
