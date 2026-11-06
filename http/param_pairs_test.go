package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRawPairsAgreesWithParseQuery(t *testing.T) {
	for _, raw := range []string{"", "a=1&a=2&b=&bare", "n=a%2Cb,c", "a+b=x+y", "x=%2B&%5Bk%5D=%E2%82%AC", "a=%zz&b=ok", "%zz=x", "a=b;c=d"} {
		t.Run(raw, func(t *testing.T) {
			want, parseErr := url.ParseQuery(raw)
			pairs, err := ParseRawPairs(raw)
			if parseErr != nil {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			for name, values := range want {
				got, err := DecodePairsParam(pairs, name, ParamCodec{Style: ParamForm, Explode: true}, ParamList, ParamNamespace{})
				require.NoError(t, err)
				require.True(t, got.Present)
				require.Equal(t, values, got.Items)
			}
		})
	}
}

func TestPairsParamCodec(t *testing.T) {
	for _, tc := range []struct {
		name  string
		codec ParamCodec
		value ParamValue
		wire  string
	}{
		{"form comma data", ParamCodec{Style: ParamForm}, ParamValue{Kind: ParamList, Present: true, Items: []string{"a,b", "c"}}, "n=a%2Cb,c"},
		{"form repeated", ParamCodec{Style: ParamForm, Explode: true}, ParamValue{Kind: ParamList, Present: true, Items: []string{"a,b", "c"}}, "n=a%2Cb&n=c"},
		{"space", ParamCodec{Style: ParamSpaceDelimited}, ParamValue{Kind: ParamList, Present: true, Items: []string{"a,b", "c"}}, "n=a%2Cb%20c"},
		{"pipe", ParamCodec{Style: ParamPipeDelimited}, ParamValue{Kind: ParamList, Present: true, Items: []string{"a,b", "c"}}, "n=a%2Cb%7Cc"},
		{"object empty value", ParamCodec{Style: ParamForm}, ParamValue{Kind: ParamPairs, Present: true, Pairs: []ParamPair{{Key: "R", Value: ""}, {Key: "G", Value: "200"}}}, "n=R,,G,200"},
		{"exploded object", ParamCodec{Style: ParamForm, Explode: true}, ParamValue{Kind: ParamPairs, Present: true, Pairs: []ParamPair{{Key: "R", Value: "100"}, {Key: "G", Value: "200"}}}, "R=100&G=200"},
		{"deep object", ParamCodec{Style: ParamDeepObject}, ParamValue{Kind: ParamPairs, Present: true, Pairs: []ParamPair{{Key: "R", Value: "100"}}}, "n%5BR%5D=100"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var builder RawPairsBuilder
			require.NoError(t, builder.Add("n", tc.codec, tc.value, ParamNamespace{Literals: []string{"control"}}))
			wire := builder.Encode(url.Values{"control": {"kept"}})
			require.Equal(t, "control=kept&"+tc.wire, wire)
			raw, err := ParseRawPairs(wire)
			require.NoError(t, err)
			got, err := DecodePairsParam(raw, "n", tc.codec, tc.value.Kind, ParamNamespace{Literals: []string{"control"}})
			require.NoError(t, err)
			require.Equal(t, tc.value, got)
		})
	}
	t.Run("literal brackets", func(t *testing.T) {
		raw, err := ParseRawPairs("n[R]=100")
		require.NoError(t, err)
		got, err := DecodePairsParam(raw, "n", ParamCodec{Style: ParamDeepObject}, ParamPairs, ParamNamespace{})
		require.NoError(t, err)
		require.Equal(t, []ParamPair{{Key: "R", Value: "100"}}, got.Pairs)
	})
	t.Run("fixed ownership", func(t *testing.T) {
		raw, err := ParseRawPairs("R=100&foreign=1&foreign=2")
		require.NoError(t, err)
		got, err := DecodePairsParam(raw, "n", ParamCodec{Style: ParamForm, Explode: true}, ParamPairs, ParamNamespace{OwnedLiterals: []string{"R"}})
		require.NoError(t, err)
		require.Equal(t, []ParamPair{{Key: "R", Value: "100"}}, got.Pairs)
	})
	for _, tc := range []struct {
		codec ParamCodec
		text  string
	}{
		{ParamCodec{Style: ParamSpaceDelimited}, "a b"},
		{ParamCodec{Style: ParamPipeDelimited}, "a|b"},
		{ParamCodec{Style: ParamForm}, ""},
	} {
		var builder RawPairsBuilder
		err := builder.Add("n", tc.codec, ParamValue{Kind: ParamList, Present: true, Items: []string{tc.text}}, ParamNamespace{})
		require.ErrorContains(t, err, "n")
	}
	for _, rawText := range []string{"n=,R,100", "n=R,100,R,200", "n=R,100&n=G,200"} {
		raw, err := ParseRawPairs(rawText)
		require.NoError(t, err)
		_, err = DecodePairsParam(raw, "n", ParamCodec{Style: ParamForm}, ParamPairs, ParamNamespace{})
		require.ErrorContains(t, err, "n")
	}
	t.Run("sibling publication rejected", func(t *testing.T) {
		var builder RawPairsBuilder
		err := builder.Add("n", ParamCodec{Style: ParamForm, Explode: true}, ParamValue{Kind: ParamPairs, Present: true, Pairs: []ParamPair{{Key: "sid", Value: "secret"}}}, ParamNamespace{Literals: []string{"sid"}})
		require.ErrorContains(t, err, "n")
		require.NotContains(t, err.Error(), "secret")
		require.Empty(t, builder.Encode(nil))
	})
}

func TestRawFormCarrier(t *testing.T) {
	const encoded = "ordinary=kept&n=a%2Cb,c&n2%5Bx%5D=y"
	req := httptest.NewRequest(http.MethodPost, "/?q=1", strings.NewReader(encoded))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	raw, err := ParseRawFormWithLimit(req, 1024)
	require.NoError(t, err)
	control := httptest.NewRequest(http.MethodPost, "/?q=1", strings.NewReader(encoded))
	control.Header = req.Header.Clone()
	require.NoError(t, ParseFormWithLimit(control, 1024))
	require.Equal(t, control.PostForm, req.PostForm)
	require.Equal(t, control.Form, req.Form)
	got, err := DecodePairsParam(raw, "n", ParamCodec{Style: ParamForm}, ParamList, ParamNamespace{})
	require.NoError(t, err)
	require.Equal(t, []string{"a,b", "c"}, got.Items)
	remaining, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	require.Empty(t, remaining)

	tooLarge := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(encoded))
	tooLarge.Header = req.Header.Clone()
	_, err = ParseRawFormWithLimit(tooLarge, 3)
	require.Error(t, err)

	outgoing := httptest.NewRequest(http.MethodPost, "/", nil)
	require.NoError(t, SetRawFormRequest(outgoing, encoded))
	require.Equal(t, int64(len(encoded)), outgoing.ContentLength)
	require.Equal(t, "application/x-www-form-urlencoded", outgoing.Header.Get("Content-Type"))
	replay, err := outgoing.GetBody()
	require.NoError(t, err)
	defer func() {
		require.NoError(t, replay.Close())
	}()
	bytes, err := io.ReadAll(replay)
	require.NoError(t, err)
	require.Equal(t, encoded, string(bytes))
}
