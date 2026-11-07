package service

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service/testdata"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestNullableErrorDescriptorOwnsFailure(t *testing.T) {
	root := codegen.RunDSL(t, testdata.NullableErrorsDSL)
	data := NewServicesData(root).Get("nullable")
	for _, method := range root.Services[0].Methods {
		for _, failure := range method.Errors {
			t.Run(method.Name+"/"+failure.Name, func(t *testing.T) {
				require.True(t, expr.IsNullable(failure.AttributeExpr))
				descriptor := BuildErrorDescriptor(data, failure.Name, failure.AttributeExpr)
				require.NotContains(t, descriptor.Type.Ref, "loom.Nullable[", "a data wrapper cannot be the native error owner")
				require.Contains(t, descriptor.Type.Ref, "*nullable.")
				require.Same(t, failure.AttributeExpr, descriptor.Payload.Attribute)
				require.Contains(t, descriptor.Payload.Ref, "loom.Nullable[")
				require.NotEmpty(t, descriptor.Constructor)
				if method.Name == "inherited" {
					require.Equal(t, "contracts", descriptor.Payload.Package)
					require.Equal(t, "nullable", descriptor.Type.Package)
				}
			})
		}
	}
}

func TestNullableErrorIdentityAndOrdinaryResult(t *testing.T) {
	root := codegen.RunDSL(t, testdata.NullableErrorsDSL)
	data := NewServicesData(root).Get("nullable")
	var refs []string
	for _, method := range root.Services[0].Methods {
		if method.Name == "scalar" {
			require.False(t, expr.IsNullable(method.Result))
			descriptor := BuildResultDescriptor(data, data.Method(method.Name), method.Result)
			errorDescriptor := BuildErrorDescriptor(data, "short", method.Errors[0].AttributeExpr)
			require.NotEqual(t, descriptor.Declared.Name, errorDescriptor.Type.Name)
		}
		if method.Name == "identified" {
			for _, failure := range method.Errors {
				descriptor := BuildErrorDescriptor(data, failure.Name, failure.AttributeExpr)
				refs = append(refs, descriptor.Type.Ref)
				require.Equal(t, descriptor, BuildErrorDescriptor(data, failure.Name, failure.AttributeExpr))
			}
		}
	}
	require.Len(t, refs, 2)
	require.NotEqual(t, refs[0], refs[1], "one nullable payload has distinct declared error owners")
}

func TestNullableDefaultErrorNativeOwner(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.Service("defaults", func() {
			dsl.Method("call", func() {
				dsl.Error("missing", func() {
					dsl.Nullable()
					dsl.Temporary()
				})
			})
		})
	})
	services := NewServicesData(root)
	files := Files("example.com/defaults/gen", root.Services[0], services, make(map[string][]string))
	var output bytes.Buffer
	for _, section := range files[0].AllSections()[1:] {
		require.NoError(t, section.Write(&output))
	}
	code := output.String()
	require.NotContains(t, code, "value.Code")
	require.Contains(t, code, "loom.Nullable[*loom.ServiceError]")
	require.Contains(t, code, "LoomServiceError() *loom.ServiceError")
	require.Contains(t, code, "Unwrap() error")
	require.Contains(t, code, "payload: loom.NullableValue(serr)")
}
