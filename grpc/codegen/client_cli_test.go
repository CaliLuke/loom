package codegen

import (
	"bytes"
	"github.com/CaliLuke/loom/codegen/testutil"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

func TestClientCLIFiles(t *testing.T) {
	cases := []struct {
		Name string
		DSL  func()
	}{
		{"payload-with-validations", testdata.PayloadWithValidationsDSL},
		{"any-error", testdata.AnyErrorDSL},
		{"named-union-field-reuse", testdata.NamedUnionFieldReuseDSL},
		{"union-branch-union", testdata.UnionBranchUnionDSL},
		{"cli-protojson", testdata.CLIProtoJSONDSL},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunGRPCDSL(t, c.DSL)
			services := CreateGRPCServices(root)
			fs := ClientCLIFiles("", services)
			require.Greater(t, len(fs), 1, "expected at least 2 files")
			require.NotEmpty(t, fs[1].AllSections())
			var buf bytes.Buffer
			for _, s := range fs[1].AllSections() {
				require.NoError(t, s.Write(&buf))
			}
			code := codegen.FormatTestCode(t, buf.String())
			testutil.AssertGo(t, "testdata/golden/client_cli_"+c.Name+".go.golden", code)
		})
	}
}

// TestClientCLIPayloadBuildersPropagateConversionErrors checks that a client
// CLI payload builder whose protobuf conversion can fail declares the
// transformErr variable the conversion writes to, imports the loomgrpc
// runtime package it calls and returns the conversion error, and that
// builders whose conversion cannot fail do neither.
func TestClientCLIPayloadBuildersPropagateConversionErrors(t *testing.T) {
	cases := []struct {
		Name       string
		DSL        func()
		ErrorAware bool
	}{
		{"any-payload", testdata.AnyErrorDSL, true},
		{"payload-with-validations", testdata.PayloadWithValidationsDSL, false},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunGRPCDSL(t, c.DSL)
			services := CreateGRPCServices(root)
			fs := ClientCLIFiles("", services)
			require.Greater(t, len(fs), 1, "expected at least 2 files")
			var buf bytes.Buffer
			for _, s := range fs[1].AllSections() {
				require.NoError(t, s.Write(&buf))
			}
			code := codegen.FormatTestCode(t, buf.String())
			checks := []string{
				`loomgrpc "github.com/CaliLuke/loom/grpc"`,
				"transformErr := new(error)",
				"return zero, *transformErr",
			}
			for _, check := range checks {
				if c.ErrorAware {
					require.Contains(t, code, check)
				} else {
					require.NotContains(t, code, check)
				}
			}
		})
	}
}
