package expr_test

import (
	"reflect"
	"regexp"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/expr/testdata"
)

func TestByPattern(t *testing.T) {
	cases := []struct {
		Name           string
		Pattern        string
		ExpectedMaxLen int
	}{
		{"not-a-regexp", "foo", 3},
		{"max-len", "foo[a-z]+", 9},
		{"max-len-2", "^/api/example/[0-9]+$", 19},
	}
	r := expr.NewRandom("test")
	for _, k := range cases {
		t.Run(k.Name, func(t *testing.T) {
			val := &expr.ValidationExpr{Pattern: k.Pattern}
			att := expr.AttributeExpr{Validation: val}

			example := att.Example(r).(string)

			if match, _ := regexp.MatchString(k.Pattern, example); !match {
				t.Errorf("got %s, expected a match for %s", example, k.Pattern)
			}
			if utf8.RuneCountInString(example) > k.ExpectedMaxLen {
				t.Errorf("got %s (len %d) exceeded expected len of %d", example, len(example), k.ExpectedMaxLen)
			}
		})
	}
}

func TestByPatternPrefersPrintableASCII(t *testing.T) {
	pattern := `^\S+$`
	attribute := expr.AttributeExpr{
		Type:       expr.String,
		Validation: &expr.ValidationExpr{Pattern: pattern},
	}
	random := &expr.ExampleGenerator{Randomizer: intRandomizer{Value: 2}}

	example := attribute.Example(random).(string)

	if !regexp.MustCompile(pattern).MatchString(example) {
		t.Errorf("got %q, expected a match for %s", example, pattern)
	}
	for _, candidate := range example {
		if !unicode.IsPrint(candidate) || unicode.IsControl(candidate) {
			t.Errorf("got non-printable pattern example %q", example)
		}
		if candidate > unicode.MaxASCII {
			t.Errorf("got non-ASCII pattern example %q", example)
		}
	}
}

func TestByPatternUsesPrintableUnicodeFallback(t *testing.T) {
	pattern := `^[\x00-\x1fα]+$`
	attribute := expr.AttributeExpr{
		Type:       expr.String,
		Validation: &expr.ValidationExpr{Pattern: pattern},
	}
	random := &expr.ExampleGenerator{Randomizer: intRandomizer{Value: 0}}

	example := attribute.Example(random).(string)

	if !regexp.MustCompile(pattern).MatchString(example) {
		t.Errorf("got %q, expected a match for %s", example, pattern)
	}
	for _, candidate := range example {
		if !unicode.IsPrint(candidate) || unicode.IsControl(candidate) {
			t.Errorf("got non-printable Unicode pattern example %q", example)
		}
		if candidate != 'α' {
			t.Errorf("got %q, expected sparse printable fallback α", example)
		}
	}
}

func TestByFormatUUID(t *testing.T) {
	val := &expr.ValidationExpr{Format: expr.FormatUUID}
	att := expr.AttributeExpr{Validation: val}
	r := expr.NewRandom("test")
	example := att.Example(r).(string)
	if !regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`).MatchString(example) {
		t.Errorf("got %s, expected a match with `[0-9A-F]{8}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{12}`", example)
	}
}

func TestExampleOmitsUnsatisfiableFractionalIntegerBounds(t *testing.T) {
	minimum := 0.2
	maximum := 0.7
	attribute := expr.AttributeExpr{
		Type: expr.Int,
		Validation: &expr.ValidationExpr{
			Minimum: &minimum,
			Maximum: &maximum,
		},
	}

	if example := attribute.Example(expr.NewRandom("test")); example != nil {
		t.Errorf("expected an omitted example, got %#v", example)
	}
}

func TestNewLengthWithOnlySmallMaxLengthDoesNotGoNegative(t *testing.T) {
	maxLength := 1
	attribute := expr.AttributeExpr{
		Type: expr.String,
		Validation: &expr.ValidationExpr{
			MaxLength: &maxLength,
		},
	}
	random := &expr.ExampleGenerator{Randomizer: intRandomizer{Value: 2}}

	if got := expr.NewLength(&attribute, random); got != 0 {
		t.Fatalf("expected length to clamp to zero, got %d", got)
	}
}

func TestExampleWithOnlySmallMaxLengthDoesNotPanic(t *testing.T) {
	maxLength := 1
	cases := map[string]struct {
		typ      expr.DataType
		expected any
	}{
		"string": {typ: expr.String, expected: ""},
		"bytes":  {typ: expr.Bytes, expected: []byte{}},
		"array": {
			typ:      &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}},
			expected: []string{},
		},
		"map": {
			typ: &expr.Map{
				KeyType:  &expr.AttributeExpr{Type: expr.String},
				ElemType: &expr.AttributeExpr{Type: expr.String},
			},
			expected: map[string]string{},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			attribute := expr.AttributeExpr{
				Type: tc.typ,
				Validation: &expr.ValidationExpr{
					MaxLength: &maxLength,
				},
			}

			example := attribute.Example(&expr.ExampleGenerator{Randomizer: intRandomizer{Value: 2}})
			if !reflect.DeepEqual(example, tc.expected) {
				t.Fatalf("expected %#v, got %#v", tc.expected, example)
			}
		})
	}
}

func TestExamplesHonorLengthConstraintsAboveDefaultCap(t *testing.T) {
	minimum := 8
	exactStringLength := 64
	exactBytesLength := 32
	tests := map[string]struct {
		attribute  *expr.AttributeExpr
		wantLength int
	}{
		"minimum-only string": {
			attribute: &expr.AttributeExpr{
				Type:       expr.String,
				Validation: &expr.ValidationExpr{MinLength: &minimum},
			},
			wantLength: minimum,
		},
		"exact-length string": {
			attribute: &expr.AttributeExpr{
				Type: expr.String,
				Validation: &expr.ValidationExpr{
					MinLength: &exactStringLength,
					MaxLength: &exactStringLength,
				},
			},
			wantLength: exactStringLength,
		},
		"exact-length bytes": {
			attribute: &expr.AttributeExpr{
				Type: expr.Bytes,
				Validation: &expr.ValidationExpr{
					MinLength: &exactBytesLength,
					MaxLength: &exactBytesLength,
				},
			},
			wantLength: exactBytesLength,
		},
		"minimum-only array": {
			attribute: &expr.AttributeExpr{
				Type:       &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}},
				Validation: &expr.ValidationExpr{MinLength: &minimum},
			},
			wantLength: minimum,
		},
		"minimum-only map": {
			attribute: &expr.AttributeExpr{
				Type: &expr.Map{
					KeyType:  &expr.AttributeExpr{Type: expr.Int},
					ElemType: &expr.AttributeExpr{Type: expr.String},
				},
				Validation: &expr.ValidationExpr{MinLength: &minimum},
			},
			wantLength: minimum,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			example := test.attribute.Example(expr.NewRandom(name))
			if example == nil {
				t.Error("expected a synthesized example")
				return
			}
			length := reflect.ValueOf(example).Len()
			if length != test.wantLength {
				t.Errorf("example length = %d, want %d", length, test.wantLength)
			}
		})
	}
}

func TestConstrainedMapOmitsUnderfilledExample(t *testing.T) {
	minimum := 8
	attribute := &expr.AttributeExpr{
		Type: &expr.Map{
			KeyType:  &expr.AttributeExpr{Type: expr.Int},
			ElemType: &expr.AttributeExpr{Type: expr.String},
		},
		Validation: &expr.ValidationExpr{MinLength: &minimum},
	}
	random := &expr.ExampleGenerator{Randomizer: intRandomizer{Value: 1}}

	if example := attribute.Example(random); example != nil {
		t.Errorf("expected an omitted example, got %#v", example)
	}
}

func TestLengthConstrainedEnumExampleUsesValidEnumValue(t *testing.T) {
	exactLength := 5
	attribute := &expr.AttributeExpr{
		Type: expr.String,
		Validation: &expr.ValidationExpr{
			Values:    []any{"abcde"},
			MinLength: &exactLength,
			MaxLength: &exactLength,
		},
	}

	if example := attribute.Example(expr.NewRandom("enum")); example != "abcde" {
		t.Errorf("example = %#v, want enum value %q", example, "abcde")
	}
}

func TestLengthConstrainedPatternExampleSatisfiesBothConstraints(t *testing.T) {
	exactLength := 5
	attribute := &expr.AttributeExpr{
		Type: expr.String,
		Validation: &expr.ValidationExpr{
			Pattern:   `^[A-Z]{5}$`,
			MinLength: &exactLength,
			MaxLength: &exactLength,
		},
	}

	example, ok := attribute.Example(expr.NewRandom("pattern-and-length")).(string)
	if !ok {
		t.Errorf("expected a string example")
		return
	}
	if len(example) != exactLength || !regexp.MustCompile(attribute.Validation.Pattern).MatchString(example) {
		t.Errorf("example %q does not satisfy exact length %d and pattern", example, exactLength)
	}
}

func TestUnsafeConstrainedLengthOmitsSynthesizedExample(t *testing.T) {
	tests := map[string]int{
		"large":   1 << 30,
		"max int": int(^uint(0) >> 1),
	}
	for name, minimum := range tests {
		t.Run(name, func(t *testing.T) {
			attribute := &expr.AttributeExpr{
				Type:       expr.String,
				Validation: &expr.ValidationExpr{MinLength: &minimum},
			}
			random := &expr.ExampleGenerator{Randomizer: panicCharactersRandomizer{}}

			if example := attribute.Example(random); example != nil {
				t.Errorf("expected an omitted example, got %#v", example)
			}
		})
	}
}

func TestExample(t *testing.T) {
	cases := []struct {
		Name     string
		DSL      func()
		Expected any
		Error    string
	}{
		{"with-example", testdata.WithExampleDSL, "example", ""},
		{"with-array-example", testdata.WithArrayExampleDSL, []int{1, 2}, ""},
		{"with-map-example", testdata.WithMapExampleDSL, map[string]int{"name": 1, "value": 2}, ""},
		{"with-multiple-examples", testdata.WithMultipleExamplesDSL, 100, ""},
		{"overriding-example", testdata.OverridingExampleDSL, map[string]any{"name": "overridden"}, ""},
		{"with-extend", testdata.WithExtendExampleDSL, map[string]any{"name": "example"}, ""},
		{"invalid-example-type", testdata.InvalidExampleTypeDSL, nil, "service \"InvalidExampleType\" method \"Method\": payload - example value map[int]int{1:1} is incompatible with type map"},
		{"empty-example", testdata.EmptyExampleDSL, nil, "too few arguments given to Example in attribute"},
		{"hiding-example", testdata.HidingExampleDSL, nil, ""},
		{"overriding-hidden-examples", testdata.OverridingHiddenExamplesDSL, "example", ""},
	}
	r := expr.NewRandom("test")
	for _, k := range cases {
		t.Run(k.Name, func(t *testing.T) {
			if k.Error == "" {
				expr.RunDSL(t, k.DSL)
				example := expr.Root.Services[0].Methods[0].Payload.Example(r)
				if !reflect.DeepEqual(example, k.Expected) {
					t.Errorf("invalid example: got %v, expected %v", example, k.Expected)
				}
			} else {
				if err := expr.RunInvalidDSL(t, k.DSL); err == nil {
					t.Error("the expected error was not returned")
				} else if !strings.Contains(err.Error(), k.Error) {
					t.Errorf("invalid error: got %q, expected %q", err.Error(), k.Error)
				}
			}
		})
	}
}

type intRandomizer struct {
	expr.DeterministicRandomizer
	Value int
}

type panicCharactersRandomizer struct {
	expr.DeterministicRandomizer
}

func (r intRandomizer) Int() int {
	return r.Value
}

func (r intRandomizer) Characters(n int) string {
	return strings.Repeat("a", n)
}

func (panicCharactersRandomizer) Characters(int) string {
	panic("Characters must not be called for an unsafe example length")
}

// TestByLengthWithAliasType tests that alias types with length validations
// can generate examples correctly. Previously, this would panic because the
// code checked a.Type.Kind() instead of the underlying type's kind.
func TestByLengthWithAliasType(t *testing.T) {
	r := expr.NewRandom("test")

	// Create an alias type based on String with length validation
	// We need to use the DSL package properly
	root := expr.RunDSL(t, testdata.AliasLengthValidationDSL)

	aliasType := root.UserType("ValidatedString")
	att := aliasType.Attribute()

	// This should not panic and should generate a string example
	// The key test is that byLength handles alias types correctly by unaliasing
	// before checking the kind. Previously this would panic with:
	// "invalid type for length validation: ValidatedString"
	example := att.Example(r)
	str, ok := example.(string)
	if !ok {
		t.Fatalf("Expected string example, got %T", example)
	}
	// Verify it's a valid non-empty string (exact length may vary due to randomness)
	if len(str) == 0 {
		t.Error("Expected non-empty string example")
	}
}

// TestByLengthWithAliasArray tests that alias array types with length
// validations generate examples correctly.
func TestByLengthWithAliasArray(t *testing.T) {
	r := expr.NewRandom("test")

	root := expr.RunDSL(t, testdata.AliasArrayLengthValidationDSL)

	aliasType := root.UserType("StringArray")
	att := aliasType.Attribute()

	// This should not panic and should generate an array example
	example := att.Example(r)
	arr, ok := example.([]string)
	if !ok {
		t.Fatalf("Expected []string example, got %T", example)
	}

	// Verify the length is within the validation constraints
	if len(arr) < 2 || len(arr) > 5 {
		t.Errorf("Generated example has length %d, expected between 2 and 5", len(arr))
	}
}
