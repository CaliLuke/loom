package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const responseValueReferenceHarness = `package responsevalue_test

import (
	"encoding/json/v2"
	"io"
	"net/http"
	"strings"
	"testing"

	client "example.com/responsevalue/gen/jsonrpc/probe/client"
	probe "example.com/responsevalue/gen/probe"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/require"
)

type presence interface {
	Present() bool
	IsNull() bool
}

func TestResponseValues(t *testing.T) {
	for _, method := range []struct {
		name string
		decode func(*http.Response) (any, error)
		valid string
		invalid string
	}{
		{"object", client.DecodeObjectResponse(loomhttp.ResponseDecoder, false), "{\"text\":\"ok\"}", "{\"text\":\"\"}"},
		{"scalar", client.DecodeScalarResponse(loomhttp.ResponseDecoder, false), "\"ok\"", "\"\""},
		{"array", client.DecodeArrayResponse(loomhttp.ResponseDecoder, false), "[\"ok\"]", "[]"},
		{"map", client.DecodeMapResponse(loomhttp.ResponseDecoder, false), "{\"key\":\"ok\"}", "{}"},
	} {
		for _, state := range []string{"valid", "null", "invalid", "absent", "malformed", "error", "bad-status"} {
			t.Run(method.name+"/"+state, func(t *testing.T) {
				body := "{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":"+method.valid+"}"
				status := http.StatusOK
				switch state {
				case "null":
					body = "{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":null}"
				case "invalid":
					body = "{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":"+method.invalid+"}"
				case "absent":
					body = "{\"jsonrpc\":\"2.0\",\"id\":1}"
				case "malformed":
					body = "{"
				case "error":
					body = "{\"jsonrpc\":\"2.0\",\"id\":1,\"error\":{\"code\":4200,\"message\":\"failed\",\"data\":{\"message\":\"rejected\"}}}"
				case "bad-status":
					status = http.StatusBadGateway
				}
				response := &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
				value, err := method.decode(response)
				if state == "error" {
					var failure *probe.Failure
					require.ErrorAs(t, err, &failure)
					require.Equal(t, "rejected", failure.Message)
					return
				}
				if state != "valid" && !(state == "null" && "NULLABLE" == "nullable") {
					require.Error(t, err)
					var failure *loomhttp.ClientError
					require.ErrorAs(t, err, &failure)
					require.Equal(t, "probe", failure.Service)
					require.Equal(t, method.name, failure.Method)
					switch state {
					case "invalid":
						require.Equal(t, "validation_error", failure.Name)
					case "absent", "malformed":
						require.Equal(t, "decoding_error", failure.Name)
					case "bad-status":
						require.Equal(t, "invalid_response", failure.Name)
					}
					require.Nil(t, value)
					return
				}
				require.NoError(t, err)
				if "NULLABLE" == "nullable" {
					present, ok := value.(presence)
					require.True(t, ok, "%T", value)
					require.True(t, present.Present())
					require.Equal(t, state == "null", present.IsNull())
				}
				encoded, err := json.Marshal(value)
				require.NoError(t, err)
				expected := method.valid
				if state == "null" {
					expected = "null"
				}
				require.JSONEq(t, expected, string(encoded))
			})
		}
	}
}
`

// TestJSONRPCResponseValueReferenceGenerated compiles and exercises response values.
func TestJSONRPCResponseValueReferenceGenerated(t *testing.T) {
	for _, nullable := range []bool{false, true} {
		mode := "ordinary"
		if nullable {
			mode = "nullable"
		}
		t.Run(mode, func(t *testing.T) {
			root := RunJSONRPCDSL(t, responseValueReferenceDSL(nullable))
			dir := t.TempDir()
			renderJSONRPCModule(t, dir, "example.com/responsevalue", root)
			modPath := filepath.Join(dir, "go.mod")
			mod, err := os.ReadFile(modPath)
			require.NoError(t, err)
			mod = []byte(strings.ReplaceAll(string(mod), "require github.com/CaliLuke/loom v1.0.0", "require github.com/CaliLuke/loom v1.10.0-alpha.9.0.20261013051529-2a8337700ffd"))
			require.NoError(t, os.WriteFile(modPath, mod, 0o600))
			harness := strings.ReplaceAll(responseValueReferenceHarness, "NULLABLE", mode)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "response_test.go"), []byte(harness), 0o600))
			for _, args := range [][]string{{"mod", "tidy"}, {"vet", "./..."}, {"test", "-count=1", "./..."}} {
				runGoJSONRPCTestCommand(t, dir, args...)
			}
		})
	}
}
