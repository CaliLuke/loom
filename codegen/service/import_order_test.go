package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service/testdata"
	grpctestdata "github.com/CaliLuke/loom/grpc/codegen/testdata"
)

// importOrderRepeats is the number of times each import list is collected.
// Go randomizes the start of every map iteration, so repeated collection in one
// process reliably detects imports emitted in map order even when the design
// yields only two imports.
const importOrderRepeats = 32

func TestImportsAreCollectedInSortedOrder(t *testing.T) {
	const genpkg = "example.com/order/gen"
	cases := []struct {
		name      string
		dsl       func()
		metaTypes []codegen.ImportSpec
		userTypes []codegen.ImportSpec
	}{
		{
			name:      "struct field meta types",
			dsl:       grpctestdata.StructMetaTypeDSL,
			metaTypes: []codegen.ImportSpec{{Path: "flag"}, {Path: "time"}},
		},
		{
			name: "multiple struct pkg paths",
			dsl:  testdata.PkgPathMultipleDSL,
			userTypes: []codegen.ImportSpec{
				{Name: "bar", Path: genpkg + "/bar"},
				{Name: "baz", Path: genpkg + "/baz"},
			},
		},
		{
			name: "multiple convert packages",
			dsl:  testdata.ConvertMultiPkgDSL,
			userTypes: []codegen.ImportSpec{
				{Name: "models", Path: genpkg + "/models"},
				{Name: "types", Path: genpkg + "/types"},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := runDSL(t, tc.dsl)
			svc := root.Services[0]
			data := NewServicesData(root).Get(svc.Name)
			for range importOrderRepeats {
				var metaTypes []codegen.ImportSpec
				for _, m := range svc.Methods {
					for _, spec := range codegen.GetMetaTypeImports(m.Payload) {
						metaTypes = append(metaTypes, *spec)
					}
				}
				require.Equal(t, tc.metaTypes, metaTypes)
				var userTypes []codegen.ImportSpec
				for _, spec := range userTypeImports(genpkg, data) {
					userTypes = append(userTypes, *spec)
				}
				require.Equal(t, tc.userTypes, userTypes)
			}
		})
	}
}
