package codegen

import (
	_ "embed"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

//go:embed testdata/optional_failure_reporting_harness.go.txt
var optionalFailureReportingHarness string

func TestJSONRPCOptionalFailureReportingGenerated(t *testing.T) {
	for _, mixed := range []bool{false, true} {
		t.Run("mixed="+strconv.FormatBool(mixed), func(t *testing.T) {
			dir := t.TempDir()
			if output := os.Getenv("LOOM_OPTIONAL_FAILURE_OUTPUT"); output != "" {
				dir = filepath.Join(output, strconv.FormatBool(mixed))
				require.NoError(t, os.MkdirAll(dir, 0o700))
			}
			root := RunJSONRPCDSL(t, optionalFailureReportingDSL(mixed))
			server := renderedJSONRPCFile(t, ServerFiles("", CreateJSONRPCServices(root)), "server.go", "server")
			require.NotContains(t, server, "errhandler(")
			require.Contains(t, server, "jsonrpc.ReportFailure(ctx, w, errhandler,")
			renderJSONRPCModule(t, dir, "example.com/optionalreport", root)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "report_test.go"), []byte(optionalFailureReportingHarness), 0o600))
			if os.Getenv("LOOM_OPTIONAL_FAILURE_GENERATE_ONLY") != "" {
				return
			}
			runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
			runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=1", "-v", "./...")
		})
	}
}

func optionalFailureReportingDSL(mixed bool) func() {
	return func() {
		API("optional-report", func() {
			JSONRPC(func() {
			})
		})
		Service("probe", func() {
			JSONRPC(func() {
				POST("/rpc")
			})
			Method("call", func() {
				Payload(func() {
					Attribute("value", String)
					Required("value")
				})
				Result(String)
				JSONRPC(func() {
				})
			})
			Method("nothing", func() {
				JSONRPC(func() {
				})
			})
			Method("captured", func() {
				Result(func() {
					Attribute("value", String)
					Attribute("header", String)
					Required("value", "header")
				})
				JSONRPC(func() {
					Response(func() {
						Header("header:X-Result")
					})
				})
			})
			if mixed {
				Method("events", func() {
					StreamingResult(String)
					JSONRPC(func() {
						ServerSentEvents()
					})
				})
			}
		})
	}
}
