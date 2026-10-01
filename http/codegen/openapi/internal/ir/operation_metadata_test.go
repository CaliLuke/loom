package ir

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
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
