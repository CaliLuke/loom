package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestInformationalServerDeclarations(t *testing.T) {
	root := RunHTTPDSL(t, responseMetadataConstructorDSL)
	data := CreateHTTPServices(root).Get("metadata")
	endpoint := data.Endpoint("read")
	sections := informationalServerSections(endpoint, data)
	require.Len(t, sections, 1)
	actual := codegen.SectionsCode(t, sections)
	require.Contains(t, actual, "func "+endpoint.Informational[0].Send+"(ctx context.Context, value *metadata.Opening) error")
	require.Empty(t, informationalServerSections(data.Endpoint("watch"), data))
}

func TestInformationalNativeValidationRegistry(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		testdata.InformationalServer(false)
	})
	data := CreateHTTPServices(root).Get("selection")
	bodyFunctions := make(map[string]*TypeData)
	for _, typ := range data.ServerBodyAttributeTypes {
		if typ.ValidateDef != "" {
			bodyFunctions[typ.validationFunction.Name] = typ
		}
	}
	require.Contains(t, bodyFunctions, "ValidateLabelRequestBody")
	require.Contains(t, bodyFunctions, "ValidateHTTPHeaderLabelRequestBody")
	var native *TypeData
	for _, typ := range data.outgoingHeaderValidationTypes {
		require.NotContains(t, bodyFunctions, typ.validationFunction.Name)
		require.True(t, data.constructorReservations(false)[typ.validationFunction.Name])
		if typ.Name == "LabelRequestBody" {
			require.Nil(t, native, "one native declaration despite shared references")
			native = typ
		}
	}
	require.NotNil(t, native)
	require.Equal(t, "*selection.LabelRequestBody", native.validationFunction.Ref)
	require.NotEqual(t, "ValidateHTTPHeaderLabelRequestBody", native.validationFunction.Name)
	require.Equal(t, bodyFunctions["ValidateLabelRequestBody"].Name, native.Name)
	var registered []*TypeData
	seen := make(map[string]struct{})
	recordValidatedType(bodyFunctions["ValidateLabelRequestBody"], seen, &registered)
	recordValidatedType(native, seen, &registered)
	recordValidatedType(native, seen, &registered)
	require.Len(t, registered, 2, "native and transport validators have distinct callable identities")
}

func TestInformationalObservationOnlyDoc(t *testing.T) {
	root := RunHTTPDSL(t, testdata.InformationalServerDSL)
	data := CreateHTTPServices(root).Get("selection")
	sections := informationalServerSections(data.Endpoint("read"), data)
	actual := codegen.SectionsCode(t, sections)
	require.Contains(t, actual, "SendReadContinue rejects the observation-only continue declaration")
	require.Contains(t, actual, "SendReadHints sends the declared hints informational response")
}
