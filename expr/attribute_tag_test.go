package expr

import "testing"

func TestAttributeExprHasTag(t *testing.T) {
	var (
		tag = "view"
	)
	cases := map[string]struct {
		attribute *AttributeExpr
		tag       string
		expected  bool
	}{
		"has tag": {
			attribute: &AttributeExpr{
				Type: &Object{
					&NamedAttributeExpr{
						Name: "foo",
						Attribute: &AttributeExpr{
							Meta: MetaExpr{
								tag: []string{"default"},
							},
						},
					},
				},
			},
			tag:      tag,
			expected: true,
		},
		"attribute expr is nil": {
			attribute: nil,
			tag:       tag,
			expected:  false,
		},
		"not object": {
			attribute: &AttributeExpr{
				Type: String,
			},
			tag:      tag,
			expected: false,
		},
		"object but has no tag": {
			attribute: &AttributeExpr{
				Type: &Object{
					&NamedAttributeExpr{
						Name:      "foo",
						Attribute: &AttributeExpr{},
					},
				},
			},
			tag: tag,
		},
	}

	for k, tc := range cases {
		if actual := tc.attribute.HasTag(tc.tag); tc.expected != actual {
			t.Errorf("%s: got %#v, expected %#v", k, actual, tc.expected)
		}
	}
}

func TestAttributeExprHasTagPrefix(t *testing.T) {
	var (
		tag    = "security:apikey:api_key"
		prefix = "security:apikey"
	)
	cases := map[string]struct {
		attribute *AttributeExpr
		prefix    string
		expected  bool
	}{
		"has tag prefix": {
			attribute: &AttributeExpr{
				Type: &Object{
					&NamedAttributeExpr{
						Name: "foo",
						Attribute: &AttributeExpr{
							Meta: MetaExpr{
								tag: []string{"key"},
							},
						},
					},
				},
			},
			prefix:   prefix,
			expected: true,
		},
		"attribute expr is nil": {
			attribute: nil,
			prefix:    prefix,
			expected:  false,
		},
		"not object": {
			attribute: &AttributeExpr{
				Type: String,
			},
			prefix:   prefix,
			expected: false,
		},
		"object but has no tag": {
			attribute: &AttributeExpr{
				Type: &Object{
					&NamedAttributeExpr{
						Name:      "foo",
						Attribute: &AttributeExpr{},
					},
				},
			},
			prefix: prefix,
		},
	}

	for k, tc := range cases {
		if actual := tc.attribute.HasTagPrefix(tc.prefix); tc.expected != actual {
			t.Errorf("%s: got %#v, expected %#v", k, actual, tc.expected)
		}
	}
}

func TestAttributeExprHasDefaultValue(t *testing.T) {
	var (
		object = &Object{
			&NamedAttributeExpr{
				Name: "foo",
				Attribute: &AttributeExpr{
					DefaultValue: 1,
				},
			},
			&NamedAttributeExpr{
				Name:      "bar",
				Attribute: &AttributeExpr{},
			},
		}
	)
	cases := map[string]struct {
		attName  string
		typ      DataType
		expected bool
	}{
		"has default value": {
			attName:  "foo",
			typ:      object,
			expected: true,
		},
		"no default value": {
			attName:  "bar",
			typ:      object,
			expected: false,
		},
		"not object": {
			typ:      Boolean,
			expected: false,
		},
	}

	for k, tc := range cases {
		attribute := AttributeExpr{
			Type: tc.typ,
		}
		if actual := attribute.HasDefaultValue(tc.attName); tc.expected != actual {
			t.Errorf("%s: got %#v, expected %#v", k, actual, tc.expected)
		}
	}
}

func TestValidationExprHasRequiredOnly(t *testing.T) {
	var (
		values           = []any{"foo"}
		pattern          = "^foo$"
		exclusiveMinimum = 1.1
		minimum          = 1.1
		exclusiveMaximum = 2.2
		maximum          = 2.2
		minLength        = 2
		maxLength        = 3
	)
	cases := map[string]struct {
		values           []any
		format           ValidationFormat
		pattern          string
		exclusiveMinimum *float64
		minimum          *float64
		exclusiveMaximum *float64
		maximum          *float64
		minLength        *int
		maxLength        *int
		expected         bool
	}{
		"has required only": {
			expected: true,
		},
		"values is not nil": {
			values:   values,
			expected: false,
		},
		"format is not empty": {
			format:   FormatDate,
			expected: false,
		},
		"pattern is not empty": {
			pattern:  pattern,
			expected: false,
		},
		"exclusiveMinimum is not nil": {
			exclusiveMinimum: &exclusiveMinimum,
			expected:         false,
		},
		"minimum is not nil": {
			minimum:  &minimum,
			expected: false,
		},
		"exclusiveMaximum is not nil": {
			exclusiveMaximum: &exclusiveMaximum,
			expected:         false,
		},
		"maximum is not nil": {
			maximum:  &maximum,
			expected: false,
		},
		"min length is not nil": {
			minLength: &minLength,
			expected:  false,
		},
		"max length is not nil": {
			maxLength: &maxLength,
			expected:  false,
		},
		"complex validation": {
			values:           values,
			format:           FormatDate,
			pattern:          pattern,
			exclusiveMinimum: &exclusiveMinimum,
			minimum:          &minimum,
			exclusiveMaximum: &exclusiveMaximum,
			maximum:          &maximum,
			minLength:        &minLength,
			maxLength:        &maxLength,
			expected:         false,
		},
	}

	for k, tc := range cases {
		validation := &ValidationExpr{
			Values:           tc.values,
			Format:           tc.format,
			Pattern:          tc.pattern,
			ExclusiveMinimum: tc.exclusiveMinimum,
			Minimum:          tc.minimum,
			ExclusiveMaximum: tc.exclusiveMaximum,
			Maximum:          tc.maximum,
			MinLength:        tc.minLength,
			MaxLength:        tc.maxLength,
		}
		if actual := validation.HasRequiredOnly(); tc.expected != actual {
			t.Errorf("%s: got %#v, expected %#v", k, actual, tc.expected)
		}
	}
}

func TestAttributeExprEvalName(t *testing.T) {
	cases := map[string]struct {
		expected string
	}{
		"testcase": {expected: "attribute"},
	}
	for key, testcase := range cases {
		attribute := AttributeExpr{}
		if actual := attribute.EvalName(); actual != testcase.expected {
			t.Errorf("%s: got %#v, expected %#v", key, actual, testcase.expected)
		}
	}
}

func TestAttributeExprValidationValidate(t *testing.T) {
	var (
		max       = 1.0
		exclMax   = 2.0
		min       = 3.0
		ExclMin   = 4.0
		MaxLength = 5
		MinLength = 6
		parent    = &UserTypeExpr{
			AttributeExpr: &AttributeExpr{Type: String},
			TypeName:      "Parent",
		}
	)
	cases := map[string]struct {
		min, max, exclMin, exclMax *float64
		minLen, maxLen             *int
		pattern                    string
		expected                   string
	}{
		"min and max":         {min: &min, max: &max, expected: "attribute: minimum is greater than maximum"},
		"min and exclMax":     {min: &min, exclMax: &exclMax, expected: "attribute: minimum is greater than or equal to exclusive maximum"},
		"exclMin and max":     {exclMin: &ExclMin, max: &max, expected: "attribute: exclusive minimum is greater than or equal to maximum"},
		"exclMin and exclMax": {exclMin: &ExclMin, exclMax: &exclMax, expected: "attribute: exclusive minimum is greater than exclusive maximum"},
		"max and exclMax":     {max: &max, exclMax: &exclMax, expected: "attribute: both maximum and exclusive maximum are defined"},
		"min and exclMin":     {min: &min, exclMin: &ExclMin, expected: "attribute: both minimum and exclusive minimum are defined"},
		"minLen and maxLen":   {minLen: &MinLength, maxLen: &MaxLength, expected: "attribute: min length is greater than max length"},
		"invalid pattern":     {pattern: "[invalid(", expected: `attribute: invalid pattern "[invalid(": error parsing regexp: missing closing ]: ` + "`[invalid(`"},
	}
	for k, tc := range cases {
		validation := &ValidationExpr{
			Minimum:          tc.min,
			Maximum:          tc.max,
			ExclusiveMinimum: tc.exclMin,
			ExclusiveMaximum: tc.exclMax,
			MinLength:        tc.minLen,
			MaxLength:        tc.maxLen,
			Pattern:          tc.pattern,
		}
		if actual := validation.Validate("", parent); actual.Error() != tc.expected {
			t.Errorf("%s: got %#v, expected %#v", k, actual.Error(), tc.expected)
		}
	}
}
