package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// UntaggedFormDSL covers field-level union codecs over a raw form carrier.
func UntaggedFormDSL() {
	roots := untaggedFormRootMethods()
	ordinary := ordinaryFormPresenceMethods()
	content := untaggedFormContentMethods()
	choice := Type("FormChoice", OneOf(func() {
		Untagged()
		Attribute("Number", Int)
		Attribute("List", ArrayOf(String))
	}))
	mapChoice := Type("FormMapChoice", OneOf(func() {
		Untagged()
		Attribute("Flag", Boolean)
		Attribute("Map", MapOf(String, Int))
	}))
	record := Type("FormRecord", func() {
		Attribute("key", String)
		Attribute("note", String, func() { Default("fallback") })
		Required("key")
	})
	objectChoice := Type("FormObjectChoice", OneOf(func() {
		Untagged()
		Attribute("Flag", Boolean)
		Attribute("Record", record)
	}))
	native := Type("FormNative", OneOf(Any, Boolean), Untagged)
	tags := taggedFormPayload(choice)

	Service("formchoices", func() {
		roots()
		ordinary()
		content()
		taggedFormMethod(tags)

		untaggedCompositeFormMethods(mapChoice, objectChoice)
		untaggedJSONFormMethods(choice, native)
		for _, item := range []struct {
			name    string
			explode string
		}{{"fields", "false"}, {"exploded", "true"}} {
			Method(item.name, func() {
				Payload(func() {
					Attribute("value", choice)
					Required("value")
				})
				HTTP(func() {
					POST("/" + item.name)
					FormRequest()
					Body(func() {
						Attribute("value")
						Meta("openapi:encoding:value:style", "form")
						Meta("openapi:encoding:value:explode", item.explode)
					})
					Response(StatusNoContent)
				})
			})
		}
	})
}

func untaggedCompositeFormMethods(mapChoice, objectChoice expr.UserType) {
	for _, item := range []struct {
		name   string
		choice any
	}{
		{"Map", mapChoice}, {"Object", objectChoice},
	} {
		for _, jsonField := range []bool{true, false} {
			name := item.name
			if jsonField {
				name += "JSON"
			}
			Method(name, func() {
				Payload(func() {
					Attribute("value", item.choice)
					Required("value")
				})
				HTTP(func() {
					POST("/" + name)
					FormRequest()
					Body(func() {
						Attribute("value")
						if jsonField {
							Meta("openapi:encoding:value:contentType", "application/json")
						} else {
							Meta("openapi:encoding:value:style", "form")
							Meta("openapi:encoding:value:explode", "false")
						}
					})
					Response(StatusNoContent)
				})
			})
		}
	}
}

func taggedFormPayload(choice expr.UserType) expr.UserType {
	return Type("TaggedFormPayload", func() {
		Attribute("value", choice, func() {
			Meta("struct:tag:form", "jsonWire")
			Meta("struct:tag:json", "jsonValue")
		})
		Attribute("plain", String, func() {
			Meta("struct:tag:form", "wire", "omitempty")
			Meta("struct:tag:json", "jsonPlain")
		})
		Attribute("styled", choice, func() {
			Meta("struct:tag:form", "selection")
		})
		Attribute("numbers", ArrayOf(Int), func() {
			Meta("struct:tag:form", "nums")
		})
		Attribute("empty", String, func() {
			Meta("struct:tag:form", "")
		})
		Attribute("skip", String, func() {
			Meta("struct:tag:form", "-")
		})
		Attribute("jsonOnly", String, func() {
			Meta("struct:tag:json", "jsonOnly")
		})
		Attribute("default", String, func() {
			Meta("struct:tag:form", "default")
			Meta("struct:tag:json", "jsonDefault")
		})
		Required("value", "plain", "styled", "numbers", "default")
	})
}

func taggedFormMethod(tags expr.UserType) {
	Method("Tags", func() {
		Payload(tags)
		HTTP(func() {
			POST("/tags")
			FormRequest()
			Body(func() {
				for _, name := range []string{"value", "plain:shared", "styled", "numbers", "empty", "skip", "jsonOnly", "default:shared"} {
					Attribute(name)
				}
				Meta("openapi:encoding:value:contentType", "application/json")
				Meta("openapi:encoding:styled:style", "form")
				Meta("openapi:encoding:styled:explode", "false")
				Meta("openapi:encoding:numbers:style", "pipeDelimited")
				Meta("openapi:encoding:numbers:explode", "false")
			})
			Response(StatusNoContent)
		})
	})
}

func untaggedJSONFormMethods(choice, native expr.UserType) {
	Method("NullableJSONField", func() {
		Payload(func() {
			Attribute("value", choice, Nullable)
			Attribute("optional", choice, Nullable)
			Attribute("native", native)
			Required("value")
		})
		HTTP(func() {
			POST("/nullablejson")
			FormRequest()
			Body(func() {
				Attribute("value")
				Attribute("optional")
				Attribute("native")
				Meta("openapi:encoding:value:contentType", "application/json")
				Meta("openapi:encoding:optional:contentType", "application/json")
				Meta("openapi:encoding:native:contentType", "application/json")
			})
			Response(StatusNoContent)
		})
	})
	Method("JSONField", func() {
		Payload(func() {
			Attribute("value", choice)
			Attribute("owner", String)
			Required("value", "owner")
		})
		HTTP(func() {
			POST("/jsonfield")
			FormRequest()
			Body(func() {
				Attribute("value")
				Attribute("owner")
				Meta("openapi:encoding:value:contentType", "application/json")
			})
			Response(StatusNoContent)
		})
	})
}

// UntaggedFormHarness checks delimiter preservation, ambiguity, and publication.
const UntaggedFormHarness = `package formchoices_test

import (
 "fmt"
 "encoding/json/v2"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	client "example.com/formchoices/gen/http/formchoices/client"
	server "example.com/formchoices/gen/http/formchoices/server"
	svc "example.com/formchoices/gen/formchoices"
	loomhttp "github.com/CaliLuke/loom/http"
 loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

func TestFormUnionFields(t *testing.T) {
	for _, tc := range []struct {
		body string
		status int
	}{
		{"value=a%2Cb,c&owner=kept", http.StatusNoContent},
		{"value=42&owner=kept", http.StatusBadRequest},
		{"owner=kept", http.StatusBadRequest},
	} {
		t.Run(tc.body, func(t *testing.T) {
			calls := 0
			endpoints := &svc.Endpoints{Fields: func(_ context.Context, raw any) (any, error) {
				calls++
				payload := raw.(*svc.FieldsPayload)
				items, ok := payload.Value.AsList()
				require.True(t, ok)
				require.Equal(t, []string{"a,b", "c"}, []string(items))
				return nil, nil
			}}
			mux := loomhttp.NewMuxer()
			server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
			request := httptest.NewRequest(http.MethodPost, "/fields", strings.NewReader(tc.body))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			require.Equal(t, tc.status, response.Code)
			if tc.status == http.StatusNoContent {
				require.Equal(t, 1, calls)
			} else {
				require.Zero(t, calls)
				require.Contains(t, response.Body.String(), "decode_payload")
			}
		})
	}
}

func TestNullableJSONFormField(t *testing.T) {
 for _, tc := range []struct{body string; status int; null bool}{
  {"value=null", http.StatusNoContent, true},
  {"value=null&optional=null", http.StatusNoContent, true},
  {"value=null&native=null", http.StatusNoContent, true},
  {"value=null&native=true", http.StatusBadRequest, false},
  {"value=%5B%22a%22%5D", http.StatusNoContent, false},
  {"value=true", http.StatusBadRequest, false},
  {"value=", http.StatusBadRequest, false},
  {"value=+%09", http.StatusBadRequest, false},
  {"value=%5B", http.StatusBadRequest, false},
  {"value=%7D", http.StatusBadRequest, false},
  {"value=null&optional=%5B", http.StatusBadRequest, false},
  {"other=x", http.StatusBadRequest, false},
 } {
  t.Run(tc.body, func(t *testing.T) {
   calls := 0
   endpoints := &svc.Endpoints{NullableJSONField: func(_ context.Context, raw any) (any, error) {
    calls++
    payload := raw.(*svc.NullableJSONFieldPayload)
    value := payload.Value
    require.Equal(t, strings.Contains(tc.body, "native="), payload.Native.Kind() != "")
    require.Equal(t, strings.Contains(tc.body, "optional="), payload.Optional.Present())
    if payload.Optional.Present() { require.True(t, payload.Optional.IsNull()) }
    require.True(t, value.Present())
    require.Equal(t, tc.null, value.IsNull())
    if !tc.null {
     choice, ok := value.Value()
     require.True(t, ok)
     items, ok := choice.AsList()
     require.True(t, ok)
     require.Equal(t, []string{"a"}, []string(items))
    }
    return nil, nil
   }}
   mux := loomhttp.NewMuxer()
   server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
   request := httptest.NewRequest(http.MethodPost, "/nullablejson", strings.NewReader(tc.body))
   request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
   response := httptest.NewRecorder()
   mux.ServeHTTP(response, request)
   require.Equal(t, tc.status, response.Code, response.Body.String())
   if tc.status == http.StatusNoContent { require.Equal(t, 1, calls) } else {
    require.Zero(t, calls)
    if tc.body == "other=x" {
     require.Contains(t, response.Body.String(), "missing_field")
     require.Contains(t, response.Body.String(), "value")
    } else {
     require.Contains(t, response.Body.String(), "decode_payload")
     require.Contains(t, response.Body.String(), "invalid request body")
    }
   }
  })
 }
 payload := &svc.NullableJSONFieldPayload{Value: loom.NullValue[svc.FormChoice]()}
 transport := client.NewClient("http", "example.test", http.DefaultClient, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
 request, err := transport.BuildNullableJSONFieldRequest(context.Background(), payload)
 require.NoError(t, err)
 require.NoError(t, client.EncodeNullableJSONFieldRequest(loomhttp.RequestEncoder)(request, payload))
 require.NoError(t, request.ParseForm())
 require.Equal(t, "null", request.PostForm.Get("value"))
 require.NotContains(t, request.PostForm, "optional")
 require.NotContains(t, request.PostForm, "native")
 require.NoError(t, json.Unmarshal([]byte("null"), &payload.Native))
 payload.Optional.SetNull()
 request, err = transport.BuildNullableJSONFieldRequest(context.Background(), payload)
 require.NoError(t, err)
 require.NoError(t, client.EncodeNullableJSONFieldRequest(loomhttp.RequestEncoder)(request, payload))
 require.NoError(t, request.ParseForm())
 require.Equal(t, "null", request.PostForm.Get("optional"))
 require.Equal(t, "null", request.PostForm.Get("native"))
 payload.Value = loom.Nullable[svc.FormChoice]{}
 request, err = transport.BuildNullableJSONFieldRequest(context.Background(), payload)
 require.NoError(t, err)
 require.Error(t, client.EncodeNullableJSONFieldRequest(loomhttp.RequestEncoder)(request, payload))
}

func TestFormJSONUnionField(t *testing.T) {
	calls := 0
	endpoints := &svc.Endpoints{JSONField: func(_ context.Context, raw any) (any, error) {
		calls++
		payload := raw.(*svc.JSONFieldPayload)
		items, ok := payload.Value.AsList()
		require.True(t, ok)
		require.Equal(t, []string{"a,b", "c"}, []string(items))
		require.Equal(t, "kept", payload.Owner)
		return nil, nil
	}}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	request := httptest.NewRequest(http.MethodPost, "/jsonfield", strings.NewReader("value=%5B%22a%2Cb%22%2C%22c%22%5D&owner=kept"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())
	require.Equal(t, 1, calls)
	var value svc.FormChoice
	value.SetList([]string{"a,b", "c"})
	payload := &svc.JSONFieldPayload{Value: value, Owner: "kept"}
	transport := client.NewClient("http", "example.test", http.DefaultClient, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	encoded, err := transport.BuildJSONFieldRequest(context.Background(), payload)
	require.NoError(t, err)
	require.NoError(t, client.EncodeJSONFieldRequest(loomhttp.RequestEncoder)(encoded, payload))
	require.NoError(t, encoded.ParseForm())
	require.Equal(t, "[\"a,b\",\"c\"]", encoded.PostForm.Get("value"))
	require.Equal(t, "kept", encoded.PostForm.Get("owner"))
}

func TestFormSharedMapAndObjectLayouts(t *testing.T) {
 for _, tc := range []struct {path, body string}{
  {"Map", "value=a,7,b,9&owner=kept"},
  {"MapJSON", "value=%7B%22a%22%3A7%2C%22b%22%3A9%7D&owner=kept"},
  {"Object", "value=key,hello&owner=kept"},
  {"Object", "value=key,hello,note,provided&owner=kept"},
  {"ObjectJSON", "value=%7B%22key%22%3A%22hello%22%7D&owner=kept"},
 } {
  t.Run(tc.path+tc.body, func(t *testing.T) {
   calls := 0
   endpoint := func(_ context.Context, raw any) (any, error) {
    calls++
    switch payload := raw.(type) {
    case *svc.MapPayload:
     value, ok := payload.Value.AsMap()
     require.True(t, ok)
     require.Equal(t, map[string]int{"a": 7, "b": 9}, map[string]int(value))
    case *svc.MapJSONPayload:
     value, ok := payload.Value.AsMap()
     require.True(t, ok)
     require.Equal(t, map[string]int{"a": 7, "b": 9}, map[string]int(value))
    case *svc.ObjectPayload:
     value, ok := payload.Value.AsRecord()
     require.True(t, ok)
     require.Equal(t, "hello", value.Key)
     note := "fallback"
     if strings.Contains(tc.body, "provided") { note = "provided" }
     require.Equal(t, note, value.Note)
    case *svc.ObjectJSONPayload:
     value, ok := payload.Value.AsRecord()
     require.True(t, ok)
     require.Equal(t, "hello", value.Key)
     require.Equal(t, "fallback", value.Note)
    default:
     t.Errorf("unexpected payload %T", raw)
    }
    return nil, nil
   }
   endpoints := &svc.Endpoints{Map: endpoint, MapJSON: endpoint, Object: endpoint, ObjectJSON: endpoint}
   mux := loomhttp.NewMuxer()
   server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
   request := httptest.NewRequest(http.MethodPost, "/"+tc.path, strings.NewReader(tc.body))
   request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
   response := httptest.NewRecorder()
   mux.ServeHTTP(response, request)
   require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())
   require.Equal(t, 1, calls)
  })
 }
}

func TestFormUnionEncoder(t *testing.T) {
	transport := client.NewClient("http", "example.test", http.DefaultClient, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	for _, items := range [][]string{{"a,b", "c"}, {"42"}} {
		var value svc.FormChoice
		value.SetList(items)
		payload := &svc.FieldsPayload{Value: value}
		request, err := transport.BuildFieldsRequest(context.Background(), payload)
		require.NoError(t, err)
		err = client.EncodeFieldsRequest(loomhttp.RequestEncoder)(request, payload)
		if len(items) == 1 {
			require.Error(t, err)
			require.Nil(t, request.Body)
		} else {
			require.NoError(t, err)
			require.NoError(t, request.ParseForm())
			require.Equal(t, "a,b,c", request.PostForm.Get("value"))
		}
	}
}
func TestCapturedFormTags(t *testing.T) {
	var value, styled svc.FormChoice
	value.SetList([]string{"a,b", "c"})
	styled.SetList([]string{"a", "b"})
	ignored := "ignored"
	payload := &svc.TaggedFormPayload{Value: value, Plain: "hello", Styled: styled, Numbers: []int{1, 2}, Empty: &ignored, Skip: &ignored, JSONOnly: &ignored, Default: "kept"}
	transport := client.NewClient("http", "example.test", http.DefaultClient, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	request, err := transport.BuildTagsRequest(context.Background(), payload)
	require.NoError(t, err)
	require.NoError(t, client.EncodeTagsRequest(loomhttp.RequestEncoder)(request, payload))
	require.NoError(t, request.ParseForm())
	require.Equal(t, 5, len(request.PostForm))
	require.Equal(t, "hello", request.PostForm.Get("wire"))
	require.JSONEq(t, "[\"a,b\",\"c\"]", request.PostForm.Get("jsonWire"))
	require.Equal(t, "a,b", request.PostForm.Get("selection"))
	require.Equal(t, "1|2", request.PostForm.Get("nums"))
	require.Equal(t, "kept", request.PostForm.Get("default"))
	calls := 0
	mux := loomhttp.NewMuxer()
	endpoint := func(_ context.Context, raw any) (any, error) {
		calls++
		actual := raw.(*svc.TaggedFormPayload)
		require.Equal(t, "hello", actual.Plain)
		require.Equal(t, []int{1, 2}, actual.Numbers)
		require.Nil(t, actual.Empty)
		require.Nil(t, actual.Skip)
		require.Nil(t, actual.JSONOnly)
		return nil, nil
	}
	server.Mount(mux, server.New(&svc.Endpoints{Tags: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	for _, body := range []string{request.PostForm.Encode(), "value=%5B%5D&plain=hello&styled=a,b&numbers=1%7C2&default=kept"} {
		req := httptest.NewRequest(http.MethodPost, "/tags", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		if strings.Contains(body, "jsonWire=") {
			require.Equal(t, 204, response.Code, response.Body.String())
		} else {
			require.Equal(t, 400, response.Code, response.Body.String())
		}
	}
	require.Equal(t, 1, calls)
}

` + untaggedFormRootsHarness + untaggedFormContentHarness + ordinaryFormPresenceHarness
