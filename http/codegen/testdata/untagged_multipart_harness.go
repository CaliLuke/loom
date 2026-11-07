package testdata

// UntaggedMultipartHarness checks generated MIME presence and literal part codecs.
const UntaggedMultipartHarness = `package multipartchoices_test

import (
 "bytes"
 "context"
 "io"
 "mime/multipart"
 "net/http"
 "net/http/httptest"
 "testing"

 client "example.com/multipartchoices/gen/http/multipartchoices/client"
 server "example.com/multipartchoices/gen/http/multipartchoices/server"
 svc "example.com/multipartchoices/gen/multipartchoices"
 loomhttp "github.com/CaliLuke/loom/http"
 loom "github.com/CaliLuke/loom/pkg"
 "github.com/stretchr/testify/require"
)

func multipartRequest(t *testing.T, path string, fields [][2]string) *http.Request {
 t.Helper()
 var buffer bytes.Buffer
 writer := multipart.NewWriter(&buffer)
 for _, field := range fields { require.NoError(t,writer.WriteField(field[0],field[1])) }
 require.NoError(t,writer.Close())
 request:=httptest.NewRequest("POST",path,&buffer)
 request.Header.Set("Content-Type",writer.FormDataContentType())
 return request
}

func TestMultipartRootPresence(t *testing.T) {
 for _, tc:= range []struct{name,path string; fields [][2]string; absent bool; status int; kind string}{
  {name:"required empty envelope",path:"/root",status:204,kind:"EmptyUpload"},
  {name:"optional empty envelope",path:"/optional",status:204,kind:"EmptyUpload"},
  {name:"required absence",path:"/root",absent:true,status:400},
  {name:"optional absence",path:"/optional",absent:true,status:204},
  {name:"named bytes",path:"/root",fields:[][2]string{{"name","a+b%2C"},{"file",string([]byte{0,255,'+'})}},status:204,kind:"NamedUpload"},
  {name:"closed unknown",path:"/root",fields:[][2]string{{"extra","value"}},status:400},
  {name:"closed empty name",path:"/root",fields:[][2]string{{"","value"}},status:400},
 } {
  t.Run(tc.name,func(t *testing.T){
   calls:=0
   handler:=func(_ context.Context,raw any)(any,error){
    calls++
    value:=raw.(*svc.UploadChoice)
    if tc.absent {require.Nil(t,value);return nil,nil}
    require.Equal(t,tc.kind,string(value.Kind()))
    if tc.kind=="NamedUpload" { record,ok:=value.AsNamedUpload();require.True(t,ok);require.Equal(t,"a+b%2C",record.Name);require.Equal(t,[]byte{0,255,'+'},record.File) }
    return nil,nil
   }
   mux:=loomhttp.NewMuxer()
   server.Mount(mux,server.New(&svc.Endpoints{Root:handler,Optional:handler},mux,loomhttp.RequestDecoder,loomhttp.ResponseEncoder,nil,nil))
   request:=multipartRequest(t,tc.path,tc.fields)
   if tc.absent {request=httptest.NewRequest("POST",tc.path,nil)}
   response:=httptest.NewRecorder();mux.ServeHTTP(response,request)
   require.Equal(t,tc.status,response.Code,response.Body.String())
   if tc.status==204 {require.Equal(t,1,calls)} else {require.Zero(t,calls);if tc.absent {require.Contains(t,response.Body.String(),"missing_payload")} else {require.Contains(t,response.Body.String(),"invalid request body")}}
  })
 }
}

func TestMultipartFieldsAndClient(t *testing.T) {
 var received *svc.FieldsPayload
 handler:=func(_ context.Context,raw any)(any,error){received=raw.(*svc.FieldsPayload);return nil,nil}
 mux:=loomhttp.NewMuxer();server.Mount(mux,server.New(&svc.Endpoints{Fields:handler},mux,loomhttp.RequestDecoder,loomhttp.ResponseEncoder,nil,nil))
 transport:=client.NewClient("http","example.test",http.DefaultClient,loomhttp.RequestEncoder,loomhttp.ResponseDecoder,false)
 payload:=&svc.FieldsPayload{Names:[]string{"a+b","%2C"},JSON:[]string{"one","two"},File:[]byte{0,255,'+'},Styled:[]string{"a%2Cb","c+d"}}
 request,err:=transport.BuildFieldsRequest(context.Background(),payload);require.NoError(t,err)
 require.NoError(t,client.EncodeFieldsRequest(loomhttp.RequestEncoder)(request,payload))
 encoded,err:=io.ReadAll(request.Body);require.NoError(t,err)
 require.Contains(t,string(encoded),"a%2Cb,c+d")
 request.Body=io.NopCloser(bytes.NewReader(encoded));response:=httptest.NewRecorder();mux.ServeHTTP(response,request)
 require.Equal(t,204,response.Code,response.Body.String());require.Equal(t,payload,received)
 payload.Styled=[]string{"unspellable,comma"}
 request,err=transport.BuildFieldsRequest(context.Background(),payload);require.NoError(t,err)
 require.Error(t,client.EncodeFieldsRequest(loomhttp.RequestEncoder)(request,payload));require.Nil(t,request.Body)
}

func TestMultipartRootClient(t *testing.T) {
 transport:=client.NewClient("http","example.test",http.DefaultClient,loomhttp.RequestEncoder,loomhttp.ResponseDecoder,false)
 value:=&svc.UploadChoice{};value.SetEmptyUpload(&svc.EmptyUpload{})
 request,err:=transport.BuildRootRequest(context.Background(),value);require.NoError(t,err)
 require.NoError(t,client.EncodeRootRequest(loomhttp.RequestEncoder)(request,value))
 form,err:=loomhttp.ReadMultipartForm(request);require.NoError(t,err);require.Empty(t,form.Parts)
 var absent *svc.UploadChoice
 request,err=transport.BuildOptionalRequest(context.Background(),absent);require.NoError(t,err)
 require.NoError(t,client.EncodeOptionalRequest(loomhttp.RequestEncoder)(request,absent));require.Nil(t,request.Body)
}

func TestMultipartNestedAndItems(t *testing.T) {
 for _,tc:=range []struct{path string;parts [][2]string;status int}{
  {"/nested",nil,204}, {"/nested",[][2]string{{"flag","true"}},204},
  {"/nested",[][2]string{{"flag","true"},{"name","both"}},400},
  {"/items",[][2]string{{"value","word"},{"value","%2C+"}},204},
  {"/items",[][2]string{{"value","1"}},400}, {"/items",nil,400},
 } {
  t.Run(tc.path,func(t *testing.T){
   calls:=0
   handler:=func(_ context.Context,raw any)(any,error){calls++;return nil,nil}
   mux:=loomhttp.NewMuxer();server.Mount(mux,server.New(&svc.Endpoints{Nested:handler,Items:handler},mux,loomhttp.RequestDecoder,loomhttp.ResponseEncoder,nil,nil))
   response:=httptest.NewRecorder();mux.ServeHTTP(response,multipartRequest(t,tc.path,tc.parts))
   require.Equal(t,tc.status,response.Code,response.Body.String())
   if tc.status==204 {require.Equal(t,1,calls)} else {require.Zero(t,calls)}
  })
 }
 transport:=client.NewClient("http","example.test",http.DefaultClient,loomhttp.RequestEncoder,loomhttp.ResponseDecoder,false)
 choice:=&svc.ItemChoice{};choice.SetTextItems(svc.TextItems{"word"})
 payload:=&svc.ItemsPayload{Value:*choice}
 request,err:=transport.BuildItemsRequest(context.Background(),payload);require.NoError(t,err)
 require.NoError(t,client.EncodeItemsRequest(loomhttp.RequestEncoder)(request,payload))
 payload.Value.SetNumberItems(svc.NumberItems{1})
 request,err=transport.BuildItemsRequest(context.Background(),payload);require.NoError(t,err)
 require.Error(t,client.EncodeItemsRequest(loomhttp.RequestEncoder)(request,payload));require.Nil(t,request.Body)
}

func TestMultipartMediaRoundTrip(t *testing.T) {
 value:=&svc.UploadChoice{};value.SetEmptyUpload(&svc.EmptyUpload{})
 nested:=&svc.NestedUpload{};nested.SetUploadChoice(value)
 envelope:=&svc.MediaUpload{};envelope.SetNestedUpload(nested)
 transport:=client.NewClient("http","example.test",http.DefaultClient,loomhttp.RequestEncoder,loomhttp.ResponseDecoder,false)
 request,err:=transport.BuildMediaRequest(context.Background(),envelope);require.NoError(t,err)
 require.NoError(t,client.EncodeMediaRequest(loomhttp.RequestEncoder)(request,envelope))
 calls:=0
 handler:=func(_ context.Context,raw any)(any,error){calls++;require.Equal(t,envelope,raw);return raw,nil}
 mux:=loomhttp.NewMuxer();server.Mount(mux,server.New(&svc.Endpoints{Media:handler},mux,loomhttp.RequestDecoder,loomhttp.ResponseEncoder,nil,nil))
 response:=httptest.NewRecorder();mux.ServeHTTP(response,request)
 require.Equal(t,200,response.Code,response.Body.String());require.Equal(t,1,calls)
 decoded,err:=client.DecodeMediaResponse(loomhttp.ResponseDecoder,false)(response.Result());require.NoError(t,err)
 require.Equal(t,envelope,decoded)
}

func TestMultipartMetadataPrecedence(t *testing.T) {
 for _,explicit:=range []bool{false,true}{
  var buffer bytes.Buffer;writer:=multipart.NewWriter(&buffer)
  file,err:=writer.CreateFormFile("file","uploaded.txt");require.NoError(t,err)
  _,err=file.Write([]byte("data"));require.NoError(t,err)
  if explicit {require.NoError(t,writer.WriteField("filename",""))}
  require.NoError(t,writer.Close())
  calls:=0
  handler:=func(_ context.Context,raw any)(any,error){
   calls++;payload:=raw.(*svc.MetadataPayload)
   if explicit {require.Equal(t,"",payload.Filename)} else {require.Equal(t,"uploaded.txt",payload.Filename)}
   require.NotNil(t,payload.ContentType);require.Equal(t,"application/octet-stream",*payload.ContentType)
   return nil,nil
  }
  mux:=loomhttp.NewMuxer();server.Mount(mux,server.New(&svc.Endpoints{Metadata:handler},mux,loomhttp.RequestDecoder,loomhttp.ResponseEncoder,nil,nil))
  request:=httptest.NewRequest("POST","/metadata",&buffer);request.Header.Set("Content-Type",writer.FormDataContentType())
  response:=httptest.NewRecorder();mux.ServeHTTP(response,request)
  require.Equal(t,204,response.Code,response.Body.String());require.Equal(t,1,calls)
 }
}

func TestMultipartProjectionDomains(t *testing.T) {
 for _,tc:=range []struct{path string;parts [][2]string;status int}{
  {"Physical",[][2]string{{"x","ok"},{"filter","extra"}},204},
  {"Closed",[][2]string{{"x","ok"}},204},
  {"Closed",[][2]string{{"x","ok"},{"filter","extra"}},400},
  {"Open",[][2]string{{"x","ok"},{"filter","extra"}},204},
  {"Numeric",[][2]string{{"count","42"}},204},
  {"Numeric",[][2]string{{"count","2147483648"}},400},
  {"Numeric",[][2]string{{"count","42"},{"label","both"}},400},
  {"Composite",[][2]string{{"value","x,ok"},{"jsonValue","null"},{"any","null"},{"tree",` + "`" + "{\"name\":\"root\",\"children\":[{\"name\":\"child\"}]}" + "`" + `},{"records",` + "`" + "{\"jsonX\":\"one\"}" + "`" + `},{"files",string([]byte{0,255})},{"files",""}},204},
  {"Composite",[][2]string{{"value","x,ok"},{"jsonValue",` + "`" + "{\"jsonX\":\"yes\"}" + "`" + `}},204},
  {"Composite",[][2]string{{"value","jsonX,wrong"},{"jsonValue","null"}},400},
  {"Composite",[][2]string{{"value","true"},{"jsonValue","null"},{"records",` + "`" + "[{\"jsonX\":\"wrong envelope\"}]" + "`" + `}},400},
  {"Selected",[][2]string{{"x","ok"}},204},
  {"WholeItems",[][2]string{{"value","[\"word\"]"}},204},
  {"WholeItems",[][2]string{{"value","[1]"}},204},
  {"WholeItems",[][2]string{{"value","word"}},400},
 } {
  t.Run(tc.path,func(t *testing.T){
   calls:=0
   handler:=func(_ context.Context,raw any)(any,error){
    calls++
    if payload,ok:=raw.(*svc.CompositePayload);ok && len(payload.Files)>0 {
     require.Equal(t,[][]byte{{0,255},{}},payload.Files)
     require.True(t,payload.JSONValue.IsNull())
     require.Equal(t,"null",string(payload.Any))
     require.Equal(t,"root",payload.Tree.Name)
     require.Equal(t,"child",payload.Tree.Children[0].Name)
     require.Equal(t,"one",payload.Records[0].X)
    }
    return nil,nil
   }
   endpoints:=&svc.Endpoints{Physical:handler,Closed:handler,Open:handler,Numeric:handler,Composite:handler,Selected:handler,WholeItems:handler}
   mux:=loomhttp.NewMuxer();server.Mount(mux,server.New(endpoints,mux,loomhttp.RequestDecoder,loomhttp.ResponseEncoder,nil,nil))
   response:=httptest.NewRecorder();mux.ServeHTTP(response,multipartRequest(t,"/"+tc.path,tc.parts))
   require.Equal(t,tc.status,response.Code,response.Body.String())
   if tc.status==204 {require.Equal(t,1,calls)} else {require.Zero(t,calls);require.Contains(t,response.Body.String(),"invalid request body")}
  })
 }
}

func TestMultipartMalformedAndSelectedAbsence(t *testing.T) {
 for _,path:=range []string{"/root","/optional"} {
  for _,wire:=range []string{"garbage","--bad","--test\r\nContent-Disposition: form-data; name=\"name\"\r\n\r\ntruncated"} {
   calls:=0
   handler:=func(_ context.Context,_ any)(any,error){calls++;return nil,nil}
   mux:=loomhttp.NewMuxer();server.Mount(mux,server.New(&svc.Endpoints{Root:handler,Optional:handler},mux,loomhttp.RequestDecoder,loomhttp.ResponseEncoder,nil,nil))
   request:=httptest.NewRequest("POST",path,bytes.NewBufferString(wire));request.Header.Set("Content-Type","multipart/form-data; boundary=test")
   response:=httptest.NewRecorder();mux.ServeHTTP(response,request)
   require.Equal(t,400,response.Code,response.Body.String());require.Zero(t,calls)
   require.Contains(t,response.Body.String(),"decode_payload");require.Contains(t,response.Body.String(),"invalid request body")
  }
 }
 calls:=0
 handler:=func(_ context.Context,raw any)(any,error){calls++;require.Nil(t,raw.(*svc.SelectedPayload).Value);return nil,nil}
 mux:=loomhttp.NewMuxer();server.Mount(mux,server.New(&svc.Endpoints{Selected:handler},mux,loomhttp.RequestDecoder,loomhttp.ResponseEncoder,nil,nil))
 request:=httptest.NewRequest("POST","/Selected",nil);response:=httptest.NewRecorder();mux.ServeHTTP(response,request)
 require.Equal(t,204,response.Code,response.Body.String());require.Equal(t,1,calls)
 payload:=&svc.SelectedPayload{}
 transport:=client.NewClient("http","example.test",http.DefaultClient,loomhttp.RequestEncoder,loomhttp.ResponseDecoder,false)
 request,err:=transport.BuildSelectedRequest(context.Background(),payload);require.NoError(t,err)
 require.NoError(t,client.EncodeSelectedRequest(loomhttp.RequestEncoder)(request,payload));require.Nil(t,request.Body)
}

func TestMultipartDerivedRootAgreement(t *testing.T) {
 for _,duplicates:=range []bool{false,true} {
  var buffer bytes.Buffer;writer:=multipart.NewWriter(&buffer)
  count:=1;if duplicates {count=2}
  for range count {part,err:=writer.CreateFormFile("file","named.txt");require.NoError(t,err);_,err=part.Write([]byte("body"));require.NoError(t,err)}
  require.NoError(t,writer.Close())
  calls:=0
  handler:=func(_ context.Context,raw any)(any,error){calls++;value,ok:=raw.(*svc.MultipartDerived).AsMultipartMetadata();require.True(t,ok);require.Equal(t,"named.txt",value.Filename);return nil,nil}
  mux:=loomhttp.NewMuxer();server.Mount(mux,server.New(&svc.Endpoints{Derived:handler},mux,loomhttp.RequestDecoder,loomhttp.ResponseEncoder,nil,nil))
  request:=httptest.NewRequest("POST","/Derived",&buffer);request.Header.Set("Content-Type",writer.FormDataContentType());response:=httptest.NewRecorder();mux.ServeHTTP(response,request)
  if duplicates {require.Equal(t,400,response.Code,response.Body.String());require.Zero(t,calls)} else {require.Equal(t,204,response.Code,response.Body.String());require.Equal(t,1,calls)}
 }
}

func TestClientRootPresenceBeforeConversion(t *testing.T) {
 var object *svc.MultipartOptionalLabel
 var union *svc.MultipartPhysical
 var multipartUnion *svc.UploadChoice
 var nullable loom.Nullable[svc.MultipartOptionalLabel]
 var rawJSON loom.JSONValue
 for _,tc:=range []struct{name string;value any;encode func(*http.Request,any)error;optional bool}{
  {"json object",object,client.EncodeJSONOptionalObjectRequest(loomhttp.RequestEncoder),true},
  {"form object",object,client.EncodeFormOptionalObjectRequest(loomhttp.RequestEncoder),true},
  {"json union",union,client.EncodeJSONOptionalUnionRequest(loomhttp.RequestEncoder),true},
  {"unselected json union",&svc.MultipartPhysical{},client.EncodeJSONOptionalUnionRequest(loomhttp.RequestEncoder),true},
  {"unselected form union",&svc.MultipartPhysical{},client.EncodeFormOptionalUnionRequest(loomhttp.RequestEncoder),true},
  {"unselected multipart union",&svc.UploadChoice{},client.EncodeOptionalRequest(loomhttp.RequestEncoder),true},
  {"required unselected json union",&svc.MultipartPhysical{},client.EncodeJSONRequiredUnionRequest(loomhttp.RequestEncoder),false},
  {"required unselected form union",&svc.MultipartPhysical{},client.EncodeFormRequiredUnionRequest(loomhttp.RequestEncoder),false},
  {"required unselected multipart union",&svc.UploadChoice{},client.EncodeRootRequest(loomhttp.RequestEncoder),false},
  {"form union",union,client.EncodeFormOptionalUnionRequest(loomhttp.RequestEncoder),true},
  {"multipart union",multipartUnion,client.EncodeOptionalRequest(loomhttp.RequestEncoder),true},
  {"required json object",object,client.EncodeJSONRequiredObjectRequest(loomhttp.RequestEncoder),false},
  {"required json union",union,client.EncodeJSONRequiredUnionRequest(loomhttp.RequestEncoder),false},
  {"required form union",union,client.EncodeFormRequiredUnionRequest(loomhttp.RequestEncoder),false},
  {"required multipart union",multipartUnion,client.EncodeRootRequest(loomhttp.RequestEncoder),false},
  {"raw JSON absent",rawJSON,client.EncodeJSONAnyRequest(loomhttp.RequestEncoder),false},
  {"nullable absent",nullable,client.EncodeJSONOptionalNullableRequest(loomhttp.RequestEncoder),true},
  {"required nullable absent",nullable,client.EncodeJSONRequiredNullableRequest(loomhttp.RequestEncoder),false},
 } {
  t.Run(tc.name,func(t *testing.T){
   request,err:=http.NewRequest("POST","http://example.test",nil);require.NoError(t,err)
   err=tc.encode(request,tc.value)
   if tc.optional {require.NoError(t,err)} else {var serviceErr *loom.ServiceError;require.ErrorAs(t,err,&serviceErr);require.Equal(t,"missing_payload",serviceErr.Name)}
   require.Nil(t,request.Body)
  })
 }
 nullable.SetNull()
 request,err:=http.NewRequest("POST","http://example.test",nil);require.NoError(t,err)
 require.NoError(t,client.EncodeJSONOptionalNullableRequest(loomhttp.RequestEncoder)(request,nullable))
 encoded,err:=io.ReadAll(request.Body);require.NoError(t,err);require.Equal(t,"null",string(bytes.TrimSpace(encoded)))
 request,err=http.NewRequest("POST","http://example.test",nil);require.NoError(t,err)
 require.NoError(t,client.EncodeJSONAnyRequest(loomhttp.RequestEncoder)(request,loom.JSONValue("null")))
 encoded,err=io.ReadAll(request.Body);require.NoError(t,err);require.Equal(t,"null",string(bytes.TrimSpace(encoded)))
}
`
