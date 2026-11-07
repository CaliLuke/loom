package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// UntaggedLocationsDSL exercises union matching over raw parameter carriers.
func UntaggedLocationsDSL() {
	untaggedLocationsDSL("3.2")
}

// UntaggedLocations31DSL uses explicit non-exploded form cookies, whose wire
// grammar is representable in OpenAPI 3.1, with the same union shapes.
func UntaggedLocations31DSL() {
	untaggedLocationsDSL("3.1")
}

func untaggedLocationsDSL(version string) {
	API("textunion", func() {
		Meta("openapi:version", version)
	})
	cookieStyle, cookieExplode := "cookie", true
	if version == "3.1" {
		cookieStyle, cookieExplode = "form", false
	}
	scalar, list, widths, data, errorRecord, objects, mapping, deep := untaggedLocationTypes()
	byteChoice, nested, nestedWidth, nestedList, nestedIdentity := untaggedNestedLocationTypes(widths, list)
	closedMethods := untaggedClosedLocationMethods(data, errorRecord, cookieStyle, cookieExplode)
	Service("locations", func() {
		closedMethods()
		parameterBindingMethod(parameterBinding{"QueryNestedList", "query", "form", nestedList, true, true})
		parameterBindingMethod(parameterBinding{"CookieList", "cookie", cookieStyle, list, cookieExplode, true})
		parameterBindingMethod(parameterBinding{"QueryNestedIdentity", "query", "form", nestedIdentity, true, true})
		parameterBindingMethod(parameterBinding{"QueryNested", "query", "form", nested, true, true})
		parameterBindingMethod(parameterBinding{"QueryNestedWidth", "query", "form", nestedWidth, true, true})
		parameterBindingMethod(parameterBinding{"QueryBytes", "query", "form", byteChoice, true, true})
		parameterBindingMethod(parameterBinding{"QueryResidual", "query", "form", mapping, true, true})
		parameterBindingMethod(parameterBinding{"QueryDeep", "query", "deepObject", deep, true, true})
		parameterBindingMethod(parameterBinding{"CookieMap", "cookie", cookieStyle, mapping, cookieExplode, true})
		parameterBindingMethod(parameterBinding{"HeaderList", "header", "simple", list, false, true})
		parameterBindingMethod(parameterBinding{"QueryOptional", "query", "form", list, true, false})
		parameterBindingMethod(parameterBinding{"PathList", "path", "label", list, true, true})
		parameterBindingMethod(parameterBinding{"HeaderScalar", "header", "simple", scalar, false, true})
		parameterBindingMethod(parameterBinding{"CookieScalar", "cookie", cookieStyle, scalar, cookieExplode, true})
		parameterBindingMethod(parameterBinding{"QueryList", "query", "form", list, false, true})
		parameterBindingMethod(parameterBinding{"QueryWidth", "query", "form", widths, true, true})
		parameterBindingMethod(parameterBinding{"QueryObjects", "query", "form", objects, true, true})
		untaggedLocationResponseMethods(objects, scalar)
	})
}

func untaggedLocationTypes() (expr.UserType, expr.UserType, expr.UserType, expr.UserType, expr.UserType, expr.UserType, expr.UserType, expr.UserType) {
	scalar := Type("TextChoice", OneOf(String, Int), Untagged)
	list := Type("ListChoice", OneOf(func() {
		Untagged()
		Attribute("Number", Int)
		Attribute("List", ArrayOf(String))
	}))
	widths := Type("WidthChoice", OneOf(Int32, Int64), Untagged)
	data := Type("DataRecord", func() {
		Attribute("data", String, func() {
			Meta("struct:tag:json", "json_data")
		})
		Required("data")
	})
	errorRecord := Type("ErrorRecord", func() {
		Attribute("error", String)
		Required("error")
	})
	objects := Type("ObjectChoice", OneOf(data, errorRecord), Untagged)
	mapping := Type("MapChoice", OneOf(func() {
		Untagged()
		Attribute("Text", String)
		Attribute("Map", MapOf(String, Int))
	}))
	deep := Type("DeepChoice", OneOf(func() {
		Untagged()
		Attribute("Data", data)
		Attribute("Map", MapOf(String, Int))
	}))
	return scalar, list, widths, data, errorRecord, objects, mapping, deep
}

func untaggedNestedLocationTypes(widths, list expr.UserType) (expr.UserType, expr.UserType, expr.UserType, expr.UserType, expr.UserType) {
	byteCode := Type("ByteCode", Bytes, func() {
		Enum("é", []byte{0xff, 0})
		MinLength(2)
		MaxLength(2)
	})
	byteChoice := Type("ByteChoice", OneOf(byteCode, Int), Untagged)
	nestedInner := Type("NestedInner", OneOf(Int, Boolean), Untagged)
	letters := Type("Letters", String, func() {
		Pattern("^[a-z]+$")
	})
	nested := Type("NestedChoice", OneOf(nestedInner, letters), Untagged)
	nestedWidth := Type("NestedWidth", OneOf(widths, String), Untagged)
	nestedList := Type("NestedList", OneOf(list, Boolean), Untagged)
	digits := Type("Digits", String, func() {
		Pattern("^[0-9]+$")
	})
	guarded := Type("GuardedInner", OneOf(digits, Boolean), Untagged)
	outerText := Type("OuterText", String, func() {
		Enum("outer")
	})
	nestedIdentity := Type("NestedIdentity", OneOf(guarded, outerText), Untagged)
	return byteChoice, nested, nestedWidth, nestedList, nestedIdentity
}

func untaggedLocationResponseMethods(objects, scalar expr.UserType) {
	Method("ResponseObject", func() {
		Result(func() {
			Attribute("value", objects)
			Required("value")
		})
		HTTP(func() {
			GET("/ResponseObject")
			Response(StatusNoContent, func() {
				Header("value:X-Value")
			})
		})
	})
	Method("ResponseUnion", func() {
		Result(func() {
			Attribute("value", scalar)
			Required("value")
		})
		HTTP(func() {
			GET("/ResponseUnion")
			Response(StatusNoContent, func() {
				Header("value:X-Value")
			})
		})
	})
	Method("query", func() {
		Payload(func() {
			Attribute("value", scalar)
			Required("value")
		})
		HTTP(func() {
			GET("/query")
			Param("value:n")
			Response(StatusNoContent)
		})
	})
}

// UntaggedLocationsHarness checks generated raw text matching at the wire.
const UntaggedLocationsHarness = `package textunion_test

import (
	"context"
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	client "example.com/textunion/gen/http/locations/client"
	server "example.com/textunion/gen/http/locations/server"
	svc "example.com/textunion/gen/locations"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

func TestQueryTextUnion(t *testing.T) {
	for _, tc := range []struct {
		name, path, want, raw string
		status int
	}{
		{"closed-object-owned-keys", "/QueryClosedObjects?data=x&outside=ignored", "{\"json_data\":\"x\"}", "", http.StatusNoContent},
		{"closed-object-ambiguous", "/QueryClosedObjects?data=x&error=y", "", "", http.StatusBadRequest},
		{"closed-map-excludes-sibling", "/QueryClosedMap?x=1&control=skip", "{\"x\":1}", "", http.StatusNoContent},
		{"closed-map-counts-alternative-keys", "/QueryClosedMap?value=42&control=skip", "", "", http.StatusBadRequest},
		{"closed-nested-map", "/QueryClosedNested?x=1", "{\"x\":1}", "", http.StatusNoContent},
		{"closed-nested-ambiguous", "/QueryClosedNested?value=true", "", "", http.StatusBadRequest},
		{"closed-header", "/HeaderClosedObjects", "{\"json_data\":\"x\"}", "data,x", http.StatusNoContent},
		{"closed-cookie", "/CookieClosedObjects", "{\"json_data\":\"x\"}", "data=x", http.StatusNoContent},
		{"closed-path", "/PathClosedObjects/data,x", "{\"json_data\":\"x\"}", "", http.StatusNoContent},
		{"nested-number", "/QueryNested?value=3", "3", "", http.StatusNoContent},
		{"nested-outer-string", "/QueryNested?value=hello", "\"hello\"", "", http.StatusNoContent},
		{"nested-outer-ambiguous", "/QueryNested?value=true", "", "", http.StatusBadRequest},
		{"nested-width-other-branch", "/QueryNestedWidth?value=2147483648", "\"2147483648\"", "", http.StatusNoContent},
		{"query-string-int", "/query?n=hello", "\"hello\"", "", http.StatusNoContent},
		{"query-42-ambiguous", "/query?n=42", "", "", http.StatusBadRequest},
		{"query-empty-string", "/query?n=", "\"\"", "", http.StatusNoContent},
		{"query-raw-comma-split", "/QueryList?value=a%2Cb,c", "[\"a,b\",\"c\"]", "", http.StatusNoContent},
		{"query-width-split", "/QueryWidth?value=2147483648", "", "", http.StatusBadRequest},
		{"query-flat-objects-probe", "/QueryObjects?data=x", "{\"json_data\":\"x\"}", "", http.StatusNoContent},
		{"query-flat-objects-ambiguous", "/QueryObjects?data=x&error=y", "", "", http.StatusBadRequest},
		{"header-string-int", "/HeaderScalar", "\"hello\"", "hello", http.StatusNoContent},
		{"header-ambiguous", "/HeaderScalar", "", "42", http.StatusBadRequest},
		{"cookie-string-int", "/CookieScalar", "\"hello\"", "value=hello", http.StatusNoContent},
		{"cookie-ambiguous", "/CookieScalar", "", "value=42", http.StatusBadRequest},
		{"path-label-exploded-array", "/PathList/.a.b", "[\"a\",\"b\"]", "", http.StatusNoContent},
		{"path-label-ambiguous", "/PathList/.42", "", "", http.StatusBadRequest},
		{"rule-a-optional-empty", "/QueryOptional", "null", "", http.StatusNoContent},
		{"query-residual-map-with-sibling", "/QueryResidual?x=1&control=skip", "{\"x\":1}", "", http.StatusNoContent},
		{"query-residual-shared-keys-ambiguous", "/QueryResidual?value=42&control=skip", "", "", http.StatusBadRequest},
		{"query-deepobject-object", "/QueryDeep?value[data]=x", "{\"json_data\":\"x\"}", "", http.StatusNoContent},
		{"query-deepobject-map", "/QueryDeep?value[x]=1", "{\"x\":1}", "", http.StatusNoContent},
		{"header-int-or-list", "/HeaderList", "[\"a\",\"b\"]", "a,b", http.StatusNoContent},
		{"cookie-string-or-map", "/CookieMap", "{\"x\":1}", "x=1; sid=skip", http.StatusNoContent},
		{"query-bytes-raw-text", "/QueryBytes?value=%C3%A9", "\"w6k=\"", "", http.StatusNoContent},
		{"query-bytes-binary", "/QueryBytes?value=%FF%00", "\"/wA=\"", "", http.StatusNoContent},
		{"query-bytes-reject-base64", "/QueryBytes?value=w6k%3D", "", "", http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			endpoints := &svc.Endpoints{}
			fields := reflect.ValueOf(endpoints).Elem()
			for index := range fields.NumField() {
				endpoint := loom.Endpoint(func(_ context.Context, payload any) (any, error) {
					calls++
					value := reflect.ValueOf(payload).Elem().FieldByName("Value").Interface()
					wire, err := json.Marshal(value)
					require.NoError(t, err)
					require.JSONEq(t, tc.want, string(wire))
					return nil, nil
				})
				fields.Field(index).Set(reflect.ValueOf(endpoint))
			}
			mux := loomhttp.NewMuxer()
			server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			if strings.HasPrefix(tc.path, "/Header") {
				req.Header.Set("X-Value", tc.raw)
			}
			if strings.HasPrefix(tc.path, "/Cookie") {
				req.Header.Set("Cookie", tc.raw)
			}
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, req)
			require.Equal(t, tc.status, response.Code)
			body, err := io.ReadAll(response.Result().Body)
			require.NoError(t, err)
			if tc.status == http.StatusNoContent {
				require.Equal(t, 1, calls)
			} else {
				require.Zero(t, calls)
				require.Contains(t, string(body), "decode_payload")
			}
		})
	}
}

func TestTextUnionClientRechecksWire(t *testing.T) {
	for _, tc := range []struct { value string; rejected bool }{
		{"hello", false}, {"42", true}, {"", false},
	} {
		var choice svc.TextChoice
		choice.SetString(tc.value)
		transport := client.NewClient("http", "example.test", http.DefaultClient, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
		req, err := transport.BuildQueryRequest(context.Background(), &svc.QueryPayload{Value: choice})
		require.NoError(t, err)
		encode := client.EncodeQueryRequest(loomhttp.RequestEncoder)
		err = encode(req, &svc.QueryPayload{Value: choice})
		if tc.rejected {
			require.Error(t, err)
			require.Empty(t, req.URL.RawQuery)
		} else {
			require.NoError(t, err)
			require.Equal(t, tc.value, req.URL.Query().Get("n"))
		}
	}
}

func TestCLIParsesTextUnionFlag(t *testing.T) {
	value, err := client.BuildQueryPayload("hello")
	require.NoError(t, err)
	text, ok := value.Value.AsString()
	require.True(t, ok)
	require.Equal(t, "hello", text)
	_, err = client.BuildQueryPayload("42")
	require.ErrorContains(t, err, "matched 2 branches in schema")
	value, err = client.BuildQueryPayload("hello&peer=1")
	require.NoError(t, err)
	text, ok = value.Value.AsString()
	require.True(t, ok)
	require.Equal(t, "hello&peer=1", text)
	object, err := client.BuildQueryObjectsPayload("data=x")
	require.NoError(t, err)
	data, ok := object.Value.AsDataRecord()
	require.True(t, ok)
	require.Equal(t, "x", data.Data)
	list, err := client.BuildQueryOptionalPayload("value=a&value=b")
	require.NoError(t, err)
	items, ok := list.Value.AsList()
	require.True(t, ok)
	require.Equal(t, []string{"a", "b"}, []string(items))
	cookie, err := client.BuildCookieMapPayload("x=1; sid=session", "session")
	require.NoError(t, err)
	mapping, ok := cookie.Value.AsMap()
	require.True(t, ok)
	require.Equal(t, map[string]int{"x": 1}, map[string]int(mapping))
	nested, err := client.BuildQueryNestedListPayload("value=a&value=b")
	require.NoError(t, err)
	inner, ok := nested.Value.AsListChoice()
	require.True(t, ok)
	items, ok = inner.AsList()
	require.True(t, ok)
	require.Equal(t, []string{"a", "b"}, []string(items))
	cookieList, err := client.BuildCookieListPayload("value=a; value=b")
	require.NoError(t, err)
	items, ok = cookieList.Value.AsList()
	require.True(t, ok)
	require.Equal(t, []string{"a", "b"}, []string(items))
	_, err = client.BuildCookieMapPayload("x=\"unterminated", "session")
	require.Error(t, err)
	_, err = client.BuildCookieScalarPayload("bad;cookie")
	require.Error(t, err)
}

func TestOptionalTextUnionPresence(t *testing.T) {
	var empty svc.ListChoice
	empty.SetList([]string{})
	transport := client.NewClient("http", "example.test", http.DefaultClient, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	for _, value := range []*svc.ListChoice{nil, &empty} {
		payload := &svc.QueryOptionalPayload{Value: value}
		req, err := transport.BuildQueryOptionalRequest(context.Background(), payload)
		require.NoError(t, err)
		require.NoError(t, client.EncodeQueryOptionalRequest(loomhttp.RequestEncoder)(req, payload))
		require.Empty(t, req.URL.RawQuery)
	}
}

func TestRequestHeaderUnionRejectsEdgeWhitespace(t *testing.T) {
 transport := client.NewClient("http", "example.test", http.DefaultClient, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
 for _, text := range []string{" leading", "trailing\t", "\t"} {
  var value svc.TextChoice
  value.SetString(text)
  payload := &svc.HeaderScalarPayload{Value: value}
  request, err := transport.BuildHeaderScalarRequest(context.Background(), payload)
  require.NoError(t, err)
  err = client.EncodeHeaderScalarRequest(loomhttp.RequestEncoder)(request, payload)
  require.Error(t, err)
  require.Empty(t, request.Header.Get("X-Value"))
 }
}

func TestResponseHeaderUnionRechecksWire(t *testing.T) {
	for _, tc := range []struct{ value string; status int }{
		{"hello", http.StatusNoContent}, {"42", http.StatusInternalServerError},
		{" leading", http.StatusInternalServerError}, {"trailing\t", http.StatusInternalServerError},
		{"in\tterior", http.StatusNoContent},
	} {
		calls := 0
		endpoints := &svc.Endpoints{ResponseUnion: func(context.Context, any) (any, error) {
			calls++
			var value svc.TextChoice
			value.SetString(tc.value)
			return &svc.ResponseUnionResult{Value: value}, nil
		}}
		mux := loomhttp.NewMuxer()
		server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ResponseUnion", nil))
		require.Equal(t, 1, calls)
		require.Equal(t, tc.status, response.Code)
		if tc.status == http.StatusNoContent {
			require.Equal(t, tc.value, response.Header().Get("X-Value"))
			decoded, err := client.DecodeResponseUnionResponse(loomhttp.ResponseDecoder, false)(response.Result())
			require.NoError(t, err)
			actual, ok := decoded.(*svc.ResponseUnionResult).Value.AsString()
			require.True(t, ok)
			require.Equal(t, tc.value, actual)
		} else {
			require.Empty(t, response.Header().Get("X-Value"))
		}
	}
}

func TestNestedTextUnionClient(t *testing.T) {
	transport := client.NewClient("http", "example.test", http.DefaultClient, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	for _, rejected := range []bool{false, true} {
		inner := new(svc.GuardedInner)
		if rejected {
			inner.SetDigits(svc.Digits("true"))
		} else {
			inner.SetBoolean(true)
		}
		var value svc.NestedIdentity
		value.SetGuardedInner(inner)
		payload := &svc.QueryNestedIdentityPayload{Value: value}
		req, err := transport.BuildQueryNestedIdentityRequest(context.Background(), payload)
		require.NoError(t, err)
		err = client.EncodeQueryNestedIdentityRequest(loomhttp.RequestEncoder)(req, payload)
		if rejected {
			require.ErrorContains(t, err, "changed selected branch")
			require.Empty(t, req.URL.RawQuery)
		} else {
			require.NoError(t, err)
			require.Equal(t, "true", req.URL.Query().Get("value"))
		}
	}
}

func TestResponseObjectUnion(t *testing.T) {
	endpoints := &svc.Endpoints{ResponseObject: func(context.Context, any) (any, error) {
		var value svc.ObjectChoice
		value.SetDataRecord(&svc.DataRecord{Data: "x"})
		return &svc.ResponseObjectResult{Value: value}, nil
	}}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ResponseObject", nil))
	require.Equal(t, http.StatusNoContent, response.Code)
	require.Equal(t, "data, x", response.Header().Get("X-Value"))
	decoded, err := client.DecodeResponseObjectResponse(loomhttp.ResponseDecoder, false)(response.Result())
	require.NoError(t, err)
	actual, ok := decoded.(*svc.ResponseObjectResult).Value.AsDataRecord()
	require.True(t, ok)
	require.Equal(t, "x", actual.Data)
}

func TestByteTextWire(t *testing.T) {
	for _, data := range [][]byte{[]byte("é"), {0xff, 0}} {
		var value svc.ByteChoice
		value.SetByteCode(svc.ByteCode(data))
		payload := &svc.QueryBytesPayload{Value: value}
		transport := client.NewClient("http", "example.test", http.DefaultClient, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
		req, err := transport.BuildQueryBytesRequest(context.Background(), payload)
		require.NoError(t, err)
		require.NoError(t, client.EncodeQueryBytesRequest(loomhttp.RequestEncoder)(req, payload))
		require.Equal(t, data, []byte(req.URL.Query().Get("value")))
	}
}
` + untaggedClosureLocationsHarness
