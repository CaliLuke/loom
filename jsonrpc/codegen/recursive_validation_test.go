package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	httptestdata "github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestRecursiveValidationGenerated(t *testing.T) {
	root := RunJSONRPCDSL(t, httptestdata.RecursiveValidationDSL("jsonrpc", "self", true, false))
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/recursivejsonrpc", root)
	mod, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	require.NoError(t, err)
	mod = []byte(strings.ReplaceAll(string(mod), "require github.com/CaliLuke/loom v1.0.0", "require github.com/CaliLuke/loom v1.10.0-alpha.9.0.20261012201303-8c7d8d711b9e"))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), mod, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "recursive_test.go"), []byte(recursiveValidationRuntime), 0o600))
	for _, args := range [][]string{{"mod", "tidy"}, {"build", "./..."}, {"vet", "./..."}, {"test", "-race", "-count=1", "./..."}} {
		runGoJSONRPCTestCommand(t, dir, args...)
	}
}

const recursiveValidationRuntime = `package recursive_test

import (
 "context"
 "encoding/json/v2"
 "encoding/json/jsontext"
 "errors"
 "fmt"
 "net/http/httptest"
 "strings"
 "testing"
 svc "example.com/recursivejsonrpc/gen/trees"
 server "example.com/recursivejsonrpc/gen/jsonrpc/trees/server"
 loomhttp "github.com/CaliLuke/loom/http"
 loom "github.com/CaliLuke/loom/pkg"
 rpc "github.com/CaliLuke/loom/jsonrpc"
 "github.com/stretchr/testify/require"
)

func TestRecursiveRequests(t *testing.T) {
 for _,invalid:=range []bool{false,true} {
  for depth:=1;depth<=6;depth++ {
   t.Run(fmt.Sprintf("invalid=%t/depth=%d",invalid,depth),func(t *testing.T) {
    leaf:="good"
    if invalid {
     leaf="BAD"
    }
    params:=fmt.Sprintf("{\"value\":%q}",leaf)
    for i:=1;i<depth;i++ {
     params="{\"value\":\"good\",\"next\":"+params+"}"
    }
    calls:=0
    endpoints:=&svc.Endpoints{Check:func(_ context.Context,p any)(any,error) {
     calls++
     node:=p.(*svc.Node)
     for i:=1;i<depth;i++ {
      require.Equal(t,svc.Label("good"),node.Value)
      require.NotNil(t,node.Next)
      node=node.Next
     }
     require.Equal(t,svc.Label("good"),node.Value)
     require.Nil(t,node.Next)
     return nil,nil
    }}
    mux:=loomhttp.NewMuxer()
    server.Mount(mux,server.New(endpoints,mux,loomhttp.RequestDecoder,loomhttp.ResponseEncoder,nil))
    body:="{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"check\",\"params\":"+params+"}"
    req:=httptest.NewRequest("POST","/rpc",strings.NewReader(body))
    req.Header.Set("Content-Type","application/json")
    response:=httptest.NewRecorder()
    mux.ServeHTTP(response,req)
    require.Equal(t,200,response.Code)
    var envelope struct {
     Error *struct {
      Code int
      Message string
      Data struct {
       Name string
      }
     }
    }
    require.NoError(t,json.Unmarshal(response.Body.Bytes(),&envelope,json.MatchCaseInsensitiveNames(true)))
    if !invalid {
     require.Nil(t,envelope.Error)
     require.Equal(t,1,calls)
     return
    }
    require.Zero(t,calls)
    require.NotNil(t,envelope.Error)
    require.Equal(t,-32602,envelope.Error.Code)
    require.Equal(t,"invalid_pattern",envelope.Error.Data.Name)
    require.Equal(t,"validation error",envelope.Error.Message)
    decodeRequest:=httptest.NewRequest("POST","/rpc",strings.NewReader(params))
    decodeRequest.Header.Set("Content-Type","application/json")
    _,decodeErr:=server.DecodeCheckRequest(mux,loomhttp.RequestDecoder)(decodeRequest,&rpc.RawRequest{Params:jsontext.Value(params)})
    var detail *loom.ServiceError
    require.True(t,errors.As(decodeErr,&detail))
    require.NotNil(t,detail.Field)
    require.Equal(t,"body"+strings.Repeat(".next",depth-1)+".value",*detail.Field)
   })
  }
 }
}
`
