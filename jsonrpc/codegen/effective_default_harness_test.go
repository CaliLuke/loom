package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const effectiveDefaultHarness = `package effectiveconstraints_test

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	constraints "example.com/effectiveconstraints/gen/constraints"
	client "example.com/effectiveconstraints/gen/jsonrpc/constraints/client"
	server "example.com/effectiveconstraints/gen/jsonrpc/constraints/server"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

func TestEffectiveDefaultAdmission(t *testing.T) {
	for _, tc := range []struct {
		name, params, inherited, overridden, nullable string
		null, valid                                   bool
	}{
		{name: "absent", params: "{}", inherited: "base", overridden: "override", nullable: "base", valid: true},
		{name: "explicit", params: "{\"inherited\":\"value\",\"overridden\":\"chosen\",\"nullable\":\"optional\"}", inherited: "value", overridden: "chosen", nullable: "optional", valid: true},
		{name: "explicit null", params: "{\"nullable\":null}", inherited: "base", overridden: "override", null: true, valid: true},
		{name: "inherited empty", params: "{\"inherited\":\"\"}"},
		{name: "overridden empty", params: "{\"overridden\":\"\"}"},
		{name: "nullable empty", params: "{\"nullable\":\"\"}"},
		{name: "inherited null", params: "{\"inherited\":null}"},
		{name: "wrong type", params: "{\"inherited\":123}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			endpoints := &constraints.Endpoints{Defaults: func(_ context.Context, value any) (any, error) {
				calls++
				payload := value.(*constraints.DefaultsPayload)
				require.Equal(t, tc.inherited, string(payload.Inherited))
				require.Equal(t, tc.overridden, string(payload.Overridden))
				require.True(t, payload.Nullable.Present())
				require.Equal(t, tc.null, payload.Nullable.IsNull())
				nullable, hasValue := payload.Nullable.Value()
				require.Equal(t, !tc.null, hasValue)
				require.Equal(t, tc.nullable, string(nullable))
				return nil, nil
			}}
			mux := loomhttp.NewMuxer()
			server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
			wire := "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"defaults\",\"params\":" + tc.params + "}"
			request := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader(wire))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			require.Equal(t, http.StatusOK, response.Code, response.Body.String())
			var reply map[string]any
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &reply))
			require.Equal(t, float64(1), reply["id"])
			if tc.valid {
				require.Nil(t, reply["error"], response.Body.String())
				require.Equal(t, 1, calls)
			} else {
				require.Zero(t, calls)
				problem, ok := reply["error"].(map[string]any)
				require.True(t, ok, response.Body.String())
				require.Equal(t, float64(-32602), problem["code"])
				require.NotEmpty(t, problem["message"])
			}
		})
	}
}

func TestEffectiveDefaultClientConstruction(t *testing.T) {
	for _, value := range []string{"", "explicit"} {
		t.Run(value, func(t *testing.T) {
			payload := &constraints.DefaultsPayload{
				Inherited:  constraints.InheritedDefault(value),
				Overridden: constraints.OverriddenDefault(value),
				Nullable:   loom.NullValue[constraints.InheritedDefault](),
			}
			body := client.NewDefaultsRequestBody(payload)
			require.Equal(t, value, body.Inherited)
			require.Equal(t, value, body.Overridden)
			require.True(t, body.Nullable.IsNull())
		})
	}
}
`

func TestEffectiveDefaultArtifacts(t *testing.T) {
	output := os.Getenv("LOOM_EFFECTIVE_DEFAULT_OUTPUT")
	if output == "" {
		t.Skip("set LOOM_EFFECTIVE_DEFAULT_OUTPUT for isolated artifact comparison")
	}
	require.NoError(t, os.MkdirAll(output, 0o700))
	renderJSONRPCModule(t, output, "example.com/effectiveconstraints", RunJSONRPCDSL(t, effectiveConstraintsDSL))
	pinEffectiveDefaultModule(t, output)
}

func TestEffectiveDefaultArtifactBuilds(t *testing.T) {
	output := os.Getenv("LOOM_EFFECTIVE_DEFAULT_OUTPUT")
	if output == "" {
		t.Skip("set LOOM_EFFECTIVE_DEFAULT_OUTPUT to compile the comparison module")
	}
	runGoJSONRPCTestCommand(t, output, "mod", "tidy")
	runGoJSONRPCTestCommand(t, output, "build", "./...")
	runGoJSONRPCTestCommand(t, output, "vet", "./...")
}

func pinEffectiveDefaultModule(t *testing.T, dir string) {
	t.Helper()
	path := filepath.Join(dir, "go.mod")
	manifest, err := os.ReadFile(path)
	require.NoError(t, err)
	manifest = []byte(strings.Replace(string(manifest), "github.com/CaliLuke/loom v1.0.0", "github.com/CaliLuke/loom v1.10.0-alpha.9.0.20261013020843-063198a2f427", 1))
	require.NoError(t, os.WriteFile(path, manifest, 0o600))
}
