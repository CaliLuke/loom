package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

const jsonRPCErrorIdentityAdmissionHarness = `package integration_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	client "example.com/jsonrpcerrorroundtrip/gen/jsonrpc/tools/client"
	tools "example.com/jsonrpcerrorroundtrip/gen/tools"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/jsonrpc"
)

type identityDecoder struct {
	base  loomhttp.Decoder
	calls *int
}

func TestRawIdentityBeforeDefaultsAndDecoder(t *testing.T) {
	for _, identity := range []string{"", "null", "7", ` + "`" + `"other"` + "`" + `} {
		t.Run(identity, func(t *testing.T) {
			data := ` + "`" + `"name":"forbidden","code":"CODE","message":"safe","remedy_code":"fix","retry_hint":"retry"` + "`" + `
			if identity != "" {
				data += ` + "`" + `,"IDENTITY_KEY":` + "`" + ` + identity
			}
			host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if _, err := w.Write([]byte(` + "`" + `{"jsonrpc":"2.0","id":"request","error":{"code":4403,"message":"safe","data":{` + "`" + ` + data + "}}}")); err != nil {
					t.Error(err)
				}
			}))
			defer host.Close()
			calls := 0
			decoder := func(r *http.Response) loomhttp.Decoder { return identityDecoder{loomhttp.ResponseDecoder(r), &calls} }
			c := client.NewClient("http", strings.TrimPrefix(host.URL, "http://"), host.Client(), loomhttp.RequestEncoder, decoder, false)
			_, err := c.Typed()(context.Background(), &tools.TypedPayload{ID: "request", Kind: "forbidden"})
			var invalid *loomhttp.ClientError
			if !errors.As(err, &invalid) || invalid.Name != "invalid_response" || calls != 0 {
				t.Errorf("identity rejected before defaults/body callback: got %T %v calls=%d", err, err, calls)
			}
		})
	}
}

func (d identityDecoder) Decode(value any) error {
	if _, envelope := value.(*jsonrpc.RawResponse); !envelope {
		*d.calls++
	}
	return d.base.Decode(value)
}
`

// TestJSONRPCErrorDiscriminatorUsesAuthoredKey checks the decoder's wire identity.
func TestJSONRPCErrorDiscriminatorUsesAuthoredKey(t *testing.T) {
	for _, field := range []string{"kind", "kind:error_kind"} {
		t.Run(field, func(t *testing.T) {
			root := RunJSONRPCDSL(t, func() {
				jsonRPCErrorDiscriminationDSL(field)
			})
			services := CreateJSONRPCServices(root)
			code := jsonrpcGeneratedCode(t, ClientFiles("", services))
			require.NotContains(t, code, "var jerrData jsonrpc.ErrorData", "designed data is not generic metadata")
		})
	}
}

// TestJSONRPCErrorDiscriminatorRoundTrip retains the existing typed/raw harness.
func TestJSONRPCErrorDiscriminatorRoundTrip(t *testing.T) {
	for _, field := range []string{"name", "kind", "kind:error_kind", "tagged"} {
		t.Run(field, func(t *testing.T) {
			root := RunJSONRPCDSL(t, func() {
				jsonRPCErrorDiscriminationDSL(field)
			})
			dir := t.TempDir()
			renderJSONRPCModule(t, dir, "example.com/jsonrpcerrorroundtrip", root)
			harness := jsonRPCTypedErrorRoundTripTestSource
			if field != "name" {
				harness = strings.ReplaceAll(harness, "Name:       ", "Name: \"unrelated resource\", Kind: ")
				harness = strings.ReplaceAll(harness, "typed.Name != name", "typed.Kind != name || typed.Name != \"unrelated resource\"")
			}
			require.NoError(t, os.WriteFile(filepath.Join(dir, "error_discrimination_test.go"), []byte(harness), 0o600))
			if field != "name" {
				key := field
				if field == "kind:error_kind" {
					key = "error_kind"
				}
				if field == "tagged" {
					key = "exact_identity"
				}
				admission := strings.ReplaceAll(jsonRPCErrorIdentityAdmissionHarness, "IDENTITY_KEY", key)
				require.NoError(t, os.WriteFile(filepath.Join(dir, "identity_test.go"), []byte(admission), 0o600))
			}
			runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
			runGoJSONRPCTestCommand(t, dir, "test", "-count=1", "./...")
		})
	}
}

func jsonRPCErrorDiscriminationDSL(field string) {
	tagged := field == "tagged"
	if tagged {
		field = "kind"
	}
	toolError := dsl.Type("ToolError", func() {
		dsl.ErrorName(field, dsl.String, func() {
			if tagged {
				dsl.Meta("struct:tag:json", "exact_identity")
				dsl.Default("forbidden")
			}
		})
		if field != "name" {
			dsl.Attribute("name", dsl.String)
			dsl.Required("name")
		}
		dsl.Attribute("code", dsl.String)
		dsl.Attribute("message", dsl.String)
		dsl.Attribute("remedy_code", dsl.String)
		dsl.Attribute("retry_hint", dsl.String)
		dsl.Required(strings.Split(field, ":")[0], "code", "message", "remedy_code", "retry_hint")
	})
	jsonRPCTypedErrorServiceDSL(toolError)
}
