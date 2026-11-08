package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestInformationalServerGenerated(t *testing.T) {
	root := RunHTTPDSL(t, testdata.InformationalServerDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/selection", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "informational_test.go"), []byte(informationalServerHarness), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "continue_test.go"), []byte(informationalContinueHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "-count=1", "./...")
}

const informationalServerHarness = `package selection_test

import (
 "context"
 "errors"
 "io"
 "net/http"
 "net/http/httptest"
 "net/http/httptrace"
 "net/textproto"
 "reflect"
 "strings"
 "testing"

 client "example.com/selection/gen/http/selection/client"
 server "example.com/selection/gen/http/selection/server"
 service "example.com/selection/gen/selection"
 loomhttp "github.com/CaliLuke/loom/http"
)

func TestInformationalWire(t *testing.T) {
 var retained context.Context
 handler := server.NewReadHandler(func(ctx context.Context, _ any) (any,error) {
  retained = ctx
  choice:=service.NewChoiceString("plain")
  if err := server.SendReadHints(ctx, &service.Hints{Status:103, Trace:"early", Labels:[]string{}, Metadata:map[string]string{"b":"two","a":"one"},Choice:&choice}); err != nil { return nil,err }
  if err := server.SendReadMarker(ctx, &service.Marker{}); err != nil { return nil,err }
  if err := server.SendReadProgress(ctx, &service.Progress{Status:104, Trace:"progress"}); err != nil { return nil,err }
  return &service.SelectionResult{Message:"message", Trace:"final"},nil
 },loomhttp.NewMuxer(),loomhttp.RequestDecoder,loomhttp.ResponseEncoder,nil,nil)
 srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request) {
  w.Header().Set("X-Pending","keep")
  handler.ServeHTTP(w,r)
 }))
 defer srv.Close()
 var codes []int
 var snapshots []http.Header
 req,err := http.NewRequest(http.MethodGet,srv.URL+"/read",nil)
 if err != nil { t.Fatal(err) }
 req = req.WithContext(httptrace.WithClientTrace(req.Context(), &httptrace.ClientTrace{Got1xxResponse:func(code int, header textproto.MIMEHeader) error {
  codes = append(codes,code)
  snapshots = append(snapshots,http.Header(header).Clone())
  return nil
 }}))
 response,err := srv.Client().Do(req)
 if err != nil { t.Fatal(err) }
 body,err := io.ReadAll(response.Body)
 if err != nil { t.Fatal(err) }
 if err := response.Body.Close(); err != nil { t.Fatal(err) }
 if !reflect.DeepEqual(codes,[]int{103,102,104}) { t.Errorf("interim statuses=%v",codes) }
 if response.StatusCode != 200 || string(body) != "\"message\"\n" { t.Errorf("final=%d %q",response.StatusCode,body) }
 if response.Header.Get("X-Trace") != "final" || response.Header.Get("X-Pending") != "keep" { t.Errorf("final headers=%v",response.Header) }
 for i,header := range snapshots {
  if header.Get("X-Pending") != "" { t.Errorf("interim %d inherited pending final headers: %v",i,header) }
 }
 if len(snapshots)==3 {
  if snapshots[0].Get("X-Trace")!="early" || len(snapshots[0].Values("X-Labels"))!=1 || snapshots[0].Get("X-Metadata")!="a, one, b, two" || snapshots[0].Get("X-Choice")!="plain" { t.Errorf("hints=%v",snapshots[0]) }
  if len(snapshots[1])!=0 { t.Errorf("marker=%v",snapshots[1]) }
  if snapshots[2].Get("X-Trace")!="progress" || snapshots[2].Get("X-Metadata")!="" { t.Errorf("progress=%v",snapshots[2]) }
 }
 if err:=server.SendReadMarker(retained,&service.Marker{}); !errors.Is(err,loomhttp.ErrInformationalClosed) { t.Errorf("late send=%v",err) }
}

func TestInformationalInvalidValues(t *testing.T) {
 invalidChoice:=service.NewChoiceLabelRequestBody(&service.LabelRequestBody{Text:"x"})
 for _,tc := range []struct{name string; value *service.Hints}{
  {"wrong status",&service.Hints{Status:102,Trace:"valid",Labels:[]string{}}},
  {"zero status",&service.Hints{Trace:"valid",Labels:[]string{}}},
  {"invalid mapped trace",&service.Hints{Status:103,Trace:"abc",Labels:[]string{}}},
  {"missing labels",&service.Hints{Status:103,Trace:"valid"}},
  {"nil carrier",nil},
  {"invalid native union branch",&service.Hints{Status:103,Trace:"valid",Labels:[]string{},Choice:&invalidChoice}},
  {"unselected union",&service.Hints{Status:103,Trace:"valid",Labels:[]string{},Choice:&service.Choice{}}},
 } {
  t.Run(tc.name,func(t *testing.T) {
   rec:=httptest.NewRecorder()
   lifecycle:=loomhttp.NewHandlerLifecycle(rec,httptest.NewRequest(http.MethodGet,"/read",nil),"selection","read")
   defer lifecycle.End()
   err:=server.SendReadHints(lifecycle.Context(),tc.value)
   var violation *loomhttp.StatusViolationError
   if !errors.As(err,&violation) { t.Errorf("error=%T %v",err,err) }
   var named interface{ LoomErrorName() string }
   if errors.As(err,&named) { t.Errorf("invalid metadata exposed designed error %q",named.LoomErrorName()) }
   if len(rec.Header())!=0 || rec.Body.Len()!=0 { t.Errorf("invalid value published headers/body: %v %s",rec.Header(),rec.Body) }
  })
 }
 if err:=server.SendReadMarker(context.Background(),&service.Marker{}); !errors.Is(err,loomhttp.ErrInformationalClosed) { t.Errorf("missing context=%v",err) }
 rec:=httptest.NewRecorder()
 lifecycle:=loomhttp.NewHandlerLifecycle(rec,httptest.NewRequest(http.MethodGet,"/read",nil),"selection","read")
 defer lifecycle.End()
 var violation *loomhttp.StatusViolationError
 if err:=server.SendReadProgress(lifecycle.Context(),&service.Progress{Status:104,Trace:"abc"}); !errors.As(err,&violation) { t.Errorf("native header constraint=%v",err) }
}

func TestInformationalGeneratedLifecycle(t *testing.T) {
 t.Run("final encoder admission",func(t *testing.T) {
  encoded:=false
  handler:=server.NewReadHandler(func(context.Context,any)(any,error) {
   return &service.SelectionResult{Message:"message",Trace:"final"},nil
  },loomhttp.NewMuxer(),loomhttp.RequestDecoder,func(ctx context.Context,w http.ResponseWriter) loomhttp.Encoder {
   encoded=true
   if err:=server.SendReadMarker(ctx,&service.Marker{}); !errors.Is(err,loomhttp.ErrInformationalClosed) { t.Errorf("send during final encoder preparation=%v",err) }
   return loomhttp.ResponseEncoder(ctx,w)
  },nil,nil)
  recorder:=httptest.NewRecorder()
  handler.ServeHTTP(recorder,httptest.NewRequest(http.MethodGet,"/read",nil))
  if !encoded || recorder.Code!=200 { t.Errorf("encoder=%t status=%d",encoded,recorder.Code) }
 })
 t.Run("runtime protocol and budget",func(t *testing.T) {
  request:=httptest.NewRequest(http.MethodGet,"/read",nil)
  request.ProtoMinor=0
  unsupported:=loomhttp.NewHandlerLifecycle(httptest.NewRecorder(),request,"selection","read")
  defer unsupported.End()
  if err:=server.SendReadMarker(unsupported.Context(),&service.Marker{}); !errors.Is(err,loomhttp.ErrInformationalUnsupported) { t.Errorf("HTTP/1.0=%v",err) }
  lifecycle:=loomhttp.NewHandlerLifecycle(httptest.NewRecorder(),httptest.NewRequest(http.MethodGet,"/read",nil),"selection","read")
  defer lifecycle.End()
  for i:=0;i<loomhttp.MaxInformationalResponses;i++ {
   if err:=server.SendReadMarker(lifecycle.Context(),&service.Marker{}); err!=nil { t.Errorf("send %d=%v",i,err) }
  }
  if err:=server.SendReadMarker(lifecycle.Context(),&service.Marker{}); !errors.Is(err,loomhttp.ErrInformationalLimit) { t.Errorf("spent budget=%v",err) }
 })
}

func TestInformationalStatusContracts(t *testing.T) {
 for _,status:=range []int{100,102,103,106} {
  recorder:=httptest.NewRecorder()
  lifecycle:=loomhttp.NewHandlerLifecycle(recorder,httptest.NewRequest(http.MethodGet,"/read",nil),"selection","read")
  err:=server.SendReadProgress(lifecycle.Context(),&service.Progress{Status:status,Trace:"valid"})
  lifecycle.End()
  var violation *loomhttp.StatusViolationError
  if !errors.As(err,&violation) {t.Errorf("status=%d error=%v",status,err)}
  if recorder.Code!=200 || len(recorder.Header())!=0 {t.Errorf("invalid status=%d published code=%d headers=%v",status,recorder.Code,recorder.Header())}
 }
}

func TestIncomingNativeHeaderConstraint(t *testing.T) {
 for _,contract:=range []struct{name string; status int; body string; decode func(*http.Response)(any,error)}{
  {"native",200,"\"message\"",client.DecodeReadResponse(loomhttp.ResponseDecoder,false)},
  {"mapped",204,"",client.DecodeMappedResponse(loomhttp.ResponseDecoder,false)},
 } {
  t.Run(contract.name,func(t *testing.T) {
   for _,tc:=range []struct{value string; valid bool}{{"abc",false},{"abcd",true}} {
    response:=&http.Response{StatusCode:contract.status,Header:http.Header{"Content-Type":{"application/json"},"X-Trace":{tc.value}},Body:io.NopCloser(strings.NewReader(contract.body))}
    _,err:=contract.decode(response)
    if (err==nil)!=tc.valid { t.Errorf("trace=%q error=%v",tc.value,err) }
   }
  })
 }
}
`
