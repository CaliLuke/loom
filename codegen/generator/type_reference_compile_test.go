package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	dsl "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/testingx"
)

// TestTypeReferenceDesignsCompile builds and vets the service and transport
// packages generated for designs whose attributes reference user types that
// the DSL has not evaluated yet at the point of reference.
func TestTypeReferenceDesignsCompile(t *testing.T) {
	cases := []struct {
		Name string
		DSL  func()
	}{
		{"union-on-recursive-inline-object-cycle", unionOnRecursiveInlineObjectCycleDSL},
		{"later-declared-types-refined-by-attribute-dsl", laterDeclaredTypesRefinedByAttributeDSL},
		{"error-refining-shared-type", errorRefiningSharedTypeDSL},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			compileAndVetDesign(t, c.DSL)
		})
	}
}

func compileAndVetDesign(t *testing.T, design func()) {
	t.Helper()
	root := codegen.RunDSL(t, design)
	roots := []eval.Root{root}
	genpkg := "example.com/unioncycle/gen"

	serviceFiles, err := Service(genpkg, roots)
	require.NoError(t, err)
	transportFiles, err := Transport(genpkg, roots)
	require.NoError(t, err)

	dir := t.TempDir()
	for _, file := range mergeFilesByPath(append(serviceFiles, transportFiles...)) {
		_, err := file.Render(dir)
		require.NoError(t, err, file.Path)
	}
	source := loomModuleSource(t)
	goMod := fmt.Sprintf("module example.com/unioncycle\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v0.0.0\n\nreplace github.com/CaliLuke/loom => %s\n", source)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))
	_, err = testingx.RunCmd(dir, "go", "mod", "tidy")
	require.NoError(t, err)
	output, err := testingx.RunCmd(dir, "go", "build", "./...")
	require.NoError(t, err, output)
	output, err = testingx.RunCmd(dir, "go", "vet", "./...")
	require.NoError(t, err, output)
}

func unionOnRecursiveInlineObjectCycleDSL() {
	dsl.API("unioncycle", func() {
		dsl.JSONRPC(func() {})
	})
	var Node = dsl.Type("Node", func() {
		dsl.Field(1, "holder", func() {
			dsl.OneOf("choice", func() {
				dsl.Field(1, "br", func() {
					dsl.Field(1, "branches", dsl.ArrayOf("Node"))
				})
			})
		})
	})
	dsl.Service("tree", func() {
		dsl.Method("walk", func() {
			dsl.Payload(Node)
			dsl.Result(Node)
			dsl.HTTP(func() {
				dsl.POST("/walk")
			})
			dsl.GRPC(func() {})
		})
	})
	dsl.Service("treerpc", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("walk", func() {
			dsl.Payload(Node)
			dsl.Result(Node)
			dsl.JSONRPC(func() {})
		})
	})
}

func laterDeclaredTypesRefinedByAttributeDSL() {
	dsl.API("latertypes", func() {
		dsl.JSONRPC(func() {})
	})
	var byVariable expr.UserType
	var Holder = dsl.Type("Holder", func() {
		dsl.Field(1, "direct", "Later", func() {
			dsl.Description("described locally")
		})
		dsl.Field(2, "variable", byVariable, func() {
			dsl.Description("described locally")
		})
		dsl.Field(3, "list", dsl.ArrayOf("LaterElem"), func() {
			dsl.MinLength(1)
		})
		dsl.Field(4, "index", dsl.MapOf(dsl.String, "LaterValue"), func() {
			dsl.MinLength(1)
		})
		dsl.Field(5, "strict", "LaterStrict", func() {
			dsl.Required("name")
		})
	})
	dsl.Type("Later", func() {
		dsl.Field(1, "name", dsl.String)
	})
	byVariable = dsl.Type("LaterVariable", func() {
		dsl.Field(1, "label", dsl.String)
	})
	dsl.Type("LaterElem", func() {
		dsl.Field(1, "age", dsl.Int)
	})
	dsl.Type("LaterValue", func() {
		dsl.Field(1, "weight", dsl.Float64)
	})
	dsl.Type("LaterStrict", func() {
		dsl.Field(1, "name", dsl.String)
	})
	dsl.Service("holder", func() {
		dsl.Method("show", func() {
			dsl.Payload(Holder)
			dsl.Result(Holder)
			dsl.HTTP(func() {
				dsl.POST("/show")
			})
			dsl.GRPC(func() {})
		})
	})
	dsl.Service("holderrpc", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("show", func() {
			dsl.Payload(Holder)
			dsl.Result(Holder)
			dsl.JSONRPC(func() {})
		})
	})
}

func errorRefiningSharedTypeDSL() {
	dsl.API("errorrefine", func() {})
	var Shared = dsl.Type("Shared", func() {
		dsl.Field(1, "name", dsl.String)
		dsl.Field(2, "detail", dsl.String)
	})
	dsl.Service("errsvc", func() {
		dsl.Method("show", func() {
			dsl.Payload(Shared)
			dsl.Result(Shared)
			dsl.Error("bad", Shared, func() {
				dsl.Required("name")
			})
			dsl.HTTP(func() {
				dsl.POST("/show")
				dsl.Response("bad", dsl.StatusBadRequest)
			})
			dsl.GRPC(func() {
				dsl.Response("bad", dsl.CodeInvalidArgument)
			})
		})
	})
}
