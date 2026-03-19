package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"goa.design/goa/v3/codegen"
	stest "goa.design/goa/v3/codegen/service/testdata"
	dsl "goa.design/goa/v3/dsl"
)

func TestAnalyzeServiceErrorsCarryRemedyMetadataToErrorTypes(t *testing.T) {
	root := codegen.RunDSL(t, stest.ErrorRemedyMethodDSL)
	services := NewServicesData(root)
	svc := services.Get("ErrorRemedyMethod")
	require.NotNil(t, svc)
	require.NotEmpty(t, svc.errorTypes)

	var found bool
	for _, ut := range svc.errorTypes {
		if ut.RemedyCode == "bad_request.fix" {
			require.Equal(t, "The request is invalid.", ut.SafeMessage)
			require.Equal(t, "Correct the payload and retry.", ut.RetryHint)
			found = true
			break
		}
	}

	require.True(t, found)
}

func TestAnalyzeWrapsRawObjectPayloadsWithUniqueSyntheticTypeNames(t *testing.T) {
	root := codegen.RunDSL(t, stest.RawObjectPayloadTypeNameCollisionDSL)
	services := NewServicesData(root)
	svc := services.Get("RawObjectPayloadTypeNameCollision")
	require.NotNil(t, svc)

	method := svc.Method("Foo")
	require.NotNil(t, method)
	require.Equal(t, "FooPayload3", method.Payload)
}

func TestAnalyzeViewedResultsDeduplicateCanonicalTypeButPreserveMethodViews(t *testing.T) {
	root := codegen.RunDSL(t, stest.WithExplicitAndDefaultViewsDSL)
	services := NewServicesData(root)
	svc := services.Get("WithExplicitAndDefaultViews")
	require.NotNil(t, svc)
	require.Len(t, svc.viewedResultTypes, 1)
	require.Len(t, svc.Methods, 2)
	require.NotSame(t, svc.Methods[0].ViewedResult, svc.Methods[1].ViewedResult)
	require.Equal(t, "", svc.Methods[0].ViewedResult.ViewName)
	require.Equal(t, "tiny", svc.Methods[1].ViewedResult.ViewName)
	require.Equal(t, svc.Methods[0].ViewedResult.FullName, svc.Methods[1].ViewedResult.FullName)
}

func TestAnalyzeForceGeneratedTypesRespectServiceFilters(t *testing.T) {
	cases := []struct {
		name     string
		dsl      func()
		service  string
		expected bool
	}{
		{
			name:     "unfiltered force generate",
			dsl:      stest.ForceGenerateTypeDSL,
			service:  "ForceGenerateType",
			expected: true,
		},
		{
			name:     "matching explicit service filter",
			dsl:      stest.ForceGenerateTypeExplicitDSL,
			service:  "ForceGenerateTypeExplicit",
			expected: true,
		},
		{
			name:     "non matching explicit service filter",
			dsl:      forceGenerateTypeMismatchDSL,
			service:  "ForceGenerateTypeMismatch",
			expected: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := codegen.RunDSL(t, c.dsl)
			services := NewServicesData(root)
			svc := services.Get(c.service)
			require.NotNil(t, svc)
			require.Equal(t, c.expected, hasServiceUserType(svc.userTypes, "ForcedType"))
		})
	}
}

func hasServiceUserType(types []*UserTypeData, name string) bool {
	for _, ut := range types {
		if ut.Name == name {
			return true
		}
	}
	return false
}

func forceGenerateTypeMismatchDSL() {
	var _ = dsl.Type("ForcedType", func() {
		dsl.Attribute("a", dsl.String)
		dsl.Meta("type:generate:force", "OtherService")
	})

	dsl.Service("ForceGenerateTypeMismatch", func() {
		dsl.Method("A", func() {})
	})
}
