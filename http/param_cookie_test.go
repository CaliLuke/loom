package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCookieParamCodec(t *testing.T) {
	for _, tc := range []struct {
		name  string
		codec ParamCodec
		value ParamValue
		wire  string
	}{
		{"scalar", ParamCodec{Style: ParamCookie}, ParamValue{Kind: ParamScalar, Present: true, Text: "a/b"}, "color=a/b"},
		{"space", ParamCodec{Style: ParamCookie}, ParamValue{Kind: ParamScalar, Present: true, Text: "hello world"}, "color=hello world"},
		{"list", ParamCodec{Style: ParamCookie}, ParamValue{Kind: ParamList, Present: true, Items: []string{"blue", "black"}}, "color=blue,black"},
		{"expanded", ParamCodec{Style: ParamCookie, Explode: true}, ParamValue{Kind: ParamList, Present: true, Items: []string{"blue", "black"}}, "color=blue; color=black"},
		{"object", ParamCodec{Style: ParamCookie}, ParamValue{Kind: ParamPairs, Present: true, Pairs: []ParamPair{{Key: "R", Value: "100"}}}, "color=R,100"},
		{"expanded object", ParamCodec{Style: ParamCookie, Explode: true}, ParamValue{Kind: ParamPairs, Present: true, Pairs: []ParamPair{{Key: "R", Value: "100"}}}, "R=100"},
		{"form", ParamCodec{Style: ParamForm}, ParamValue{Kind: ParamList, Present: true, Items: []string{"a/b", "a,b", "a b"}}, "color=a%2Fb,a%2Cb,a%20b"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Cookie", "sid=kept")
			ns := ParamNamespace{Literals: []string{"sid"}}
			require.NoError(t, AddRequestCookieParam(req, "color", tc.codec, tc.value, ns))
			require.Equal(t, "sid=kept; "+tc.wire, req.Header.Get("Cookie"))
			got, err := DecodeCookieParam(req.Cookies(), "color", tc.codec, tc.value.Kind, ns)
			require.NoError(t, err)
			require.Equal(t, tc.value, got)
			for _, pair := range strings.Split(req.Header.Get("Cookie"), "; ") {
				require.NotContains(t, pair, `"`, "verbatim peer must not receive quoted strings")
			}
		})
	}
	for _, value := range []string{"a,b", `a"b`, `a\b`, "café", " leading", "trailing "} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Add("Cookie", "sid=kept")
		before := req.Header.Clone()
		err := AddRequestCookieParam(req, "color", ParamCodec{Style: ParamCookie}, ParamValue{Kind: ParamScalar, Present: true, Text: value}, ParamNamespace{})
		require.ErrorContains(t, err, "color")
		require.Equal(t, before, req.Header)
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Cookie", `color="hello world"; sid=kept; theme=dark; theme=ignored`)
	got, err := DecodeCookieParam(req.Cookies(), "color", ParamCodec{Style: ParamCookie}, ParamScalar, ParamNamespace{})
	require.NoError(t, err)
	require.Equal(t, "hello world", got.Text)
	got, err = DecodeCookieParam(req.Cookies(), "prefs", ParamCodec{Style: ParamCookie, Explode: true}, ParamPairs, ParamNamespace{Literals: []string{"sid", "color"}})
	require.NoError(t, err)
	require.Equal(t, []ParamPair{{Key: "theme", Value: "dark"}}, got.Pairs)
	before := req.Header.Clone()
	err = AddRequestCookieParam(req, "prefs", ParamCodec{Style: ParamCookie, Explode: true}, ParamValue{Kind: ParamPairs, Present: true, Pairs: []ParamPair{{Key: "sid", Value: "secret"}}}, ParamNamespace{Literals: []string{"sid"}})
	require.ErrorContains(t, err, "prefs")
	require.NotContains(t, err.Error(), "secret")
	require.Equal(t, before, req.Header)
}

func TestAddRequestCookieVerbatim(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Add("Cookie", "first=one")
	req.Header.Add("Cookie", "second=two")
	require.NoError(t, AddRequestCookie(req, "greeting", "hello world"))
	require.Equal(t, []string{"first=one; second=two; greeting=hello world"}, req.Header.Values("Cookie"))
}
