package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	cg "github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/stretchr/testify/require"
)

const customTextClientHarness = `package customtext_test

import (
 "context"
 "net/http"
 "net/http/httptest"
 "net/url"
 "strings"
 "sync/atomic"
 "testing"

 client "example.com/customtext/gen/http/SERVICE/client"
 loomhttp "github.com/CaliLuke/loom/http"
)

func TestCustomTextCLIAndWire(t *testing.T) {
 const value = "10290aeb-ea52-40ab-acb7-538dd06d9d1e"
 p, err := client.BuildMETHODPayload(value)
 if err != nil {
  t.Fatal(err)
 }
 if p.ID.String() != value {
  t.Errorf("parsed value=%v", p.ID)
 }
 if _, err := client.BuildMETHODPayload("invalid"); err == nil {
  t.Error("invalid UUID accepted")
 }
 var want atomic.Value
 want.Store(value)
 srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
  var got string
  switch "LOCATION" {
  case "path":
   got = strings.TrimPrefix(r.URL.Path, "/")
  case "query":
   got = r.URL.Query().Get("id")
   if want.Load().(string) == "" && r.URL.Query().Has("id") {
    t.Error("omitted query is present")
   }
  case "header":
   got = r.Header.Get("id")
   if want.Load().(string) == "" && r.Header.Values("id") != nil {
    t.Error("omitted header is present")
   }
  case "cookie":
   cookie, err := r.Cookie("id")
   if err != nil {
    t.Error(err)
   } else {
    got = cookie.Value
   }
  }
  if got != want.Load().(string) {
   t.Errorf("wire value=%q, want %q", got, want.Load())
  }
  w.WriteHeader(http.StatusNoContent)
 }))
 defer srv.Close()
 addr, err := url.Parse(srv.URL)
 if err != nil {
  t.Fatal(err)
 }
 c := client.NewClient(addr.Scheme, addr.Host, http.DefaultClient, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
 if _, err := c.METHOD()(context.Background(), p); err != nil {
  t.Fatal(err)
 }
 EMPTY
}
`

const customTextTokenSource = `package scalar
import "strings"
// Token is a custom scalar with a canonical wire representation.
type Token struct {
 // Text is the parsed token value.
 Text string
}
// String returns the token's wire representation.
func (t Token) String() string {
 return "token:"+t.Text
}
// UnmarshalText parses a token; DSL validation owns length constraints.
func (t *Token) UnmarshalText(b []byte) error {
 t.Text = strings.TrimPrefix(string(b), "token:")
 return nil
}
`

// TestCustomTextClientGeneratedModules compiles and exercises custom scalar CLI and wire encoding.
func TestCustomTextClientGeneratedModules(t *testing.T) {
	cases := []struct {
		name, location string
		design         func()
	}{
		{"path", "path", testdata.PayloadPathCustomTextUnmarshalerDSL},
		{"path-validate", "path", testdata.PayloadPathCustomTextUnmarshalerValidateDSL},
		{"query", "query", testdata.PayloadQueryCustomTextUnmarshalerDSL},
		{"query-optional", "query", testdata.PayloadQueryCustomTextUnmarshalerOptionalDSL},
		{"query-optional-validate", "query", testdata.PayloadQueryCustomTextUnmarshalerOptionalValidateDSL},
		{"header", "header", testdata.PayloadHeaderCustomTextUnmarshalerDSL},
		{"header-optional-validate", "header", testdata.PayloadHeaderCustomTextUnmarshalerOptionalValidateDSL},
		{"cookie", "cookie", testdata.PayloadCookieCustomTextUnmarshalerDSL},
		{"cookie-default", "cookie", testdata.PayloadCookieCustomTextUnmarshalerDefaultDSL},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := RunHTTPDSL(t, tc.design)
			dir := t.TempDir()
			renderHTTPModule(t, dir, "example.com/customtext", root)
			svc := root.Services[0]
			method := svc.Methods[0]
			checkEmpty := ""
			if strings.Contains(tc.name, "optional") {
				checkEmpty = `empty, err := client.BuildMETHODPayload("")
 if err != nil {
  t.Fatal(err)
 }
 if empty.ID != nil {
  t.Errorf("omitted field=%v", empty.ID)
 }
 p = empty
 want.Store("")
 if _, err := c.METHOD()(context.Background(), p); err != nil {
  t.Fatal(err)
 }`
			}
			if tc.name == "cookie-default" {
				checkEmpty = `empty, err := client.BuildMETHODPayload("")
 if err != nil {
  t.Fatal(err)
 }
 if empty.ID.String() != "00000000-0000-0000-0000-000000000000" {
  t.Errorf("default=%v", empty.ID)
 }`
			}
			harness := strings.NewReplacer("SERVICE", cg.SnakeCase(svc.Name), "METHOD", cg.Goify(method.Name, true), "LOCATION", tc.location, "EMPTY", checkEmpty).Replace(customTextClientHarness)
			// Replacement text is expanded separately because Replacer does not recurse.
			harness = strings.ReplaceAll(harness, "METHOD", cg.Goify(method.Name, true))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "client_test.go"), []byte(harness), 0600))
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "vet", "./...")
			runGoCommand(t, dir, "test", "-race", "-count=1", "./...")
		})
	}
}

// TestCustomTextClientStringer verifies a non-UUID type and raw text constraints.
func TestCustomTextClientStringer(t *testing.T) {
	for _, location := range []string{"query", "cookie"} {
		t.Run(location, func(t *testing.T) {
			root := RunHTTPDSL(t, func() {
				Service("token", func() {
					Method("send", func() {
						Payload(func() {
							Attribute("id", String, func() {
								MinLength(6)
								if location == "cookie" {
									Default("token:default")
								}
								Meta("struct:field:type", "scalar.Token", "example.com/customtext/scalar")
							})
						})
						HTTP(func() {
							GET("/")
							if location == "cookie" {
								Cookie("id")
							} else {
								Param("id")
							}
						})
					})
				})
			})
			dir := t.TempDir()
			renderHTTPModule(t, dir, "example.com/customtext", root)
			require.NoError(t, os.MkdirAll(filepath.Join(dir, "scalar"), 0700))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "scalar", "token.go"), []byte(customTextTokenSource), 0600))
			checkDefault := ""
			if location == "cookie" {
				checkDefault = `defaultPayload, err := client.BuildSendPayload("")
 if err != nil {
  t.Fatal(err)
 }
 if defaultPayload.ID.String() != "token:default" {
  t.Errorf("default=%v", defaultPayload.ID)
 }
 want.Store("token:default")
 if _, err := c.Send()(context.Background(), defaultPayload); err != nil {
  t.Fatal(err)
 }`
			}
			harness := strings.NewReplacer("SERVICE", "token", "METHOD", "Send", "LOCATION", location, "EMPTY", checkDefault).Replace(customTextClientHarness)
			harness = strings.ReplaceAll(harness, "10290aeb-ea52-40ab-acb7-538dd06d9d1e", "token:abc")
			harness = strings.ReplaceAll(harness, "invalid UUID accepted", "invalid raw text accepted")
			harness = strings.ReplaceAll(harness, `BuildSendPayload("invalid")`, `BuildSendPayload("x")`)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "client_test.go"), []byte(harness), 0600))
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "vet", "./...")
			runGoCommand(t, dir, "test", "-race", "-count=1", "./...")
		})
	}
}
