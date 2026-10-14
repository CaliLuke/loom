package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func TestRawTextValidationCarrier(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		named, custom, format bool
	}{
		{name: "direct", custom: true, format: true},
		{name: "without format", custom: true},
		{name: "inherited", named: true, custom: true, format: true},
		{name: "native", format: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			minimum, maximum := 2, 20
			base := &expr.AttributeExpr{
				Type: expr.String,
				Validation: &expr.ValidationExpr{
					MinLength:      &minimum,
					Pattern:        "^[0-9.]+$",
					PatternClauses: []string{"^[1-9]"},
					MaxLength:      &maximum,
					Values:         []any{"1.2.3.4", "2.3.4.5"},
				},
			}
			if tc.format {
				base.Validation.Format = expr.FormatIP
			}
			attribute := base
			if tc.named {
				attribute = &expr.AttributeExpr{Type: &expr.UserTypeExpr{
					TypeName: "Token", UID: "raw-carrier-token", AttributeExpr: base,
				}}
			}
			if tc.custom {
				attribute.Meta = expr.MetaExpr{"struct:field:type": {"scalar.Token", "example.com/scalar"}}
			}
			original := expr.DupAtt(attribute)
			raw := rawTextValidationAttribute(attribute)
			require.Equal(t, expr.String, raw.Type)
			require.Empty(t, raw.Meta["struct:field:type"])
			rules, err := expr.EffectiveConstraintsFor(raw)
			require.NoError(t, err)
			validation := rules.Validation().Lowered()
			require.Empty(t, validation.Formats())
			require.Equal(t, []string{"^[0-9.]+$", "^[1-9]"}, validation.Patterns())
			require.Equal(t, 2, *validation.MinLength)
			require.Equal(t, 20, *validation.MaxLength)
			require.Equal(t, [][]any{{"1.2.3.4", "2.3.4.5"}}, validation.Enums())
			ctx := codegen.NewAttributeContext(false, false, false, "", codegen.NewNameScope())
			rendered := codegen.AttributeValidationCode(raw, nil, ctx, true, false, "rawValue", "token")
			require.Contains(t, rendered, "utf8.RuneCountInString(rawValue)")
			require.NotContains(t, rendered, "loom.StringValue")
			require.NotContains(t, rendered, "loom.ValidateFormat")
			require.Equal(t, original.Meta, attribute.Meta)
			require.Equal(t, original.Validation, attribute.Validation)
			if tc.format {
				require.Equal(t, expr.ValidationFormat(expr.FormatIP), base.Validation.Format)
			} else {
				require.Empty(t, base.Validation.Format)
			}
			*validation.MinLength = 99
			require.Equal(t, 2, *base.Validation.MinLength)
		})
	}
}
