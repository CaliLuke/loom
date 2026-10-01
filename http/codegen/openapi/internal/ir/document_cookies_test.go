package ir

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

func TestResponseCookieHeaderExamplesAreStable(t *testing.T) {
	generator := expr.NewRandom("cookies")
	cookie := preparedResponseCookie(t, "session", &expr.AttributeExpr{Type: expr.String}, generator)

	first := responseCookieHeader([]*transportir.Cookie{cookie})
	second := responseCookieHeader([]*transportir.Cookie{cookie})

	require.Equal(t, first.Example, second.Example)
}

func TestResponseCookieHeaderOmitsOnlySynthesizedExamples(t *testing.T) {
	generator := expr.NewRandom("cookies")
	generator.Randomizer = nil
	synthesized := preparedResponseCookie(t, "synthesized", &expr.AttributeExpr{Type: expr.String}, generator)
	authored := preparedResponseCookie(t, "authored", &expr.AttributeExpr{
		Type:         expr.String,
		UserExamples: []*expr.ExampleExpr{{Value: "authored-value"}},
	}, generator)

	header := responseCookieHeader([]*transportir.Cookie{synthesized, authored})

	require.NotContains(t, header.Examples, synthesized.HTTPName)
	require.Equal(
		t,
		"authored=authored-value",
		header.Examples[authored.HTTPName].Value.Value,
	)
}

func TestResponseCookieHeaderExampleHonorsValueLength(t *testing.T) {
	exactLength := 32
	attribute := &expr.AttributeExpr{
		Type: expr.String,
		Validation: &expr.ValidationExpr{
			MinLength: &exactLength,
			MaxLength: &exactLength,
		},
	}
	generator := expr.NewRandom("cookies")
	cookie := preparedResponseCookie(t, "csrftoken", attribute, generator)
	header := responseCookieHeader([]*transportir.Cookie{cookie})
	example, ok := header.Example.(string)
	require.True(t, ok)
	pair := strings.SplitN(example, ";", 2)[0]
	value, ok := strings.CutPrefix(pair, "csrftoken=")
	require.True(t, ok)
	require.Len(t, value, exactLength)
}

func TestResponseCookieHeaderFormatsPreparedBytesAsText(t *testing.T) {
	attribute := &expr.AttributeExpr{
		Type: expr.Bytes, UserExamples: []*expr.ExampleExpr{{Value: "plain-text"}},
	}
	cookie := preparedResponseCookie(t, "data", attribute, expr.NewRandom("cookies"))

	header := responseCookieHeader([]*transportir.Cookie{cookie})

	require.Equal(t, "data=plain-text", header.Example)
}

func TestResponseCookieHeaderUsesPreparedRepresentative(t *testing.T) {
	attribute := &expr.AttributeExpr{
		Type:         expr.String,
		UserExamples: []*expr.ExampleExpr{{Value: "prepared-value"}},
	}
	target, err := representation.PrepareStandaloneExamples(attribute, expr.NewRandom("unused"))
	require.NoError(t, err)
	cookie := &transportir.Cookie{HTTPName: "session", Attribute: attribute, Value: target}

	header := responseCookieHeader([]*transportir.Cookie{cookie})

	require.Equal(t, "session=prepared-value", header.Example)
}

func TestResponseCookieHeaderRejectsMissingPreparedValue(t *testing.T) {
	attribute := &expr.AttributeExpr{
		Type:         expr.String,
		UserExamples: []*expr.ExampleExpr{{Value: "must-not-fallback"}},
	}

	require.Panics(t, func() {
		responseCookieHeader([]*transportir.Cookie{{HTTPName: "missing", Attribute: attribute}})
	})
	require.Panics(t, func() {
		responseCookieHeader([]*transportir.Cookie{{
			HTTPName: "unprepared", Attribute: attribute, Value: &transportir.ValueTarget{},
		}})
	})
}

func TestResponseCookieHeaderOmitsInvalidPreparedValue(t *testing.T) {
	cookie := &transportir.Cookie{
		HTTPName: "invalid",
		Attribute: &expr.AttributeExpr{
			Type:         expr.String,
			UserExamples: []*expr.ExampleExpr{{Value: "must-not-fallback"}},
		},
		Value: &transportir.ValueTarget{
			ExamplesPrepared: true,
			Representative:   &transportir.ValueExample{},
		},
	}

	header := responseCookieHeader([]*transportir.Cookie{cookie})

	require.Nil(t, header.Example)
}

func preparedResponseCookie(
	t *testing.T,
	name string,
	attribute *expr.AttributeExpr,
	generator *expr.ExampleGenerator,
) *transportir.Cookie {
	t.Helper()
	target, err := representation.PrepareStandaloneExamples(attribute, generator)
	require.NoError(t, err)
	return &transportir.Cookie{HTTPName: name, Attribute: attribute, Value: target}
}
