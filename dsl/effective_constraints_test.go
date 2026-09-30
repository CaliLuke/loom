package dsl

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestNamedEnumRefinementDesignValidation(t *testing.T) {
	tests := []struct {
		name      string
		values    []any
		wantError bool
	}{
		{name: "equal", values: []any{1, 2}},
		{name: "subset", values: []any{2}},
		{name: "partial overlap", values: []any{2, 3}, wantError: true},
		{name: "disjoint", values: []any{3}, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			design := func() {
				base := Type("Base", Int, func() { Enum(1, 2) })
				derived := Type("Derived", base, func() { Enum(test.values...) })
				Service("Service", func() {
					Method("Method", func() { Payload(func() { Attribute("value", derived) }) })
				})
			}
			if !test.wantError {
				expr.RunDSL(t, design)
				return
			}
			err := expr.RunInvalidDSL(t, design)
			require.ErrorContains(t, err, "is not admitted by ancestor")
			require.ErrorContains(t, err, "Base")
		})
	}
}

func TestInheritedDefaultMustSatisfyDerivedEnum(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		base := Type("Base", Int, func() {
			Enum(1, 2)
			Default(1)
		})
		derived := Type("Derived", base, func() { Enum(2) })
		Service("Service", func() {
			Method("Method", func() { Payload(func() { Attribute("value", derived) }) })
		})
	})
	require.ErrorContains(t, err, `default value 1 declared by "Base" violates the effective contract for "Derived"`)
}

func TestNamedEnumRefinementDiagnosticsIncludeOccurrencePath(t *testing.T) {
	tests := []struct {
		name     string
		wrap     func(expr.DataType) expr.DataType
		wantPath string
	}{
		{name: "object", wrap: func(dataType expr.DataType) expr.DataType {
			return &expr.Object{{Name: "value", Attribute: &expr.AttributeExpr{Type: dataType}}}
		}, wantPath: "outer.value"},
		{name: "array", wrap: func(dataType expr.DataType) expr.DataType {
			return ArrayOf(dataType)
		}, wantPath: "outer[]"},
		{name: "map value", wrap: func(dataType expr.DataType) expr.DataType {
			return MapOf(String, dataType)
		}, wantPath: "outer[value]"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				base := Type("Base", Int, func() { Enum(1, 2) })
				derived := Type("Derived", base, func() { Enum(2, 3) })
				Service("Svc", func() {
					Method("M", func() {
						Payload(func() { Attribute("outer", test.wrap(derived)) })
					})
				})
			})
			require.ErrorContains(t, err, test.wantPath)
			require.ErrorContains(t, err, "enum member 3")
		})
	}
}

func TestNamedPatternAndFormatClausesUseConjunction(t *testing.T) {
	expr.RunDSL(t, func() {
		email := Type("Email", String, func() { Format(FormatEmail) })
		hostname := Type("Hostname", email, func() { Format(FormatHostname) })
		Service("Service", func() {
			Method("Method", func() { Payload(hostname) })
		})
	})

	for _, test := range []struct {
		name   string
		design func()
	}{
		{name: "pattern default", design: func() {
			base := Type("Base", String, func() { Pattern("^a") })
			derived := Type("Derived", base, func() {
				Pattern("b$")
				Default("xb")
			})
			Service("Service", func() { Method("Method", func() { Payload(derived) }) })
		}},
		{name: "format default", design: func() {
			base := Type("Base", String, func() { Format(FormatEmail) })
			derived := Type("Derived", base, func() {
				Format(FormatHostname)
				Default("example.com")
			})
			Service("Service", func() { Method("Method", func() { Payload(derived) }) })
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, test.design)
			require.ErrorContains(t, err, "violates the effective contract")
		})
	}
}

func TestReferenceAndExtendCopyExplicitValidationClauses(t *testing.T) {
	var referenced, extended expr.UserType
	expr.RunDSL(t, func() {
		base := Type("Base", func() {
			Attribute("value", String, func() { Pattern("^a") })
		})
		referenced = Type("Referenced", func() {
			Reference(base)
			Attribute("value")
		})
		extended = Type("Extended", func() { Extend(base) })
		Service("Service", func() {
			Method("Reference", func() { Payload(referenced) })
			Method("Extend", func() { Payload(extended) })
		})
	})
	for _, dataType := range []expr.UserType{referenced, extended} {
		field := dataType.Attribute().Find("value")
		require.NotNil(t, field)
		constraints, err := expr.EffectiveConstraintsFor(field)
		require.NoError(t, err)
		require.Equal(t, []string{"^a"}, constraints.Validation().Lowered().Patterns())
	}
}

func TestExtendConjoinsObjectEnumsWithoutChangingReferenceSemantics(t *testing.T) {
	member := func(value string) map[string]any {
		return map[string]any{"value": value}
	}
	for _, test := range []struct {
		name      string
		values    []any
		wantError bool
	}{
		{name: "equal", values: []any{member("base"), member("other")}},
		{name: "subset", values: []any{member("base")}},
		{name: "widening", values: []any{member("base"), member("outside")}, wantError: true},
		{name: "disjoint", values: []any{member("outside")}, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			design := func() {
				base := Type("Base", func() {
					Attribute("value", String)
					Enum(member("base"), member("other"))
				})
				derived := Type("Derived", func() {
					Extend(base)
					Enum(test.values...)
				})
				Service("Service", func() {
					Method("Method", func() { Payload(derived) })
				})
			}
			if test.wantError {
				err := expr.RunInvalidDSL(t, design)
				require.ErrorContains(t, err, "enum member")
				return
			}
			expr.RunDSL(t, design)
		})
	}

	var referenced, extended expr.UserType
	expr.RunDSL(t, func() {
		base := Type("Base", func() {
			Attribute("value", String)
			Enum(member("base"))
		})
		referenced = Type("Referenced", func() {
			Reference(base)
			Attribute("value")
		})
		extended = Type("Extended", func() { Extend(base) })
		Service("Service", func() {
			Method("Referenced", func() { Payload(referenced) })
			Method("Extended", func() { Payload(extended) })
		})
	})
	_, referenceHasEnum := mustEffectiveConstraints(t, referenced.Attribute()).EnumCandidates()
	require.False(t, referenceHasEnum)
	extendedValues, extendHasEnum := mustEffectiveConstraints(t, extended.Attribute()).EnumCandidates()
	require.True(t, extendHasEnum)
	require.Equal(t, []any{member("base")}, extendedValues)
}

func mustEffectiveConstraints(t *testing.T, attribute *expr.AttributeExpr) expr.EffectiveConstraints {
	t.Helper()
	constraints, err := expr.EffectiveConstraintsFor(attribute)
	require.NoError(t, err)
	return constraints
}
