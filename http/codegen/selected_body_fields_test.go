package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	servicedata "github.com/CaliLuke/loom/codegen/service/testdata"
	"github.com/CaliLuke/loom/expr"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/CaliLuke/loom/internal/testingx"
)

// TestSelectedBodyContractsGenerated compiles all body directions and carriers
// together while retaining the client/server and CLI assertions for each case.
func TestSelectedBodyContractsGenerated(t *testing.T) {
	type bodyCase struct {
		name, direction, shape string
		renamed                bool
	}
	cases := make([]bodyCase, 0, 18)
	for _, direction := range []string{"request", "response", "error", "whole", "viewed"} {
		for _, renamed := range []bool{false, true} {
			name := direction + "/ordinary"
			if renamed {
				name = direction + "/renamed"
			}
			cases = append(cases, bodyCase{name, direction, "string", renamed})
		}
	}
	for _, shape := range []string{"named", "optional", "defaulted", "nullable", "array", "object"} {
		cases = append(cases, bodyCase{"request/" + shape, "request", shape, true})
	}
	for _, renamed := range []bool{false, true} {
		name := "viewed/optional/ordinary"
		if renamed {
			name = "viewed/optional/renamed"
		}
		cases = append(cases, bodyCase{name, "viewed", "optional", renamed})
	}
	batch := testingx.NewModuleBatch(t, checkoutPinnedLoomModule(t), "")
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := RunHTTPDSL(t, servicedata.SelectedBodyFieldsDSL("http", tc.direction, tc.shape, tc.renamed))
			if tc.direction == "request" && tc.renamed {
				request := CreateHTTPServices(root).Get("fields").Endpoint("call").Payload.Request
				require.Equal(t, "ActualValue", request.PayloadAttr)
				require.Equal(t, "ActualValue", request.PayloadInit.ReturnTypeAttribute)
			}
			c := batch.Case(t, tc.name, "example.com/bodyfields")
			renderHTTPModuleFiles(t, c.Dir, c.ModulePath+"/gen", root)
			files, err := openapiv3.Files(root)
			require.NoError(t, err)
			renderGeneratedFiles(t, c.Dir, files)
			if tc.direction == "request" || tc.direction == "response" {
				c.WriteFile(t, "wire_test.go", servicedata.SelectedBodyFieldHarness("http", tc.direction, tc.shape, tc.renamed))
			}
		})
	}
	batch.Check(t)
}

func TestSelectedBodyFieldArtifacts(t *testing.T) {
	output := os.Getenv("LOOM_SELECTED_BODY_OUTPUT")
	if output == "" {
		t.Skip("set LOOM_SELECTED_BODY_OUTPUT for isolated artifact comparison")
	}
	for _, direction := range []string{"request", "response", "error", "whole", "viewed"} {
		for _, renamed := range []bool{false, true} {
			name := direction + "-ordinary"
			if renamed {
				name = direction + "-renamed"
			}
			renderSelectedBodyFields(t, filepath.Join(output, "http", name), direction, "string", renamed)
		}
	}
	for _, shape := range []string{"named", "optional", "defaulted", "nullable", "array", "object"} {
		renderSelectedBodyFields(t, filepath.Join(output, "http", shape), "request", shape, true)
	}
}

func renderSelectedBodyFields(t *testing.T, dir, direction, shape string, renamed bool) {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o700))
	root := RunHTTPDSL(t, servicedata.SelectedBodyFieldsDSL("http", direction, shape, renamed))
	renderHTTPModule(t, dir, "example.com/bodyfields", root)
	files, err := openapiv3.Files(root)
	require.NoError(t, err)
	renderGeneratedFiles(t, dir, files)
	path := filepath.Join(dir, "go.mod")
	module, err := os.ReadFile(path)
	require.NoError(t, err)
	module = []byte(strings.Replace(string(module), "github.com/CaliLuke/loom v1.0.0", "github.com/CaliLuke/loom v1.10.0-alpha.9.0.20261013020843-063198a2f427", 1))
	require.NoError(t, os.WriteFile(path, module, 0o600))
}

func TestSelectedBodyFieldMappedIdentity(t *testing.T) {
	field := &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:field:name": []string{"ActualValue"}}, DefaultValue: "fallback"}
	parent := &expr.AttributeExpr{Type: &expr.Object{{Name: "value:wire", Attribute: field}}, Validation: &expr.ValidationExpr{Required: []string{"value:wire"}}}
	body := &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"origin:attribute": []string{"value"}}}
	got := selectBodyField(parent, body)
	require.Equal(t, "value", got.Name)
	require.Equal(t, "value:wire", got.Key)
	require.Equal(t, "ActualValue", got.GoName)
	require.True(t, parent.IsRequired(got.Key))
	require.Equal(t, "fallback", got.Attribute.DefaultValue)
	require.Same(t, parent, selectBodyField(parent, &expr.AttributeExpr{Type: expr.String}).Attribute)
}
