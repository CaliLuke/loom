package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	servicecodegen "github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

func TestServiceClientStreamReadinessControls(t *testing.T) {
	for _, tc := range []struct {
		name   string
		design func()
	}{
		{"ServerStreamingRPCDSL", testdata.ServerStreamingRPCDSL},
		{"ClientStreamingRPCDSL", testdata.ClientStreamingRPCDSL},
		{"ClientStreamingNoResultDSL", testdata.ClientStreamingNoResultDSL},
		{"BidirectionalStreamingRPCDSL", testdata.BidirectionalStreamingRPCDSL},
		{"shared-http-sse", serviceClientSharedSSEDSL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := RunGRPCDSL(t, tc.design)
			dir := t.TempDir()
			const modulePath = "example.com/streamcontract"
			renderGRPCResponseContractModule(t, dir, modulePath, root)
			if tc.name == "shared-http-sse" {
				client, err := os.ReadFile(filepath.Join(dir, "gen/grpc/feed/client/client.go"))
				require.NoError(t, err)
				require.Contains(t, string(client), "func (s *WatchClientStream) Close() error")
				require.Contains(t, string(client), "return s.stream.CloseSend()")
			}
			data := servicecodegen.NewServicesData(root)
			for _, svc := range root.Services {
				sd := data.Get(svc.Name)
				for _, method := range sd.Methods {
					if method.ClientStream == nil {
						continue
					}
					assertion := fmt.Sprintf("package contract_test\nimport (svc %q; client %q)\nvar _ svc.%s = (*client.%s)(nil)\n", modulePath+"/gen/"+sd.PathName, modulePath+"/gen/grpc/"+sd.PathName+"/client", method.ClientStream.Interface, method.ClientStream.VarName)
					require.NoError(t, os.WriteFile(filepath.Join(dir, method.VarName+"_test.go"), []byte(assertion), 0o600))
				}
			}
			runGRPCGoCommand(t, dir, "mod", "tidy")
			runGRPCGoCommand(t, dir, "test", "-run", "^$", "./...")
			runGRPCGoCommand(t, dir, "vet", "./...")
		})
	}
}

func serviceClientSharedSSEDSL() {
	dsl.Service("feed", func() {
		dsl.Method("watch", func() {
			dsl.StreamingResult(dsl.String)
			dsl.HTTP(func() {
				dsl.GET("/watch")
				dsl.ServerSentEvents()
			})
			dsl.GRPC(func() {
			})
		})
	})
}
