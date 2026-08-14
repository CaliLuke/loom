package ir

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
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
			parameter := paramFor(
				&expr.AttributeExpr{Type: expr.String},
				"value",
				test.location,
				false,
				expr.NewRandom("operation-metadata-test"),
				false,
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

	parameter := paramFor(attribute, "value", "query", false, expr.NewRandom("operation-metadata-test"), false)

	require.False(t, parameter.Value.AllowEmptyValue)
}

func TestParamForIgnoresOpenAPIAllowEmptyValueMetadataOutsideQuery(t *testing.T) {
	attribute := &expr.AttributeExpr{
		Type: expr.String,
		Meta: expr.MetaExpr{
			"openapi:allowEmptyValue": {"true"},
		},
	}

	parameter := paramFor(attribute, "value", "header", false, expr.NewRandom("operation-metadata-test"), false)

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

	parameter := paramFor(attribute, "api-version", "header", false, expr.NewRandom("operation-metadata-test"), false)

	require.Equal(t, "API version of the request.", parameter.Value.Description)
	require.Equal(t, "1.0", parameter.Value.Example)
	require.Empty(t, parameter.Value.Schema.Description)
	require.Nil(t, parameter.Value.Schema.Example)
}
