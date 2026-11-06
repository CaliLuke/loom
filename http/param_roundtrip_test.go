package http

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestParamCodecAlphabet checks implementation correspondence with CodecRoundTrip's
// delimiter alphabet, extended with quote, backslash and real UTF-8 bytes.
func TestParamCodecAlphabet(t *testing.T) {
	alphabet := []string{"", "a", ",", ".", ";", "=", " ", "|", "é", "%2E", `"`, `\`, "[", "]"}
	values := make([]ParamValue, 0, 2*(len(alphabet)+1)*len(alphabet))
	for _, text := range alphabet {
		values = append(values, ParamValue{Kind: ParamScalar, Present: true, Text: text})
		for _, other := range alphabet {
			values = append(values,
				ParamValue{Kind: ParamList, Present: true, Items: []string{text, other}},
				ParamValue{Kind: ParamPairs, Present: true, Pairs: []ParamPair{{Key: text, Value: other}}},
			)
		}
		values = append(values, ParamValue{Kind: ParamList, Present: true, Items: []string{text}})
	}
	for _, style := range []ParamStyle{ParamSimple, ParamMatrix, ParamLabel, ParamForm, ParamSpaceDelimited, ParamPipeDelimited, ParamDeepObject, ParamCookie} {
		for _, explode := range []bool{false, true} {
			codec := ParamCodec{Style: style, Explode: explode}
			for i, value := range values {
				t.Run(fmt.Sprintf("%d/%t/%d", style, explode, i), func(t *testing.T) {
					switch style {
					case ParamSimple, ParamMatrix, ParamLabel:
						raw, err := EncodePathParam("n", codec, value)
						if err == nil {
							got, err := DecodePathParam("n", raw, codec, value.Kind)
							require.NoError(t, err)
							require.Equal(t, value, got)
						}
						if style == ParamSimple {
							raw, err := EncodeHeaderParam("X-N", codec, value)
							if err == nil {
								got, err := DecodeHeaderParam("X-N", []string{strings.Trim(raw, " \t")}, true, codec, value.Kind)
								require.NoError(t, err)
								require.Equal(t, value, got)
							}
						}
					case ParamForm, ParamSpaceDelimited, ParamPipeDelimited, ParamDeepObject:
						var builder RawPairsBuilder
						if err := builder.Add("n", codec, value, ParamNamespace{}); err == nil {
							pairs, err := ParseRawPairs(builder.Encode(nil))
							require.NoError(t, err)
							got, err := DecodePairsParam(pairs, "n", codec, value.Kind, ParamNamespace{})
							require.NoError(t, err)
							require.Equal(t, value, got)
						}
					}
					if style == ParamCookie || style == ParamForm {
						req := httptest.NewRequest(http.MethodGet, "/", nil)
						if err := AddRequestCookieParam(req, "n", codec, value, ParamNamespace{}); err == nil {
							got, err := DecodeCookieParam(req.Cookies(), "n", codec, value.Kind, ParamNamespace{})
							require.NoError(t, err)
							require.Equal(t, value, got)
						}
					}
				})
			}
		}
	}
}
