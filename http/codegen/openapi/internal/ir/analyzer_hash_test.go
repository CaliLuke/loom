package ir

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestAttributeFingerprintIsCanonical(t *testing.T) {
	t.Parallel()

	first := fingerprintObject(
		&expr.NamedAttributeExpr{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}},
		&expr.NamedAttributeExpr{Name: "age", Attribute: &expr.AttributeExpr{Type: expr.Int}},
	)
	first.Validation = &expr.ValidationExpr{
		Values:   []any{"member", "admin", "member"},
		Required: []string{"name", "age", "name"},
	}
	second := fingerprintObject(
		&expr.NamedAttributeExpr{Name: "age", Attribute: &expr.AttributeExpr{Type: expr.Int}},
		&expr.NamedAttributeExpr{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}},
	)
	second.Validation = &expr.ValidationExpr{
		Values:   []any{"admin", "member"},
		Required: []string{"age", "name"},
	}

	require.Equal(t, fingerprintAttribute(first, false), fingerprintAttribute(second, false))
}

func TestAttributeFingerprintDefinesDuplicateObjectMemberBehavior(t *testing.T) {
	t.Parallel()

	withDuplicate := fingerprintObject(
		&expr.NamedAttributeExpr{Name: "value", Attribute: &expr.AttributeExpr{Type: expr.Int}},
		&expr.NamedAttributeExpr{Name: "value", Attribute: &expr.AttributeExpr{Type: expr.String}},
	)
	lastMember := fingerprintObject(
		&expr.NamedAttributeExpr{Name: "value", Attribute: &expr.AttributeExpr{Type: expr.String}},
	)

	require.Equal(t, fingerprintAttribute(lastMember, false), fingerprintAttribute(withDuplicate, false))
}

func TestAttributeFingerprintUsesProjectedJSONNames(t *testing.T) {
	t.Parallel()

	first := fingerprintObject(&expr.NamedAttributeExpr{
		Name: "value",
		Attribute: &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{
			"struct:tag:json": {"first"},
		}},
	})
	second := fingerprintObject(&expr.NamedAttributeExpr{
		Name: "value",
		Attribute: &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{
			"struct:tag:json": {"second"},
		}},
	})

	require.NotEqual(t, fingerprintAttribute(first, false), fingerprintAttribute(second, false))
}

func TestAttributeFingerprintSeparatesTaggedAndUntaggedUnions(t *testing.T) {
	t.Parallel()

	values := []*expr.NamedAttributeExpr{{Name: "value", Attribute: &expr.AttributeExpr{Type: expr.String}}}
	tagged := &expr.AttributeExpr{Type: &expr.Union{Values: values}}
	untagged := &expr.AttributeExpr{Type: &expr.Union{Values: values, Untagged: true}}

	require.NotEqual(t, fingerprintAttribute(tagged, false), fingerprintAttribute(untagged, false))
}

func TestAttributeFingerprintCanonicalizesMapValues(t *testing.T) {
	t.Parallel()

	first := &expr.AttributeExpr{
		Type: expr.Any,
		Validation: &expr.ValidationExpr{Values: []any{map[string]any{
			"name": "loom",
			"rank": 1,
		}}},
	}
	second := &expr.AttributeExpr{
		Type: expr.Any,
		Validation: &expr.ValidationExpr{Values: []any{map[string]any{
			"rank": 1,
			"name": "loom",
		}}},
	}

	require.Equal(t, fingerprintAttribute(first, false), fingerprintAttribute(second, false))
}

func TestAttributeFingerprintIncludesSchemaTitle(t *testing.T) {
	t.Parallel()

	plain := &expr.AttributeExpr{Type: expr.String}
	titled := &expr.AttributeExpr{Type: expr.String, Title: "Display Name"}

	require.NotEqual(t, fingerprintAttribute(plain, false), fingerprintAttribute(titled, false))
}

func TestAttributeFingerprintIsStableForRecursiveAttributes(t *testing.T) {
	t.Parallel()

	first := recursiveFingerprintAttribute("First")
	second := recursiveFingerprintAttribute("Second")

	require.Equal(t, fingerprintAttribute(first, false), fingerprintAttribute(second, false))
	require.Equal(t, fingerprintAttribute(first, false), fingerprintAttribute(first, false))
}

func TestAttributeFingerprintChangesForEveryValidationField(t *testing.T) {
	t.Parallel()

	float := func(value float64) *float64 {
		return &value
	}
	integer := func(value int) *int {
		return &value
	}

	cases := []struct {
		name       string
		validation *expr.ValidationExpr
	}{
		{name: "enum", validation: &expr.ValidationExpr{Values: []any{"admin"}}},
		{name: "format", validation: &expr.ValidationExpr{Format: expr.FormatEmail}},
		{name: "pattern", validation: &expr.ValidationExpr{Pattern: "^[a-z]+$"}},
		{name: "exclusive minimum", validation: &expr.ValidationExpr{ExclusiveMinimum: float(1)}},
		{name: "minimum", validation: &expr.ValidationExpr{Minimum: float(1)}},
		{name: "maximum", validation: &expr.ValidationExpr{Maximum: float(1)}},
		{name: "exclusive maximum", validation: &expr.ValidationExpr{ExclusiveMaximum: float(1)}},
		{name: "minimum length", validation: &expr.ValidationExpr{MinLength: integer(1)}},
		{name: "maximum length", validation: &expr.ValidationExpr{MaxLength: integer(1)}},
		{name: "required", validation: &expr.ValidationExpr{Required: []string{"value"}}},
	}

	base := fingerprintAttribute(&expr.AttributeExpr{Type: expr.String}, false)
	seen := map[string]string{base: "no validation"}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			fingerprint := fingerprintAttribute(&expr.AttributeExpr{
				Type:       expr.String,
				Validation: testCase.validation,
			}, false)
			if previous, ok := seen[fingerprint]; ok {
				t.Errorf("fingerprint matches %s", previous)
			}
			seen[fingerprint] = testCase.name
		})
	}
}

func TestAttributeFingerprintHasExactVersionedValues(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		attr *expr.AttributeExpr
		want string
	}{
		{
			name: "string",
			attr: &expr.AttributeExpr{Type: expr.String},
			want: "aa96cb3cb12860b2d4833d8b18d7a9898144c39dc283852d9ab7abbf64782091",
		},
		{
			name: "constrained object",
			attr: &expr.AttributeExpr{
				Type: &expr.Object{
					{Name: "id", Attribute: &expr.AttributeExpr{
						Type:       expr.String,
						Validation: &expr.ValidationExpr{Format: expr.FormatUUID},
					}},
				},
				Validation: &expr.ValidationExpr{Required: []string{"id"}},
			},
			want: "3fc1e878dbd5798d586cc312955a7e63b645b6963fe6c8b87c60ea626063a9be",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.want, fingerprintAttribute(testCase.attr, false))
		})
	}
}

func TestAttributeFingerprintDoesNotDuplicateInheritedUserTypeNullability(t *testing.T) {
	t.Parallel()

	base := &expr.UserTypeExpr{
		TypeName: "NullableText",
		AttributeExpr: &expr.AttributeExpr{
			Type:     expr.String,
			Nullable: true,
		},
	}
	wrapper := &expr.AttributeExpr{Type: base, Nullable: true}

	require.Equal(t, fingerprintAttribute(base.Attribute(), false), fingerprintAttribute(wrapper, false))
}

func TestAttributeFingerprintMatchesMaterializedNullableUserType(t *testing.T) {
	t.Parallel()

	widget := &expr.UserTypeExpr{
		TypeName: "Widget",
		AttributeExpr: &expr.AttributeExpr{
			Type: &expr.Object{{
				Name:      "name",
				Attribute: &expr.AttributeExpr{Type: expr.String},
			}},
			Validation: &expr.ValidationExpr{Required: []string{"name"}},
		},
	}
	occurrence := &expr.AttributeExpr{Type: widget, Nullable: true}
	materialized := expr.DupAtt(widget.Attribute())
	materialized.Nullable = true

	require.Equal(t, fingerprintAttribute(materialized, false), fingerprintAttribute(occurrence, false))
}
func TestAttributeFingerprintCanonicalizesEquivalentUserTypeValidation(t *testing.T) {
	t.Parallel()

	base := &expr.UserTypeExpr{
		TypeName: "Role",
		AttributeExpr: &expr.AttributeExpr{
			Type:       expr.String,
			Validation: &expr.ValidationExpr{Values: []any{"admin", "member"}},
		},
	}
	first := &expr.AttributeExpr{
		Type:       base,
		Validation: &expr.ValidationExpr{Values: []any{"admin", "member"}},
	}
	second := &expr.AttributeExpr{
		Type:       base,
		Validation: &expr.ValidationExpr{Values: []any{"member", "admin"}},
	}

	require.False(t, hasUserTypeValidationOverlay(first, base.Attribute()))
	require.False(t, hasUserTypeValidationOverlay(second, base.Attribute()))
	require.Equal(t, fingerprintAttribute(first, false), fingerprintAttribute(second, false))
}
func TestExplicitComponentNameDoesNotChangeAttributeFingerprint(t *testing.T) {
	t.Parallel()

	plain := &expr.AttributeExpr{Type: expr.String}
	named := &expr.AttributeExpr{
		Type: expr.String,
		Meta: expr.MetaExpr{"openapi:typename": []string{"PublicName"}},
	}

	require.Equal(t, fingerprintAttribute(plain, false), fingerprintAttribute(named, false))
}

func TestAnalyzerCollisionNamesAreStableForReorderedInputs(t *testing.T) {
	t.Parallel()

	collisionRef := func(members ...*expr.NamedAttributeExpr) string {
		analyzer := NewAnalyzer(expr.NewRandom("collision"), false)
		analyzer.AnalyzeSchema(&expr.AttributeExpr{Type: &expr.UserTypeExpr{
			TypeName: "Payload",
			AttributeExpr: fingerprintObject(&expr.NamedAttributeExpr{
				Name:      "collision",
				Attribute: &expr.AttributeExpr{Type: expr.Boolean},
			}),
		}})
		candidate := fingerprintObject(members...)
		candidate.Validation = &expr.ValidationExpr{Required: []string{"name", "age"}}
		userType := &expr.UserTypeExpr{TypeName: "Payload", AttributeExpr: candidate}
		first := analyzer.AnalyzeSchema(&expr.AttributeExpr{Type: userType})
		second := analyzer.AnalyzeSchema(&expr.AttributeExpr{Type: userType})
		require.Equal(t, first.Ref, second.Ref)
		return first.Ref
	}

	first := collisionRef(
		&expr.NamedAttributeExpr{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}},
		&expr.NamedAttributeExpr{Name: "age", Attribute: &expr.AttributeExpr{Type: expr.Int}},
	)
	second := collisionRef(
		&expr.NamedAttributeExpr{Name: "age", Attribute: &expr.AttributeExpr{Type: expr.Int}},
		&expr.NamedAttributeExpr{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}},
	)

	require.Equal(t, first, second)
	require.Regexp(t, `^#/components/schemas/Payload_[0-9a-f]{16}$`, first)
}

func fingerprintObject(members ...*expr.NamedAttributeExpr) *expr.AttributeExpr {
	object := expr.Object(members)
	return &expr.AttributeExpr{Type: &object}
}

func recursiveFingerprintAttribute(name string) *expr.AttributeExpr {
	userType := &expr.UserTypeExpr{TypeName: name}
	object := fingerprintObject(&expr.NamedAttributeExpr{
		Name:      "next",
		Attribute: &expr.AttributeExpr{Type: userType},
	})
	userType.AttributeExpr = object
	return &expr.AttributeExpr{Type: userType}
}
