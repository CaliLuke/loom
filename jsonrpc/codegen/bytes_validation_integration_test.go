package codegen

import (
	_ "embed"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	servicecodegen "github.com/CaliLuke/loom/codegen/service"
	codegentestdata "github.com/CaliLuke/loom/codegen/testdata"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/schematest"
)

//go:embed testdata/bytes_validation_jsonrpc_test.go.txt
var bytesValidationJSONRPCHarness string

// TestBytesValidationGeneratedJSONRPC compiles request, client response and
// service view validation with retained named byte slices.
func TestBytesValidationGeneratedJSONRPC(t *testing.T) {
	root := RunJSONRPCDSL(t, codegentestdata.BytesValidationJSONRPCDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/bytesvalidation", root)
	data := servicecodegen.NewServicesData(root)
	for _, svc := range root.Services {
		if views := servicecodegen.ViewsFile("example.com/bytesvalidation/gen", svc, data); views != nil {
			_, err := views.Render(dir)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bytes_test.go"), []byte(bytesValidationJSONRPCHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "build", "./...")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=1", "./...")
	if os.Getenv("LOOM_OPENAPI_CONTRACT") != "1" {
		return
	}
	observed, err := os.ReadFile(filepath.Join(dir, "schema_observations.json"))
	require.NoError(t, err)
	var observations []struct {
		Name     string `json:"name"`
		Params   string `json:"params"`
		Accepted bool   `json:"accepted"`
	}
	require.NoError(t, json.Unmarshal(observed, &observations, json.RejectUnknownMembers(true)))
	require.Len(t, observations, 15)
	inline, err := expr.InlineJSONSchema(root.Services[0].Method("check").Payload)
	require.NoError(t, err)
	instances := make([]any, len(observations))
	for index, observation := range observations {
		instances[index] = jsontext.Value(observation.Params)
	}
	validator := schematest.New(t)
	results := validator.Check(t, []schematest.Batch{{Schema: jsontext.Value(inline), Instances: instances}})[0]
	for index, observation := range observations {
		if results[index].Valid != observation.Accepted {
			t.Errorf("%s: generated decoder accepted=%t schema accepted=%t: %s", observation.Name, observation.Accepted, results[index].Valid, results[index].Errors)
		}
	}
}
