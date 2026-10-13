package testdata

import . "github.com/CaliLuke/loom/dsl"

// ConstructorOwnership declares lexical constructor collisions or shared scalar
// response bodies with different projected sources. Ordinary and distinct-alias
// modes are controls; reverse varies source declaration order.
func ConstructorOwnership(mode string, rpc, reverse bool) {
	API("constructors", func() {
		if rpc {
			JSONRPC(func() {})
		}
	})
	var payload any
	switch mode {
	case "handler":
		payload = Type("Handler", func() {
			Attribute("name", String)
		})
	case "response":
		payload = Type("MResponseBody", func() {
			Attribute("name", String)
		})
	case "ordinary":
		payload = Type("Input", func() {
			Attribute("name", String)
		})
	}
	text := Type("Text", String)
	optionalText := text
	if mode == "distinct" {
		optionalText = Type("OptionalText", String)
	}
	required := ResultType("application/vnd.required", func() {
		Attribute("value", text)
		Required("value")
		View("default", func() {
			Attribute("value")
		})
	})
	optional := ResultType("application/vnd.optional", func() {
		Attribute("value", optionalText)
		View("default", func() {
			Attribute("value")
		})
	})
	pair := ResultType("application/vnd.pair", func() {
		Attribute("left", text)
		Attribute("right", text)
		Required("left", "right")
		View("default", func() {
			Attribute("left")
			Attribute("right")
		})
	})
	Service("constructors", func() {
		if rpc {
			JSONRPC(func() {
				POST("/rpc")
			})
		}
		if payload != nil {
			Method("m", func() {
				Payload(payload)
				if mode == "response" {
					Result(payload)
				} else {
					Result(String)
				}
				if rpc {
					JSONRPC(func() {})
				} else {
					HTTP(func() {
						POST("/m")
					})
				}
			})
			return
		}
		if mode == "selector" {

			for _, field := range []string{"left", "right"} {
				Method(field, func() {
					Result(pair)
					response := func() {
						Response(StatusOK, func() {
							Body(field)
						})
					}
					if rpc {
						JSONRPC(response)
					} else {
						HTTP(func() {
							GET("/" + field)
							response()
						})
					}
				})
			}
			return
		}
		names := []string{"required", "optional"}
		if reverse {
			names[0], names[1] = names[1], names[0]
		}
		for _, name := range names {
			Method(name, func() {
				if name == "required" || mode == "same" {
					Result(required)
				} else {
					Result(optional)
				}
				response := func() {
					Response(StatusOK, func() {
						Body("value")
					})
				}
				if rpc {
					JSONRPC(response)
				} else {
					HTTP(func() {
						GET("/" + name)
						response()
					})
				}
			})
		}
	})
}

// ConstructorOwnershipWireHarness verifies that two selected fields of the same
// projected source retain separate conversions at generated response encoders.
const ConstructorOwnershipWireHarness = `package constructors_test

import (
 "context"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"

 "github.com/stretchr/testify/require"
 loomhttp "github.com/CaliLuke/loom/http"
 views "example.com/constructors/gen/constructors/views"
 server "example.com/constructors/gen/%s/constructors/server"
)

func TestSelectedConstructorSource(t *testing.T) {
 left,right:=views.TextView("left"),views.TextView("right")
 value:=&views.Pair{Projected:&views.PairView{Left:&left,Right:&right},View:"default"}
 for _,tc:=range []struct {
  name string
  encode func(context.Context,http.ResponseWriter,any) error
  want string
 }{
  {"left",server.EncodeLeftResponse(loomhttp.ResponseEncoder),"\"left\""},
  {"right",server.EncodeRightResponse(loomhttp.ResponseEncoder),"\"right\""},
 } {
  t.Run(tc.name,func(t *testing.T) {
   response:=httptest.NewRecorder()
   require.NoError(t,tc.encode(context.Background(),response,value))
   require.Equal(t,http.StatusOK,response.Code)
   require.Equal(t,tc.want,strings.TrimSpace(response.Body.String()))
  })
 }
}
`

// ConstructorOwnershipSharedHarness exercises both selected alias conversions
// through mounted generated servers, preserving absent, empty and nonempty values.
const ConstructorOwnershipSharedHarness = `package constructors_test

import (
 "context"
 "encoding/json/jsontext"
 "encoding/json/v2"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"

 "github.com/stretchr/testify/require"
 loomhttp "github.com/CaliLuke/loom/http"
 svc "example.com/constructors/gen/constructors"
 server "example.com/constructors/gen/%s/constructors/server"
)

type service struct {
 value *svc.Text
 required, optional int
}
func (s *service) Required(context.Context) (*svc.Required,error) {
 s.required++
 return &svc.Required{Value:svc.Text("required")},nil
}
func (s *service) Optional(context.Context) (*svc.Optional,error) {
 s.optional++
 return &svc.Optional{Value:s.value},nil
}
func TestMountedSharedConstructor(t *testing.T) {
 empty,full:=svc.Text(""),svc.Text("value")
 for _,tc:=range []struct {
 name string
 value *svc.Text
 want string
}{
  {"absent",nil,"null"},{"empty",&empty,"\"\""},{"value",&full,"\"value\""},
 } {
  t.Run(tc.name,func(t *testing.T){
   implementation:=&service{value:tc.value}
   mux:=loomhttp.NewMuxer()
   endpoint:=server.New(svc.NewEndpoints(implementation),mux,loomhttp.RequestDecoder,loomhttp.ResponseEncoder,nil%s)
   server.Mount(mux,endpoint)
   for _,method:=range []string{"required","optional"} {
    request:=httptest.NewRequest(http.MethodGet,"/"+method,nil)
    if %t {
     request=httptest.NewRequest(http.MethodPost,"/rpc",strings.NewReader("{\"jsonrpc\":\"2.0\",\"method\":\""+method+"\",\"id\":1}"))
     request.Header.Set("Content-Type","application/json")
    }
    response:=httptest.NewRecorder()
    mux.ServeHTTP(response,request)
    require.Equal(t,http.StatusOK,response.Code,response.Body.String())
    body:=response.Body.Bytes()
    if %t {
     var envelope map[string]jsontext.Value
     require.NoError(t,json.Unmarshal(body,&envelope))
     require.Empty(t,envelope["error"])
     body=envelope["result"]
    }
    want:=tc.want
    if method=="required" {
     want="\"required\""
    }
    require.JSONEq(t,want,string(body))
   }
   require.Equal(t,1,implementation.required)
   require.Equal(t,1,implementation.optional)
  })
 }
}
`
