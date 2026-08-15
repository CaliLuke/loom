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
			want: "116abbf0a2f187124e44c4094105d64aa41f78d5d467a2df6b070b414d9b5dea",
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
			want: "8afc24034e477566e9aef8cd678c2b6e399053cbcb0c9f15aef6f89b69a5fd35",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.want, fingerprintAttribute(testCase.attr, false))
		})
	}
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
