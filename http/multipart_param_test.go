package http

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMultipartLiteralParam(t *testing.T) {
	for _, tc := range []struct {
		name      string
		codec     ParamCodec
		value     ParamValue
		wantError bool
	}{
		{"scalar percent plus", ParamCodec{Style: ParamForm}, ParamValue{Kind: ParamScalar, Present: true, Text: "a%2Cb+%broken"}, false},
		{"repeated delimiter data", ParamCodec{Style: ParamForm, Explode: true}, ParamValue{Kind: ParamList, Present: true, Items: []string{"a,b", "x+y", "%2C"}}, false},
		{"delimited literal escapes", ParamCodec{Style: ParamForm}, ParamValue{Kind: ParamList, Present: true, Items: []string{"a%2Cb", "x+y"}}, false},
		{"delimited comma", ParamCodec{Style: ParamForm}, ParamValue{Kind: ParamList, Present: true, Items: []string{"a,b"}}, true},
		{"exploded object", ParamCodec{Style: ParamForm, Explode: true}, ParamValue{Kind: ParamPairs, Present: true, Pairs: []ParamPair{{Key: "x+y", Value: "%2C+"}}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var form MultipartForm
			err := form.AddParam("value", "text/plain", tc.codec, tc.value, ParamNamespace{})
			if tc.wantError {
				require.Error(t, err)
				require.Empty(t, form.Parts)
				return
			}
			require.NoError(t, err)
			value, err := DecodeMultipartParam(&form, "value", tc.codec, tc.value.Kind, ParamNamespace{})
			require.NoError(t, err)
			require.Equal(t, tc.value, value)
		})
	}
}
