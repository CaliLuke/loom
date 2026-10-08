package generator

import (
	"fmt"
	"strings"
	"testing"

	servicedata "github.com/CaliLuke/loom/codegen/service/testdata"
)

func TestBearerCredentials(t *testing.T) {
	var entries strings.Builder
	for i := range 16 {
		fmt.Fprintf(&entries, "{%d, &svc.Token%dPayload{}, hc.EncodeToken%dRequest(nil), erase(hs.DecodeToken%dRequest(lh.NewMuxer(), nil)), gc.EncodeToken%dRequest, gs.DecodeToken%dRequest, hs.NewToken%dHandler},\n", i, i, i, i, i, i, i)
	}
	runDesignHarness(t, "example.com/bearer", servicedata.BearerCredentialsDSL, strings.ReplaceAll(bearerCredentialsHarness, "ENTRIES", entries.String()))
}

const bearerCredentialsHarness = `package bearer_test
import (
 "context"
 "net/http"
 "net/http/httptest"
 "reflect"
 "strings"
 "testing"

 svc "example.com/bearer/gen/credentials"
 hc "example.com/bearer/gen/http/credentials/client"
 hs "example.com/bearer/gen/http/credentials/server"
 gc "example.com/bearer/gen/grpc/credentials/client"
 gs "example.com/bearer/gen/grpc/credentials/server"
 lh "github.com/CaliLuke/loom/http"
 loom "github.com/CaliLuke/loom/pkg"
 "github.com/stretchr/testify/require"
 "google.golang.org/grpc/metadata"
)
func erase[T any](f func(*http.Request)(T,error)) func(*http.Request)(any,error) {
 return func(r *http.Request)(any,error) {return f(r)}
}
func tokenPayload(template any, token *string) any {
 p := reflect.New(reflect.TypeOf(template).Elem())
 f := p.Elem().FieldByName("Token")
 if token != nil {
  if f.Kind() == reflect.Pointer {f.Set(reflect.New(f.Type().Elem())); f.Elem().SetString(*token)} else {f.SetString(*token)}
 }
 return p.Interface()
}
func TestCredentialRoundTrip(t *testing.T) {
 for _, tc := range []struct {
  index int
  template any
  httpEncode func(*http.Request,any) error
  httpDecode func(*http.Request)(any,error)
  grpcEncode func(context.Context,any,*metadata.MD)(any,error)
  grpcDecode func(context.Context,any,metadata.MD)(any,error)
 handler func(loom.Endpoint,lh.Muxer,func(*http.Request)lh.Decoder,func(context.Context,http.ResponseWriter)lh.Encoder,func(context.Context,http.ResponseWriter,error),func(context.Context,error)lh.StatusCoder)http.Handler
 }{ENTRIES} {
  t.Run(reflect.TypeOf(tc.template).Elem().Name(),func(t *testing.T){
   ctx := context.Background()
   custom, optional := tc.index&8 != 0, tc.index&4 != 0
   header := "Authorization"
   if custom {header="X-Token"}
   for _, token := range []string{"valid", "x", "UPPER", "Bearer valid"} {
    p := tokenPayload(tc.template,&token)
    req := httptest.NewRequest("GET", "/", nil)
    require.NoError(t,tc.httpEncode(req,p))
    md := metadata.MD{}
    msg,err := tc.grpcEncode(ctx,p,&md)
    require.NoError(t,err)
    wire := token
    if !custom {wire="Bearer "+token}
    require.Equal(t,wire,req.Header.Get(header))
    require.Equal(t,[]string{wire},md.Get(header))
    calls := 0
    endpoint := func(context.Context,any)(any,error) {calls++;return "ok",nil}
    rec:=httptest.NewRecorder()
    tc.handler(endpoint,lh.NewMuxer(),lh.RequestDecoder,lh.ResponseEncoder,nil,nil).ServeHTTP(rec,req)
    if token=="valid" {require.Equal(t,200,rec.Code);require.Equal(t,1,calls)} else {require.Equal(t,400,rec.Code);require.Zero(t,calls)}
    decoded,herr := tc.httpDecode(req)
    gdecoded,gerr := tc.grpcDecode(ctx,msg,md)
    if token=="valid" {
     require.NoError(t,herr);require.NoError(t,gerr)
     require.Equal(t,p,decoded);require.Equal(t,p,gdecoded)
    } else {
     require.Error(t,herr);require.Error(t,gerr)
    }
   }
   // Explicit wire probes: missing versus malformed envelopes and casing.
   for _, wire := range []string{"", "valid", "Basic valid", "Bearer ", "bearer valid", "Bearer  valid"} {
    req := httptest.NewRequest("GET", "/", nil)
    md := metadata.MD{}
    if wire!="" {req.Header.Set(header,wire);md.Set(strings.ToLower(header),wire)}
    valid := wire=="" && optional || custom && wire=="valid" || !custom && (wire=="bearer valid" || wire=="Bearer  valid")
    _,herr := tc.httpDecode(req)
    _,gerr := tc.grpcDecode(ctx,nil,md)
    if valid {require.NoError(t,herr);require.NoError(t,gerr)} else {require.Error(t,herr);require.Error(t,gerr)}
   }
  })
 }
}
func TestOpaqueAPIKey(t *testing.T) {
 p:=&svc.KeyPayload{Token:"Bearer opaque key"}
 req:=httptest.NewRequest("GET","/key",nil)
 require.NoError(t,hc.EncodeKeyRequest(nil)(req,p))
 decoded,err:=hs.DecodeKeyRequest(lh.NewMuxer(),nil)(req)
 require.NoError(t,err);require.Equal(t,p,decoded)
 md:=metadata.MD{}
 msg,err:=gc.EncodeKeyRequest(context.Background(),p,&md)
 require.NoError(t,err)
 actual,err:=gs.DecodeKeyRequest(context.Background(),msg,md)
 require.NoError(t,err);require.Equal(t,p,actual)
}
`
