package http

import (
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
