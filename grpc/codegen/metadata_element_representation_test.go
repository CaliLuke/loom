package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	loomcodegen "github.com/CaliLuke/loom/codegen"
	servicecodegen "github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestMetadataElementPhysicalStorage(t *testing.T) {
	root := RunGRPCDSL(t, func() {
		metadataElementDSL(true)
	})
	services := CreateGRPCServices(root)
	endpoint := services.Get("elements").Endpoint("request")
	require.Len(t, endpoint.Request.Metadata, 6)
	for _, field := range endpoint.Request.Metadata {
		t.Run(field.AttributeName, func(t *testing.T) {
			array := expr.AsArray(field.Type)
			require.NotNil(t, array)
			_, nominal := array.ElemType.Type.(expr.UserType)
			require.False(t, nominal, "transport variables must have the primitive element representation")
			original := expr.AsArray(field.FieldType)
			require.NotNil(t, original)
			_, named := original.ElemType.Type.(expr.UserType)
			require.True(t, named, "the service declaration must remain nominal")
		})
	}
}

func TestMetadataElementGeneratedModules(t *testing.T) {
	for _, named := range []bool{false, true} {
		name := map[bool]string{false: "native", true: "named"}[named]
		t.Run(name, func(t *testing.T) {
			root := RunGRPCDSL(t, func() {
				metadataElementDSL(named)
			})
			dir := t.TempDir()
			if output := os.Getenv("LOOM_METADATA_ELEMENTS_OUTPUT"); output != "" {
				dir = filepath.Join(output, name)
				require.NoError(t, os.MkdirAll(dir, 0o750))
			}
			renderGRPCResponseContractModule(t, dir, "example.com/metadataelements", root)
			services := servicecodegen.NewServicesData(root)
			for _, service := range root.Services {
				require.NoError(t, servicecodegen.SetUserTypeImports("example.com/metadataelements/gen", services.Get(service.Name)))
			}
			files := ClientCLIFiles("example.com/metadataelements/gen", NewServicesData(services))
			require.GreaterOrEqual(t, len(files), 2)
			for _, file := range files {
				_, err := file.Render(dir)
				require.NoError(t, err)
			}
			require.NoError(t, os.WriteFile(filepath.Join(dir, "metadata_test.go"), []byte(metadataElementHarness), 0o600))
			if os.Getenv("LOOM_METADATA_ELEMENTS_GENERATE_ONLY") == "1" {
				return
			}
			runGRPCGoCommand(t, dir, "mod", "tidy")
			runGRPCGoCommand(t, dir, "build", "./...")
			runGRPCGoCommand(t, dir, "vet", "./...")
			runGRPCGoCommand(t, dir, "test", "-race", "-count=1", "-v", ".")
		})
	}
}

func TestMetadataElementConstructorProjection(t *testing.T) {
	root := RunGRPCDSL(t, func() {
		metadataElementDSL(true)
	})
	services := CreateGRPCServices(root)
	service := services.Get("elements")
	for _, field := range service.Endpoint("request").Request.Metadata {
		t.Run(field.AttributeName, func(t *testing.T) {
			original := expr.AsArray(field.FieldType)
			primitive := unalias(original.ElemType)
			physical := &expr.Array{ElemType: &expr.AttributeExpr{Type: primitive.Type}}
			plan, err := loomcodegen.InitStructFields([]*loomcodegen.InitArgData{{
				Name: field.VarName, Type: physical, FieldName: field.FieldName, FieldType: field.FieldType,
			}}, "result", "", service.Service.PkgName, service.Service.Scope)
			require.NoError(t, err)
			pkg := service.Service.PkgName
			if location := loomcodegen.UserTypeLocation(original.ElemType.Type); location != nil {
				pkg = service.Service.Scope.PackageName(location)
			}
			name := service.Service.Scope.GoFullTypeRef(original.ElemType, pkg)
			require.Contains(t, plan.Code, name, "shared constructor must restore qualified service element identity")
		})
	}
}

func TestMetadataElementDefaultAndValidationStorage(t *testing.T) {
	root := RunGRPCDSL(t, func() {
		metadataElementDSL(true)
	})
	services := CreateGRPCServices(root)
	found := false
	for _, md := range services.Get("elements").Endpoint("request").Request.Metadata {
		if md.AttributeName != "optional_text" {
			continue
		}
		found = true
		require.Equal(t, []string{"base"}, md.DefaultValue)
		require.Contains(t, md.DefaultLiteral, "[]string")
		require.NotContains(t, md.DefaultLiteral, "Elementtext")
		require.Contains(t, md.Validate, "3")
		require.Contains(t, md.Validate, "8")
	}
	require.True(t, found, "optional default field must be analyzed")
}

func metadataElementDSL(named bool) {
	primitive := []expr.DataType{dsl.Bytes, dsl.String, dsl.Boolean, dsl.UInt32, dsl.Float64}
	names := []string{"raw", "text", "flags", "counts", "ratios"}
	fields := make([]any, len(primitive))
	for i, kind := range primitive {
		fields[i] = kind
		if named {
			fields[i] = dsl.Type("Element"+names[i], kind, func() {
				if kind == dsl.Float64 {
					dsl.Meta("struct:pkg:path", "numeric")
				}
				if kind == dsl.String {
					dsl.MinLength(2)
					dsl.MaxLength(8)
				}
			})
		}
	}
	arrays := make([]*expr.Array, len(fields))
	for i, field := range fields {
		arrays[i] = dsl.ArrayOf(field, func() {
			if primitive[i] == dsl.String {
				dsl.MinLength(3)
				if !named {
					dsl.MaxLength(8)
				}
			}
		})
	}
	var defaultTexts any = arrays[1]
	if named {
		defaultTexts = dsl.Type("DefaultTexts", arrays[1], func() {
			dsl.Default([]string{"base"})
		})
	}
	var rawValues any = arrays[0]
	if named {
		rawValues = dsl.Type("RawValues", arrays[0])
	}
	value := dsl.Type("Value", func() {
		dsl.Field(1, "body", dsl.String)
		for i, name := range names {
			var array any = arrays[i]
			if i == 0 {
				array = rawValues
			}
			dsl.Field(i+2, name, array)
		}
		dsl.Field(7, "optional_text", defaultTexts, func() {
			if !named {
				dsl.Default([]string{"base"})
			}
		})
		dsl.Required("body", "raw", "text", "flags", "counts", "ratios")
	})
	mapping := func() {
		for _, name := range append(names, "optional_text") {
			key := "x-" + name
			if name == "raw" {
				key += "-bin"
			}
			dsl.Attribute(name + ":" + key)
		}
	}
	dsl.Service("elements", func() {
		for _, placement := range []string{"request", "header", "trailer"} {
			dsl.Method(placement, func() {
				dsl.Payload(value)
				dsl.Result(value)
				dsl.GRPC(func() {
					if placement == "request" {
						dsl.Metadata(mapping)
					}
					dsl.Response(dsl.CodeOK, func() {
						if placement == "header" {
							dsl.Headers(mapping)
						} else if placement == "trailer" {
							dsl.Trailers(mapping)
						}
					})
				})
			})
		}
	})
}
