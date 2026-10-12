package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestRecursiveValidationGeneratedReadiness(t *testing.T) {
	for _, test := range []struct {
		shape           string
		first, nullable bool
	}{
		{"self", false, false},
		{"self", true, false},
		{"self", false, true},
		{"self", true, true},
		{"array", false, false},
		{"map", true, false},
		{"mutual", false, false},
		{"named indirection", true, false},
	} {
		t.Run(fmt.Sprintf("%s/first=%t/nullable=%t", test.shape, test.first, test.nullable), func(t *testing.T) {
			root := RunHTTPDSL(t, testdata.RecursiveValidationDSL("http", test.shape, test.first, test.nullable))
			dir := t.TempDir()
			renderHTTPModule(t, dir, "example.com/recursivevalidation", root)
			mod, err := os.ReadFile(filepath.Join(dir, "go.mod"))
			require.NoError(t, err)
			mod = []byte(strings.ReplaceAll(string(mod), "require github.com/CaliLuke/loom v1.0.0", "require github.com/CaliLuke/loom v1.10.0-alpha.9.0.20261012201303-8c7d8d711b9e"))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), mod, 0o600))
			harness := strings.ReplaceAll(recursiveValidationReadinessHarness, "SHAPE", test.shape)
			harness = strings.ReplaceAll(harness, "NULLABLE", fmt.Sprint(test.nullable))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "recursive_test.go"), []byte(harness), 0o600))
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "test", "-race", "-count=1", "./...")
		})
	}
}

const recursiveValidationReadinessHarness = `package recursive_test
import (
 "context"
 "encoding/json/v2"
 "errors"
 "fmt"
 "net/http/httptest"
 "strings"
 "testing"
 svc "example.com/recursivevalidation/gen/trees"
 server "example.com/recursivevalidation/gen/http/trees/server"
 loomhttp "github.com/CaliLuke/loom/http"
 loom "github.com/CaliLuke/loom/pkg"
)
type service struct {
 calls int
}
func (s *service) Check(ctx context.Context, p *svc.Node) error {
 s.calls++
 if p==nil || p.Value!="good" {
  return fmt.Errorf("unexpected decoded payload: %#v",p)
 }
 return nil
}
func TestRecursiveValidation(t *testing.T) {
 for _, invalid := range []bool{false,true} {
  for depth:=1;depth<=6;depth++ {
   t.Run(fmt.Sprintf("invalid=%t/depth=%d",invalid,depth),func(t *testing.T) {
    value:="good"
    if invalid {
     value="BAD"
    }
    body:=fmt.Sprintf("{\"value\":%q}",value)
    for level:=1;level<depth;level++ {
     switch "SHAPE" {
     case "array":
      body="["+body+"]"
     case "map":
      body="{\"key\":"+body+"}"
     }
     body="{\"value\":\"good\",\"next\":"+body+"}"
    }
    runRequest(t,body,invalid)
    if invalid {
     checkDiagnostic(t,body,depth)
    }
   })
  }
 }
 runRequest(t,"{\"value\":\"good\",\"next\":null}",!NULLABLE)
}
func checkDiagnostic(t *testing.T, body string, depth int) {
 t.Helper()
 req:=httptest.NewRequest("POST","/check",strings.NewReader(body))
 req.Header.Set("Content-Type","application/json")
 _,err:=server.DecodeCheckRequest(loomhttp.NewMuxer(),loomhttp.RequestDecoder)(req)
 if err==nil {
  return
 }
 var serviceError *loom.ServiceError
 if !errors.As(err,&serviceError) || serviceError.Field==nil {
  t.Errorf("missing validation field: %v",err)
  return
 }
 edge:=".next"
 switch "SHAPE" {
 case "array":
  edge+="[*]"
 case "map":
  edge+="[key]"
 }
 want:="body"+strings.Repeat(edge,depth-1)+".value"
 if *serviceError.Field!=want {
  t.Errorf("diagnostic field=%q want %q",*serviceError.Field,want)
 }
}
func runRequest(t *testing.T, body string, invalid bool) {
 t.Helper()
 impl:=&service{}
 mux:=loomhttp.NewMuxer()
 transport:=server.New(svc.NewEndpoints(impl),mux,loomhttp.RequestDecoder,loomhttp.ResponseEncoder,nil,nil)
 server.Mount(mux,transport)
 req:=httptest.NewRequest("POST","/check",strings.NewReader(body))
 req.Header.Set("Content-Type","application/json")
 response:=httptest.NewRecorder()
 mux.ServeHTTP(response,req)
 wantStatus,wantCalls:=204,1
 if invalid {
  wantStatus,wantCalls=400,0
 }
 if response.Code!=wantStatus || impl.calls!=wantCalls {
  t.Errorf("status=%d calls=%d want status=%d calls=%d, response=%s",response.Code,impl.calls,wantStatus,wantCalls,response.Body.String())
 }
 if invalid {
  var problem loomhttp.ProblemResponse
  if err:=json.Unmarshal(response.Body.Bytes(),&problem);err!=nil {
   t.Fatal(err)
  }
  code,detail:="invalid_pattern","validation error"
  if !strings.Contains(body,"BAD") {
   code,detail="decode_payload","invalid request body"
  }
  if problem.Code!=code || problem.Detail!=detail || problem.Status!=400 {
   t.Errorf("problem=%+v want code=%s detail=%s",problem,code,detail)
  }
 }
}
`
