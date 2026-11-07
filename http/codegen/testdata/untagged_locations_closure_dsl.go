package testdata

import . "github.com/CaliLuke/loom/dsl"

func untaggedClosedLocationMethods(data, failure any, cookieStyle string, cookieExplode bool) func() {
	objects := Type("ClosedObjectChoice", OneOf(data, failure), func() {
		Untagged()
		Meta("openapi:unevaluatedProperties", "false")
	})
	mapping := Type("ClosedMapChoice", OneOf(func() {
		Untagged()
		Attribute("Text", String)
		Attribute("Map", MapOf(String, Int))
		Meta("openapi:unevaluatedProperties", "false")
	}))
	nested := Type("NestedClosedMapChoice", OneOf(mapping, Boolean), func() {
		Untagged()
		Meta("openapi:unevaluatedProperties", "false")
	})
	return func() {
		for _, binding := range []parameterBinding{
			{"QueryClosedObjects", "query", "form", objects, true, true},
			{"QueryClosedNested", "query", "form", nested, true, true},
			{"HeaderClosedObjects", "header", "simple", objects, false, true},
			{"CookieClosedObjects", "cookie", cookieStyle, objects, cookieExplode, true},
			{"PathClosedObjects", "path", "simple", objects, false, true},
		} {
			parameterBindingMethod(binding)
		}
		Method("QueryClosedMap", func() {
			Payload(func() {
				Attribute("value", mapping)
				Attribute("control", String)
				Required("value", "control")
			})
			HTTP(func() {
				GET("/QueryClosedMap")
				Param("value", func() {
					Meta("openapi:style", "form")
					Meta("openapi:explode", "true")
				})
				Param("control")
				Response(StatusNoContent)
			})
		})
		Method("ResponseClosedObject", func() {
			Result(func() {
				Attribute("value", objects)
				Required("value")
			})
			HTTP(func() {
				GET("/ResponseClosedObject")
				Response(StatusNoContent, func() {
					Header("value:X-Value")
				})
			})
		})
	}
}

const untaggedClosureLocationsHarness = `
func TestClosedTextUnionClientAndCLI(t *testing.T) {
 value, err := client.BuildQueryClosedObjectsPayload("data=x&outside=ignored")
 require.NoError(t, err)
 object, ok := value.Value.AsDataRecord()
 require.True(t, ok)
 require.Equal(t, "x", object.Data)
 _, err = client.BuildQueryClosedObjectsPayload("data=x&error=y")
 require.ErrorContains(t, err, "matched 2 branches in schema")
 transport := client.NewClient("http", "example.test", http.DefaultClient, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
 request, err := transport.BuildQueryClosedObjectsRequest(t.Context(), value)
 require.NoError(t, err)
 require.NoError(t, client.EncodeQueryClosedObjectsRequest(loomhttp.RequestEncoder)(request, value))
 require.Equal(t, "data=x", request.URL.RawQuery)
 mapping, err := client.BuildQueryClosedMapPayload("x=1&control=skip", "skip")
 require.NoError(t, err)
 request, err = transport.BuildQueryClosedMapRequest(t.Context(), mapping)
 require.NoError(t, err)
 require.NoError(t, client.EncodeQueryClosedMapRequest(loomhttp.RequestEncoder)(request, mapping))
 require.Equal(t, "1", request.URL.Query().Get("x"))
 require.Equal(t, "skip", request.URL.Query().Get("control"))
}

func TestClosedResponseTextUnion(t *testing.T) {
 endpoints := &svc.Endpoints{ResponseClosedObject: func(context.Context, any) (any, error) {
  var value svc.ClosedObjectChoice
  value.SetDataRecord(&svc.DataRecord{Data: "x"})
  return &svc.ResponseClosedObjectResult{Value: value}, nil
 }}
 mux := loomhttp.NewMuxer()
 server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
 response := httptest.NewRecorder()
 mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ResponseClosedObject", nil))
 require.Equal(t, http.StatusNoContent, response.Code)
 require.Equal(t, "data, x", response.Header().Get("X-Value"))
 decoded, err := client.DecodeResponseClosedObjectResponse(loomhttp.ResponseDecoder, false)(response.Result())
 require.NoError(t, err)
 actual, ok := decoded.(*svc.ResponseClosedObjectResult).Value.AsDataRecord()
 require.True(t, ok)
 require.Equal(t, "x", actual.Data)
}
`
