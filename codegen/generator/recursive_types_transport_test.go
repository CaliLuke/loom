package generator

import (
	"bytes"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	servicetestdata "github.com/CaliLuke/loom/codegen/service/testdata"
	codegentestdata "github.com/CaliLuke/loom/codegen/testdata"
	dsl "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
	exprtestdata "github.com/CaliLuke/loom/expr/testdata"
	grpctestdata "github.com/CaliLuke/loom/grpc/codegen/testdata"
	httptestdata "github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/CaliLuke/loom/internal/loomsource"
	"github.com/CaliLuke/loom/internal/testingx"
)

// TestTransportRecursiveTypes runs the full transport generator over every
// valid testdata DSL that declares a recursive user or result type and over a
// design exposed on HTTP, gRPC, and JSON-RPC whose types recurse mutually and
// through collections alone. It renders every file section in memory and
// parses the Go output; it does not run protoc because some of these designs
// are valid Loom designs that protoc rejects.
func TestTransportRecursiveTypes(t *testing.T) {
	cases := []struct {
		Name string
		DSL  func()
	}{
		{"service-bidirectional-streaming", servicetestdata.BidirectionalStreamingMethodDSL},
		{"service-multiple-methods", servicetestdata.MultipleMethodsDSL},
		{"service-recursive-collection-of-result-type", servicetestdata.ResultWithRecursiveCollectionOfResultTypeDSL},
		{"service-recursive-result-type", servicetestdata.ResultWithRecursiveResultTypeDSL},
		{"service-streaming-payload", servicetestdata.StreamingPayloadMethodDSL},
		{"codegen-recursive-validation", codegentestdata.RecursiveValidationDSL},
		{"codegen-types", codegentestdata.TestTypesDSL},
		{"expr-grpc-endpoint-with-any-type", exprtestdata.GRPCEndpointWithAnyType},
		{"grpc-nested-user-types", grpctestdata.MessageUserTypeWithNestedUserTypesDSL},
		{"http-multi", httptestdata.MultiDSL},
		{"http-payload-body-inline-recursive-user", httptestdata.PayloadBodyInlineRecursiveUserDSL},
		{"all-transports-mutually-recursive", mutuallyRecursiveAllTransportsDSL},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := codegen.RunDSL(t, c.DSL)
			require.True(t, hasRecursiveUserType(root), "DSL no longer declares a recursive type")

			files, err := Transport("example.com/recursive/gen", []eval.Root{root})
			require.NoError(t, err)

			for _, f := range files {
				var buf bytes.Buffer
				for _, s := range f.AllSections() {
					require.NoError(t, s.Write(&buf), f.Path)
				}
				if filepath.Ext(f.Path) == ".go" {
					_, err := parser.ParseFile(token.NewFileSet(), f.Path, buf.Bytes(), parser.SkipObjectResolution)
					require.NoError(t, err, f.Path)
				}
			}
		})
	}
}

// TestRecursiveTypesGeneratedCodeCompiles compiles the service and transport
// packages generated for the recursive design exposed on every transport.
func TestRecursiveTypesGeneratedCodeCompiles(t *testing.T) {
	root := codegen.RunDSL(t, mutuallyRecursiveAllTransportsDSL)
	roots := []eval.Root{root}
	genpkg := "example.com/recursive/gen"

	serviceFiles, err := Service(genpkg, roots)
	require.NoError(t, err)
	transportFiles, err := Transport(genpkg, roots)
	require.NoError(t, err)

	dir := t.TempDir()
	for _, file := range mergeFilesByPath(append(serviceFiles, transportFiles...)) {
		_, err := file.Render(dir)
		require.NoError(t, err, file.Path)
	}
	repoRoot, err := loomsource.RepositoryRoot(".")
	require.NoError(t, err)
	source, err := loomsource.Resolve(repoRoot, filepath.Join(t.TempDir(), "loom-pinned"))
	require.NoError(t, err)
	goMod := fmt.Sprintf("module example.com/recursive\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v0.0.0\n\nreplace github.com/CaliLuke/loom => %s\n", source)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))
	_, err = testingx.RunCmd(dir, "go", "mod", "tidy")
	require.NoError(t, err)
	output, err := testingx.RunCmd(dir, "go", "build", "./...")
	require.NoError(t, err, output)
}

func mutuallyRecursiveAllTransportsDSL() {
	dsl.API("recursive", func() {
		dsl.JSONRPC(func() {})
	})
	var Node = dsl.Type("Node", func() {
		dsl.Field(1, "name", dsl.String)
		dsl.Field(2, "edges", dsl.ArrayOf("Edge"))
		dsl.Field(3, "index", dsl.MapOf(dsl.String, "Edge"))
	})
	var Edge = dsl.Type("Edge", func() {
		dsl.Field(1, "target", Node)
	})
	var Tree = dsl.Type("Tree", func() {
		dsl.Field(1, "children", dsl.ArrayOf("Tree"))
		dsl.Field(2, "lookup", dsl.MapOf(dsl.String, "Tree"))
	})
	dsl.Service("graph", func() {
		dsl.Method("walk", func() {
			dsl.Payload(Node)
			dsl.Result(Edge)
			dsl.HTTP(func() {
				dsl.POST("/walk")
			})
			dsl.GRPC(func() {})
		})
		dsl.Method("grow", func() {
			dsl.Payload(Tree)
			dsl.Result(Tree)
			dsl.HTTP(func() {
				dsl.POST("/grow")
			})
			dsl.GRPC(func() {})
		})
	})
	dsl.Service("graphrpc", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("walk", func() {
			dsl.Payload(Node)
			dsl.Result(Edge)
			dsl.JSONRPC(func() {})
		})
		dsl.Method("grow", func() {
			dsl.Payload(Tree)
			dsl.Result(Tree)
			dsl.JSONRPC(func() {})
		})
	})
}

// hasRecursiveUserType reports whether any user or result type in root can
// reach itself through its attributes.
func hasRecursiveUserType(root *expr.RootExpr) bool {
	types := make([]expr.UserType, 0, len(root.Types)+len(root.ResultTypes))
	types = append(types, root.Types...)
	for _, rt := range root.ResultTypes {
		types = append(types, rt)
	}
	for _, ut := range types {
		if reachesUserType(ut.Attribute(), ut.ID(), make(map[string]struct{})) {
			return true
		}
	}
	return false
}

func reachesUserType(att *expr.AttributeExpr, target string, seen map[string]struct{}) bool {
	if att == nil {
		return false
	}
	switch dt := att.Type.(type) {
	case expr.UserType:
		if dt.ID() == target {
			return true
		}
		if _, ok := seen[dt.ID()]; ok {
			return false
		}
		seen[dt.ID()] = struct{}{}
		return reachesUserType(dt.Attribute(), target, seen)
	case *expr.Object:
		for _, nat := range *dt {
			if reachesUserType(nat.Attribute, target, seen) {
				return true
			}
		}
	case *expr.Array:
		return reachesUserType(dt.ElemType, target, seen)
	case *expr.Map:
		return reachesUserType(dt.KeyType, target, seen) || reachesUserType(dt.ElemType, target, seen)
	case *expr.Union:
		for _, nat := range dt.Values {
			if reachesUserType(nat.Attribute, target, seen) {
				return true
			}
		}
	}
	return false
}
