package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestFormMapFieldsUseCapturedCodec(t *testing.T) {
	root := RunHTTPDSL(t, testdata.FormMapFieldsDSL)
	services := CreateHTTPServices(root)
	functions := renderTransportFuncs(t, append(ServerFiles("", services), ClientFiles("", services)...))
	for _, name := range []string{"Records", "Nested"} {
		require.Contains(t, functions["Encode"+name+"Request"], "loomhttp.EncodeJSONField(parameterForm, formKey, formValue, loomhttp.FieldItems)")
		require.Contains(t, functions["Decode"+name+"Request"], "loomhttp.DecodeJSONField(parameterForm, formKey, &formValue, false, loomhttp.FieldItems)")
	}
	require.Contains(t, functions["EncodeChoiceRequest"], "matchedPath")
	require.NotContains(t, functions["EncodeRecordsRequest"], "SetFormRequest(req")
}

func TestFormMapFieldsGeneratedRoundTrip(t *testing.T) {
	root := RunHTTPDSL(t, testdata.FormMapFieldsDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/mapfields", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "map_fields_test.go"), []byte(formMapFieldsHarness), 0600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

const formMapFieldsHarness = `package mapfields
import (
 "encoding/json/v2"
 "context"
 "io"
 "net/http"
 "net/http/httptest"
 "net/url"
 "strings"
 "testing"
 "github.com/stretchr/testify/require"
 client "example.com/mapfields/gen/http/mapfields/client"
 server "example.com/mapfields/gen/http/mapfields/server"
 svc "example.com/mapfields/gen/mapfields"
 loomhttp "github.com/CaliLuke/loom/http"
)
func adapt[T any](decode func(*http.Request)(T,error)) func(*http.Request)(any,error) {
 return func(r *http.Request)(any,error) { return decode(r) }
}
func TestMapFields(t *testing.T) {
 mux:=loomhttp.NewMuxer()
 for _,tc:=range []struct { name,wire,json string; decode func(*http.Request)(any,error); encode func(*http.Request,any)error }{
 {"scalar","=ok&a%5Bb%5D=text",` + "`" + `{"":"ok","a[b]":"text"}` + "`" + `,adapt(server.DecodeScalarRequest(mux,loomhttp.RequestDecoder)),client.EncodeScalarRequest(loomhttp.RequestEncoder)},
 {"strings","=a&=b",` + "`" + `{"": ["a","b"]}` + "`" + `,adapt(server.DecodeStringsRequest(mux,loomhttp.RequestDecoder)),client.EncodeStringsRequest(loomhttp.RequestEncoder)},
 {"object","k=%7B%22x%22%3A%22ok%22%7D",` + "`" + `{"k":{"x":"ok"}}` + "`" + `,adapt(server.DecodeObjectRequest(mux,loomhttp.RequestDecoder)),client.EncodeObjectRequest(loomhttp.RequestEncoder)},
 {"records","=%7B%22x%22%3A%22a%22%7D&=%7B%22x%22%3A%22b%22%7D",` + "`" + `{"": [{"x":"a"},{"x":"b"}]}` + "`" + `,adapt(server.DecodeRecordsRequest(mux,loomhttp.RequestDecoder)),client.EncodeRecordsRequest(loomhttp.RequestEncoder)},
 {"nested","k=%5B1%2C2%5D&k=%5B3%5D",` + "`" + `{"k":[[1,2],[3]]}` + "`" + `,adapt(server.DecodeNestedRequest(mux,loomhttp.RequestDecoder)),client.EncodeNestedRequest(loomhttp.RequestEncoder)},
 {"choice","k=%7B%22x%22%3A%22ok%22%7D",` + "`" + `{"k":[{"x":"ok"}]}` + "`" + `,adapt(server.DecodeChoiceRequest(mux,loomhttp.RequestDecoder)),client.EncodeChoiceRequest(loomhttp.RequestEncoder)},
 {"root","k=%7B%22x%22%3A%22ok%22%7D",` + "`" + `{"k":[{"x":"ok"}]}` + "`" + `,adapt(server.DecodeRootRequest(mux,loomhttp.RequestDecoder)),client.EncodeRootRequest(loomhttp.RequestEncoder)},
 {"word","key=first",` + "`" + `{"key":"first"}` + "`" + `,adapt(server.DecodeWordRequest(mux,loomhttp.RequestDecoder)),client.EncodeWordRequest(loomhttp.RequestEncoder)},
 {"empty","",` + "`" + `{}` + "`" + `,adapt(server.DecodeScalarRequest(mux,loomhttp.RequestDecoder)),client.EncodeScalarRequest(loomhttp.RequestEncoder)},
 } {
 t.Run(tc.name,func(t *testing.T){
  req:=httptest.NewRequest("POST","/",strings.NewReader(tc.wire));req.Header.Set("Content-Type","application/x-www-form-urlencoded")
  payload,err:=tc.decode(req);require.NoError(t,err)
  data,err:=json.Marshal(payload);require.NoError(t,err);require.JSONEq(t,tc.json,string(data))
  outgoing:=httptest.NewRequest("POST","/",nil);require.NoError(t,tc.encode(outgoing,payload));require.NotNil(t,outgoing.GetBody)
  first,err:=io.ReadAll(outgoing.Body);require.NoError(t,err)
  replay,err:=outgoing.GetBody();require.NoError(t,err);again,err:=io.ReadAll(replay);require.NoError(t,err);require.NoError(t,replay.Close());require.Equal(t,first,again)
  for range 3 { repeated:=httptest.NewRequest("POST","/",nil);require.NoError(t,tc.encode(repeated,payload));data,err:=io.ReadAll(repeated.Body);require.NoError(t,err);require.Equal(t,first,data) }
  expected,err:=url.ParseQuery(tc.wire);require.NoError(t,err);actual,err:=url.ParseQuery(string(first));require.NoError(t,err);require.Equal(t,expected,actual)
 })
 }
}
func TestOptionalMapPresence(t *testing.T) {
 decode:=server.DecodeOptionalRequest(loomhttp.NewMuxer(),loomhttp.RequestDecoder)
 absent:=httptest.NewRequest("POST","/",nil);absent.Header.Set("Content-Type","application/x-www-form-urlencoded")
 payload,err:=decode(absent);require.NoError(t,err);require.Nil(t,payload.Value)
 supplied:=httptest.NewRequest("POST","/",strings.NewReader("=ok"));supplied.Header.Set("Content-Type","application/x-www-form-urlencoded")
 payload,err=decode(supplied);require.NoError(t,err);require.Equal(t,"ok",payload.Value[""])
}
func TestMapMedia(t *testing.T) {
 mux:=loomhttp.NewMuxer()
 calls:=0
 endpoint:=func(_ context.Context,payload any)(any,error){calls++;value:=payload.(*svc.MapMedia);mapping,ok:=value.AsForm();require.True(t,ok);require.Len(t,mapping[""],1);return payload,nil}
 server.Mount(mux,server.New(&svc.Endpoints{Media:endpoint},mux,loomhttp.RequestDecoder,loomhttp.ResponseEncoder,nil,nil))
 host:=httptest.NewServer(mux);defer host.Close()
 transport:=client.NewClient("http",strings.TrimPrefix(host.URL,"http://"),host.Client(),loomhttp.RequestEncoder,loomhttp.ResponseDecoder,false)
 var payload svc.MapMedia;payload.SetForm(svc.Maprecords{"":{{X:"ok"}}})
 value,err:=transport.Media()(context.Background(),&payload);require.NoError(t,err);require.Equal(t,1,calls)
 mapping,ok:=value.(*svc.MapMedia).AsForm();require.True(t,ok);require.Equal(t,"ok",mapping[""][0].X)
}
func TestMapFieldsRejectInvalid(t *testing.T) {
 mux:=loomhttp.NewMuxer()
 for _,tc:=range []struct {wire string; decode func(*http.Request)(any,error)}{
 {"x=%XX",adapt(server.DecodeScalarRequest(mux,loomhttp.RequestDecoder))},
 {"x=%7B",adapt(server.DecodeRecordsRequest(mux,loomhttp.RequestDecoder))},
 {"x=null",adapt(server.DecodeRecordsRequest(mux,loomhttp.RequestDecoder))},
 } {
 req:=httptest.NewRequest("POST","/",strings.NewReader(tc.wire));req.Header.Set("Content-Type","application/x-www-form-urlencoded")
 _,err:=tc.decode(req);require.Error(t,err,tc.wire)
 }
 req:=httptest.NewRequest("POST","/",nil)
 require.Error(t,client.EncodeStringsRequest(loomhttp.RequestEncoder)(req,svc.Mapstrings{"key":{}}))
}
`
