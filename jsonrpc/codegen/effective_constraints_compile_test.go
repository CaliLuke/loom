package codegen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	numericdata "github.com/CaliLuke/loom/codegen/testdata"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

func TestEffectiveConstraintsGeneratedJSONRPCModule(t *testing.T) {
	root := RunJSONRPCDSL(t, effectiveConstraintsDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/effectiveconstraints", root)
	pinEffectiveDefaultModule(t, dir)
	generated := readGeneratedGo(t, dir)
	require.Contains(t, generated, "Inherited  InheritedDefault")
	require.Contains(t, generated, "Overridden OverriddenDefault")
	require.Contains(t, generated, "loom.Nullable[InheritedDefault]")
	serverConstructor := generatedDefaultConstructor(t, dir, "server", "NewDefaultsPayload")
	require.Contains(t, serverConstructor, `body.Inherited.Value()`)
	require.Contains(t, serverConstructor, `body.Overridden.Value()`)
	require.Contains(t, serverConstructor, `v.Inherited = "base"`)
	require.Contains(t, serverConstructor, `v.Overridden = "override"`)
	clientConstructor := generatedDefaultConstructor(t, dir, "client", "NewDefaultsRequestBody")
	require.Contains(t, clientConstructor, `string(p.Inherited)`)
	require.Contains(t, clientConstructor, `string(p.Overridden)`)
	require.NotContains(t, clientConstructor, `body.Inherited = "base"`)
	require.NotContains(t, clientConstructor, `body.Overridden = "override"`)
	require.Contains(t, generated, `ValidateRequiredRequestBody(&body)`)
	require.Contains(t, generated, `loom.MissingFieldError("base", validationPath)`)
	require.Contains(t, generated, `loom.MissingFieldError("derived", validationPath)`)
	require.GreaterOrEqual(t, strings.Count(generated, "loom.ValidatePattern"), 2)
	require.Contains(t, generated, `"b$"`)
	require.Contains(t, generated, `"^a"`)
	require.Contains(t, generated, `loom.FormatIPv4`)
	require.Contains(t, generated, `loom.FormatIP`)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "numeric_test.go"), []byte(exactNumericHarness), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "defaults_test.go"), []byte(effectiveDefaultHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "./...")
}

func effectiveConstraintsDSL() {
	dsl.API("effective", func() {
		dsl.JSONRPC(func() {})
	})
	base := dsl.Type("DefaultBase", dsl.String, func() {
		dsl.Default("base")
		dsl.MinLength(2)
	})
	inherited := dsl.Type("InheritedDefault", base)
	overridden := dsl.Type("OverriddenDefault", base, func() {
		dsl.Default("override")
	})
	baseRequired := dsl.Type("BaseRequired", func() {
		dsl.Attribute("base", dsl.String)
		dsl.Attribute("derived", dsl.String)
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
			dsl.JSONRPC(func() {})
		})

		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("defaults", func() {
			dsl.Payload(func() {
				dsl.Attribute("inherited", inherited)
				dsl.Attribute("overridden", overridden)
				dsl.Attribute("nullable", inherited, func() {
					dsl.Nullable()
				})
			})
			dsl.JSONRPC(func() {})
		})
		dsl.Method("required", func() {
			dsl.Payload(derivedRequired)
			dsl.JSONRPC(func() {})
		})
		dsl.Method("clauses", func() {
			dsl.Payload(func() {
				dsl.Attribute("pattern", patternDerived)
				dsl.Attribute("ip", formatDerived)
				dsl.Required("pattern", "ip")
			})
			dsl.JSONRPC(func() {})
		})
	})
}

func readGeneratedGo(t *testing.T, root string) string {
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

func generatedDefaultConstructor(t *testing.T, dir, side, name string) string {
	t.Helper()
	path := filepath.Join(dir, "gen", "jsonrpc", "constraints", side, "types.go")
	source, err := os.ReadFile(path)
	require.NoError(t, err)
	positions := token.NewFileSet()
	file, err := parser.ParseFile(positions, path, source, 0)
	require.NoError(t, err)
	var matches []string
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Recv == nil && function.Name.Name == name {
			matches = append(matches, string(source[positions.Position(function.Pos()).Offset:positions.Position(function.End()).Offset]))
		}
	}
	require.Len(t, matches, 1, "%s constructor %s", side, name)
	return matches[0]
}

const exactNumericHarness = `package effectiveconstraints_test
import (
 "context"
 "encoding/json/v2"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"
 "github.com/stretchr/testify/require"
 constraints "example.com/effectiveconstraints/gen/constraints"
 server "example.com/effectiveconstraints/gen/jsonrpc/constraints/server"
 loomhttp "github.com/CaliLuke/loom/http"
)
func TestExactNumericBounds(t *testing.T) {
 calls := 0
 endpoints := &constraints.Endpoints{Numeric: func(context.Context, any) (any, error) { calls++; return nil, nil }}
 mux := loomhttp.NewMuxer()
 server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
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
  wire := "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"numeric\",\"params\":"+tc.wire+"}"
  request := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader(wire))
  request.Header.Set("Content-Type", "application/json")
  response := httptest.NewRecorder()
  mux.ServeHTTP(response, request)
  require.Equal(t, 200, response.Code, response.Body.String())
  var reply map[string]any
  require.NoError(t, json.Unmarshal(response.Body.Bytes(), &reply))
  if tc.valid {
   require.Nil(t, reply["error"], "%s: %s", tc.wire, response.Body.String())
   require.Equal(t, before+1, calls)
  } else {
   require.NotNil(t, reply["error"], tc.wire)
   require.Equal(t, float64(-32602), reply["error"].(map[string]any)["code"])
   require.Equal(t, "validation error", reply["error"].(map[string]any)["message"])
   require.Equal(t, before, calls)
  }
 }
}

func TestInheritedRequiredFields(t *testing.T) {
	calls := 0
	endpoints := &constraints.Endpoints{Required: func(context.Context, any) (any, error) {
		calls++
		return nil, nil
	}}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
	for _, tc := range []struct {
		name    string
		params  string
		missing []string
	}{
		{"both missing", "{}", []string{"base", "derived"}},
		{"inherited missing", "{\"derived\":\"value\"}", []string{"base"}},
		{"local missing", "{\"base\":\"value\"}", []string{"derived"}},
		{"complete", "{\"base\":\"value\",\"derived\":\"value\"}", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := calls
			wire := "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"required\",\"params\":" + tc.params + "}"
			request := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader(wire))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			require.Equal(t, http.StatusOK, response.Code, response.Body.String())
			var reply map[string]any
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &reply))
			if len(tc.missing) == 0 {
				require.Nil(t, reply["error"], response.Body.String())
				require.Equal(t, before+1, calls)
				return
			}
			require.NotNil(t, reply["error"], response.Body.String())
			problem := reply["error"].(map[string]any)
			require.Equal(t, float64(-32602), problem["code"])
			require.Contains(t, problem["message"], "Missing required field:")
			if len(tc.missing) == 1 {
				require.Contains(t, response.Body.String(), tc.missing[0])
			}
			require.Equal(t, before, calls)
		})
	}
}
`
