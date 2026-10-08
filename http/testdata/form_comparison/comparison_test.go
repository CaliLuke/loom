package formcomparison

import (
	"fmt"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	loomhttp "github.com/CaliLuke/loom/http"
	form "github.com/go-playground/form/v4"
	"github.com/stretchr/testify/require"
)

type (
	child struct {
		Name string `form:"name"`
	}
	payload struct {
		Child child            `form:"child"`
		Items map[string]child `form:"items"`
		Tags  []string         `form:"tags"`
		Flag  *bool            `form:"flag,omitempty"`
	}
	fields struct {
		Number *int              `form:"number,omitempty"`
		Flag   *bool             `form:"flag,omitempty"`
		Text   *string           `form:"text,omitempty"`
		Tags   []string          `form:"tags,omitempty"`
		Items  map[string]string `form:"items,omitempty"`
	}
	codec struct {
		name   string
		encode func(any) (url.Values, error)
		decode func(url.Values, any) error
	}
)

func codecs() []codec {
	e, d := form.NewEncoder(), form.NewDecoder()
	e.SetMode(form.ModeExplicit)
	d.SetMode(form.ModeExplicit)
	be, bd := form.NewEncoder(), form.NewDecoder()
	be.SetMode(form.ModeExplicit)
	bd.SetMode(form.ModeExplicit)
	be.SetNamespacePrefix("[")
	be.SetNamespaceSuffix("]")
	bd.SetNamespacePrefix("[")
	bd.SetNamespaceSuffix("]")
	return []codec{
		{"loom", loomhttp.EncodeFormValues, loomhttp.DecodeFormValues},
		{"library", e.Encode, func(v url.Values, p any) error {
			return d.Decode(p, v)
		}},
		{"library-brackets", be.Encode, func(v url.Values, p any) error {
			return bd.Decode(p, v)
		}},
	}
}

func TestNestedRoundTrip(t *testing.T) {
	flag := false
	in := payload{Child: child{"a"}, Items: map[string]child{"": {"empty"}, "k": {"b"}}, Tags: []string{"x", "y"}, Flag: &flag}
	for _, c := range codecs() {
		t.Run(c.name, func(t *testing.T) {
			v, err := c.encode(in)
			require.NoError(t, err)
			var out payload
			require.NoError(t, c.decode(v, &out))
			require.Equal(t, in, out)
			key := "items[k][name]"
			if c.name == "library" {
				key = "items[k].name"
			}
			require.Equal(t, "b", v.Get(key))
			t.Log(v.Encode())
		})
	}
}

func TestPresenceAndErrors(t *testing.T) {
	for _, c := range codecs() {
		for _, tc := range []struct {
			name         string
			values       url.Values
			loomError    bool
			libraryError bool
		}{
			{"missing", url.Values{}, false, false},
			{"empty-integer", url.Values{"number": {""}}, true, false},
			{"empty-boolean", url.Values{"flag": {""}}, true, false},
			{"invalid-integer", url.Values{"number": {"bad"}}, true, true},
			{"overflow-integer", url.Values{"number": {"9999999999999999999999999"}}, true, true},
			{"empty-string", url.Values{"text": {""}}, false, false},
			{"zero-and-false", url.Values{"number": {"0"}, "flag": {"false"}}, false, false},
			{"unknown", url.Values{"unknown": {"x"}}, false, false},
			{"duplicates", url.Values{"number": {"1", "2"}, "tags": {"x", "y"}}, false, false},
		} {
			t.Run(c.name+"/"+tc.name, func(t *testing.T) {
				var out fields
				err := c.decode(tc.values, &out)
				wantErr := tc.libraryError
				if c.name == "loom" {
					wantErr = tc.loomError
				}
				require.Equal(t, wantErr, err != nil)
				switch tc.name {
				case "missing", "unknown", "empty-integer":
					require.Equal(t, fields{}, out)
				case "empty-boolean":
					if c.name == "loom" {
						require.Nil(t, out.Flag)
					} else {
						require.NotNil(t, out.Flag)
						require.False(t, *out.Flag)
					}
				case "empty-string":
					require.NotNil(t, out.Text)
					require.Empty(t, *out.Text)
				case "zero-and-false":
					require.NotNil(t, out.Number)
					require.Zero(t, *out.Number)
					require.NotNil(t, out.Flag)
					require.False(t, *out.Flag)
				case "duplicates":
					require.Equal(t, 1, *out.Number)
					require.Equal(t, []string{"x", "y"}, out.Tags)
				}
				t.Logf("error=%v output=%#v", err, out)
			})
		}
	}
}

func TestBytesAndRootMap(t *testing.T) {
	for _, c := range codecs() {
		t.Run(c.name, func(t *testing.T) {
			type bytesBody struct {
				Data []byte `form:"data"`
			}
			in := bytesBody{[]byte("hi")}
			v, err := c.encode(in)
			require.NoError(t, err)
			var out bytesBody
			require.NoError(t, c.decode(v, &out))
			require.Equal(t, in, out)
			if c.name == "loom" {
				require.Equal(t, []string{"hi"}, v["data"])
			} else {
				require.Equal(t, []string{"104", "105"}, v["data"])
			}
			t.Log("bytes:", v.Encode())
			m := map[string]string{"k": "v"}
			v, err = c.encode(m)
			require.NoError(t, err)
			var decoded map[string]string
			require.NoError(t, c.decode(v, &decoded))
			require.Equal(t, m, decoded)
			key := "k"
			if c.name != "loom" {
				key = "[k]"
			}
			require.Equal(t, "v", v.Get(key))
			t.Log("root map:", v.Encode())
		})
	}
}

func TestRawBytesCustomCallback(t *testing.T) {
	e, d := form.NewEncoder(), form.NewDecoder()
	e.RegisterCustomTypeFunc(func(v any) ([]string, error) {
		return []string{string(v.([]byte))}, nil
	}, []byte{})
	d.RegisterCustomTypeFunc(func(v []string) (any, error) {
		return []byte(v[0]), nil
	}, []byte{})
	type body struct {
		Data []byte `form:"data"`
	}
	v, err := e.Encode(body{[]byte("hi")})
	require.NoError(t, err)
	require.Equal(t, url.Values{"data": {"hi"}}, v)
	var out body
	require.NoError(t, d.Decode(&out, v))
	require.Equal(t, []byte("hi"), out.Data)
}

func TestLimitsAndMalformedKeys(t *testing.T) {
	for _, c := range codecs() {
		for _, key := range []string{"tags[2]", "tags[10000]", "tags[" + strconv.Itoa(int(^uint(0)>>1)) + "]", "unknown[", "unknown]"} {
			t.Run(c.name+"/"+key, func(t *testing.T) {
				var out fields
				err, panicValue := observe(func() error {
					return c.decode(url.Values{key: {"x"}}, &out)
				})
				t.Logf("error=%v panic=%v tags-length=%d", err, panicValue, len(out.Tags))
				if c.name == "loom" {
					require.Nil(t, panicValue)
					require.NoError(t, err)
					require.Empty(t, out.Tags)
				} else {
					switch key {
					case "tags[2]":
						require.Nil(t, panicValue)
						require.NoError(t, err)
						require.Equal(t, []string{"", "", "x"}, out.Tags)
					case "tags[10000]":
						require.Nil(t, panicValue)
						require.Error(t, err)
					case "unknown[":
						require.Contains(t, fmt.Sprint(panicValue), "missing ']' bracket")
					case "unknown]":
						require.Contains(t, fmt.Sprint(panicValue), "missing '[' bracket")
					default:
						require.Contains(t, fmt.Sprint(panicValue), "reflect.MakeSlice: negative len")
					}
				}
			})
		}
	}
	t.Run("indexed-limit-does-not-bound-repeated-values", func(t *testing.T) {
		d := form.NewDecoder()
		d.SetMaxArraySize(2)
		var out fields
		require.NoError(t, d.Decode(&out, url.Values{"tags": {"a", "b", "c"}}))
		require.Len(t, out.Tags, 3)
	})
}

// observe catches panics only in this investigation to assert unsafe library behavior.
// It is not a proposed production recovery wrapper.
func observe(f func() error) (err error, panicValue any) {
	defer func() {
		panicValue = recover()
	}()
	err = f()
	return
}

func TestUnionHookCannotReadSiblingKeys(t *testing.T) {
	type union struct{ hidden string }
	type body struct {
		Grant union `form:"grant"`
	}
	d := form.NewDecoder()
	called := false
	d.RegisterCustomTypeFunc(func(v []string) (any, error) {
		called = true
		return union{fmt.Sprint(v)}, nil
	}, union{})
	var out body
	require.NoError(t, d.Decode(&out, url.Values{"grant.type": {"code"}, "grant.code": {"abc"}}))
	require.False(t, called, "callbacks only receive values at the exact field key")
	require.NoError(t, d.Decode(&out, url.Values{"grant": {"opaque"}}))
	require.True(t, called)
}

func TestEmptyRootMapKey(t *testing.T) {
	for _, c := range codecs() {
		t.Run(c.name, func(t *testing.T) {
			in := map[string]string{"": "empty"}
			v, err := c.encode(in)
			if c.name == "loom" {
				require.ErrorContains(t, err, "requires a field name")
				return
			}
			require.NoError(t, err)
			var out map[string]string
			require.NoError(t, c.decode(v, &out))
			require.Equal(t, in, out)
		})
	}
}

func TestCollectionOfObjects(t *testing.T) {
	type body struct {
		Children []child `form:"children"`
	}
	for _, c := range codecs() {
		t.Run(c.name, func(t *testing.T) {
			in := body{[]child{{"a"}, {"b"}}}
			v, err := c.encode(in)
			if c.name == "loom" {
				require.ErrorContains(t, err, "unsupported form slice element type")
				return
			}
			require.NoError(t, err)
			var out body
			require.NoError(t, c.decode(v, &out))
			require.Equal(t, in, out)
			t.Log(v.Encode())
		})
	}
}

func TestMalformedKeySurvivesHTTPParsing(t *testing.T) {
	req := httptest.NewRequest("POST", "/form", strings.NewReader("unknown%5B=x"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	require.NoError(t, loomhttp.ParseFormWithLimit(req, 1024))
	var out fields
	err, p := observe(func() error {
		return form.NewDecoder().Decode(&out, req.PostForm)
	})
	require.NoError(t, err)
	require.Contains(t, fmt.Sprint(p), "missing ']' bracket")
}
