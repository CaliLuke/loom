package ir

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestParamForDefaultsAllowEmptyValueByLocation(t *testing.T) {
	tests := []struct {
		location string
		want     bool
	}{
		{location: "path", want: false},
		{location: "query", want: true},
		{location: "header", want: false},
		{location: "cookie", want: false},
	}

	for _, test := range tests {
		t.Run(test.location, func(t *testing.T) {
			attribute := &expr.AttributeExpr{Type: expr.String}
			generator := expr.NewRandom("operation-metadata-test")
			parameter := paramFor(
				attribute,
				preparedOperationValue(t, attribute, generator),
				"value",
				test.location,
				generator,
			)

			require.Equal(t, test.want, parameter.Value.AllowEmptyValue)
		})
	}
}

func TestParamForHonorsOpenAPIAllowEmptyValueMetadata(t *testing.T) {
	attribute := &expr.AttributeExpr{
		Type: expr.String,
		Meta: expr.MetaExpr{
			"openapi:allowEmptyValue": {"false"},
		},
	}

	generator := expr.NewRandom("operation-metadata-test")
	parameter := paramFor(attribute, preparedOperationValue(t, attribute, generator),
		"value", "query", generator)

	require.False(t, parameter.Value.AllowEmptyValue)
}

func TestParamForIgnoresOpenAPIAllowEmptyValueMetadataOutsideQuery(t *testing.T) {
	attribute := &expr.AttributeExpr{
		Type: expr.String,
		Meta: expr.MetaExpr{
			"openapi:allowEmptyValue": {"true"},
		},
	}

	generator := expr.NewRandom("operation-metadata-test")
	parameter := paramFor(attribute, preparedOperationValue(t, attribute, generator),
		"value", "header", generator)

	require.False(t, parameter.Value.AllowEmptyValue)
}

func TestParamForKeepsPresentationMetadataOffSchema(t *testing.T) {
	attribute := &expr.AttributeExpr{
		Type:         expr.String,
		Description:  "API version of the request.",
		DefaultValue: "1.0",
		UserExamples: []*expr.ExampleExpr{
			{Value: "1.0"},
		},
	}

	generator := expr.NewRandom("operation-metadata-test")
	parameter := paramFor(attribute, preparedOperationValue(t, attribute, generator),
		"api-version", "header", generator)

	require.Equal(t, "API version of the request.", parameter.Value.Description)
	require.Equal(t, "1.0", parameter.Value.Example)
	require.Empty(t, parameter.Value.Schema.Description)
	require.Nil(t, parameter.Value.Schema.Example)
}

func preparedOperationValue(
	t *testing.T,
	attribute *expr.AttributeExpr,
	generator *expr.ExampleGenerator,
) *transportir.ValueTarget {
	t.Helper()
	target, err := representation.PrepareStandaloneExamples(attribute, generator)
	require.NoError(t, err)
	return target
}

func TestCanonicalOperationIDComponent(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "service name", input: "test service", expected: "test_service"},
		{name: "camel case", input: "OAuth2UserInfo", expected: "oauth2_user_info"},
		{name: "path-like", input: "/assets/{*filepath}", expected: "assets_filepath"},
		{name: "punctuation-only", input: "{}/*-", expected: "operation"},
		{name: "latin accents", input: "ÉtéFoo", expected: "été_foo"},
		{name: "cjk", input: "日本", expected: "日本"},
		{name: "other cjk", input: "中国", expected: "中国"},
		{name: "cjk between words", input: "Foo日本Bar", expected: "foo日本_bar"},
		{name: "upper without lower", input: "ϒϒA", expected: "ϒϒ_a"},
		{name: "title case", input: "ǅemal", expected: "ǆemal"},
		{name: "combining mark", input: "CaféBar", expected: "café_bar"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, canonicalOperationIDComponent(tc.input))
		})
	}
}

func TestParseOperationIDTemplateKeepsCaselessNamesDistinct(t *testing.T) {
	first := ParseOperationIDTemplate("{service}/{method}", "服务", "日本", 0)
	second := ParseOperationIDTemplate("{service}/{method}", "服务", "中国", 0)
	require.Equal(t, "服务/日本", first)
	require.Equal(t, "服务/中国", second)
}

func TestParseOperationIDTemplatePreservesStaticLiteral(t *testing.T) {
	require.Equal(t, "widgets-list", ParseOperationIDTemplate("widgets-list", "widgets", "list", 1))
	require.Equal(t, "widgets{literal}", ParseOperationIDTemplate("widgets{literal}", "widgets", "list", 1))
}

func TestBuildRouteOperationSelectsExpandedServiceBasePath(t *testing.T) {
	root := codegen.RunDSL(t, testdata.ServerMultiBasesDSL)
	service := root.API.HTTP.Services[0]
	route := service.HTTPEndpoints[0].Routes[0]
	bodies := BuildBodyTypes(root.API, root.Types, root.ResultTypes)
	endpointBodies := bodies.Services[service.Name()][service.HTTPEndpoints[0].Name()]

	operation, err := BuildRouteOperation(
		route,
		"/base_2/{id}",
		endpointBodies,
		root.API.ExampleGenerator,
		root.API.Meta,
		false,
	)
	require.NoError(t, err)
	require.Equal(t, "service_multi_bases.method_multi_bases.1", operation.OperationID)

	_, err = BuildRouteOperation(
		route,
		"/base_3/{id}",
		endpointBodies,
		root.API.ExampleGenerator,
		root.API.Meta,
		false,
	)
	require.ErrorContains(t, err, `HTTP path "/base_3/{id}" is not emitted`)
}
