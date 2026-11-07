package testdata

// UntaggedRootsHarness checks selection, retained values and HTTP error contracts.
const UntaggedRootsHarness = `package rootunion_test

import (
 "context"
 "encoding/json/v2"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"

 server "example.com/rootunion/gen/http/roots/server"
 svc "example.com/rootunion/gen/roots"
 loomhttp "github.com/CaliLuke/loom/http"
 "github.com/stretchr/testify/require"
)

func TestRootWireContracts(t *testing.T) {
 calls:=0
 var observed any
 echo:=func(_ context.Context,value any)(any,error){
  calls++
  observed=value
  return value,nil
 }
 mux:=loomhttp.NewMuxer()
 endpoints:=&svc.Endpoints{Scalar:echo,Number:echo,Any:echo,Alias:echo,Mapping:echo,Nested:echo,Nullable:echo,Widths:echo,Typed:echo,Closure:echo,Taggedclosure:echo,Nullableobject:echo,Nullablemap:echo}
 server.Mount(mux,server.New(endpoints,mux,loomhttp.RequestDecoder,loomhttp.ResponseEncoder,nil,nil))
 for _,tc:=range []struct{name,route,wire string;status int;detail string}{
  {"scalar-string","scalar","\"abc\"",200,""},
  {"scalar-int","scalar","42",200,""},
  {"scalar-ambiguous-int-number","number","42",400,"matched 2 branches in schema"},
  {"any-non-string","any","{\"x\":1}",200,""},
  {"any-null","any","null",200,""},
  {"any-string-ambiguous","any","\"abc\"",400,"matched 2 branches in schema"},
  {"alias-pattern","alias","\"abc\"",200,""},
  {"alias-rejected","alias","\"123\"",400,"matched 0 branches in schema"},
  {"map-root","mapping","{\"a\":1}",200,""},
  {"map-invalid","mapping","{\"a\":false}",400,"matched 0 branches in schema"},
  {"nested-union","nested","true",200,""},
  {"nested-string","nested","\"abc\"",200,""},
  {"nullable-branch-null","nullable","null",200,""},
  {"nullable-branch-string","nullable","\"abc\"",200,""},
  {"nullable-object-null","nullableobject","null",200,""},
  {"nullable-object-value","nullableobject","{\"x\":0}",200,""},
  {"nullable-object-required","nullableobject","{}",400,"matched 0 branches in schema"},
  {"nullable-map-null","nullablemap","null",200,""},
  {"nullable-map-empty","nullablemap","{}",200,""},
  {"null-not-nullable-400","scalar","null",400,"matched 0 branches in schema"},
  {"width-split-400","widths","2147483648",400,"matched 2 branches in schema"},
  {"typed-extra","typed","{\"name\":\"ok\",\"count\":1,\"extra\":2}",200,""},
  {"typed-invalid-extra","typed","{\"name\":\"ok\",\"count\":1,\"extra\":false}",400,"matched 0 branches in schema"},
  {"typed-missing-required","typed","{\"name\":\"ok\"}",400,"matched 0 branches in schema"},
 } {
  t.Run(tc.name,func(t *testing.T){
   if tc.status!=200 {
    require.ErrorContains(t,json.Unmarshal([]byte(tc.wire),rootCandidate(tc.route)),tc.detail)
   }
   before:=calls
   observed=nil
   request:=httptest.NewRequest(http.MethodPost,"/"+tc.route,strings.NewReader(tc.wire))
   request.Header.Set("Content-Type","application/json")
   response:=httptest.NewRecorder()
   mux.ServeHTTP(response,request)
   require.Equal(t,tc.status,response.Code,response.Body.String())
   if tc.status==200 {
    require.Equal(t,before+1,calls)
    require.NotNil(t,observed)
    actual,err:=json.Marshal(observed,json.Deterministic(true))
    require.NoError(t,err)
    require.JSONEq(t,tc.wire,string(actual))
    require.JSONEq(t,tc.wire,response.Body.String())
   } else {
    require.Equal(t,before,calls)
    require.Nil(t,observed)
    var problem map[string]any
    require.NoError(t,json.Unmarshal(response.Body.Bytes(),&problem))
    require.Equal(t,"decode_payload",problem["code"])
    require.Equal(t,"invalid request body",problem["detail"])
   }
  })
 }
}

func rootCandidate(route string) any {
 switch route {
 case "scalar":
  return new(svc.ScalarChoice)
 case "number":
  return new(svc.NumberChoice)
 case "any":
  return new(svc.AnyChoice)
 case "alias":
  return new(svc.AliasChoice)
 case "mapping":
  return new(svc.MapChoice)
 case "nested":
  return new(svc.NestedChoice)
 case "nullable":
  return new(svc.NullableChoice)
 case "widths":
  return new(svc.WidthChoice)
 case "nullableobject":
  return new(svc.NullableObjectChoice)
 case "nullablemap":
  return new(svc.NullableMapChoice)
 default:
  return new(svc.RecordChoice)
 }
}

func TestRootEncodeRejectsAmbiguity(t *testing.T) {
 value:=svc.NewNumberChoiceInt(42)
 _,err:=json.Marshal(value)
 require.ErrorContains(t,err,"matched 2 branches in schema")
 var retained svc.ScalarChoice
 require.NoError(t,json.Unmarshal([]byte("\"keep\""),&retained))
 before:=retained
 require.Error(t,json.Unmarshal([]byte("null"),&retained))
 require.Equal(t,before,retained)
}

func TestOccurrenceClosure(t *testing.T) {
 calls:=0
 var observed any
 echo:=func(_ context.Context,value any)(any,error){
  calls++
  observed=value
  return value,nil
 }
 mux:=loomhttp.NewMuxer()
 endpoints:=&svc.Endpoints{Closure:echo,Taggedclosure:echo}
 server.Mount(mux,server.New(endpoints,mux,loomhttp.RequestDecoder,loomhttp.ResponseEncoder,nil,nil))
 for _,tc:=range []struct{route,wire string;status int;result string}{
  {"closure","{\"open\":{\"x\":1,\"extra\":2},\"closed\":{\"x\":3}}",200,"{\"open\":{\"x\":1},\"closed\":{\"x\":3}}"},
  {"closure","{\"open\":{\"x\":1},\"closed\":{\"x\":3,\"extra\":2}}",400,""},
  {"closure","{\"open\":\"ok\",\"closed\":\"ok\"}",200,"{\"open\":\"ok\",\"closed\":\"ok\"}"},
  {"taggedclosure","{\"open\":{\"kind\":\"String\",\"content\":\"ok\",\"extra\":2},\"closed\":{\"kind\":\"String\",\"content\":\"ok\"}}",200,"{\"open\":{\"kind\":\"String\",\"content\":\"ok\"},\"closed\":{\"kind\":\"String\",\"content\":\"ok\"}}"},
  {"taggedclosure","{\"open\":{\"kind\":\"String\",\"content\":\"ok\"},\"closed\":{\"kind\":\"String\",\"content\":\"ok\",\"extra\":2}}",400,""},
 } {
  before:=calls
  observed=nil
  request:=httptest.NewRequest(http.MethodPost,"/"+tc.route,strings.NewReader(tc.wire))
  request.Header.Set("Content-Type","application/json")
  response:=httptest.NewRecorder()
  mux.ServeHTTP(response,request)
  require.Equal(t,tc.status,response.Code,response.Body.String())
  if tc.status==200 {
   require.Equal(t,before+1,calls)
   require.NotNil(t,observed)
   actual,err:=json.Marshal(observed,json.Deterministic(true))
   require.NoError(t,err)
   require.JSONEq(t,tc.result,string(actual))
   require.JSONEq(t,tc.result,response.Body.String())
  } else {
   require.Equal(t,before,calls)
   require.Nil(t,observed)
   var problem map[string]any
   require.NoError(t,json.Unmarshal(response.Body.Bytes(),&problem))
   require.Equal(t,"decode_payload",problem["code"])
   require.Equal(t,"invalid request body",problem["detail"])
  }
 }
 var retained svc.ClosureEnvelope
 require.NoError(t,json.Unmarshal([]byte("{\"open\":\"ok\",\"closed\":{\"x\":3}}"),&retained))
 before,err:=json.Marshal(retained.Closed,json.Deterministic(true))
 require.NoError(t,err)
 require.ErrorContains(t,json.Unmarshal([]byte("{\"x\":4,\"extra\":2}"),&retained.Closed),"unevaluated")
 after,err:=json.Marshal(retained.Closed,json.Deterministic(true))
 require.NoError(t,err)
 require.Equal(t,string(before),string(after))
}
`
