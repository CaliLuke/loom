package ir

import (
	"testing"

	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

func TestDiscriminatorMetadataAdmission(t *testing.T) {
	for _, tc := range []struct {
		name          string
		meta          expr.MetaExpr
		discriminator bool
		wantOptional  bool
		wantFallback  string
		wantError     string
	}{
		{name: "required", discriminator: true},
		{name: "optional missing fallback", discriminator: true, meta: expr.MetaExpr{"openapi:discriminator:optional": {"true"}}, wantError: "requires an explicit defaultMapping"},
		{name: "optional blank fallback", discriminator: true, meta: expr.MetaExpr{"openapi:discriminator:optional": {"true"}, "openapi:discriminator:defaultMapping": {" "}}, wantError: "defaultMapping"},
		{name: "invalid boolean", discriminator: true, meta: expr.MetaExpr{"openapi:discriminator:optional": {"maybe"}}, wantError: "optional must be true or false"},
		{name: "non union", meta: expr.MetaExpr{"openapi:discriminator:optional": {"true"}}, wantError: "requires a union discriminator"},
		{name: "local name", discriminator: true, meta: expr.MetaExpr{"openapi:discriminator:optional": {"true"}, "openapi:discriminator:defaultMapping": {"Fallback"}}, wantOptional: true, wantFallback: "#/components/schemas/Fallback"},
		{name: "local ref", discriminator: true, meta: expr.MetaExpr{"openapi:discriminator:optional": {"true"}, "openapi:discriminator:defaultMapping": {"#/components/schemas/Fallback"}}, wantOptional: true, wantFallback: "#/components/schemas/Fallback"},
		{name: "encoded local ref", discriminator: true, meta: expr.MetaExpr{"openapi:discriminator:optional": {"true"}, "openapi:discriminator:defaultMapping": {"#/components/schemas/Fall%62ack"}}, wantOptional: true, wantFallback: "#/components/schemas/Fallback"},
		{name: "required fallback", discriminator: true, meta: expr.MetaExpr{"openapi:discriminator:optional": {"false"}, "openapi:discriminator:defaultMapping": {"Fallback"}}, wantFallback: "#/components/schemas/Fallback"},
		{name: "missing target", discriminator: true, meta: expr.MetaExpr{"openapi:discriminator:defaultMapping": {"Missing"}}, wantError: "does not resolve"},
		{name: "invalid local path", discriminator: true, meta: expr.MetaExpr{"openapi:discriminator:defaultMapping": {"#/paths/example"}}, wantError: "component schema"},
		{name: "malformed uri", discriminator: true, meta: expr.MetaExpr{"openapi:discriminator:defaultMapping": {"https://example.com/%zz"}}, wantError: "defaultMapping"},
		{name: "external uri", discriminator: true, meta: expr.MetaExpr{"openapi:discriminator:optional": {"true"}, "openapi:discriminator:defaultMapping": {"https://example.com/schemas.json#/$defs/fallback"}}, wantOptional: true, wantFallback: "https://example.com/schemas.json#/$defs/fallback"},
		{name: "relative uri", discriminator: true, meta: expr.MetaExpr{"openapi:discriminator:defaultMapping": {"./schemas.json#/$defs/fallback"}}, wantFallback: "./schemas.json#/$defs/fallback"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := NewAnalyzer(expr.NewRandom("discriminator"), false, WithExampleValue(nil))
			a.schemas["Fallback"] = &Schema{Type: "object"}
			s := &Schema{Required: []string{"kind"}}
			if tc.discriminator {
				s.Discriminator = &Discriminator{PropertyName: "kind"}
			}
			a.applySchemaAnnotations(s, &expr.AttributeExpr{Type: expr.String, Meta: tc.meta}, "", "test union")
			if tc.wantError != "" {
				require.ErrorContains(t, a.Err(), tc.wantError)
				return
			}
			require.NoError(t, a.Err())
			require.Equal(t, tc.wantOptional, s.Discriminator.Optional)
			require.Equal(t, tc.wantFallback, s.Discriminator.DefaultMapping)
			require.Equal(t, !tc.wantOptional, len(s.Required) > 0)
		})
	}
}

func TestDiscriminatorFallbackURICharacters(t *testing.T) {
	for _, value := range []string{`./schemas\fallback.json`, "./schema|fallback.json", "./schema{fallback}.json", "./schema[fallback].json", "./schema.json?q=%zz", "./schema.json#first#second"} {
		t.Run(value, func(t *testing.T) {
			_, err := discriminatorFallback(value)
			require.Error(t, err)
		})
	}
	for _, value := range []string{"./schemas%5Cfallback.json", "./schema%7Cfallback.json", "./schema%7Bfallback%7D.json", "./schema%5Bfallback%5D.json", "https://[::1]/schema.json", "HTTPS://[::1]/schema.json", "urn:example:fallback"} {
		t.Run(value, func(t *testing.T) {
			got, err := discriminatorFallback(value)
			require.NoError(t, err)
			require.Equal(t, value, got)
		})
	}
}
