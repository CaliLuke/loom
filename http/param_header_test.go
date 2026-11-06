package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHeaderParamCodec(t *testing.T) {
	for _, tc := range []struct {
		name    string
		value   ParamValue
		explode bool
		wire    string
	}{
		{"list", ParamValue{Kind: ParamList, Present: true, Items: []string{"blue", "black"}}, false, "blue, black"},
		{"object", ParamValue{Kind: ParamPairs, Present: true, Pairs: []ParamPair{{Key: "R", Value: "100"}, {Key: "G", Value: "200"}}}, false, "R, 100, G, 200"},
		{"exploded", ParamValue{Kind: ParamPairs, Present: true, Pairs: []ParamPair{{Key: "R", Value: "100"}, {Key: "G", Value: "200"}}}, true, "R=100, G=200"},
		{"empty value", ParamValue{Kind: ParamPairs, Present: true, Pairs: []ParamPair{{Key: "R", Value: ""}}}, true, "R="},
		{"leading value space", ParamValue{Kind: ParamPairs, Present: true, Pairs: []ParamPair{{Key: "R", Value: " 100"}}}, true, "R= 100"},
		{"trailing key space", ParamValue{Kind: ParamPairs, Present: true, Pairs: []ParamPair{{Key: "R ", Value: "100"}}}, true, "R =100"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			codec := ParamCodec{Style: ParamSimple, Explode: tc.explode}
			wire, err := EncodeHeaderParam("X-Value", codec, tc.value)
			require.NoError(t, err)
			require.Equal(t, tc.wire, wire)
			got, err := DecodeHeaderParam("X-Value", []string{", " + wire + ", "}, true, codec, tc.value.Kind)
			require.NoError(t, err)
			require.Equal(t, tc.value, got)
		})
	}
	for _, value := range []string{"", "a,b", " a", "a "} {
		_, err := EncodeHeaderParam("X-Value", ParamCodec{Style: ParamSimple}, ParamValue{Kind: ParamList, Present: true, Items: []string{value}})
		require.ErrorContains(t, err, "X-Value")
	}
	for _, pair := range []ParamPair{{Key: "a=b", Value: "x"}, {Key: " a", Value: "x"}, {Key: "a", Value: "x "}, {Key: "a", Value: "x,y"}} {
		_, err := EncodeHeaderParam("X-Value", ParamCodec{Style: ParamSimple, Explode: true}, ParamValue{Kind: ParamPairs, Present: true, Pairs: []ParamPair{pair}})
		require.ErrorContains(t, err, "X-Value")
	}
	for _, raw := range []string{"R,,G,200", ",R,100,G", "R,1,R,2"} {
		_, err := DecodeHeaderParam("X-Value", []string{raw}, true, ParamCodec{Style: ParamSimple}, ParamPairs)
		require.ErrorContains(t, err, "X-Value")
	}
}

func TestHeaderParamHTTPFraming(t *testing.T) {
	for _, tc := range []struct {
		name     string
		value    ParamValue
		explode  bool
		accepted bool
	}{
		{"scalar empty", ParamValue{Kind: ParamScalar, Present: true}, false, true},
		{"scalar internal space", ParamValue{Kind: ParamScalar, Present: true, Text: "hello world"}, false, true},
		{"scalar internal tab", ParamValue{Kind: ParamScalar, Present: true, Text: "hello\tworld"}, false, true},
		{"scalar quoted space", ParamValue{Kind: ParamScalar, Present: true, Text: `" hello "`}, false, true},
		{"scalar non-OWS Unicode", ParamValue{Kind: ParamScalar, Present: true, Text: "\u00a0hello\u00a0"}, false, true},
		{"scalar space", ParamValue{Kind: ParamScalar, Present: true, Text: " "}, false, false},
		{"scalar leading space", ParamValue{Kind: ParamScalar, Present: true, Text: " hello"}, false, false},
		{"scalar trailing space", ParamValue{Kind: ParamScalar, Present: true, Text: "hello "}, false, false},
		{"scalar leading tab", ParamValue{Kind: ParamScalar, Present: true, Text: "\thello"}, false, false},
		{"scalar trailing tab", ParamValue{Kind: ParamScalar, Present: true, Text: "hello\t"}, false, false},
		{"list internal spaces", ParamValue{Kind: ParamList, Present: true, Items: []string{"hello world", `" next "`}}, false, true},
		{"list leading space", ParamValue{Kind: ParamList, Present: true, Items: []string{" hello", "world"}}, false, false},
		{"list trailing tab", ParamValue{Kind: ParamList, Present: true, Items: []string{"hello", "world\t"}}, false, false},
		{"object interior spaces", ParamValue{Kind: ParamPairs, Present: true, Pairs: []ParamPair{{Key: "key name", Value: "some value"}}}, false, true},
		{"object value edge space", ParamValue{Kind: ParamPairs, Present: true, Pairs: []ParamPair{{Key: "key", Value: " value"}}}, false, false},
		{"exploded interior spaces", ParamValue{Kind: ParamPairs, Present: true, Pairs: []ParamPair{{Key: "key ", Value: " value"}}}, true, true},
		{"exploded key edge space", ParamValue{Kind: ParamPairs, Present: true, Pairs: []ParamPair{{Key: " key", Value: "value"}}}, true, false},
		{"exploded value edge tab", ParamValue{Kind: ParamPairs, Present: true, Pairs: []ParamPair{{Key: "key", Value: "value\t"}}}, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			codec := ParamCodec{Style: ParamSimple, Explode: tc.explode}
			wire, err := EncodeHeaderParam("X-Value", codec, tc.value)
			if !tc.accepted {
				require.ErrorContains(t, err, "X-Value")
				require.Empty(t, wire)
				return
			}
			require.NoError(t, err)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header()["X-Value"] = r.Header.Values("X-Value")
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()
			request, err := http.NewRequest(http.MethodGet, server.URL, nil)
			require.NoError(t, err)
			request.Header.Set("X-Value", wire)
			response, err := server.Client().Do(request)
			require.NoError(t, err)
			_, err = io.Copy(io.Discard, response.Body)
			require.NoError(t, err)
			require.NoError(t, response.Body.Close())
			lines, present := response.Header["X-Value"]
			got, err := DecodeHeaderParam("X-Value", lines, present, codec, tc.value.Kind)
			require.NoError(t, err)
			require.Equal(t, tc.value, got)
		})
	}
}

func TestHeaderParamScalarWhitespaceIsLostOnWire(t *testing.T) {
	for _, value := range []string{" hello ", "\thello\t", " ", "\t"} {
		t.Run(value, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header()["X-Value"] = r.Header.Values("X-Value")
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()
			request, err := http.NewRequest(http.MethodGet, server.URL, nil)
			require.NoError(t, err)
			request.Header.Set("X-Value", value)
			response, err := server.Client().Do(request)
			require.NoError(t, err)
			require.NoError(t, response.Body.Close())
			require.Equal(t, strings.Trim(value, " \t"), response.Header.Get("X-Value"))
			require.NotEqual(t, value, response.Header.Get("X-Value"))
		})
	}
}
