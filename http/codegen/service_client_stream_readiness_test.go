package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	servicecodegen "github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestServiceClientStreamReadinessControls(t *testing.T) {
	for _, tc := range []struct {
		name   string
		design func()
	}{
		{"StreamingResultDSL", testdata.StreamingResultDSL},
		{"StreamingPayloadDSL", testdata.StreamingPayloadDSL},
		{"StreamingPayloadNoResultDSL", testdata.StreamingPayloadNoResultDSL},
		{"BidirectionalStreamingDSL", testdata.BidirectionalStreamingDSL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := RunHTTPDSL(t, tc.design)
			dir := t.TempDir()
			const modulePath = "example.com/streamcontract"
			renderHTTPModule(t, dir, modulePath, root)
			data := servicecodegen.NewServicesData(root)
			for _, svc := range root.Services {
				sd := data.Get(svc.Name)
				for _, method := range sd.Methods {
					if method.ClientStream == nil {
						continue
					}
					assertion := fmt.Sprintf("package contract_test\nimport (svc %q; client %q)\nvar _ svc.%s = (*client.%s)(nil)\n", modulePath+"/gen/"+sd.PathName, modulePath+"/gen/http/"+sd.PathName+"/client", method.ClientStream.Interface, method.ClientStream.VarName)
					require.NoError(t, os.WriteFile(filepath.Join(dir, method.VarName+"_test.go"), []byte(assertion), 0o600))
				}
			}
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "test", "-run", "^$", "./...")
			runGoCommand(t, dir, "vet", "./...")
		})
	}
}

func TestServiceClientMixedSSEReadiness(t *testing.T) {
	root := RunHTTPDSL(t, testdata.MixedResultsDSL)
	dir := t.TempDir()
	const modulePath = "example.com/mixedcontract"
	renderHTTPModule(t, dir, modulePath, root)
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "build", "./...")
	const assertions = `package contract_test
import (
 svc "example.com/mixedcontract/gen/mixed_results_service"
 client "example.com/mixedcontract/gen/http/mixed_results_service/client"
)
var _ svc.CreateClientStream = (*client.CreateStreamImpl)(nil)
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "contract_test.go"), []byte(assertions), 0o600))
	runGoCommand(t, dir, "test", "-run", "^$", "./...")
}
