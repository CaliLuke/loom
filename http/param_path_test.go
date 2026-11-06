package http

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPathParamCodec(t *testing.T) {
	scalar := func(value string) ParamValue {
		return ParamValue{Kind: ParamScalar, Present: true, Text: value}
	}
	list := func(values ...string) ParamValue {
		return ParamValue{Kind: ParamList, Present: true, Items: values}
	}
	pairs := ParamValue{Kind: ParamPairs, Present: true, Pairs: []ParamPair{{Key: "R", Value: "100"}, {Key: "G", Value: "200"}}}
	for _, tc := range []struct {
		name  string
		codec ParamCodec
		value ParamValue
		wire  string
	}{
		{"simple", ParamCodec{Style: ParamSimple}, scalar("blue"), "blue"},
		{"simple dot", ParamCodec{Style: ParamSimple}, scalar("."), "%2E"},
		{"simple dots", ParamCodec{Style: ParamSimple}, scalar(".."), "%2E%2E"},
		{"simple list", ParamCodec{Style: ParamSimple}, list("a,b", "c"), "a%2Cb,c"},
		{"simple list dots", ParamCodec{Style: ParamSimple}, list(".", ".."), "%2E,%2E%2E"},
		{"simple object", ParamCodec{Style: ParamSimple}, pairs, "R,100,G,200"},
		{"simple exploded", ParamCodec{Style: ParamSimple, Explode: true}, pairs, "R=100,G=200"},
		{"matrix", ParamCodec{Style: ParamMatrix}, scalar("blue"), ";id=blue"},
		{"matrix dot", ParamCodec{Style: ParamMatrix}, scalar("."), ";id=."},
		{"matrix empty", ParamCodec{Style: ParamMatrix}, scalar(""), ";id"},
		{"matrix list", ParamCodec{Style: ParamMatrix}, list("a,b", "c"), ";id=a%2Cb,c"},
		{"matrix exploded list", ParamCodec{Style: ParamMatrix, Explode: true}, list("blue", "black"), ";id=blue;id=black"},
		{"matrix empty list", ParamCodec{Style: ParamMatrix}, list(), ";id"},
		{"matrix one empty", ParamCodec{Style: ParamMatrix}, list(""), ";id="},
		{"matrix object", ParamCodec{Style: ParamMatrix}, pairs, ";id=R,100,G,200"},
		{"matrix exploded object", ParamCodec{Style: ParamMatrix, Explode: true}, pairs, ";R=100;G=200"},
		{"label", ParamCodec{Style: ParamLabel}, scalar("blue"), ".blue"},
		{"label empty", ParamCodec{Style: ParamLabel}, scalar(""), "%2E"},
		{"label dot", ParamCodec{Style: ParamLabel}, scalar("."), "%2E%2E"},
		{"label list", ParamCodec{Style: ParamLabel}, list("a,b", "c"), ".a%2Cb,c"},
		{"label exploded", ParamCodec{Style: ParamLabel, Explode: true}, list("blue", "black"), ".blue.black"},
		{"label one empty", ParamCodec{Style: ParamLabel, Explode: true}, list(""), "%2E"},
		{"label object", ParamCodec{Style: ParamLabel}, pairs, ".R,100,G,200"},
		{"label exploded object", ParamCodec{Style: ParamLabel, Explode: true}, pairs, ".R=100.G=200"},
		{"literal protected spelling", ParamCodec{Style: ParamLabel}, scalar("%2E"), ".%252E"},
		{"matrix escaped equals", ParamCodec{Style: ParamMatrix}, scalar("a=b"), ";id=a%3Db"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wire, err := EncodePathParam("id", tc.codec, tc.value)
			require.NoError(t, err)
			require.Equal(t, tc.wire, wire)
			got, err := DecodePathParam("id", tc.wire, tc.codec, tc.value.Kind)
			require.NoError(t, err)
			require.Equal(t, tc.value.Kind, got.Kind)
			require.True(t, got.Present)
			require.Equal(t, tc.value.Text, got.Text)
			require.Equal(t, len(tc.value.Items), len(got.Items))
			if len(tc.value.Items) > 0 {
				require.Equal(t, tc.value.Items, got.Items)
			}
			require.Equal(t, tc.value.Pairs, got.Pairs)
		})
	}
	for _, tc := range []struct {
		codec ParamCodec
		value ParamValue
	}{
		{ParamCodec{Style: ParamSimple}, scalar("")},
		{ParamCodec{Style: ParamSimple}, list()},
		{ParamCodec{Style: ParamSimple}, list("")},
		{ParamCodec{Style: ParamLabel}, list()},
		{ParamCodec{Style: ParamLabel, Explode: true}, list("1.5")},
		{ParamCodec{Style: ParamMatrix, Explode: true}, ParamValue{Kind: ParamPairs, Present: true}},
	} {
		_, err := EncodePathParam("id", tc.codec, tc.value)
		require.ErrorContains(t, err, "id")
	}
	for _, raw := range []string{"%2e", "%2E%2e"} {
		got, err := DecodePathParam("id", raw, ParamCodec{Style: ParamLabel}, ParamScalar)
		require.NoError(t, err)
		want := ""
		if len(raw) == 6 {
			want = "."
		}
		require.Equal(t, want, got.Text)
	}
	for _, raw := range []string{"R,100,G", "R,100,R,200", "R=%zz"} {
		_, err := DecodePathParam("id", raw, ParamCodec{Style: ParamSimple}, ParamPairs)
		require.ErrorContains(t, err, "id")
	}
}
