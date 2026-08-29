package service

import (
	"bytes"
	"go/format"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service/testdata"
	"github.com/CaliLuke/loom/codegen/testutil"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestService(t *testing.T) {
	cases := []struct {
		Name string
		DSL  func()
	}{
		{"service-name-with-spaces", testdata.NamesWithSpacesDSL},
		{"service-single", testdata.SingleMethodDSL},
		{"service-multiple", testdata.MultipleMethodsDSL},
		{"service-union", testdata.UnionMethodDSL},
		{"service-multi-union", testdata.MultiUnionMethodDSL},
		{"service-union-alias-cross-pkg", testdata.UnionWithAliasCrossPkgDSL},
		{"service-no-payload-no-result", testdata.EmptyMethodDSL},
		{"service-payload-no-result", testdata.EmptyResultMethodDSL},
		{"service-no-payload-result", testdata.EmptyPayloadMethodDSL},
		{"service-payload-result-with-default", testdata.WithDefaultDSL},
		{"service-result-with-multiple-views", testdata.MultipleMethodsResultMultipleViewsDSL},
		{"service-result-with-explicit-and-default-views", testdata.WithExplicitAndDefaultViewsDSL},
		{"service-result-collection-multiple-views", testdata.ResultCollectionMultipleViewsMethodDSL},
		{"service-result-with-other-result", testdata.ResultWithOtherResultMethodDSL},
		{"service-result-with-result-collection", testdata.ResultWithResultCollectionMethodDSL},
		{"service-result-with-dashed-mime-type", testdata.ResultWithDashedMimeTypeMethodDSL},
		{"service-result-with-one-of-type", testdata.ResultWithOneOfTypeMethodDSL},
		{"service-result-with-inline-validation", testdata.ResultWithInlineValidationDSL},
		{"service-service-level-error", testdata.ServiceErrorDSL},
		{"service-error-remedy-method", testdata.ErrorRemedyMethodDSL},
		{"service-custom-errors", testdata.CustomErrorsDSL},
		{"service-custom-errors-custom-field", testdata.CustomErrorsCustomFieldDSL},
		{"service-force-generate-type", testdata.ForceGenerateTypeDSL},
		{"service-force-generate-type-explicit", testdata.ForceGenerateTypeExplicitDSL},
		{"service-streaming-result", testdata.StreamingResultMethodDSL},
		{"service-mixed-results", testdata.MixedResultsEndpointDSL},
		{"service-streaming-result-with-views", testdata.StreamingResultWithViewsMethodDSL},
		{"service-streaming-result-with-explicit-view", testdata.StreamingResultWithExplicitViewMethodDSL},
		{"service-streaming-result-no-payload", testdata.StreamingResultNoPayloadMethodDSL},
		{"service-streaming-payload", testdata.StreamingPayloadMethodDSL},
		{"service-streaming-payload-no-payload", testdata.StreamingPayloadNoPayloadMethodDSL},
		{"service-streaming-payload-no-result", testdata.StreamingPayloadNoResultMethodDSL},
		{"service-streaming-payload-result-with-views", testdata.StreamingPayloadResultWithViewsMethodDSL},
		{"service-streaming-payload-result-with-explicit-view", testdata.StreamingPayloadResultWithExplicitViewMethodDSL},
		{"service-bidirectional-streaming", testdata.BidirectionalStreamingMethodDSL},
		{"service-bidirectional-streaming-no-payload", testdata.BidirectionalStreamingNoPayloadMethodDSL},
		{"service-bidirectional-streaming-result-with-views", testdata.BidirectionalStreamingResultWithViewsMethodDSL},
		{"service-bidirectional-streaming-result-with-explicit-view", testdata.BidirectionalStreamingResultWithExplicitViewMethodDSL},
		{"service-multiple-api-key-security", testdata.MultipleAPIKeySecurityDSL},
		{"service-mixed-and-multiple-api-key-security", testdata.MixedAndMultipleAPIKeySecurityDSL},
		{"service-raw-object-payload-type-name-collision", testdata.RawObjectPayloadTypeNameCollisionDSL},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := codegen.RunDSL(t, c.DSL)
			services := NewServicesData(root)
			require.Len(t, root.Services, 1)
			files := Files("github.com/CaliLuke/loom/example", root.Services[0], services, make(map[string][]string))
			require.Greater(t, len(files), 0)

			// Generate the code
			buf := new(bytes.Buffer)
			for _, s := range files[0].AllSections()[1:] {
				require.NoError(t, s.Write(buf))
			}
			bs, err := format.Source(buf.Bytes())
			require.NoError(t, err, buf.String())
			code := string(bs)

			// Compare with golden file
			testutil.AssertGo(t, "testdata/golden/service_"+c.Name+".go.golden", code)
		})
	}
}

func TestRenderErrorMethodsUsesLoomErrorNameOnly(t *testing.T) {
	code := renderErrorMethods(&UserTypeData{
		Ref:         "*ExampleError",
		Name:        "example_error",
		Description: "example error",
		Type: &expr.UserTypeExpr{
			AttributeExpr: &expr.AttributeExpr{Type: expr.String},
			TypeName:      "ExampleError",
		},
	})

	require.Contains(t, code, ") LoomErrorName() string")
	require.NotContains(t, code, ") ErrorName() string")
}

func TestRenderErrorMethodsUsesInstanceMessage(t *testing.T) {
	tests := []struct {
		name     string
		fields   expr.Object
		required []string
		contains []string
		excludes []string
	}{
		{
			name: "optional message",
			fields: expr.Object{
				&expr.NamedAttributeExpr{Name: "message", Attribute: &expr.AttributeExpr{Type: expr.String}},
			},
			contains: []string{
				`if e != nil && e.Message != nil && *e.Message != ""`,
				`return *e.Message`,
			},
		},
		{
			name: "conventional field priority",
			fields: expr.Object{
				&expr.NamedAttributeExpr{Name: "reason", Attribute: &expr.AttributeExpr{Type: expr.String}},
				&expr.NamedAttributeExpr{Name: "message", Attribute: &expr.AttributeExpr{Type: expr.String}},
			},
			required: []string{"reason", "message"},
			contains: []string{
				`if e != nil && e.Message != ""`,
				`return e.Message`,
				`if e != nil && e.Reason != ""`,
			},
		},
		{
			name: "non-string conventional field",
			fields: expr.Object{
				&expr.NamedAttributeExpr{Name: "message", Attribute: &expr.AttributeExpr{Type: expr.Int}},
			},
			excludes: []string{"e.Message"},
		},
		{
			name: "optional string alias",
			fields: expr.Object{
				&expr.NamedAttributeExpr{
					Name: "detail",
					Attribute: &expr.AttributeExpr{Type: &expr.UserTypeExpr{
						AttributeExpr: &expr.AttributeExpr{Type: expr.String},
						TypeName:      "ErrorDetail",
					}},
				},
			},
			contains: []string{
				`if e != nil && e.Detail != nil && string(*e.Detail) != ""`,
				`return string(*e.Detail)`,
			},
		},
		{
			name: "error name field",
			fields: expr.Object{
				&expr.NamedAttributeExpr{Name: "error", Attribute: &expr.AttributeExpr{
					Type: expr.String,
					Meta: expr.MetaExpr{"struct:error:name": nil},
				}},
			},
			required: []string{"error"},
			excludes: []string{"if e != nil && e.Error"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			errorType := &expr.UserTypeExpr{
				AttributeExpr: &expr.AttributeExpr{
					Type:       &test.fields,
					Validation: &expr.ValidationExpr{Required: test.required},
				},
				TypeName: "ExceptionResponse",
			}
			code := renderErrorMethods(&UserTypeData{
				Ref:         "*ExceptionResponse",
				Name:        "exception_response",
				Description: "Error shape returned by every non-2xx response.",
				Type:        errorType,
			})

			for _, expected := range test.contains {
				require.Contains(t, code, expected)
			}
			for _, unexpected := range test.excludes {
				require.NotContains(t, code, unexpected)
			}
			if test.name == "conventional field priority" {
				require.Less(t, strings.Index(code, "e.Message"), strings.Index(code, "e.Reason"))
			}
			require.Contains(t, code, `return "Error shape returned by every non-2xx response."`)
		})
	}
}

func TestStructPkgPath(t *testing.T) {
	fooPath := filepath.Join("gen", "foo", "foo.go")
	recursiveFooPath := filepath.Join("gen", "foo", "recursive_foo.go")
	barPath := filepath.Join("gen", "bar", "bar.go")
	bazPath := filepath.Join("gen", "baz", "baz.go")
	cases := []struct {
		Name      string
		DSL       func()
		TypeFiles []string
	}{
		{"none", testdata.SingleMethodDSL, nil},
		{"single", testdata.PkgPathDSL, []string{fooPath}},
		{"array", testdata.PkgPathArrayDSL, []string{fooPath}},
		{"recursive", testdata.PkgPathRecursiveDSL, []string{fooPath, recursiveFooPath}},
		{"multiple", testdata.PkgPathMultipleDSL, []string{barPath, bazPath}},
		{"nopkg", testdata.PkgPathNoDirDSL, nil},
		{"dupes", testdata.PkgPathDupeDSL, []string{fooPath}},
		{"payload_attribute", testdata.PkgPathPayloadAttributeDSL, []string{fooPath}},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			userTypePkgs := make(map[string][]string)
			root := codegen.RunDSL(t, c.DSL)
			services := NewServicesData(root)
			files := Files("github.com/CaliLuke/loom/example", root.Services[0], services, userTypePkgs)

			// Check file count
			expectedFiles := len(c.TypeFiles) + 1
			require.Len(t, files, expectedFiles, "unexpected number of files")

			// First file is always the service file
			buf := new(bytes.Buffer)
			for _, s := range files[0].AllSections()[1:] {
				require.NoError(t, s.Write(buf))
			}
			bs, err := format.Source(buf.Bytes())
			require.NoError(t, err)
			testutil.AssertGo(t, "testdata/golden/pkg_path_"+c.Name+"_service.go.golden", string(bs))

			// Type files
			for i, typeFile := range c.TypeFiles {
				buf := new(bytes.Buffer)
				for _, s := range files[i+1].AllSections()[1:] {
					require.NoError(t, s.Write(buf))
				}
				bs, err := format.Source(buf.Bytes())
				require.NoError(t, err)
				goldenName := filepath.Base(typeFile)
				testutil.AssertGo(t, "testdata/golden/pkg_path_"+c.Name+"_"+goldenName+".golden", string(bs))
			}

			// For dupes case, test the second service
			if c.Name == "dupes" && len(root.Services) > 1 {
				files = Files("github.com/CaliLuke/loom/example", root.Services[1], services, userTypePkgs)
				require.Len(t, files, 1)
				buf := new(bytes.Buffer)
				for _, s := range files[0].AllSections()[1:] {
					require.NoError(t, s.Write(buf))
				}
				bs, err := format.Source(buf.Bytes())
				require.NoError(t, err)
				testutil.AssertGo(t, "testdata/golden/pkg_path_"+c.Name+"_service2.go.golden", string(bs))
			}
		})
	}
}

func TestStructPkgPath_UnionImportsJSON(t *testing.T) {
	root := codegen.RunDSL(t, testdata.PkgPathUnionDSL)
	services := NewServicesData(root)
	require.Len(t, root.Services, 1)

	files := Files("github.com/CaliLuke/loom/example", root.Services[0], services, make(map[string][]string))
	require.GreaterOrEqual(t, len(files), 2, "expected at least service.go + one struct:pkg:path file")

	var typeFile *codegen.File
	for _, f := range files {
		if strings.HasSuffix(f.Path, filepath.Join("gen", "types", "type_with_union.go")) {
			typeFile = f
			break
		}
	}
	require.NotNil(t, typeFile, "expected generated type file for struct:pkg:path type_with_union")

	buf := new(bytes.Buffer)
	for _, s := range typeFile.AllSections() {
		require.NoError(t, s.Write(buf))
	}
	code := buf.String()
	require.Contains(t, code, "\"encoding/json/jsontext\"", "expected jsontext import in generated file:\n%s", code)
	require.Contains(t, code, "json \"encoding/json/v2\"", "expected JSON v2 import in generated file:\n%s", code)
}

func TestStructPkgPath_UnionJSONFieldBranchesGenerateAliases(t *testing.T) {
	root := codegen.RunDSL(t, testdata.PkgPathUnionJSONFieldDSL)
	services := NewServicesData(root)
	require.Len(t, root.Services, 1)

	files := Files("github.com/CaliLuke/loom/example", root.Services[0], services, make(map[string][]string))
	require.GreaterOrEqual(t, len(files), 2, "expected at least service.go + one struct:pkg:path file")

	var typeFile *codegen.File
	for _, f := range files {
		if strings.HasSuffix(f.Path, filepath.Join("gen", "types", "type_with_json_field_union.go")) {
			typeFile = f
			break
		}
	}
	require.NotNil(t, typeFile, "expected generated type file for struct:pkg:path type_with_json_field_union")

	buf := new(bytes.Buffer)
	for _, s := range typeFile.AllSections() {
		require.NoError(t, s.Write(buf))
	}
	code := buf.String()
	require.Regexp(t, `(?m)^\s*A\s+ValuesA$`, code, "expected union field A to use generated alias type:\n%s", code)
	require.Regexp(t, `(?m)^\s*B\s+ValuesB$`, code, "expected union field B to use generated alias type:\n%s", code)

	var hasValuesAFile, hasValuesBFile bool
	for _, f := range files {
		if strings.HasSuffix(f.Path, filepath.Join("gen", "types", "values_a.go")) {
			hasValuesAFile = true
		}
		if strings.HasSuffix(f.Path, filepath.Join("gen", "types", "values_b.go")) {
			hasValuesBFile = true
		}
	}
	require.True(t, hasValuesAFile, "expected generated alias file in struct:pkg:path package: gen/types/values_a.go")
	require.True(t, hasValuesBFile, "expected generated alias file in struct:pkg:path package: gen/types/values_b.go")
}

func TestServiceDataImportsAreCachedAndDeduped(t *testing.T) {
	root := codegen.RunDSL(t, serviceDataImportCacheDSL)
	services := NewServicesData(root)
	require.Len(t, root.Services, 1)

	data := services.Get(root.Services[0].Name)
	SetUserTypeImports("github.com/CaliLuke/loom/example", data)
	SetUserTypeImports("github.com/CaliLuke/loom/example", data)
	require.Len(t, data.UserTypeImports, 1)

	header := codegen.Header("cached imports", "cache", nil)
	AddServiceDataMetaTypeImports(header, data)
	AddServiceDataMetaTypeImports(header, data)
	AddUserTypeImports(header, data)
	AddUserTypeImports(header, data)

	imports := codegen.HeaderSectionData(header).Imports
	seen := make(map[codegen.ImportSpec]struct{}, len(imports))
	for _, imp := range imports {
		if _, ok := seen[*imp]; ok {
			t.Fatalf("duplicate import %#v in %v", *imp, imports)
		}
		seen[*imp] = struct{}{}
	}
	require.Contains(t, seen, codegen.ImportSpec{Path: "encoding/json/jsontext"})
	require.Contains(t, seen, codegen.ImportSpec{Name: "types", Path: "github.com/CaliLuke/loom/example/types"})
}

var serviceDataImportCacheDSL = func() {
	var CachedPayload = dsl.Type("CachedPayload", func() {
		dsl.Meta("struct:pkg:path", "types")
		dsl.Attribute("raw", dsl.String, func() {
			dsl.Meta("struct:field:type", "jsontext.Value", "encoding/json/jsontext")
		})
	})

	dsl.Service("cache", func() {
		dsl.Method("show", func() {
			dsl.Payload(CachedPayload)
			dsl.Result(CachedPayload)
		})
	})
}
