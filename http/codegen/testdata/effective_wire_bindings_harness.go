package testdata

import "strings"

// EffectiveWireBindingsHarness checks transport decoding and actual auth/service
// invocation for the shared effective mapping fixture.
func EffectiveWireBindingsHarness(transport string) string {
	return strings.NewReplacer("SERVERARGS", map[bool]string{false: ",nil", true: ""}[transport == "jsonrpc"], "TRANSPORT", transport, "IS_RPC", map[bool]string{false: "false", true: "true"}[transport == "jsonrpc"]).Replace(effectiveWireBindingsHarness)
}

const effectiveWireBindingsHarness = `package binding_test

import (
 "context"
 "encoding/json/v2"
 "fmt"
 "net/http"
 "net/http/httptest"
 "reflect"
 "strings"
 "testing"

 svc "example.com/binding/gen/binding"
 server "example.com/binding/gen/TRANSPORT/binding/server"
 loomhttp "github.com/CaliLuke/loom/http"
 "github.com/CaliLuke/loom/security"
 "github.com/stretchr/testify/require"
)

type implementation struct {
 svc.Service
 calls int
 payload any
 accepted []string
}
func (s *implementation) record(p any) error {
 s.calls++
 s.payload=p
 return nil
}
func (s *implementation) Response(context.Context) (*svc.ResponseResult,error) {
 s.calls++
 return &svc.ResponseResult{A:"first", B:"second"},nil
}
func (s *implementation) Header(_ context.Context,p *svc.HeaderPayload) error {
 return s.record(p)
}
func (s *implementation) Either(_ context.Context,p *svc.EitherPayload) error {
 return s.record(p)
}
func (s *implementation) Explicit(_ context.Context,p *svc.ExplicitPayload) error {
 return s.record(p)
}
func (s *implementation) AndKey(_ context.Context,p *svc.AndKeyPayload) error {
 return s.record(p)
}
func (s *implementation) OrKey(_ context.Context,p *svc.OrKeyPayload) error {
 return s.record(p)
}
func (s *implementation) SharedFrame(_ context.Context,p *svc.SharedFramePayload) error {
 return s.record(p)
}
func (s *implementation) check(ctx context.Context, name, got, want string) (context.Context,error) {
 if got != want {
  return ctx,fmt.Errorf("invalid %s",name)
 }
 s.accepted=append(s.accepted,name)
 return ctx,nil
}
func (s *implementation) BasicAuth(ctx context.Context,user,pass string,_ *security.BasicScheme) (context.Context,error) {
 return s.check(ctx,"basic",user+":"+pass,"user:pass")
}
func (s *implementation) JWTAuth(ctx context.Context,token string,_ *security.JWTScheme) (context.Context,error) {
 return s.check(ctx,"jwt",token,"token")
}
func (s *implementation) APIKeyAuth(ctx context.Context,key string,_ *security.APIKeyScheme) (context.Context,error) {
 return s.check(ctx,"key",key,"key")
}
func (s *implementation) OAuth2Auth(ctx context.Context,token string,_ *security.OAuth2Scheme) (context.Context,error) {
 return s.check(ctx,"oauth",token,"token")
}
func TestEffectiveResponseBindings(t *testing.T) {
 impl:=&implementation{}
 mux:=loomhttp.NewMuxer()
 server.Mount(mux,server.New(svc.NewEndpoints(impl),mux,loomhttp.RequestDecoder,loomhttp.ResponseEncoder,nilSERVERARGS))
 path,body:="/response","{}"
 if IS_RPC {
  path="/rpc"
  body=` + "`" + `{"jsonrpc":"2.0","method":"response","id":1}` + "`" + `
 }
 req:=httptest.NewRequest(http.MethodPost,path,strings.NewReader(body))
 req.Header.Set("Content-Type","application/json")
 rec:=httptest.NewRecorder()
 mux.ServeHTTP(rec,req)
 require.Equal(t,1,impl.calls,rec.Body.String())
 require.Equal(t,http.StatusOK,rec.Code,rec.Body.String())
 if IS_RPC {
  require.Empty(t,rec.Header().Get("X-A"))
  require.Empty(t,rec.Header().Get("X-B"))
  require.JSONEq(t,` + "`" + `{"jsonrpc":"2.0","id":1,"result":{"a":"first","b":"second"}}` + "`" + `,rec.Body.String())
 } else {
  require.Equal(t,"first",rec.Header().Get("X-A"))
  require.Equal(t,"second",rec.Header().Get("X-B"))
  require.Empty(t,rec.Body.String())
 }
}
func TestEffectiveRequestHeaderBindings(t *testing.T) {
 impl:=&implementation{}
 mux:=loomhttp.NewMuxer()
 server.Mount(mux,server.New(svc.NewEndpoints(impl),mux,loomhttp.RequestDecoder,loomhttp.ResponseEncoder,nilSERVERARGS))
 path,body:="/header","{}"
 if IS_RPC {
  path="/rpc"
  body=` + "`" + `{"jsonrpc":"2.0","method":"header","params":{},"id":1}` + "`" + `
 }
 req:=httptest.NewRequest(http.MethodPost,path,strings.NewReader(body))
 req.Header.Set("Content-Type","application/json")
 req.Header.Set("X-A","first")
 req.Header.Set("X-B","second")
 rec:=httptest.NewRecorder()
 mux.ServeHTTP(rec,req)
 require.Equal(t,1,impl.calls,rec.Body.String())
 require.Equal(t,&svc.HeaderPayload{A:"first",B:"second"},impl.payload)
 if IS_RPC {
  require.Equal(t,http.StatusOK,rec.Code)
  var envelope map[string]any
  require.NoError(t,json.Unmarshal(rec.Body.Bytes(),&envelope))
  require.NotContains(t,envelope,"error")
 } else {
  require.Equal(t,http.StatusNoContent,rec.Code)
 }
}
func TestEffectiveWireAuth(t *testing.T) {
 for _,tc:=range []struct {
  method,authorization,key string
  want []string
  fields map[string]string
 }{
  {"either","Basic dXNlcjpwYXNz","",[]string{"basic"},map[string]string{"Username":"user","Password":"pass"}},
  {"either","Bearer token","",[]string{"jwt"},map[string]string{"Token":"token"}},
  {"explicit","Basic dXNlcjpwYXNz","",[]string{"basic"},map[string]string{"Username":"user","Password":"pass"}},
  {"explicit","Bearer token","",[]string{"jwt"},map[string]string{"Token":"token"}},
  {"and-key","Bearer token","key",[]string{"jwt","key"},map[string]string{"Token":"token","Key":"key"}},
  {"or-key","Bearer token","",[]string{"jwt"},map[string]string{"Token":"token"}},
  {"or-key","","key",[]string{"key"},map[string]string{"Key":"key"}},
  {"shared-frame","Bearer token","",[]string{"jwt","oauth"},map[string]string{"Token":"token"}},
 } {
  t.Run(tc.method+tc.authorization+tc.key,func(t *testing.T) {
   impl:=&implementation{}
   mux:=loomhttp.NewMuxer()
   server.Mount(mux,server.New(svc.NewEndpoints(impl),mux,loomhttp.RequestDecoder,loomhttp.ResponseEncoder,nilSERVERARGS))
   path,body:="/"+tc.method,"{}"
   if IS_RPC {
    path="/rpc"
    body=fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"method\":%q,\"params\":{},\"id\":1}",tc.method)
   }
   req:=httptest.NewRequest(http.MethodPost,path,strings.NewReader(body))
   req.Header.Set("Content-Type","application/json")
   if tc.authorization!="" {
    req.Header.Set("Authorization",tc.authorization)
   }
   if tc.key!="" {
    req.Header.Set("X-Api-Key",tc.key)
   }
   rec:=httptest.NewRecorder()
   mux.ServeHTTP(rec,req)
   require.Equal(t,1,impl.calls,rec.Body.String())
   require.Equal(t,tc.want,impl.accepted)
   if IS_RPC {
    require.Equal(t,http.StatusOK,rec.Code)
    require.NotContains(t,rec.Body.String(),"\"error\"")
   } else {
    require.Equal(t,http.StatusNoContent,rec.Code)
   }
   value:=reflect.ValueOf(impl.payload).Elem()
   for index:=range value.NumField() {
    field:=value.Field(index)
    name:=value.Type().Field(index).Name
    expected,present:=tc.fields[name]
    if !present {
     require.True(t,field.IsNil(),name)
     continue
    }
    if field.Kind()==reflect.Pointer {
     require.False(t,field.IsNil(),name)
     field=field.Elem()
    }
    require.Equal(t,expected,field.String(),name)
   }
  })
 }
}
`

// EffectiveWireBindingScopeHarness checks that replaced inherited credentials
// remain ordinary body data while only selected security authenticates requests.
func EffectiveWireBindingScopeHarness(transport string) string {
	return strings.NewReplacer("SERVERARGS", map[bool]string{false: ",nil", true: ""}[transport == "jsonrpc"], "TRANSPORT", transport, "IS_RPC", map[bool]string{false: "false", true: "true"}[transport == "jsonrpc"]).Replace(effectiveWireBindingScopeHarness)
}

const effectiveWireBindingScopeHarness = `package scope_test

import (
 "context"
 "fmt"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"
 svc "example.com/scope/gen/scope"
 server "example.com/scope/gen/TRANSPORT/scope/server"
 loomhttp "github.com/CaliLuke/loom/http"
 "github.com/CaliLuke/loom/security"
 "github.com/stretchr/testify/require"
)

type implementation struct {
 svc.Service
 calls, auth int
 payload *svc.ReadPayload
}
func (s *implementation) Read(_ context.Context,p *svc.ReadPayload) error {
 s.calls++
 s.payload=p
 return nil
}
func (s *implementation) APIKeyAuth(ctx context.Context,key string,scheme *security.APIKeyScheme) (context.Context,error) {
 s.auth++
 if scheme.Name!="local" || key!="credential" {
  return ctx,fmt.Errorf("unexpected credential %q for %q",key,scheme.Name)
 }
 return ctx,nil
}
func TestSelectedWireScope(t *testing.T) {
 impl:=&implementation{}
 mux:=loomhttp.NewMuxer()
 server.Mount(mux,server.New(svc.NewEndpoints(impl),mux,loomhttp.RequestDecoder,loomhttp.ResponseEncoder,nilSERVERARGS))
 path,body:="/","{\"other\":\"ordinary data\"}"
 if IS_RPC {
  path="/rpc"
  body="{\"jsonrpc\":\"2.0\",\"method\":\"read\",\"params\":{\"other\":\"ordinary data\"},\"id\":1}"
 }
 req:=httptest.NewRequest(http.MethodPost,path,strings.NewReader(body))
 req.Header.Set("Content-Type","application/json")
 req.Header.Set("Authorization","credential")
 rec:=httptest.NewRecorder()
 mux.ServeHTTP(rec,req)
 require.Equal(t,1,impl.calls,rec.Body.String())
 require.Equal(t,1,impl.auth)
 require.Equal(t,"credential",impl.payload.Active)
 require.Equal(t,"ordinary data",impl.payload.Other)
 if IS_RPC {
  require.Equal(t,http.StatusOK,rec.Code)
  require.NotContains(t,rec.Body.String(),"\"error\"")
 } else {
  require.Equal(t,http.StatusNoContent,rec.Code)
 }
}
`
