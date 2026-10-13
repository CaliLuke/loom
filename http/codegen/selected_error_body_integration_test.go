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
 mappedProperty := %q
 for _, hint := range []string{"", "retry later"} {
  for _, safe := range []string{"", "public detail"} {
   for _, formatted := range []bool{false, true} {
    original := loom.NewServiceError(errors.New("private credential"), "oops", true, true, true)
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
    if mappedProperty!="" { delete(fields,"temporary") }
    if mappedProperty=="header" { require.Equal(t,"true",response.Header.Get("X-Temporary")) }
    if mappedProperty=="cookie" { require.Contains(t,response.Header.Get("Set-Cookie"),"transient=true") }
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
     fullProperties := !shaped
     require.Equal(t,fullProperties || selected=="timeout",actual.Timeout)
     require.Equal(t,fullProperties || selected=="temporary" || mappedProperty!="",actual.Temporary)
     require.Equal(t,fullProperties || selected=="fault",actual.Fault)
    }
    host.Close()
    if hint=="" && safe=="" && !formatted && !empty {
     invalidBodies:=[]struct{body, kind string}{{"{", "decoding_error"}, {"[]", "decoding_error"}}
     if selected=="" {
      invalidBodies=append(invalidBodies,struct{body, kind string}{"{}","validation_error"})
     }
     if selected=="" && !shaped {
      for _, property:=range []string{"timeout","temporary","fault"} {
       if property=="temporary" && mappedProperty!="" {
        continue
       }
       for _, mutation:=range []string{"missing","null","string"} {
        malformed:=make(map[string]any,len(fields))
        for key,value:=range fields {
         malformed[key]=value
        }
        kind:="validation_error"
        switch mutation {
        case "missing":
         delete(malformed,property)
        case "null":
         malformed[property]=nil
        case "string":
         malformed[property]="true"
         kind="decoding_error"
        }
        encoded,err:=json.Marshal(malformed)
        require.NoError(t,err)
        invalidBodies=append(invalidBodies,struct{body, kind string}{string(encoded),kind})
       }
      }
     }
     for _, invalid:=range invalidBodies {
      badHost:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,_ *http.Request){
       w.Header().Set("Content-Type","application/json")
       w.Header().Set("loom-error","oops")
       w.WriteHeader(400)
       if _,err:=io.WriteString(w,invalid.body); err!=nil { t.Errorf("write malformed response: %%v",err) }
      }))
      address,err:=url.Parse(badHost.URL)
      require.NoError(t,err)
      cli:=client.NewClient(address.Scheme,address.Host,badHost.Client(),loomhttp.RequestEncoder,loomhttp.ResponseDecoder,false)
      result,err:=cli.Selected()(t.Context(),nil)
      require.Nil(t,result)
      require.Error(t,err)
      var clientError *loomhttp.ClientError
      require.ErrorAs(t,err,&clientError)
      require.Equal(t,invalid.kind,clientError.Name,"body: %%s",invalid.body)
      badHost.Close()
     }
    }
   }
  }
 }
}
`
