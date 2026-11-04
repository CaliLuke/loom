package codegen

import (
	"strings"
	"testing"

	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestNamedNullableValidationGenerated(t *testing.T) {
	batch := testingx.NewModuleBatch(t, resolveJSONRPCTestLoomModulePath(t, t.TempDir()), "")
	for _, shape := range []string{"object", "array", "map"} {
		t.Run(shape, func(t *testing.T) {
			root := RunJSONRPCDSL(t, testdata.NamedNullableValidationDSL("jsonrpc", shape))
			c := batch.Case(t, shape, "example.com/namednullablevalidation")
			renderJSONRPCModuleFiles(t, c.Dir, c.ModulePath+"/gen", root)
			c.WriteFile(t, "validation_test.go", strings.ReplaceAll(namedNullableValidationHarness, "SHAPE", shape))
		})
	}
	batch.Check(t)
}

const namedNullableValidationHarness = `package validation_test
import (
 "context"
 "net/http/httptest"
 "reflect"
 "encoding/json/v2"
 "strings"
 "testing"
 svc "example.com/namednullablevalidation/gen/trees"
 server "example.com/namednullablevalidation/gen/jsonrpc/trees/server"
 loomhttp "github.com/CaliLuke/loom/http"
 "github.com/stretchr/testify/require"
)
func TestNamedNullable(t *testing.T) {
 for _,method:=range []string{"nested_required","nested_optional","root_required","root_optional"} {
  for _,state:=range []string{"valid","invalid","absent","null"} {
   t.Run(method+"/"+state,func(t *testing.T) {
    value:="good"
    if state=="invalid" {
     value="BAD"
    }
    body:="{\"value\":\""+value+"\"}"
    switch "SHAPE" {
    case "array":
     body="["+body+"]"
    case "map":
     body="{\"key\":"+body+"}"
    }
    if state=="null" {
     body="null"
    }
    if strings.HasPrefix(method,"nested") {
     body="{\"tree\":"+body+"}"
     if state=="absent" {
      body="{}"
     }
    } else if state=="absent" {
     body=""
    }
    calls:=0
    endpoint:=func(_ context.Context,payload any)(any,error) {
     calls++
     tree:=reflect.ValueOf(payload).Elem().FieldByName("Tree")
     present:=tree.MethodByName("Present").Call(nil)[0].Bool()
     isNull:=tree.MethodByName("IsNull").Call(nil)[0].Bool()
     require.Equal(t,state!="absent",present)
     require.Equal(t,state=="null",isNull)
     return nil,nil
    }
    endpoints:=&svc.Endpoints{NestedRequired:endpoint,NestedOptional:endpoint,RootRequired:endpoint,RootOptional:endpoint}
    mux:=loomhttp.NewMuxer()
    server.Mount(mux,server.New(endpoints,mux,loomhttp.RequestDecoder,loomhttp.ResponseEncoder,nil))
    envelope := "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\""+method+"\""
    if body!="" {
     envelope+=",\"params\":"+body
    }
    envelope+="}"
    req:=httptest.NewRequest("POST","/rpc",strings.NewReader(envelope))
    req.Header.Set("Content-Type","application/json")
    response:=httptest.NewRecorder()
    mux.ServeHTTP(response,req)
    rejected:=state=="invalid" || state=="absent" && strings.HasSuffix(method,"required")
    require.Equal(t,200,response.Code,response.Body.String())
    var result map[string]any
    require.NoError(t,json.Unmarshal(response.Body.Bytes(), &result))
    if rejected {
     require.NotNil(t,result["error"],response.Body.String())
     require.Equal(t,float64(-32602),result["error"].(map[string]any)["code"])
     message:="validation error"
     if state=="absent" && method=="nested_required" {
      message="Missing required field: tree"
     }
     require.Equal(t,message,result["error"].(map[string]any)["message"])
     require.Equal(t,0,calls)
    } else {
     require.Nil(t,result["error"],response.Body.String())
     require.Equal(t,1,calls)
    }
   })
  }
 }
}
`
