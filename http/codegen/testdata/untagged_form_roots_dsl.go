package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func untaggedFormRootMethods() func() {
	count, empty, choice, label, nested, mapping := untaggedFormRootChoices()
	zeroFields := untaggedFormZeroFieldRootMethods(count)
	filter, fieldRoot, itemsRoot := untaggedFormRootFieldChoices(label)
	open, physical, closed, closedNested := untaggedFormRootClosureChoices(filter, label)
	ambiguous := Type("FormRootAmbiguous", OneOf(func() {
		Untagged()
		Attribute("Count", count)
		Attribute("Labels", MapOf(String, String))
	}))
	otherEmpty := Type("FormRootOtherEmpty", func() {
		Attribute("note", String)
	})
	absent := Type("FormRootAbsent", OneOf(func() {
		Untagged()
		Attribute("Empty", empty)
		Attribute("Other", otherEmpty)
	}))
	media := Type("FormRootMedia", OneOf(func() {
		Attribute("Form", choice)
		Attribute("Text", String)
	}))
	return func() {
		for _, item := range []struct {
			name string
			typ  any
		}{{"RootRequired", choice}, {"RootNested", nested}, {"RootMap", mapping}, {"RootPhysical", physical}, {"RootClosed", closed}, {"RootAmbiguous", ambiguous}, {"RootFields", fieldRoot}, {"RootItems", itemsRoot}, {"RootOpen", open}, {"RootClosedNested", closedNested}} {
			Method(item.name, func() {
				Payload(item.typ)
				HTTP(func() {
					POST("/" + item.name)
					FormRequest()
					Response(StatusNoContent)
				})
			})
		}
		untaggedFormOptionalRootMethods(media, absent, choice)
		zeroFields()
	}
}

func untaggedFormZeroFieldRootMethods(count expr.UserType) func() {
	empty := Type("FormRootZeroFields", func() {
		Meta("openapi:additionalProperties", "false")
	})
	choice := Type("FormRootZeroChoice", OneOf(func() {
		Untagged()
		Attribute("Count", count)
		Attribute("Empty", empty)
	}))
	return func() {
		Method("RootZeroRequired", func() {
			Payload(choice)
			HTTP(func() {
				POST("/RootZeroRequired")
				FormRequest()
				Response(StatusNoContent)
			})
		})
		Method("RootZeroOptional", func() {
			Payload(func() { Attribute("value", choice) })
			HTTP(func() {
				POST("/RootZeroOptional")
				Body("value")
				OptionalRequestBody()
				FormRequest()
				Response(StatusNoContent)
			})
		})
	}
}

func untaggedFormRootChoices() (expr.UserType, expr.UserType, expr.UserType, expr.UserType, expr.UserType, expr.UserType) {
	count := Type("FormRootCount", func() {
		Attribute("count", Int64)
		Required("count")
	})
	empty := Type("FormRootEmpty", func() {
		Attribute("empty", Boolean)
		Meta("openapi:additionalProperties", "false")
	})
	choice := Type("FormRootChoice", OneOf(func() {
		Untagged()
		Attribute("Count", count)
		Attribute("Empty", empty)
	}))
	label := Type("FormRootLabel", func() {
		Attribute("label", String)
		Required("label")
	})
	nested := Type("FormRootNested", OneOf(func() {
		Untagged()
		Attribute("Inner", choice)
		Attribute("Label", label)
	}))
	mapping := Type("FormRootMap", OneOf(func() {
		Untagged()
		Attribute("Count", count)
		Attribute("Flags", MapOf(String, Boolean))
	}))
	return count, empty, choice, label, nested, mapping
}

func untaggedFormRootFieldChoices(label expr.UserType) (expr.UserType, expr.UserType, expr.UserType) {
	filterValue := Type("FormRootFilterValue", func() {
		Attribute("x", String, func() { Meta("struct:tag:json", "jsonX") })
		Required("x")
	})
	filter := Type("FormRootFilter", func() {
		Attribute("filter", filterValue)
		Required("filter")
		Meta("openapi:encoding:filter:style", "form")
		Meta("openapi:encoding:filter:explode", "true")
	})
	fieldChoice := Type("FormRootFieldChoice", OneOf(func() {
		Untagged()
		Attribute("Flag", Boolean)
		Attribute("Record", filterValue)
	}))
	fields := Type("FormRootFields", func() {
		Attribute("value", fieldChoice)
		Attribute("jsonValue", fieldChoice)
		Attribute("records", ArrayOf(filterValue))
		Required("value", "jsonValue")
		Meta("openapi:encoding:value:style", "form")
		Meta("openapi:encoding:value:explode", "false")
		Meta("openapi:encoding:jsonValue:contentType", "application/json")
	})
	items := Type("FormRootItems", func() {
		Attribute("records", ArrayOf(filterValue))
		Required("records")
	})
	itemsRoot := Type("FormRootItemUnion", OneOf(func() {
		Untagged()
		Attribute("Items", items)
		Attribute("Label", label)
	}))
	fieldRoot := Type("FormRootFieldUnion", OneOf(func() {
		Untagged()
		Attribute("Fields", fields)
		Attribute("Label", label)
	}))
	return filter, fieldRoot, itemsRoot
}

func untaggedFormRootClosureChoices(filter, label expr.UserType) (expr.UserType, expr.UserType, expr.UserType, expr.UserType) {
	openFilter := Type("FormRootOpenFilter", filter, func() {
		Meta("openapi:additionalProperties", "true")
	})
	open := Type("FormRootOpen", OneOf(func() {
		Untagged()
		Attribute("Filter", openFilter)
		Attribute("Label", label)
		Meta("openapi:unevaluatedProperties", "false")
	}))
	physical := Type("FormRootPhysical", OneOf(func() {
		Untagged()
		Attribute("Filter", filter)
		Attribute("Label", label)
	}))
	closed := Type("FormRootClosed", OneOf(func() {
		Untagged()
		Attribute("Filter", filter)
		Attribute("Label", label)
		Meta("openapi:unevaluatedProperties", "false")
	}))
	closedNested := Type("FormRootClosedNested", OneOf(func() {
		Untagged()
		Attribute("Closed", closed)
		Attribute("Labels", MapOf(String, String))
	}))
	return open, physical, closed, closedNested
}

func untaggedFormOptionalRootMethods(media, absent, choice expr.UserType) {
	Method("RootMedia", func() {
		Payload(media)
		Result(media)
		HTTP(func() {
			POST("/RootMedia")
			MediaType("application/x-www-form-urlencoded", "Form")
			MediaType("text/plain", "Text")
			Response(StatusOK, func() {
				MediaType("application/x-www-form-urlencoded", "Form")
				MediaType("text/plain", "Text")
			})
		})
	})
	Method("RootOptionalAmbiguous", func() {
		Payload(func() { Attribute("value", absent) })
		HTTP(func() {
			POST("/RootOptionalAmbiguous")
			Body("value")
			OptionalRequestBody()
			FormRequest()
			Response(StatusNoContent)
		})
	})
	Method("RootOptional", func() {
		Payload(func() {
			Attribute("value", choice)
		})
		HTTP(func() {
			POST("/RootOptional")
			Body("value")
			OptionalRequestBody()
			FormRequest()
			Response(StatusNoContent)
		})
	})
}

const untaggedFormRootsHarness = `
func TestFormRootPresence(t *testing.T) {
 for _, tc := range []struct{name, wire, kind string; optional bool}{
  {"RootRequired", "", "Empty", false},
  {"RootRequired", "&&", "Empty", false},
  {"RootRequired", "count=7", "Count", false},
  {"RootRequired", "count=0", "Count", false},
  {"RootOptional", "", "", true},
  {"RootOptional", "&&", "Empty", true},
  {"RootOptional", "count=7", "Count", true},
 } {
  t.Run(tc.name+"/"+tc.wire,func(t *testing.T){
   calls:=0
   handler:=func(_ context.Context, raw any)(any,error){
    calls++
    var value *svc.FormRootChoice
    if tc.optional {value=raw.(*svc.RootOptionalPayload).Value} else {value=raw.(*svc.FormRootChoice)}
    if tc.kind=="" { require.Nil(t,value) } else {require.NotNil(t,value);require.Equal(t,tc.kind,string(value.Kind()))}
    return nil,nil
   }
   endpoints:=&svc.Endpoints{RootRequired:handler,RootOptional:handler}
   mux:=loomhttp.NewMuxer();server.Mount(mux,server.New(endpoints,mux,loomhttp.RequestDecoder,loomhttp.ResponseEncoder,nil,nil))
   request:=httptest.NewRequest("POST","/"+tc.name,strings.NewReader(tc.wire));request.Header.Set("Content-Type","application/x-www-form-urlencoded")
   response:=httptest.NewRecorder();mux.ServeHTTP(response,request)
   require.Equal(t,204,response.Code,response.Body.String());require.Equal(t,1,calls)
  })
 }
}

func TestFormRootDomains(t *testing.T) {
 for _, tc := range []struct{name, wire, kind string; status int}{
  {"RootNested", "count=7", "Inner", 204},
  {"RootNested", "label=ok", "Label", 204},
  {"RootNested", "count=7&label=ok", "", 400},
  {"RootMap", "=true&flag=false", "Flags", 204},
  {"RootMap", "", "Flags", 204},
  {"RootMap", "count=999999999999999999999999", "", 400},
  {"RootPhysical", "x=ok&filter=extra", "Filter", 204},
  {"RootClosed", "x=ok", "Filter", 204},
  {"RootOpen", "x=ok&filter=extra", "Filter", 204},
  {"RootClosedNested", "x=ok&filter=extra", "Labels", 204},
  {"RootFields", "value=x,ok&jsonValue=%7B%22jsonX%22:%22yes%22%7D", "Fields", 204},
  {"RootFields", "value=x,ok&jsonValue=%7B%22jsonX%22:%22yes%22%7D&records=%7B%22jsonX%22:%22one%22%7D&records=%7B%22jsonX%22:%22two%22%7D", "Fields", 204},
  {"RootFields", "value=x,ok&jsonValue=%7B%22jsonX%22:%22yes%22%7D&records=%5B%7B%22jsonX%22:%22one%22%7D%5D", "", 400},
  {"RootFields", "value=jsonX,ok&jsonValue=%7B%22jsonX%22:%22yes%22%7D", "", 400},
  {"RootClosed", "x=ok&filter=extra", "", 400},
  {"RootClosed", "x=ok&unknown=extra", "", 400},
  {"RootRequired", "count=7&count=8", "Count", 204},
  {"RootRequired", "count=%FF", "", 400},
  {"RootRequired", "count=bad", "", 400},
  {"RootAmbiguous", "count=999999999999999999999999", "", 400},
 } {
  t.Run(tc.name+"/"+tc.wire,func(t *testing.T){
   calls:=0
   endpoint:=func(_ context.Context,raw any)(any,error){
    calls++
    switch value:=raw.(type){
    case *svc.FormRootNested: require.Equal(t,tc.kind,string(value.Kind())); if tc.kind=="Inner" {inner,ok:=value.AsInner(); require.True(t,ok); require.Equal(t,"Count",string(inner.Kind()))}
    case *svc.FormRootMap: require.Equal(t,tc.kind,string(value.Kind())); if tc.wire!="" {flags,ok:=value.AsFlags();require.True(t,ok);require.True(t,flags[""]);require.False(t,flags["flag"])}
    case *svc.FormRootFieldUnion:
     require.Equal(t,tc.kind,string(value.Kind()));fields,ok:=value.AsFields();require.True(t,ok)
     text,ok:=fields.Value.AsRecord();require.True(t,ok);require.Equal(t,"ok",text.X)
     jsonValue,ok:=fields.JSONValue.AsRecord();require.True(t,ok);require.Equal(t,"yes",jsonValue.X)
     if strings.Contains(tc.wire,"records=") {require.Len(t,fields.Records,2);require.Equal(t,"one",fields.Records[0].X);require.Equal(t,"two",fields.Records[1].X)}
    case *svc.FormRootOpen: require.Equal(t,tc.kind,string(value.Kind()))
    case *svc.FormRootClosedNested: require.Equal(t,tc.kind,string(value.Kind()))
    case *svc.FormRootPhysical: require.Equal(t,tc.kind,string(value.Kind()))
    case *svc.FormRootClosed: require.Equal(t,tc.kind,string(value.Kind()))
    case *svc.FormRootChoice: require.Equal(t,tc.kind,string(value.Kind()));count,ok:=value.AsCount();require.True(t,ok);require.EqualValues(t,7,count.Count)
    default: t.Errorf("unexpected payload %T",raw)
    }
    return nil,nil
   }
   endpoints:=&svc.Endpoints{RootNested:endpoint,RootMap:endpoint,RootPhysical:endpoint,RootClosed:endpoint,RootRequired:endpoint,RootAmbiguous:endpoint,RootFields:endpoint,RootOpen:endpoint,RootClosedNested:endpoint}
   mux:=loomhttp.NewMuxer();server.Mount(mux,server.New(endpoints,mux,loomhttp.RequestDecoder,loomhttp.ResponseEncoder,nil,nil))
   request:=httptest.NewRequest("POST","/"+tc.name,strings.NewReader(tc.wire));request.Header.Set("Content-Type","application/x-www-form-urlencoded")
   response:=httptest.NewRecorder();mux.ServeHTTP(response,request)
   require.Equal(t,tc.status,response.Code,response.Body.String())
   if tc.status==204 {require.Equal(t,1,calls)} else {require.Zero(t,calls);require.Contains(t,response.Body.String(),"decode_payload");require.Contains(t,response.Body.String(),"invalid request body")}
  })
 }
}

func TestFormRootOutgoing(t *testing.T) {
 transport:=client.NewClient("http","example.test",http.DefaultClient,loomhttp.RequestEncoder,loomhttp.ResponseDecoder,false)
 for _, count:= range []int64{-1,0,7} {
  empty:=count<0
  var choice svc.FormRootChoice
  if empty {choice.SetEmpty(&svc.FormRootEmpty{})} else {choice.SetCount(&svc.FormRootCount{Count:count})}
  request,err:=transport.BuildRootRequiredRequest(context.Background(),&choice);require.NoError(t,err)
  require.NoError(t,client.EncodeRootRequiredRequest(loomhttp.RequestEncoder)(request,&choice))
  require.NoError(t,request.ParseForm())
  if empty {require.Empty(t,request.PostForm)} else {require.Equal(t,fmt.Sprint(count),request.PostForm.Get("count"))}
 }
 var ambiguous svc.FormRootAmbiguous
 ambiguous.SetCount(&svc.FormRootCount{Count:7})
 request,err:=transport.BuildRootAmbiguousRequest(context.Background(),&ambiguous);require.NoError(t,err)
 require.Error(t,client.EncodeRootAmbiguousRequest(loomhttp.RequestEncoder)(request,&ambiguous))
 require.Nil(t,request.Body);require.Empty(t,request.Header.Get("Content-Type"))
}

func TestFormRootMediaRoundTrip(t *testing.T) {
 calls:=0
 mux:=loomhttp.NewMuxer();endpoint:=func(_ context.Context,value any)(any,error){calls++;return value,nil}
 server.Mount(mux,server.New(&svc.Endpoints{RootMedia:endpoint},mux,loomhttp.RequestDecoder,loomhttp.ResponseEncoder,nil,nil))
 remote:=httptest.NewServer(mux);defer remote.Close()
 transport:=client.NewClient("http",strings.TrimPrefix(remote.URL,"http://"),remote.Client(),loomhttp.RequestEncoder,loomhttp.ResponseDecoder,false)
 for _,empty:=range []bool{false,true}{
  var choice svc.FormRootChoice
  if empty {choice.SetEmpty(&svc.FormRootEmpty{})} else {choice.SetCount(&svc.FormRootCount{Count:7})}
  var media svc.FormRootMedia;media.SetForm(&choice)
  got,err:=transport.RootMedia()(context.Background(),&media);require.NoError(t,err)
  result:=got.(*svc.FormRootMedia);value,ok:=result.AsForm();require.True(t,ok);require.Equal(t,choice.Kind(),value.Kind())
 }
 require.Equal(t,2,calls)
}

func TestFormRootOptionalEmptyEncoding(t *testing.T) {
 var choice svc.FormRootAbsent;choice.SetEmpty(&svc.FormRootEmpty{})
 payload:=&svc.RootOptionalAmbiguousPayload{Value:&choice}
 transport:=client.NewClient("http","example.test",http.DefaultClient,loomhttp.RequestEncoder,loomhttp.ResponseDecoder,false)
 request,err:=transport.BuildRootOptionalAmbiguousRequest(context.Background(),payload);require.NoError(t,err)
 require.NoError(t,client.EncodeRootOptionalAmbiguousRequest(loomhttp.RequestEncoder)(request,payload))
 calls:=0
 endpoint:=func(_ context.Context,value any)(any,error){calls++;require.Nil(t,value.(*svc.RootOptionalAmbiguousPayload).Value);return nil,nil}
 mux:=loomhttp.NewMuxer();server.Mount(mux,server.New(&svc.Endpoints{RootOptionalAmbiguous:endpoint},mux,loomhttp.RequestDecoder,loomhttp.ResponseEncoder,nil,nil))
 response:=httptest.NewRecorder();mux.ServeHTTP(response,request);require.Equal(t,204,response.Code,response.Body.String());require.Equal(t,1,calls)
 // A non-empty spelling of the same empty value set is present and ambiguous.
 request=httptest.NewRequest("POST","/RootOptionalAmbiguous",strings.NewReader("&&"));request.Header.Set("Content-Type","application/x-www-form-urlencoded")
 response=httptest.NewRecorder();mux.ServeHTTP(response,request);require.Equal(t,400,response.Code);require.Equal(t,1,calls)
}

func TestFormRootJSONItemsOutgoing(t *testing.T) {
 var text,jsonValue svc.FormRootFieldChoice
 text.SetRecord(&svc.FormRootFilterValue{X:"ok"});jsonValue.SetRecord(&svc.FormRootFilterValue{X:"yes"})
 var choice svc.FormRootFieldUnion
 choice.SetFields(&svc.FormRootFields{Value:text,JSONValue:jsonValue,Records:[]*svc.FormRootFilterValue{{X:"one"},{X:"two"}}})
 transport:=client.NewClient("http","example.test",http.DefaultClient,loomhttp.RequestEncoder,loomhttp.ResponseDecoder,false)
 request,err:=transport.BuildRootFieldsRequest(context.Background(),&choice);require.NoError(t,err)
 require.NoError(t,client.EncodeRootFieldsRequest(loomhttp.RequestEncoder)(request,&choice))
 require.NoError(t,request.ParseForm());require.Equal(t,[]string{"{\"jsonX\":\"one\"}","{\"jsonX\":\"two\"}"},request.PostForm["records"])
}

func TestFormRootRequiredJSONItems(t *testing.T) {
 transport:=client.NewClient("http","example.test",http.DefaultClient,loomhttp.RequestEncoder,loomhttp.ResponseDecoder,false)
 for _,records:=range [][]*svc.FormRootFilterValue{nil,{},{{X:"one"}}}{
  var choice svc.FormRootItemUnion;choice.SetItems(&svc.FormRootItems{Records:records})
  request,err:=transport.BuildRootItemsRequest(context.Background(),&choice);require.NoError(t,err)
  err=client.EncodeRootItemsRequest(loomhttp.RequestEncoder)(request,&choice)
  if len(records)==0 {require.Error(t,err);require.Nil(t,request.Body);continue}
  require.NoError(t,err)
  calls:=0
  endpoint:=func(_ context.Context,value any)(any,error){calls++;payload:=value.(*svc.FormRootItemUnion);items,ok:=payload.AsItems();require.True(t,ok);require.Len(t,items.Records,1);require.Equal(t,"one",items.Records[0].X);return nil,nil}
  mux:=loomhttp.NewMuxer();server.Mount(mux,server.New(&svc.Endpoints{RootItems:endpoint},mux,loomhttp.RequestDecoder,loomhttp.ResponseEncoder,nil,nil))
  response:=httptest.NewRecorder();mux.ServeHTTP(response,request);require.Equal(t,204,response.Code,response.Body.String());require.Equal(t,1,calls)
 }
}

func TestZeroFieldFormRootIncoming(t *testing.T) {
	for _, tc := range []struct {
		name, wire, kind string
		optional         bool
		status           int
	}{
		{"required-empty", "", "Empty", false, 204},
		{"required-present-empty", "&&", "Empty", false, 204},
		{"required-count-zero", "count=0", "Count", false, 204},
		{"required-invalid", "count=bad", "", false, 400},
		{"required-unknown", "unexpected=1", "", false, 400},
		{"optional-absent", "", "", true, 204},
		{"optional-present-empty", "&&", "Empty", true, 204},
		{"optional-count-zero", "count=0", "Count", true, 204},
		{"optional-invalid", "count=bad", "", true, 400},
		{"optional-unknown", "unexpected=1", "", true, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			endpoint := func(_ context.Context, raw any) (any, error) {
				calls++
				var choice *svc.FormRootZeroChoice
				if tc.optional {
					choice = raw.(*svc.RootZeroOptionalPayload).Value
				} else {
					choice = raw.(*svc.FormRootZeroChoice)
				}
				if tc.kind == "" {
					require.Nil(t, choice)
				} else {
					require.NotNil(t, choice)
					require.Equal(t, tc.kind, string(choice.Kind()))
					if tc.kind == "Empty" {
						value, ok := choice.AsEmpty()
						require.True(t, ok)
						require.NotNil(t, value)
					} else {
						value, ok := choice.AsCount()
						require.True(t, ok)
						require.Zero(t, value.Count)
					}
				}
				return nil, nil
			}
			mux := loomhttp.NewMuxer()
			server.Mount(mux, server.New(&svc.Endpoints{RootZeroRequired: endpoint, RootZeroOptional: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
			route := "/RootZeroRequired"
			if tc.optional {
				route = "/RootZeroOptional"
			}
			request := httptest.NewRequest(http.MethodPost, route, strings.NewReader(tc.wire))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			require.Equal(t, tc.status, response.Code, response.Body.String())
			if tc.status == 204 {
				require.Equal(t, 1, calls)
			} else {
				require.Zero(t, calls)
				var problem map[string]any
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &problem))
				require.Equal(t, "decode_payload", problem["code"])
				require.Equal(t, "invalid request body", problem["detail"])
			}
		})
	}
}

func TestZeroFieldFormRootOutgoing(t *testing.T) {
	for _, optional := range []bool{false, true} {
		for _, selection := range []string{"Empty", "Count", "absent"} {
			if !optional && selection == "absent" {
				continue
			}
			t.Run(fmt.Sprintf("optional=%t/%s", optional, selection), func(t *testing.T) {
				testZeroFieldFormRootOutgoing(t, optional, selection)
			})
		}
	}
}

func TestZeroFieldFormRootInvalidSelection(t *testing.T) {
	for _, optional := range []bool{false, true} {
		for _, selection := range []string{"nil branch", "unselected"} {
			t.Run(fmt.Sprintf("optional=%t/%s", optional, selection), func(t *testing.T) {
				testZeroFieldFormRootOutgoing(t, optional, selection)
			})
		}
	}
}

func testZeroFieldFormRootOutgoing(t *testing.T, optional bool, selection string) {
	t.Helper()
	transport := client.NewClient("http", "example.test", http.DefaultClient, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	var choice svc.FormRootZeroChoice
	switch selection {
	case "Empty":
		choice.SetEmpty(&svc.FormRootZeroFields{})
	case "Count":
		choice.SetCount(&svc.FormRootCount{Count: 7})
	case "nil branch":
		choice.SetEmpty(nil)
	}
	var payload any = &choice
	build := transport.BuildRootZeroRequiredRequest
	encode := client.EncodeRootZeroRequiredRequest(loomhttp.RequestEncoder)
	if optional {
		envelope := &svc.RootZeroOptionalPayload{}
		if selection != "absent" {
			envelope.Value = &choice
		}
		payload = envelope
		build = transport.BuildRootZeroOptionalRequest
		encode = client.EncodeRootZeroOptionalRequest(loomhttp.RequestEncoder)
	}
	request, err := build(context.Background(), payload)
	require.NoError(t, err)
	err = encode(request, payload)
	if selection == "nil branch" || selection == "unselected" && !optional {
		if selection == "nil branch" {
			require.ErrorContains(t, err, "selected form union branch is nil")
		} else {
			require.Error(t, err)
		}
		require.Nil(t, request.Body)
		require.Empty(t, request.Header.Get("Content-Type"))
		return
	}
	require.NoError(t, err)
	if selection == "absent" || optional && selection == "unselected" {
		require.Nil(t, request.Body)
	} else {
		require.NotNil(t, request.Body)
		require.Equal(t, "application/x-www-form-urlencoded", request.Header.Get("Content-Type"))
		if selection == "Empty" {
			require.Zero(t, request.ContentLength)
		} else {
			require.Positive(t, request.ContentLength)
		}
	}
	calls := 0
	endpoint := func(_ context.Context, raw any) (any, error) {
		calls++
		var received *svc.FormRootZeroChoice
		if optional {
			received = raw.(*svc.RootZeroOptionalPayload).Value
		} else {
			received = raw.(*svc.FormRootZeroChoice)
		}
		if optional && selection != "Count" {
			require.Nil(t, received)
		} else {
			require.NotNil(t, received)
			require.Equal(t, selection, string(received.Kind()))
			if selection == "Count" {
				value, ok := received.AsCount()
				require.True(t, ok)
				require.EqualValues(t, 7, value.Count)
			}
		}
		return nil, nil
	}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(&svc.Endpoints{RootZeroRequired: endpoint, RootZeroOptional: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	// The HTTP server supplies a nonnil body even when the client sends none.
	if request.Body == nil {
		request.Body = http.NoBody
	}
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	require.Equal(t, 204, response.Code, response.Body.String())
	require.Equal(t, 1, calls)
}

`
