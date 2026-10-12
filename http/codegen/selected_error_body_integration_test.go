package codegen

const selectedProblemWireHarness = `package problems_test
import (
 "context"
 "encoding/json/v2"
 "errors"
 "io"
 "net/http"
 "net/http/httptest"
 "net/url"
 "strings"
 "testing"
 "github.com/stretchr/testify/require"
 service "example.com/problems/gen/problems"
 client "example.com/problems/gen/http/problems/client"
 server "example.com/problems/gen/http/problems/server"
 loomhttp "github.com/CaliLuke/loom/http"
 loom "github.com/CaliLuke/loom/pkg"
)
func TestSelectedWire(t *testing.T) {
 selected, header, cookie, empty, shaped, override := %q, %t, %t, %t, %t, %t
 for _, hint := range []string{"", "retry later"} {
  for _, safe := range []string{"", "public detail"} {
   for _, formatted := range []bool{false, true} {
    original := loom.NewServiceError(errors.New("private credential"), "oops", false, false, true)
    original.ID = "instance-id"
    original.Message = "public message"
    original = loom.WithErrorRemedy(original, &loom.ErrorRemedy{SafeMessage:safe, RetryHint:hint})
    problemType, problemTitle := "", ""
    if override { problemType="https://example.com/problem"; problemTitle="Explicit title" }
    expected := loomhttp.NewProblemResponse(t.Context(), original, 400, problemType, problemTitle)
    var formatter func(context.Context, error) loomhttp.StatusCoder
    if formatted {
     formatter = func(_ context.Context, err error) loomhttp.StatusCoder {
      require.Same(t, original, err)
      return &loomhttp.ProblemResponse{Status:418, Detail:"formatter document"}
     }
    }
    mux := loomhttp.NewMuxer()
    generated := server.New(&service.Endpoints{Selected:func(context.Context,any)(any,error){return nil,original}}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, func(_ context.Context,_ http.ResponseWriter,err error){t.Errorf("encoding failure: %%v",err)}, formatter)
    server.Mount(mux,generated)
    host := httptest.NewServer(mux)
    response, err := host.Client().Get(host.URL+"/selected")
    require.NoError(t,err)
    data,err := io.ReadAll(response.Body)
    require.NoError(t,err)
    require.NoError(t,response.Body.Close())
    require.Equal(t,400,response.StatusCode)
    require.Equal(t,"oops",response.Header.Get("loom-error"))
    require.NotContains(t,string(data),"private credential")
    if header { require.Equal(t,"oops",response.Header.Get("X-Code")) }
    if cookie { require.Contains(t,response.Header.Get("Set-Cookie"),"error_code=oops") }
    var fields map[string]any
    raw,err := json.Marshal(expected)
    require.NoError(t,err)
    require.NoError(t,json.Unmarshal(raw,&fields))
    var wanted any = fields
    if selected!="" { wanted=fields[selected] }
    if shaped { wanted=map[string]any{"detail":expected.Detail} }
    if header || cookie { delete(fields,"code") }
    if formatted { wanted=&loomhttp.ProblemResponse{Status:418, Detail:"formatter document"} }
    if empty { require.Empty(t,data) } else {
     contentType := loomhttp.ProblemJSONContentType
     if selected!="" || shaped { contentType="application/json" }
     require.Equal(t,contentType,response.Header.Get("Content-Type"))
     wantJSON,err:=json.Marshal(wanted)
     require.NoError(t,err)
     require.JSONEq(t,string(wantJSON),strings.TrimSpace(string(data)))
    }
    if !formatted {
     address,err:=url.Parse(host.URL)
     require.NoError(t,err)
     cli:=client.NewClient(address.Scheme,address.Host,host.Client(),loomhttp.RequestEncoder,loomhttp.ResponseDecoder,false)
     result,err:=cli.Selected()(t.Context(),nil)
     require.Nil(t,result)
     var actual *loom.ServiceError
     require.ErrorAs(t,err,&actual)
     wantCode,wantDetail,wantID,wantHint:="","","",""
     full:= !shaped
     if full { wantCode="oops" }
     if full || selected=="detail" || shaped { wantDetail=expected.Detail }
     if full || selected=="instance" { wantID="instance-id" }
     if full || selected=="retry_hint" { wantHint=hint }
     require.Equal(t,wantCode,actual.Name)
     require.Equal(t,wantDetail,actual.Message)
     require.Equal(t,wantID,actual.ID)
     require.Equal(t,wantHint,loom.ErrorRetryHint(actual))
    }
    host.Close()
    if hint=="" && safe=="" && !formatted && !empty {
     invalidBodies:=[]string{"{", "[]"}
     if selected=="" { invalidBodies=append(invalidBodies,"{}") }
     for _, invalid:=range invalidBodies {
      badHost:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,_ *http.Request){
       w.Header().Set("Content-Type","application/json")
       w.Header().Set("loom-error","oops")
       w.WriteHeader(400)
       if _,err:=io.WriteString(w,invalid); err!=nil { t.Errorf("write malformed response: %%v",err) }
      }))
      address,err:=url.Parse(badHost.URL)
      require.NoError(t,err)
      cli:=client.NewClient(address.Scheme,address.Host,badHost.Client(),loomhttp.RequestEncoder,loomhttp.ResponseDecoder,false)
      result,err:=cli.Selected()(t.Context(),nil)
      require.Nil(t,result)
      require.Error(t,err)
      var clientError *loomhttp.ClientError
      require.ErrorAs(t,err,&clientError)
      expectedKind:="decoding_error"
      if invalid=="{}" { expectedKind="validation_error" }
      require.Equal(t,expectedKind,clientError.Name)
      badHost.Close()
     }
    }
   }
  }
 }
}
`
