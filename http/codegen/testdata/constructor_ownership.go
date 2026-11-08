package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// ConstructorOwnership declares lexical constructor collisions or shared scalar
// response bodies with different projected sources. Ordinary and distinct-alias
// modes are controls; reverse varies source declaration order.
func ConstructorOwnership(mode string, rpc, reverse bool) {
	API("constructors", func() {
		if rpc {
			JSONRPC(func() {})
		}
	})
	if mode == "metadata" || mode == "metadata-handshake" || mode == "metadata-multi-exact" {
		constructorMetadataService(reverse, mode == "metadata-handshake", mode == "metadata-multi-exact")
		return
	}
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
		View("left", func() {
			Attribute("left")
		})
		View("right", func() {
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
			constructorPayloadMethod(mode, rpc, payload)
			return
		}
		if mode == "selector" {
			constructorSelectedMethods(rpc, pair)
			return
		}
		constructorAliasMethods(mode, rpc, reverse, required, optional)
	})
}

func constructorPayloadMethod(mode string, rpc bool, payload any) {
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
}

func constructorSelectedMethods(rpc bool, pair *expr.ResultTypeExpr) {
	for _, field := range []string{"left", "right"} {
		Method(field, func() {
			Result(pair, func() {
				View(field)
			})
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
}

func constructorAliasMethods(mode string, rpc, reverse bool, required, optional *expr.ResultTypeExpr) {
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
}

// constructorMetadataService extends the maintained ownership specimen with
// native metadata-only types. Multi-exact mode uses one typed handshake for
// both 201 and 202; cookie mode retains the separate cookie-mapped method.
func constructorMetadataService(reverse, cookies, multiExact bool) {
	detail := Type("MetadataDetail", func() {
		Attribute("message", String)
		Meta("struct:pkg:path", "metadata")
	})
	opening := Type("Opening", func() {
		Attribute("status", Int)
		Attribute("session", String)
		Attribute("detail", detail)
		Required("status", "session")
	})
	Service("constructors", func() {
		names := []string{"m", "watch"}
		if cookies {
			names = append(names, "cookie")
		}
		if reverse {
			names[0], names[1] = names[1], names[0]
		}
		for _, name := range names {
			Method(name, func() {
				if name != "m" {
					StreamingResult(String)
				} else {
					Result(String)
				}
				HTTP(func() {
					GET("/" + name)
					if name != "m" {
						ServerSentEvents()
						statuses := []int{StatusCreated}
						if multiExact {
							statuses = append(statuses, StatusAccepted)
						}
						for _, status := range statuses {
							Response(status, opening, func() {
								StatusField("status")
								if name == "cookie" {
									Cookie("session:session")
								} else {
									Header("session:X-Session")
								}
							})
						}
						return
					}
					Informational("hints", StatusEarlyHints, opening, func() {
						StatusField("status")
						Header("session:X-Session")
					})
				})
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
 value:=&views.Pair{Projected:&views.PairView{Left:&left,Right:&right}}
 for _,tc:=range []struct {
  name string
  encode func(context.Context,http.ResponseWriter,any) error
  want string
 }{
  {"left",server.EncodeLeftResponse(loomhttp.ResponseEncoder),"\"left\""},
  {"right",server.EncodeRightResponse(loomhttp.ResponseEncoder),"\"right\""},
 } {
  t.Run(tc.name,func(t *testing.T) {
   value.View=tc.name
   response:=httptest.NewRecorder()
   require.NoError(t,tc.encode(context.Background(),response,value))
   require.Equal(t,http.StatusOK,response.Code)
   require.Equal(t,tc.want,strings.TrimSpace(response.Body.String()))
  })
 }
}
`

// ConstructorOwnershipSharedHarness exercises both selected alias conversions
// through mounted generated servers: absent selected values fail safely, while
// supplied empty and nonempty values preserve their payloads.
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
    body:=response.Body.Bytes()
    absent:=method=="optional" && tc.value==nil
    if %t {
     require.Equal(t,http.StatusOK,response.Code,response.Body.String())
     var envelope map[string]jsontext.Value
     require.NoError(t,json.Unmarshal(body,&envelope))
     if absent {
      require.Empty(t,response.Header().Get("loom-view"))
      var failure map[string]jsontext.Value
      require.NoError(t,json.Unmarshal(envelope["error"],&failure))
      require.JSONEq(t,"-32603",string(failure["code"]))
      require.JSONEq(t,"\"internal server error\"",string(failure["message"]))
      require.Empty(t,envelope["result"])
      require.JSONEq(t,"1",string(envelope["id"]))
      continue
     }
     require.Empty(t,envelope["error"])
     body=envelope["result"]
    } else if absent {
     require.Equal(t,http.StatusInternalServerError,response.Code,response.Body.String())
     var problem loomhttp.ProblemResponse
     require.NoError(t,json.Unmarshal(body,&problem))
     require.Equal(t,http.StatusInternalServerError,problem.Status)
     require.Equal(t,"fault",problem.Code)
     require.Equal(t,"internal server error",problem.Detail)
     require.Contains(t,response.Header().Get("Content-Type"),"application/problem+json")
     continue
    } else {
     require.Equal(t,http.StatusOK,response.Code,response.Body.String())
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

// MixedMetadataDSL extends the constructor specimen with separate native unary,
// event and response-metadata values whose logical field names overlap.
func MixedMetadataDSL() {
	Interceptor("metadataAudit", func() {
		Description("Observes the complete client return value.")
	})
	opening := Type("MixedOpening", func() {
		Attribute("status", Int, func() {
			Enum(201, 202, 299, 303, 500)
		})
		Attribute("session", String)
		Required("status", "session")
	})
	result := Type("MixedResult", func() {
		Attribute("status", String)
		Attribute("session", String)
		Required("status", "session")
	})
	viewed := ResultType("application/vnd.mixed+json", func() {
		TypeName("MixedView")
		Attribute("status", String)
		Attribute("session", String)
		Attribute("hidden", String)
		Required("status", "session")
		View("default", func() {
			Attribute("status")
			Attribute("session")
		})
		View("full", func() {
			Attribute("status")
			Attribute("session")
			Attribute("hidden")
		})
	})
	event := ResultType("application/vnd.mixed-event+json", func() {
		TypeName("MixedEvent")
		Attribute("status", String)
		Attribute("session", String)
		Attribute("hidden", String)
		Required("status", "session", "hidden")
		View("default", func() {
			Attribute("status")
			Attribute("session")
		})
		View("full", func() {
			Attribute("status")
			Attribute("session")
			Attribute("hidden")
		})
	})
	Service("mixed", func() {
		mixedMetadataReadMethod(opening, result)
		mixedMetadataVariantMethods(opening, viewed)
		mixedMetadataViewedMethods(opening, viewed, event)
	})
}

// ImplicitHandshakeDSL declares optional handshake metadata with authored
// defaults on SSE and typed mixed methods, plus an optional member without one.
// Token members use the module-local custom representation package; the invalid
// carrier's default fails that representation's parser at runtime.
func ImplicitHandshakeDSL() {
	token := func(fallback string) func() {
		return func() {
			Meta("struct:field:type", "scalar.Token", "example.com/handshake/scalar")
			Default(fallback)
		}
	}
	opening := Type("ImplicitOpening", func() {
		Attribute("trace", String, func() {
			Default("abc")
		})
		Attribute("count", Int, func() {
			Default(7)
		})
		Attribute("session", String, func() {
			Default("sess")
		})
		Attribute("note", String)
		Attribute("token", String, token("token:default"))
		Attribute("hint", String, token("token:hint"))
	})
	invalid := Type("InvalidOpening", func() {
		Attribute("token", String, token("bad:default"))
		Attribute("session", String, func() {
			Default("sess")
		})
	})
	response := func() {
		Response(StatusOK, opening, func() {
			Header("trace:X-Trace")
			Header("count:X-Count")
			Header("note:X-Note")
			Header("token:X-Token")
			Cookie("session:sid")
		})
	}
	invalidResponse := func() {
		Response(StatusOK, invalid, func() {
			Header("token:X-Token")
			Cookie("session:sid")
		})
	}
	Service("implicit", func() {
		Method("watch", func() {
			Payload(func() {
				Attribute("explicit", Boolean)
			})
			StreamingResult(String)
			HTTP(func() {
				GET("/implicit")
				Param("explicit")
				ServerSentEvents()
				response()
			})
		})
		Method("mixed", func() {
			Result(String)
			StreamingResult(String)
			HTTP(func() {
				GET("/implicit/mixed")
				ServerSentEvents()
				response()
			})
		})
		Method("invalid", func() {
			StreamingResult(String)
			HTTP(func() {
				GET("/implicit/invalid")
				ServerSentEvents()
				invalidResponse()
			})
		})
		Method("invalidMixed", func() {
			Result(String)
			StreamingResult(String)
			HTTP(func() {
				GET("/implicit/invalid/mixed")
				ServerSentEvents()
				invalidResponse()
			})
		})
	})
}

func mixedMetadataReadMethod(opening, result expr.UserType) {
	Method("read", func() {
		ClientInterceptor("metadataAudit")
		Result(result)
		StreamingResult(String)
		HTTP(func() {
			GET("/mixed")
			ServerSentEvents()
			for _, status := range []any{StatusCreated, Status2XX, StatusFallback} {
				Response(status, opening, func() {
					StatusField("status")
					Header("session:X-Session")
				})
			}
		})
	})
}

func mixedMetadataVariantMethods(opening expr.UserType, viewed *expr.ResultTypeExpr) {
	for _, shape := range []struct {
		name   string
		result any
	}{
		{"primitive", String},
		{"collection", ArrayOf(String)},
		{"viewed", viewed},
	} {
		Method(shape.name, func() {
			if shape.name == "viewed" {
				Result(shape.result, func() {
					View("default")
				})
			} else {
				Result(shape.result)
			}
			StreamingResult(String)
			HTTP(func() {
				GET("/mixed/" + shape.name)
				ServerSentEvents()
				Response(StatusAccepted, opening, func() {
					StatusField("status")
					Header("session:X-Session")
				})
			})
		})
	}
}

func mixedMetadataViewedMethods(opening expr.UserType, viewed, event *expr.ResultTypeExpr) {
	Method("event", func() {
		Result(String)
		StreamingResult(event, func() {
			View("default")
		})
		HTTP(func() {
			GET("/mixed/event")
			ServerSentEvents()
			Response(StatusAccepted, opening, func() {
				StatusField("status")
				Header("session:X-Session")
			})
		})
	})
	Method("dynamic", func() {
		Result(viewed)
		StreamingResult(String)
		HTTP(func() {
			GET("/mixed/dynamic")
			ServerSentEvents()
			Response(StatusAccepted, opening, func() {
				StatusField("status")
				Header("session:X-Session")
			})
		})
	})
}
