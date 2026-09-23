package http

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type (
	// formFuzzInner is a nested object in a generated form body.
	formFuzzInner struct {
		Label string `form:"label"`
		N     *int   `form:"n"`
	}

	// formFuzzBody is a representative generated form body.
	formFuzzBody struct {
		Name   string                   `form:"name"`
		Count  *int64                   `form:"count"`
		Tags   []string                 `form:"tags"`
		Attrs  map[string]string        `form:"attrs"`
		Nested map[string]formFuzzInner `form:"nested"`
		Inner  *formFuzzInner           `form:"inner"`
	}
)

// FuzzFormValuesRoundTrip checks that form bodies survive EncodeFormValues,
// URL-encoded serialization, parsing, and DecodeFormValues unchanged, and that
// decoding arbitrary parsed query strings never panics.
func FuzzFormValuesRoundTrip(f *testing.F) {
	f.Add("alice", int64(3), "a,b", "k", "v", "n1", "label", 7, uint8(0xff), "name=x&tags=a&tags=b&attrs[k]=v&nested[a][label]=l")
	f.Add("", int64(0), "", "", "", "", "", 0, uint8(0), "")
	f.Add("a&b=c", int64(-1), "x y,%zz", "sp ace", "=&", "k2", "", -5, uint8(0x0f), "inner[n]=1&inner[label]=x&count=bad")
	f.Add("unicode é", int64(1<<62), ",", "k.dot", "v", "k", "l", 1, uint8(0x3f), "nested[a]=1&attrs=flat&attrs[a][b]=c")

	f.Fuzz(func(t *testing.T, name string, count int64, tags, attrKey, attrValue, nestedKey, label string, n int, mask uint8, query string) {
		body := formFuzzBody{Name: name}
		if mask&1 != 0 {
			body.Count = &count
		}
		if mask&2 != 0 && tags != "" {
			body.Tags = strings.Split(tags, ",")
		}
		if mask&4 != 0 {
			body.Attrs = map[string]string{attrKey: attrValue}
		}
		if mask&8 != 0 {
			inner := formFuzzInner{Label: label}
			if mask&16 != 0 {
				inner.N = &n
			}
			body.Nested = map[string]formFuzzInner{nestedKey: inner}
		}
		if mask&32 != 0 {
			body.Inner = &formFuzzInner{Label: label}
		}
		if strings.ContainsAny(attrKey+nestedKey, "[]") {
			// Bracket nesting has no escape for '[' or ']' inside a map key,
			// so such keys cannot round-trip by construction.
			t.Skip()
		}

		values, err := EncodeFormValues(body)
		require.NoError(t, err)
		parsed, err := url.ParseQuery(values.Encode())
		require.NoError(t, err)
		var decoded formFuzzBody
		require.NoError(t, DecodeFormValues(parsed, &decoded))
		require.Equal(t, body, decoded, "encoded as %q", values.Encode())

		if arbitrary, err := url.ParseQuery(query); err == nil {
			var target formFuzzBody
			if err := DecodeFormValues(arbitrary, &target); err != nil {
				require.NotEmpty(t, err.Error())
			}
			var flat map[string]string
			if err := DecodeFormValues(arbitrary, &flat); err != nil {
				require.NotEmpty(t, err.Error())
			}
		}
	})
}
