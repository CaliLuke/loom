package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	httptestdata "github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestJSONRPCUntaggedRoots(t *testing.T) {
	root := RunJSONRPCDSL(t, httptestdata.UntaggedRootsJSONRPCDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/rootunion", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "roots_test.go"), []byte(jsonRPCUntaggedRootsHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "test", "./...")
}

const jsonRPCUntaggedRootsHarness = `package rootunion_test

import (
 "context"
 "encoding/json/jsontext"
 "encoding/json/v2"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"

 client "example.com/rootunion/gen/jsonrpc/roots/client"
 server "example.com/rootunion/gen/jsonrpc/roots/server"
 svc "example.com/rootunion/gen/roots"
 loomhttp "github.com/CaliLuke/loom/http"
 "github.com/stretchr/testify/require"
)

func TestJSONRPCRoots(t *testing.T) {
 calls:=0
 var observed any
 echo:=func(_ context.Context,value any)(any,error){
  calls++
  observed=value
  return value,nil
 }
 mux:=loomhttp.NewMuxer()
 endpoints:=&svc.Endpoints{Scalar:echo,Number:echo,Any:echo,Alias:echo,Mapping:echo,Nested:echo,Nullable:echo,Widths:echo,Typed:echo,Closure:echo,Taggedclosure:echo,Nullableobject:echo,Nullablemap:echo}
 server.Mount(mux,server.New(endpoints,mux,loomhttp.RequestDecoder,loomhttp.ResponseEncoder,nil))
 for _,tc:=range []struct{method,wire string;valid bool;result string}{
  {"scalar","42",true,"42"},
  {"scalar","\"text\"",true,"\"text\""},
  {"number","42",false,""},
  {"any","null",true,"null"},
  {"any","{\"x\":1}",true,"{\"x\":1}"},
  {"alias","\"abc\"",true,"\"abc\""},
  {"alias","\"123\"",false,""},
  {"mapping","{\"a\":1}",true,"{\"a\":1}"},
  {"nested","true",true,"true"},
  {"nullable","null",true,"null"},
  {"nullableobject","null",true,"null"},
  {"nullableobject","{\"x\":0}",true,"{\"x\":0}"},
  {"nullableobject","{}",false,""},
  {"nullablemap","null",true,"null"},
  {"nullablemap","{}",true,"{}"},
  {"widths","2147483648",false,""},
  {"typed","{\"name\":\"ok\",\"count\":1,\"extra\":2}",true,"{\"name\":\"ok\",\"count\":1,\"extra\":2}"},
  {"typed","{\"name\":\"ok\",\"count\":1,\"extra\":false}",false,""},
  {"closure","{\"open\":{\"x\":1,\"extra\":2},\"closed\":{\"x\":3}}",true,"{\"open\":{\"x\":1},\"closed\":{\"x\":3}}"},
  {"closure","{\"open\":{\"x\":1},\"closed\":{\"x\":3,\"extra\":2}}",false,""},
  {"taggedclosure","{\"open\":{\"kind\":\"String\",\"content\":\"ok\"},\"closed\":{\"kind\":\"String\",\"content\":\"ok\",\"extra\":2}}",false,""},
 } {
  t.Run(tc.method+tc.wire,func(t *testing.T){
   before:=calls
   observed=nil
   wire:="{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\""+tc.method+"\",\"params\":"+tc.wire+"}"
   request:=httptest.NewRequest(http.MethodPost,"/rpc",strings.NewReader(wire))
   request.Header.Set("Content-Type","application/json")
   response:=httptest.NewRecorder()
   mux.ServeHTTP(response,request)
   require.Equal(t,http.StatusOK,response.Code,response.Body.String())
   var envelope map[string]jsontext.Value
   require.NoError(t,json.Unmarshal(response.Body.Bytes(),&envelope))
   require.Equal(t,"1",string(envelope["id"]))
   require.Equal(t,"\"2.0\"",string(envelope["jsonrpc"]))
   if tc.valid {
    require.Equal(t,before+1,calls,response.Body.String())
    require.NotNil(t,observed)
    require.JSONEq(t,tc.result,string(envelope["result"]))
    actual,err:=json.Marshal(observed,json.Deterministic(true))
    require.NoError(t,err)
    require.JSONEq(t,tc.result,string(actual))
    require.NotContains(t,envelope,"error")
   } else {
    require.Equal(t,before,calls)
    require.Nil(t,observed)
    require.NotContains(t,envelope,"result")
    var problem map[string]any
    require.NoError(t,json.Unmarshal(envelope["error"],&problem))
    require.Equal(t,float64(-32602),problem["code"])
    require.Equal(t,"invalid request body",problem["message"])
   }
  })
 }
 host:=httptest.NewServer(mux)
 defer host.Close()
 c:=client.NewClient("http",strings.TrimPrefix(host.URL,"http://"),host.Client(),loomhttp.RequestEncoder,loomhttp.ResponseDecoder,false)
 for _,wire:=range []string{"42","\"client\""} {
  var payload svc.ScalarChoice
  require.NoError(t,json.Unmarshal([]byte(wire),&payload))
  result,err:=c.Scalar()(t.Context(),&payload)
  require.NoError(t,err)
  require.Equal(t,&payload,result)
 }
}
`
