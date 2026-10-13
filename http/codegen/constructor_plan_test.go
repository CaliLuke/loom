package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	servicecodegen "github.com/CaliLuke/loom/codegen/service"
	codegentestdata "github.com/CaliLuke/loom/codegen/testdata"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestConstructorPlanIdentity(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		testdata.ConstructorOwnership("selector", false, false)
	})
	sd := CreateHTTPServices(root).Get("constructors")
	left := sd.Endpoints[0].Result.Responses[0].ServerBody[0].Init
	right := sd.Endpoints[1].Result.Responses[0].ServerBody[0].Init
	require.NotEqual(t, left.constructor.Hash(), right.constructor.Hash())
	copy := *left
	copy.Name = "DifferentName"
	copy.Description = "Different documentation"
	copy.ServerArgs = append([]*InitArgData(nil), left.ServerArgs...)
	arg := *copy.ServerArgs[0]
	arg.Ref = "anotherCall.Projected"
	copy.ServerArgs[0] = &arg
	require.Equal(t, left.constructor.Hash(), newConstructorPlan(&copy, false, true, sd.Service.Scope).Hash())
}

func TestConstructorAllocationReservesOccupiedSuffix(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		testdata.ConstructorOwnership("shared", false, false)
	})
	sd := CreateHTTPServices(root).Get("constructors")
	for _, endpoint := range sd.Endpoints {
		endpoint.Result.Responses[0].ServerBody[0].Init.Name = "NewText"
	}
	sd.ServerBodyAttributeTypes = append(sd.ServerBodyAttributeTypes, &TypeData{VarName: "NewText2", Def: "string"})
	sd.finalizeConstructors()
	require.Equal(t, "NewText", sd.Endpoints[0].Result.Responses[0].ServerBody[0].Init.Name)
	require.Equal(t, "NewText3", sd.Endpoints[1].Result.Responses[0].ServerBody[0].Init.Name)
	require.NotContains(t, sd.constructorReservations(true), "NewText2", "server declarations do not reserve client names")
}

func TestConstructorPlansRetainFileImportIdentity(t *testing.T) {
	root := RunHTTPDSL(t, codegentestdata.UserTypePackageTransportsDSL)
	services := CreateHTTPServices(root)
	for _, service := range services.Expressions.Services {
		require.NoError(t, servicecodegen.SetUserTypeImports("example.com/imports/gen", services.ServicesData.Get(service.Name())))
		original := services.Get(service.Name())
		imports := append([]*codegen.ImportSpec{{Path: "example.com/conflict", Name: "common"}}, original.Service.UserTypeImports...)
		renamed, _ := services.FileData(service.Name(), imports)
		require.NotSame(t, original, renamed)
		for _, client := range []bool{false, true} {
			var before, after []string
			original.walkConstructors(client, func(init *InitData, _ bool) {
				before = append(before, init.Name)
			})
			renamed.walkConstructors(client, func(init *InitData, body bool) {
				after = append(after, init.Name)
				require.Equal(t, newConstructorPlan(init, client, body, renamed.Service.Scope).Hash(), init.constructor.Hash())
			})
			require.Equal(t, before, after, "%s client=%t", service.Name(), client)
		}
	}
}

func TestConstructorPlanReuse(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		testdata.ConstructorOwnership("same", false, false)
	})
	sd := CreateHTTPServices(root).Get("constructors")
	first := sd.Endpoints[0].Result.Responses[0].ServerBody[0].Init
	second := sd.Endpoints[1].Result.Responses[0].ServerBody[0].Init
	require.Equal(t, first.constructor.Hash(), second.constructor.Hash())
	require.Equal(t, "NewText", first.Name)
	require.Equal(t, first.Name, second.Name)
}
