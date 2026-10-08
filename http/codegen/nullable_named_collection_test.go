package codegen

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestNullableNamedCollectionBodyRoundTrip preserves the destination array or
// map type through transport conversion, including empty values and invalid null
// elements, for both optional and required bodies.
func TestNullableNamedCollectionBodyRoundTrip(t *testing.T) {
	cases := []struct {
		name                                 string
		typ                                  func() expr.DataType
		empty, populated                     string
		invalid, invalidCode, invalidMessage string
	}{
		{
			"array", func() expr.DataType {
				return ArrayOf(String)
			}, "[]", `["a","b"]`, `[null]`, "decode_payload", "invalid request body",
		},
		{
			"map", func() expr.DataType {
				return MapOf(String, Int)
			}, "{}", `{"a":7}`, `{"a":null}`, "decode_payload", "invalid request body",
		},
		{
			"array nullable elements", func() expr.DataType {
				element := Type("Element", String, func() {
					Nullable()
				})
				return ArrayOf(element)
			}, "[]", `[null,"a"]`, `[1]`, "decode_payload", "invalid request body",
		},
		{
			"map nullable elements", func() expr.DataType {
				element := Type("Element", Int, func() {
					Nullable()
				})
				return MapOf(String, element)
			}, "{}", `{"a":null}`, `{"a":"bad"}`, "decode_payload", "invalid request body",
		},
	}
	for _, c := range cases {
		for _, required := range []bool{false, true} {
			for _, value := range []string{c.empty, c.populated} {
				t.Run(c.name+"/required="+strconv.FormatBool(required)+"/"+value, func(t *testing.T) {
					root := RunHTTPDSL(t, func() {
						nullableNamedBodyDSL(c.typ(), required)()
					})
					dir := t.TempDir()
					renderHTTPModule(t, dir, "example.com/namedbody", root)
					harness := strings.NewReplacer(
						"REQUIRED", strconv.FormatBool(required),
						"VALID_JSON", strconv.Quote(value),
						"INVALID_JSON", strconv.Quote(c.invalid),
						"VALIDATION_CODE", strconv.Quote(c.invalidCode),
						"VALIDATION_DETAIL", strconv.Quote(c.invalidMessage),
					).Replace(nullableNamedBodyHarness)
					require.NoError(t, os.WriteFile(filepath.Join(dir, "named_body_test.go"), []byte(harness), 0o600))
					runGoCommand(t, dir, "mod", "tidy")
					runGoCommand(t, dir, "vet", "./...")
					runGoCommand(t, dir, "test", "-count=1", "./...")
				})
			}
		}
	}
}

// TestNullableNamedCollectionDefaults applies defaults only to absent fields.
func TestNullableNamedCollectionDefaults(t *testing.T) {
	cases := []struct {
		name             string
		typ              func() expr.DataType
		value            any
		populated, empty string
	}{
		{"array", func() expr.DataType {
			return ArrayOf(String)
		}, []string{"a"}, `["a"]`, `[]`},
		{"map", func() expr.DataType {
			return MapOf(String, Int)
		}, map[string]int{"a": 7}, `{"a":7}`, `{}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := RunHTTPDSL(t, func() {
				value := Type("Value", c.typ(), func() {
					Nullable()
				})
				Service("sender", func() {
					Method("send", func() {
						Payload(func() {
							Attribute("b", value, func() {
								Default(c.value)
							})
						})
						HTTP(func() {
							POST("/")
						})
					})
				})
			})
			dir := t.TempDir()
			renderHTTPModule(t, dir, "example.com/namedbody", root)
			harness := strings.NewReplacer("POPULATED", strconv.Quote(c.populated), "EMPTY", strconv.Quote(c.empty)).Replace(nullableNamedCollectionDefaultHarness)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "defaults_test.go"), []byte(harness), 0o600))
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "vet", "./...")
			runGoCommand(t, dir, "test", "-count=1", "./...")
		})
	}
}

const nullableNamedCollectionDefaultHarness = `package namedbody_test

import (
 "context"
 "encoding/json/v2"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"

 server "example.com/namedbody/gen/http/sender/server"
 "example.com/namedbody/gen/sender"
 loomhttp "github.com/CaliLuke/loom/http"
)

func TestDefaults(t *testing.T) {
 const populated, empty = POPULATED, EMPTY
 for _, tc := range []struct{ name, body, want string }{
  {"absent", "{}", populated},
  {"null", ` + "`{\"b\":null}`" + `, "null"},
  {"empty", ` + "`{\"b\":`" + `+empty+"}", empty},
  {"value", ` + "`{\"b\":`" + `+populated+"}", populated},
 } {
  t.Run(tc.name, func(t *testing.T) {
   var got *sender.SendPayload
   endpoints := &sender.Endpoints{Send: func(_ context.Context, p any) (any,error) {
    got = p.(*sender.SendPayload)
    return nil,nil
   }}
   mux := loomhttp.NewMuxer()
   server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
   req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body))
   req.Header.Set("Content-Type", "application/json")
   recorder := httptest.NewRecorder()
   mux.ServeHTTP(recorder, req)
   if recorder.Code != http.StatusNoContent || got == nil {
    t.Fatalf("status=%d invoked=%v body=%s", recorder.Code, got!=nil,recorder.Body.String())
   }
   value, err := json.Marshal(got.B)
   if err != nil || string(value) != tc.want {
    t.Errorf("value=%s err=%v want=%s",value,err,tc.want)
   }
  })
 }
}
`
