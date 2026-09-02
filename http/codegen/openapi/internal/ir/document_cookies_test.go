package ir

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

func TestResponseCookieHeaderExamplesAreStable(t *testing.T) {
	cookie := &transportir.Cookie{
		HTTPName:  "session",
		Attribute: &expr.AttributeExpr{Type: expr.String},
	}
	generator := expr.NewRandom("cookies")

	first := responseCookieHeader([]*transportir.Cookie{cookie}, generator, false)
	second := responseCookieHeader([]*transportir.Cookie{cookie}, generator, false)

	require.Equal(t, first.Example, second.Example)
}

func TestResponseCookieHeaderOmitsOnlySynthesizedExamples(t *testing.T) {
	synthesized := &transportir.Cookie{
		HTTPName:  "synthesized",
		Attribute: &expr.AttributeExpr{Type: expr.String},
	}
	authored := &transportir.Cookie{
		HTTPName: "authored",
		Attribute: &expr.AttributeExpr{
			Type:         expr.String,
			UserExamples: []*expr.ExampleExpr{{Value: "authored-value"}},
		},
	}
	generator := expr.NewRandom("cookies")
	generator.Randomizer = nil

	header := responseCookieHeader([]*transportir.Cookie{synthesized, authored}, generator, false)

	require.NotContains(t, header.Examples, synthesized.HTTPName)
	require.Equal(
		t,
		"authored=authored-value",
		header.Examples[authored.HTTPName].Value.Value,
	)
}

func TestResponseCookieHeaderExampleHonorsValueLength(t *testing.T) {
	exactLength := 32
	cookie := &transportir.Cookie{
		HTTPName: "csrftoken",
		Attribute: &expr.AttributeExpr{
			Type: expr.String,
			Validation: &expr.ValidationExpr{
				MinLength: &exactLength,
				MaxLength: &exactLength,
			},
		},
	}

	header := responseCookieHeader([]*transportir.Cookie{cookie}, expr.NewRandom("cookies"), false)
	example, ok := header.Example.(string)
	require.True(t, ok)
	pair := strings.SplitN(example, ";", 2)[0]
	value, ok := strings.CutPrefix(pair, "csrftoken=")
	require.True(t, ok)
	require.Len(t, value, exactLength)
}
