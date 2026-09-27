package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/codegentest"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

const primitiveFieldHarness = `package fields_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	loomhttp "github.com/CaliLuke/loom/http"
	client %q
)

func TestWireValue(t *testing.T) {
	requests := make(chan *http.Request, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Clone(context.Background())
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	u, err := url.Parse(server.URL)
	require.NoError(t, err)
	c := client.NewClient(u.Scheme, u.Host, server.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = c.%s()(ctx, %s)
	require.NoError(t, err)
	select {
	case request := <-requests:
		if %t {
			cookie, err := request.Cookie("c")
			require.NoError(t, err)
			require.Equal(t, %s, []string{cookie.Value})
		} else {
			require.Equal(t, %s, request.Header.Values("h"))
		}
	default:
		t.Fatal("client did not send a request")
	}
}
`

var primitiveFieldCases = []struct {
	name    string
	design  func()
	payload string
	values  string
	cookie  bool
}{
	{"header string", testdata.PayloadHeaderPrimitiveStringValidateDSL, `"val"`, `[]string{"val"}`, false},
	{"header bool", testdata.PayloadHeaderPrimitiveBoolValidateDSL, `true`, `[]string{"true"}`, false},
	{"header strings", testdata.PayloadHeaderPrimitiveArrayStringValidateDSL, `[]string{"val-one", "val-two"}`, `[]string{"val-one", "val-two"}`, false},
	{"header bools", testdata.PayloadHeaderPrimitiveArrayBoolValidateDSL, `[]bool{true, true}`, `[]string{"true", "true"}`, false},
	{"header default", testdata.PayloadHeaderPrimitiveStringDefaultDSL, `"override"`, `[]string{"override"}`, false},
	{"cookie string", testdata.PayloadCookiePrimitiveStringValidateDSL, `"val"`, `[]string{"val"}`, true},
	{"cookie bool", testdata.PayloadCookiePrimitiveBoolValidateDSL, `true`, `[]string{"true"}`, true},
	{"cookie default", testdata.PayloadCookiePrimitiveStringDefaultDSL, `"override"`, `[]string{"override"}`, true},
}

func TestPrimitiveHeaderCookieEncoder(t *testing.T) {
	for _, tc := range primitiveFieldCases {
		t.Run(tc.name, func(t *testing.T) {
			root := RunHTTPDSL(t, tc.design)
			files := ClientFiles("gen", CreateHTTPServices(root))
			sections := codegentest.Sections(files, "encode_decode.go", "request-encoder")
			require.Len(t, sections, 1)
			code := codegen.SectionCode(t, sections[0])
			if tc.cookie {
				require.Contains(t, code, "req.AddCookie(")
			} else {
				require.Contains(t, code, "req.Header.")
			}
		})
	}
}

func TestPrimitiveHeaderCookieWireValues(t *testing.T) {
	for _, tc := range primitiveFieldCases {
		t.Run(tc.name, func(t *testing.T) {
			root := RunHTTPDSL(t, tc.design)
			data := CreateHTTPServices(root).Get(root.Services[0].Name)
			harness := fmt.Sprintf(primitiveFieldHarness, "example.com/fields/gen/http/"+data.Service.PathName+"/client", data.Endpoints[0].Method.VarName, tc.payload, tc.cookie, tc.values, tc.values)
			dir := t.TempDir()
			renderHTTPModule(t, dir, "example.com/fields", root)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "fields_test.go"), []byte(harness), 0o600))
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "test", "./...")
		})
	}
}
