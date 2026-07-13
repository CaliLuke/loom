package service

import (
	"bytes"
	"go/format"
	"strings"
	"testing"

	projectiontestutil "github.com/CaliLuke/loom/codegen/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service/testdata"
	"github.com/CaliLuke/loom/expr"
)

func TestViews(t *testing.T) {
	cases := []struct {
		Name string
		DSL  func()
		Code string
	}{
		{"result-with-multiple-views", testdata.ResultWithMultipleViewsDSL, testdata.ResultWithMultipleViewsCode},
		{"result-collection-multiple-views", testdata.ResultCollectionMultipleViewsDSL, testdata.ResultCollectionMultipleViewsCode},
		{"result-with-user-type", testdata.ResultWithUserTypeDSL, testdata.ResultWithUserTypeCode},
		{"result-with-result-type", testdata.ResultWithResultTypeDSL, testdata.ResultWithResultTypeCode},
		{"result-with-recursive-result-type", testdata.ResultWithRecursiveResultTypeDSL, testdata.ResultWithRecursiveResultTypeCode},
		{"result-type-with-custom-fields", testdata.ResultWithCustomFieldsDSL, testdata.ResultWithCustomFieldsCode},
		{"result-with-recursive-collection-of-result-type", testdata.ResultWithRecursiveCollectionOfResultTypeDSL, testdata.ResultWithRecursiveCollectionOfResultTypeCode},
		{"result-with-multiple-methods", testdata.ResultWithMultipleMethodsDSL, testdata.ResultWithMultipleMethodsCode},
		{"result-with-enum-type", testdata.ResultWithEnumTypeDSL, testdata.ResultWithEnumType},
		{"result-with-pkg-path", testdata.ResultWithPkgPathDSL, testdata.ResultWithPkgPathCode},
		{"result-with-oneof-in-result-type", testdata.ResultWithOneOfInResultTypeDSL, testdata.ResultWithOneOfInResultTypeCode},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := codegen.RunDSL(t, c.DSL)
			services := NewServicesData(root)
			require.Len(t, root.Services, 1)
			fs := ViewsFile("github.com/CaliLuke/loom/example", root.Services[0], services)
			require.NotNil(t, fs)
			buf := new(bytes.Buffer)
			for _, s := range fs.AllSections()[1:] {
				require.NoError(t, s.Write(buf))
			}
			bs, err := format.Source(buf.Bytes())
			require.NoError(t, err, buf.String())
			code := string(bs)
			code = strings.ReplaceAll(code, "\r\n", "\n")
			assert.Equal(t, c.Code, code)
		})
	}
}

func TestViewValidationRequiresNestedResultTypeFields(t *testing.T) {
	root := codegen.RunDSL(t, testdata.ResultWithResultTypeDSL)
	services := NewServicesData(root)
	require.Len(t, root.Services, 1)
	fs := ViewsFile("github.com/CaliLuke/loom/example", root.Services[0], services)
	require.NotNil(t, fs)

	buf := new(bytes.Buffer)
	for _, s := range fs.AllSections()[1:] {
		require.NoError(t, s.Write(buf))
	}
	bs, err := format.Source(buf.Bytes())
	require.NoError(t, err, buf.String())
	code := strings.ReplaceAll(string(bs), "\r\n", "\n")

	require.Contains(t, code, "if result.B == nil {\n\t\terr = loom.MergeErrors(err, loom.MissingFieldError(\"b\", \"result\"))")
	require.Contains(t, code, "if result.C == nil {\n\t\terr = loom.MergeErrors(err, loom.MissingFieldError(\"c\", \"result\"))")
}

func TestViewValidationUsesViewRequirednessOverrides(t *testing.T) {
	root := codegen.RunDSL(t, testdata.ViewRequirednessOverridesDSL)
	services := NewServicesData(root)
	fs := ViewsFile("github.com/CaliLuke/loom/example", root.Services[0], services)
	require.NotNil(t, fs)

	var buf bytes.Buffer
	for _, section := range fs.AllSections()[1:] {
		require.NoError(t, section.Write(&buf))
	}
	code := codegen.FormatTestCode(t, "package views\n"+buf.String())

	inherited := generatedFunction(t, code, "ValidateViewRequirednessViewInherited")
	overridden := generatedFunction(t, code, "ValidateViewRequirednessViewOverridden")
	require.Contains(t, inherited, `MissingFieldError("canonical_required", "result")`)
	require.NotContains(t, inherited, `MissingFieldError("canonical_optional", "result")`)
	require.NotContains(t, overridden, `MissingFieldError("canonical_required", "result")`)
	require.Contains(t, overridden, `MissingFieldError("canonical_optional", "result")`)
}

func generatedFunction(t *testing.T, code, name string) string {
	t.Helper()
	start := strings.Index(code, "func "+name+"(")
	require.NotEqual(t, -1, start)
	end := strings.Index(code[start:], "\n}\n")
	require.NotEqual(t, -1, end)
	return code[start : start+end+3]
}

func TestProjectionParity(t *testing.T) {
	cases := []struct {
		Name string
		DSL  func()
	}{
		{"result-with-multiple-views", testdata.ResultWithMultipleViewsDSL},
		{"result-with-result-type", testdata.ResultWithResultTypeDSL},
		{"projection-parity-nested-views", testdata.ProjectionParityNestedViewsDSL},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := codegen.RunDSL(t, c.DSL)
			services := NewServicesData(root)
			require.Len(t, root.Services, 1)
			svc := services.Get(root.Services[0].Name)
			require.NotNil(t, svc)
			assertServiceProjectionParity(t, root.Services[0], svc)
		})
	}
}

func assertServiceProjectionParity(t *testing.T, service *expr.ServiceExpr, data *Data) {
	t.Helper()
	for _, method := range service.Methods {
		rt, ok := method.Result.Type.(*expr.ResultTypeExpr)
		if !ok {
			continue
		}
		projected := projectedTypeDataFor(t, data, rt)
		projectedAttr := &expr.AttributeExpr{Type: projected.Type, Validation: projected.Type.Attribute().Validation}
		projectiontestutil.AssertProjectionParity(t, method.Result, projectedAttr)
		for _, view := range rt.Views {
			projectiontestutil.AssertProjectionViewParity(t, rt, view.Name, projectedAttr)
		}
	}
}

func projectedTypeDataFor(t *testing.T, data *Data, rt *expr.ResultTypeExpr) *ProjectedTypeData {
	t.Helper()
	for _, projected := range data.projectedTypes {
		if projected.Type.ID() == rt.ID() {
			return projected
		}
	}
	t.Fatalf("missing projected type data for result type %q", rt.TypeName)
	return nil
}
