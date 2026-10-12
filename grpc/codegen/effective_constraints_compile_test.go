package codegen

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	numericdata "github.com/CaliLuke/loom/codegen/testdata"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

func TestEffectiveConstraintsGeneratedGRPCModule(t *testing.T) {
	root := RunGRPCDSL(t, func() {
		base := dsl.Type("DefaultBase", dsl.String, func() {
			dsl.Default("base")
			dsl.MinLength(2)
		})
		inherited := dsl.Type("InheritedDefault", base)
		overridden := dsl.Type("OverriddenDefault", base, func() {
			dsl.Default("override")
		})
		baseRequired := dsl.Type("BaseRequired", func() {
			dsl.Field(1, "base", dsl.String)
			dsl.Field(2, "derived", dsl.String)
			dsl.Required("base")
		})
		derivedRequired := dsl.Type("DerivedRequired", baseRequired, func() {
			dsl.Required("derived")
		})
		patternBase := dsl.Type("PatternBase", dsl.String, func() { dsl.Pattern("^a") })
		patternDerived := dsl.Type("PatternDerived", patternBase, func() { dsl.Pattern("b$") })
		formatBase := dsl.Type("FormatBase", dsl.String, func() { dsl.Format(dsl.FormatIP) })
		formatDerived := dsl.Type("FormatDerived", formatBase, func() { dsl.Format(dsl.FormatIPv4) })
		numeric := numericdata.ExactNumericPayload()
		dsl.Service("constraints", func() {
			dsl.Method("numeric", func() {
				dsl.Payload(numeric)
				dsl.GRPC(func() {})
			})

			dsl.Method("defaults", func() {
				dsl.Payload(func() {
					dsl.Field(1, "inherited", inherited)
					dsl.Field(2, "overridden", overridden)
				})
				dsl.GRPC(func() {})
			})
			dsl.Method("required", func() {
				dsl.Payload(derivedRequired)
				dsl.GRPC(func() {})
			})
			dsl.Method("clauses", func() {
				dsl.Payload(func() {
					dsl.Field(1, "pattern", patternDerived)
					dsl.Field(2, "ip", formatDerived)
					dsl.Required("pattern", "ip")
				})
				dsl.GRPC(func() {})
			})
		})
	})
	dir := t.TempDir()
	renderGRPCResponseContractModule(t, dir, "example.com/effectiveconstraints", root)
	generated := readGeneratedGRPCGo(t, dir)
	require.Contains(t, generated, "InheritedDefault")
	require.Contains(t, generated, "OverriddenDefault")
	require.Contains(t, generated, `"base"`)
	require.Contains(t, generated, `"override"`)
	require.Contains(t, generated, "Base    string")
	require.Contains(t, generated, "Derived string")
	require.GreaterOrEqual(t, strings.Count(generated, "loom.ValidatePattern"), 2)
	require.Contains(t, generated, `"b$"`)
	require.Contains(t, generated, `"^a"`)
	require.Contains(t, generated, `loom.FormatIPv4`)
	require.Contains(t, generated, `loom.FormatIP`)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "numeric_test.go"), []byte(exactNumericHarness), 0o600))
	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "vet", "./gen/...")
	runGRPCGoCommand(t, dir, "test", ".", "./gen/...")
}

func TestGRPCMetadataUsesEffectiveNamedDefaults(t *testing.T) {
	root := RunGRPCDSL(t, func() {
		base := dsl.Type("MetadataDefaultBase", dsl.String, func() {
			dsl.Default("base")
		})
		middle := dsl.Type("MetadataDefaultMiddle", base)
		override := dsl.Type("MetadataDefaultOverride", middle, func() {
			dsl.Default("override")
		})
		payload := dsl.Type("MetadataPayload", func() {
			dsl.Field(1, "inherited", middle)
			dsl.Field(2, "overridden", override)
		})
		dsl.Service("metadata", func() {
			dsl.Method("check", func() {
				dsl.Payload(payload)
				dsl.GRPC(func() {
					dsl.Metadata(func() {
						dsl.Attribute("inherited:X-Inherited")
						dsl.Attribute("overridden:X-Overridden")
					})
				})
			})
		})
	})
	metadata := CreateGRPCServices(root).Get("metadata").Endpoint("check").Request.Metadata
	require.Len(t, metadata, 2)
	require.Equal(t, "base", metadata[0].DefaultValue)
	require.Equal(t, "override", metadata[1].DefaultValue)
}

func readGeneratedGRPCGo(t *testing.T, root string) string {
	t.Helper()
	var source strings.Builder
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		source.Write(contents)
		return nil
	})
	require.NoError(t, err)
	return source.String()
}

const exactNumericHarness = `package effectiveconstraints_test
import (
 "context"
 "encoding/json/v2"
 "testing"
 "github.com/stretchr/testify/require"
 "google.golang.org/grpc/codes"
 "google.golang.org/grpc/status"
 constraints "example.com/effectiveconstraints/gen/constraints"
 pb "example.com/effectiveconstraints/gen/grpc/constraints/pb"
 server "example.com/effectiveconstraints/gen/grpc/constraints/server"
)
func TestExactNumericBounds(t *testing.T) {
 calls := 0
 endpoints := &constraints.Endpoints{Numeric: func(context.Context, any) (any, error) { calls++; return nil, nil }}
 handler := server.New(endpoints, nil)
 for _, tc := range []struct { wire string; valid bool }{
{"{}",true},
{"{\"signed\":9007199254740993}",true},
{"{\"signed\":9007199254740992}",false},
{"{\"signed\":9223372036854775807}",true},
{"{\"unsigned\":18446744073709551615}",true},
{"{\"unsigned\":18446744073709551614}",false},
{"{\"negative\":-9223372036854775808}",false},
{"{\"negative\":-9223372036854775807}",true},
{"{\"upper\":9007199254740992}",true},
{"{\"upper\":9007199254740993}",false},
{"{\"decimal\":0.09999999999999999}",true},
{"{\"decimal\":0.1}",false},
{"{\"decimal32\":0.099999994}",true},
{"{\"decimal32\":0.1}",false},
{"{\"tiny\":5e-324}",true},
{"{\"tiny\":0}",false},
{"{\"control\":2}",true},
{"{\"control\":1}",false},
} {

  before := calls
  var request pb.NumericRequest
  require.NoError(t, json.Unmarshal([]byte(tc.wire), &request))
  _, err := handler.Numeric(context.Background(), &request)
  if tc.valid {
   require.NoError(t, err, tc.wire)
   require.Equal(t, before+1, calls)
  } else {
   require.Equal(t, codes.InvalidArgument, status.Code(err), tc.wire)
   require.Equal(t, "validation error", status.Convert(err).Message())
   require.Equal(t, before, calls)
  }
 }
}
`
