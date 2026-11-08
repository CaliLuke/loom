package codegen

import (
	"fmt"
	"os"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestNullableUnionValidationGenerated(t *testing.T) {
	batch := testingx.NewModuleBatch(t, checkoutPinnedLoomModule(t), os.Getenv("LOOM_NULLABLE_UNION_OUTPUT"))
	for _, intrinsic := range []bool{false, true} {
		for _, hinted := range []bool{false, true} {
			name := fmt.Sprintf("intrinsic=%t/hinted=%t", intrinsic, hinted)
			t.Run(name, func(t *testing.T) {
				root := RunHTTPDSL(t, nullableUnionValidationDSL(intrinsic, hinted))
				c := batch.Case(t, name, "example.com/nullableunion")
				renderHTTPModuleFiles(t, c.Dir, c.ModulePath+"/gen", root)
				c.WriteFile(t, "validation_test.go", nullableUnionValidationHarness)
			})
		}
	}
	batch.Check(t)
}

func nullableUnionValidationDSL(intrinsic, hinted bool) func() {
	return func() {
		API("nullable union", func() {
			Meta("openapi:example", "false")
		})
		record := Type("Record", func() {
			if intrinsic {
				Nullable()
			}
			Attribute("kind", String, func() {
				Enum("cat")
			})
			Attribute("name", String)
			Required("kind", "name")
		})
		other := Type("Other", func() {
			Attribute("kind", String, func() {
				Enum("dog")
			})
			Attribute("age", Int)
			Required("kind", "age")
		})
		choice := Type("Choice", OneOf(func() {
			Untagged()
			Attribute("Record", record, func() {
				if !intrinsic {
					Nullable()
				}
			})
			Attribute("Other", other)
		}), func() {
			if hinted {
				Meta("openapi:discriminator:propertyName", "kind")
				Meta("openapi:discriminator:defaultMapping", "Record")
			}
		})
		holder := Type("Holder", func() {
			Attribute("choice", choice)
			Required("choice")
		})
		viewed := ResultType("application/vnd.nullable-holder", func() {
			TypeName("ViewedHolder")
			Attribute("choice", choice)
			Required("choice")
			View("default", func() {
				Attribute("choice")
			})
		})
		Service("nullable", func() {
			Method("echo", func() {
				Payload(holder)
				Result(holder)
				HTTP(func() {
					POST("/echo")
				})
			})
			Method("show", func() {
				Result(viewed)
				HTTP(func() {
					GET("/show")
				})
			})
		})
	}
}

const nullableUnionValidationHarness = `package nullableunion_test

import (
 "context"
 "encoding/json/v2"
 "io"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"

 client "example.com/nullableunion/gen/http/nullable/client"
 server "example.com/nullableunion/gen/http/nullable/server"
 svc "example.com/nullableunion/gen/nullable"
 loomhttp "github.com/CaliLuke/loom/http"
 "github.com/stretchr/testify/require"
)

func TestNullableBranchWire(t *testing.T) {
 calls:=0
 var observed any
 echo:=func(_ context.Context,value any)(any,error) {
  calls++
  observed=value
  return value,nil
 }
 mux:=loomhttp.NewMuxer()
 server.Mount(mux,server.New(&svc.Endpoints{Echo:echo},mux,loomhttp.RequestDecoder,loomhttp.ResponseEncoder,nil,nil))
 for _,tc:=range []struct{name,wire string;valid bool}{
  {"null","{\"choice\":null}",true},
  {"cat","{\"choice\":{\"kind\":\"cat\",\"name\":\"M\"}}",true},
  {"dog","{\"choice\":{\"kind\":\"dog\",\"age\":1}}",true},
  {"enum","{\"choice\":{\"kind\":\"bird\",\"name\":\"M\"}}",false},
  {"missing kind","{\"choice\":{\"name\":\"M\"}}",false},
  {"missing name","{\"choice\":{\"kind\":\"cat\"}}",false},
 } {
  t.Run(tc.name,func(t *testing.T) {
   before:=calls
   observed=nil
   request:=httptest.NewRequest(http.MethodPost,"/echo",strings.NewReader(tc.wire))
   request.Header.Set("Content-Type","application/json")
   response:=httptest.NewRecorder()
   mux.ServeHTTP(response,request)
   viewed:=&http.Response{StatusCode:200,Header:http.Header{"Content-Type":{"application/json"}},Body:io.NopCloser(strings.NewReader(tc.wire))}
   viewedResult,viewedErr:=client.DecodeShowResponse(loomhttp.ResponseDecoder,false)(viewed)
   if tc.valid {
    require.Equal(t,200,response.Code,response.Body.String())
    require.Equal(t,before+1,calls)
    raw,err:=json.Marshal(observed,json.Deterministic(true))
    require.NoError(t,err)
    require.JSONEq(t,tc.wire,string(raw))
    result,err:=client.DecodeEchoResponse(loomhttp.ResponseDecoder,false)(response.Result())
    require.NoError(t,err)
    raw,err=json.Marshal(result,json.Deterministic(true))
    require.NoError(t,err)
    require.JSONEq(t,tc.wire,string(raw))
    require.NoError(t,viewedErr)
    raw,err=json.Marshal(viewedResult,json.Deterministic(true))
    require.NoError(t,err)
    require.JSONEq(t,tc.wire,string(raw))
   } else {
    require.Error(t,viewedErr)
    require.Equal(t,400,response.Code,response.Body.String())
    require.Equal(t,before,calls)
    require.Nil(t,observed)
    var problem map[string]any
    require.NoError(t,json.Unmarshal(response.Body.Bytes(),&problem))
    require.Equal(t,"decode_payload",problem["code"])
    require.Equal(t,"invalid request body",problem["detail"])
    invalid:=httptest.NewRecorder()
    invalid.Header().Set("Content-Type","application/json")
    invalid.WriteHeader(200)
    _,err:=invalid.WriteString(tc.wire)
    require.NoError(t,err)
    _,err=client.DecodeEchoResponse(loomhttp.ResponseDecoder,false)(invalid.Result())
    require.Error(t,err)
   }
  })
 }
}
`
