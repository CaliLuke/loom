package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func untaggedFormContentMethods() func() {
	word, record, choice, broad, narrow, nested := untaggedFormContentChoices()
	records, closedItems, nulls, arrays := untaggedFormContentCollections(record, word)
	form := Type("FormContentForm", func() {
		Attribute("value", choice)
		Required("value")
	})
	media := Type("FormContentMedia", OneOf(func() {
		Attribute("Form", form)
		Attribute("Text", String)
	}))
	return func() {
		for _, item := range []struct {
			name           string
			typ            any
			required, json bool
		}{
			{"Content", choice, true, false}, {"ContentOptional", choice, false, false},
			{"ContentOnly", choice, true, false},
			{"ContentBroad", broad, true, false}, {"ContentJSON", broad, true, true},
			{"ContentWidth", narrow, true, false}, {"ContentNested", nested, true, false},
			{"ContentRecords", records, true, false}, {"ContentNulls", nulls, true, true},
			{"ContentArrays", arrays, true, true}, {"ContentClosedItems", closedItems, true, false},
		} {
			Method(item.name, func() {
				Payload(func() {
					Attribute("value", item.typ)
					if item.name != "ContentOnly" {
						Attribute("owner", String)
					}
					if item.required {
						Required("value")
					}
				})
				HTTP(func() {
					POST("/" + item.name)
					FormRequest()
					Body(func() {
						Attribute("value")
						if item.name != "ContentOnly" {
							Attribute("owner")
						}
						if item.json {
							Meta("openapi:encoding:value:contentType", "application/json")
						}
					})
					Response(StatusNoContent)
				})
			})
		}
		untaggedFormContentMediaMethod(media)
	}
}

func untaggedFormContentChoices() (expr.UserType, expr.UserType, expr.UserType, expr.UserType, expr.UserType, expr.UserType) {
	word := Type("FormContentWord", String, func() {
		Pattern("^text:")
	})
	record := Type("FormContentRecord", func() {
		Attribute("number", Int64, func() {
			Meta("struct:tag:json", "n")
		})
		Required("number")
	})
	choice := Type("FormContentChoice", OneOf(func() {
		Untagged()
		Attribute("Words", ArrayOf(word))
		Attribute("Records", ArrayOf(record))
	}), func() {
		Meta("http:encoding:sourceType", "array")
	})
	broad := Type("FormContentBroad", OneOf(func() {
		Untagged()
		Attribute("Words", ArrayOf(String))
		Attribute("Records", ArrayOf(record))
	}), func() {
		Meta("http:encoding:sourceType", "array")
	})
	narrow := Type("FormContentWidth", OneOf(func() {
		Untagged()
		Attribute("Ints", ArrayOf(Int32))
		Attribute("Floats", ArrayOf(Float64))
	}), func() {
		Meta("http:encoding:sourceType", "array")
	})
	nested := Type("FormContentNested", OneOf(func() {
		Untagged()
		Attribute("Inner", choice)
		Attribute("Flags", ArrayOf(Boolean))
		Meta("openapi:unevaluatedProperties", "false")
	}), func() {
		Meta("http:encoding:sourceType", "array")
	})
	return word, record, choice, broad, narrow, nested
}

func untaggedFormContentCollections(record, word expr.UserType) (expr.UserType, expr.UserType, expr.UserType, expr.UserType) {
	otherRecord := Type("FormContentOtherRecord", func() {
		Attribute("label", String, func() {
			Meta("struct:tag:json", "m")
		})
		Required("label")
	})
	records := Type("FormContentRecords", OneOf(func() {
		Untagged()
		Attribute("Number", ArrayOf(record))
		Attribute("Label", ArrayOf(otherRecord))
	}), func() {
		Meta("http:encoding:sourceType", "array")
	})
	closedItem := Type("FormContentClosedItem", OneOf(func() {
		Untagged()
		Attribute("Number", record)
		Attribute("Label", otherRecord)
		Meta("openapi:unevaluatedProperties", "false")
	}), func() {
		Meta("http:encoding:sourceType", "object")
	})
	closedItems := Type("FormContentClosedItems", OneOf(func() {
		Untagged()
		Attribute("Objects", ArrayOf(closedItem))
		Attribute("Words", ArrayOf(word))
	}), func() {
		Meta("http:encoding:sourceType", "array")
	})
	nulls := Type("FormContentNulls", OneOf(func() {
		Untagged()
		Attribute("Anything", ArrayOf(Any))
		Attribute("Flags", ArrayOf(Boolean))
	}), func() {
		Meta("http:encoding:sourceType", "array")
	})
	arrays := Type("FormContentArrays", OneOf(func() {
		Untagged()
		Attribute("Matrices", ArrayOf(ArrayOf(Int)))
		Attribute("Words", ArrayOf(String))
	}), func() {
		Meta("http:encoding:sourceType", "array")
	})
	return records, closedItems, nulls, arrays
}

func untaggedFormContentMediaMethod(media expr.UserType) {
	Method("ContentMedia", func() {
		Payload(media)
		Result(media)
		HTTP(func() {
			POST("/ContentMedia")
			MediaType("application/x-www-form-urlencoded", "Form")
			MediaType("text/plain", "Text")
			Response(StatusOK, func() {
				MediaType("application/x-www-form-urlencoded", "Form")
				MediaType("text/plain", "Text")
			})
		})
	})
}

const untaggedFormContentHarness = `
func TestFormContentIncoming(t *testing.T) {
 for _,tc:=range []struct{name,wire,kind string;status int}{
  {"Content","value=text:a&value=text:b&owner=kept","Words",204},
  {"Content","value=%7B%22n%22:7%7D&value=%7B%22n%22:8%7D","Records",204},
  {"Content","value=%7B%22n%22:7%7D&value=%7B","",400},
  {"Content","value=text:a&value=%7B%22n%22:7%7D","",400},
  {"Content","owner=kept","",400},
  {"Content","value=invalid","",400},
  {"ContentOptional","owner=kept","",204},
  {"ContentOnly","value=%7B%22n%22:7%7D","Records",204},
  {"ContentBroad","value=","Words",204},
  {"ContentBroad","value=%22%22","Words",204},
  {"ContentBroad","value=%7B","Words",204},
  {"ContentBroad","value=%7B%22n%22:7%7D","",400},
  {"ContentJSON","value=%22%22","Words",204},
  {"ContentJSON","value=","",400},
  {"ContentJSON","value=%7B%22n%22:7%7D","Records",204},
  {"ContentWidth","value=2147483648","",400},
  {"ContentWidth","value=1.5","Floats",204},
  {"ContentNested","value=false","Flags",204},
  {"ContentRecords","value=%7B%22n%22:7%7D","Number",204},
  {"ContentRecords","value=%7B%22m%22:%22ok%22%7D","Label",204},
  {"ContentRecords","value=%7B%7D","",400},
  {"ContentClosedItems","value=%7B%22n%22:7%7D","Objects",204},
  {"ContentClosedItems","value=%7B%22n%22:7,%22unknown%22:1%7D","",400},
  {"ContentClosedItems","value=%7B%22n%22:7,%22m%22:%22ok%22%7D","",400},
  {"ContentNulls","value=null","Anything",204},
  {"ContentNulls","value=true","",400},
  {"ContentArrays","value=%5B%5D&value=%5B1,2%5D","Matrices",204},
  {"ContentNested","value=%7B%22n%22:7%7D","Inner",204},
 }{
  t.Run(tc.name+"/"+tc.wire,func(t *testing.T){
   calls:=0
   endpoint:=func(_ context.Context,raw any)(any,error){
    calls++
    switch value:=raw.(type){
    case *svc.ContentPayload:
     require.Equal(t,tc.kind,string(value.Value.Kind()))
     if words,ok:=value.Value.AsWords();ok {require.Equal(t,[]svc.FormContentWord{"text:a","text:b"},[]svc.FormContentWord(words));require.NotNil(t,value.Owner);require.Equal(t,"kept",*value.Owner)}
     if records,ok:=value.Value.AsRecords();ok {require.Len(t,records,2);require.EqualValues(t,7,records[0].Number);require.EqualValues(t,8,records[1].Number)}
    case *svc.ContentOptionalPayload: require.Nil(t,value.Value)
    case *svc.ContentOnlyPayload:items,ok:=value.Value.AsRecords();require.True(t,ok);require.Len(t,items,1);require.EqualValues(t,7,items[0].Number)
    case *svc.ContentBroadPayload:
     words,ok:=value.Value.AsWords();require.True(t,ok);require.Len(t,words,1)
     expected:="";if tc.wire=="value=%22%22" {expected="\"\""};if tc.wire=="value=%7B" {expected="{"};require.Equal(t,expected,string(words[0]))
    case *svc.ContentJSONPayload:
     require.Equal(t,tc.kind,string(value.Value.Kind()))
     if words,ok:=value.Value.AsWords();ok {require.Len(t,words,1);require.Empty(t,words[0])}
    case *svc.ContentWidthPayload:require.Equal(t,tc.kind,string(value.Value.Kind()));items,ok:=value.Value.AsFloats();require.True(t,ok);require.Equal(t,[]float64{1.5},[]float64(items))
    case *svc.ContentArraysPayload:require.Equal(t,tc.kind,string(value.Value.Kind()));items,ok:=value.Value.AsMatrices();require.True(t,ok);require.Len(t,items,2);require.Empty(t,items[0]);require.Equal(t,[]int{1,2},items[1])
    case *svc.ContentRecordsPayload:
     require.Equal(t,tc.kind,string(value.Value.Kind()));if items,ok:=value.Value.AsNumber();ok {require.Len(t,items,1);require.EqualValues(t,7,items[0].Number)};if items,ok:=value.Value.AsLabel();ok {require.Len(t,items,1);require.Equal(t,"ok",items[0].Label)}
    case *svc.ContentClosedItemsPayload:require.Equal(t,tc.kind,string(value.Value.Kind()));items,ok:=value.Value.AsObjects();require.True(t,ok);require.Len(t,items,1);item,ok:=items[0].AsNumber();require.True(t,ok);require.EqualValues(t,7,item.Number)
    case *svc.ContentNullsPayload:require.Equal(t,tc.kind,string(value.Value.Kind()));items,ok:=value.Value.AsAnything();require.True(t,ok);require.Len(t,items,1);raw,err:=json.Marshal(items[0]);require.NoError(t,err);require.JSONEq(t,"null",string(raw))
    case *svc.ContentNestedPayload:require.Equal(t,tc.kind,string(value.Value.Kind()))
    default:t.Errorf("unexpected payload %T",raw)
    }
    return nil,nil
   }
   endpoints:=&svc.Endpoints{Content:endpoint,ContentOnly:endpoint,ContentOptional:endpoint,ContentBroad:endpoint,ContentJSON:endpoint,ContentWidth:endpoint,ContentNested:endpoint,ContentRecords:endpoint,ContentNulls:endpoint,ContentArrays:endpoint,ContentClosedItems:endpoint}
   mux:=loomhttp.NewMuxer();server.Mount(mux,server.New(endpoints,mux,loomhttp.RequestDecoder,loomhttp.ResponseEncoder,nil,nil))
   request:=httptest.NewRequest("POST","/"+tc.name,strings.NewReader(tc.wire));request.Header.Set("Content-Type","application/x-www-form-urlencoded")
   response:=httptest.NewRecorder();mux.ServeHTTP(response,request)
   require.Equal(t,tc.status,response.Code,response.Body.String())
   if tc.status==204 {require.Equal(t,1,calls)} else {require.Zero(t,calls);require.Contains(t,response.Body.String(),"decode_payload");require.Contains(t,response.Body.String(),"invalid request body")}
  })
 }
}

func TestFormContentOutgoing(t *testing.T) {
 transport:=client.NewClient("http","example.test",http.DefaultClient,loomhttp.RequestEncoder,loomhttp.ResponseDecoder,false)
 for _,jsonItems:=range []bool{false,true}{
  for _,size:=range []int{-1,0,2}{
   empty:=size<2
   var choice svc.FormContentChoice
   if jsonItems {var records []*svc.FormContentRecord;if size==0 {records=[]*svc.FormContentRecord{}};if !empty {records=[]*svc.FormContentRecord{{Number:7},{Number:8}}};choice.SetRecords(records)} else {var words []svc.FormContentWord;if size==0 {words=[]svc.FormContentWord{}};if !empty {words=[]svc.FormContentWord{"text:a","text:b"}};choice.SetWords(words)}
   required:=&svc.ContentPayload{Value:choice}
   request,err:=transport.BuildContentRequest(context.Background(),required);require.NoError(t,err)
   err=client.EncodeContentRequest(loomhttp.RequestEncoder)(request,required)
   if empty {require.Error(t,err);require.Nil(t,request.Body)} else {
    require.NoError(t,err);require.NoError(t,request.ParseForm())
    if jsonItems {require.Equal(t,[]string{"{\"n\":7}","{\"n\":8}"},request.PostForm["value"])} else {require.Equal(t,[]string{"text:a","text:b"},request.PostForm["value"])}
   }
   owner:="kept";optional:=&svc.ContentOptionalPayload{Value:&choice,Owner:&owner}
   request,err=transport.BuildContentOptionalRequest(context.Background(),optional);require.NoError(t,err)
   require.NoError(t,client.EncodeContentOptionalRequest(loomhttp.RequestEncoder)(request,optional));require.NoError(t,request.ParseForm())
   require.Equal(t,!empty,request.PostForm.Has("value"));require.Equal(t,"kept",request.PostForm.Get("owner"))
  }
 }
 var ambiguous svc.FormContentBroad;ambiguous.SetRecords([]*svc.FormContentRecord{{Number:7}})
 payload:=&svc.ContentBroadPayload{Value:ambiguous}
 request,err:=transport.BuildContentBroadRequest(context.Background(),payload);require.NoError(t,err)
 require.Error(t,client.EncodeContentBroadRequest(loomhttp.RequestEncoder)(request,payload));require.Nil(t,request.Body);require.Empty(t,request.Header.Get("Content-Type"))
 for _,jsonItems:=range []bool{false,true}{
  var broad svc.FormContentBroad;broad.SetWords([]string{""})
  if jsonItems {
   payload:=&svc.ContentJSONPayload{Value:broad};request,err=transport.BuildContentJSONRequest(context.Background(),payload);require.NoError(t,err)
   require.NoError(t,client.EncodeContentJSONRequest(loomhttp.RequestEncoder)(request,payload));require.NoError(t,request.ParseForm());require.Equal(t,[]string{"\"\""},request.PostForm["value"])
  } else {
   payload:=&svc.ContentBroadPayload{Value:broad};request,err=transport.BuildContentBroadRequest(context.Background(),payload);require.NoError(t,err)
   require.NoError(t,client.EncodeContentBroadRequest(loomhttp.RequestEncoder)(request,payload));require.NoError(t,request.ParseForm());require.Equal(t,[]string{""},request.PostForm["value"])
  }
 }
}

func TestFormContentMediaRoundTrip(t *testing.T) {
 calls:=0
 endpoint:=func(_ context.Context,value any)(any,error){calls++;return value,nil}
 mux:=loomhttp.NewMuxer();server.Mount(mux,server.New(&svc.Endpoints{ContentMedia:endpoint},mux,loomhttp.RequestDecoder,loomhttp.ResponseEncoder,nil,nil))
 remote:=httptest.NewServer(mux);defer remote.Close()
 transport:=client.NewClient("http",strings.TrimPrefix(remote.URL,"http://"),remote.Client(),loomhttp.RequestEncoder,loomhttp.ResponseDecoder,false)
 for _,jsonItems:=range []bool{false,true}{
  var choice svc.FormContentChoice
  if jsonItems {choice.SetRecords([]*svc.FormContentRecord{{Number:7}})} else {choice.SetWords([]svc.FormContentWord{"text:a"})}
  var media svc.FormContentMedia;media.SetForm(&svc.FormContentForm{Value:choice})
  got,err:=transport.ContentMedia()(context.Background(),&media);require.NoError(t,err)
  result:=got.(*svc.FormContentMedia);body,ok:=result.AsForm();require.True(t,ok);require.Equal(t,choice.Kind(),body.Value.Kind())
 }
 require.Equal(t,2,calls)
}

func TestFormContentRejectedResponse(t *testing.T) {
 for _,words:=range [][]svc.FormContentWord{nil,{}, {"invalid"}}{
  var choice svc.FormContentChoice;choice.SetWords(words)
  var media svc.FormContentMedia;media.SetForm(&svc.FormContentForm{Value:choice})
  response:=httptest.NewRecorder();response.Code=0
  require.Error(t,server.EncodeContentMediaResponse(loomhttp.ResponseEncoder)(context.Background(),response,&media))
  require.Empty(t,response.Body.String());require.Zero(t,response.Code)
 }
}

func TestFormContentNestedAndRequiredKeysOutgoing(t *testing.T) {
 transport:=client.NewClient("http","example.test",http.DefaultClient,loomhttp.RequestEncoder,loomhttp.ResponseDecoder,false)
 var inner svc.FormContentChoice;inner.SetRecords([]*svc.FormContentRecord{{Number:7}})
 var nested svc.FormContentNested;nested.SetInner(&inner)
 payload:=&svc.ContentNestedPayload{Value:nested}
 request,err:=transport.BuildContentNestedRequest(context.Background(),payload);require.NoError(t,err)
 require.NoError(t,client.EncodeContentNestedRequest(loomhttp.RequestEncoder)(request,payload));require.NoError(t,request.ParseForm());require.Equal(t,[]string{"{\"n\":7}"},request.PostForm["value"])
 nested.SetInner(nil);payload.Value=nested
 request,err=transport.BuildContentNestedRequest(context.Background(),payload);require.NoError(t,err)
 require.Error(t,client.EncodeContentNestedRequest(loomhttp.RequestEncoder)(request,payload));require.Nil(t,request.Body)
 for _,label:=range []bool{false,true}{
  var choice svc.FormContentRecords
  expected:="{\"n\":0}"
  if label {choice.SetLabel([]*svc.FormContentOtherRecord{{Label:"ok"}});expected="{\"m\":\"ok\"}"} else {choice.SetNumber([]*svc.FormContentRecord{{Number:0}})}
  payload:=&svc.ContentRecordsPayload{Value:choice}
  request,err:=transport.BuildContentRecordsRequest(context.Background(),payload);require.NoError(t,err)
  require.NoError(t,client.EncodeContentRecordsRequest(loomhttp.RequestEncoder)(request,payload));require.NoError(t,request.ParseForm());require.Equal(t,[]string{expected},request.PostForm["value"])
 }
}

func TestFormContentNestedItemEncoding(t *testing.T) {
 var choice svc.FormContentArrays;choice.SetMatrices([][]int{{},{1,2}})
 payload:=&svc.ContentArraysPayload{Value:choice}
 transport:=client.NewClient("http","example.test",http.DefaultClient,loomhttp.RequestEncoder,loomhttp.ResponseDecoder,false)
 request,err:=transport.BuildContentArraysRequest(context.Background(),payload);require.NoError(t,err)
 require.NoError(t,client.EncodeContentArraysRequest(loomhttp.RequestEncoder)(request,payload));require.NoError(t,request.ParseForm());require.Equal(t,[]string{"[]","[1,2]"},request.PostForm["value"])
}
`
