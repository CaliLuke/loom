package codegen

import (
	"strings"
	"testing"

	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestNamedNullableValidationGenerated(t *testing.T) {
	batch := testingx.NewModuleBatch(t, checkoutPinnedLoomModule(t), "")
	for _, shape := range []string{"object", "array", "map"} {
		t.Run(shape, func(t *testing.T) {
			root := RunHTTPDSL(t, testdata.NamedNullableValidationDSL("http", shape))
			c := batch.Case(t, shape, "example.com/namednullablevalidation")
			renderHTTPModuleFiles(t, c.Dir, c.ModulePath+"/gen", root)
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
 "strings"
 "testing"
 svc "example.com/namednullablevalidation/gen/trees"
 server "example.com/namednullablevalidation/gen/http/trees/server"
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
    server.Mount(mux,server.New(endpoints,mux,loomhttp.RequestDecoder,loomhttp.ResponseEncoder,nil,nil))
    req:=httptest.NewRequest("POST","/"+method,strings.NewReader(body))
    req.Header.Set("Content-Type","application/json")
    response:=httptest.NewRecorder()
    mux.ServeHTTP(response,req)
    rejected:=state=="invalid" || state=="absent" && strings.HasSuffix(method,"required")
    if rejected {
     require.Equal(t,400,response.Code,response.Body.String())
     require.Equal(t,0,calls)
    } else {
     require.Equal(t,204,response.Code,response.Body.String())
     require.Equal(t,1,calls)
    }
   })
  }
 }
}
`
