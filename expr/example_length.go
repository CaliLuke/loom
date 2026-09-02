package expr

import (
	"reflect"
	"unicode/utf8"
)

const (
	maxLength          = 3    // Preferred maximum length when constraints permit it.
	maxGeneratedLength = 1024 // Maximum safe allocation for a synthesized example.
)

// NewLength returns an int that validates the generator attribute length
// validations if any.
func NewLength(a *AttributeExpr, r *ExampleGenerator) int {
	if hasLengthValidation(a) {
		minlength, maxlength := 0, 0
		hasMinimum := a.Validation.MinLength != nil
		hasMaximum := a.Validation.MaxLength != nil
		limit := maxLength
		if hasMinimum {
			minlength = max(0, *a.Validation.MinLength)
			limit = max(limit, minlength)
		}
		if hasMaximum {
			maxlength = max(0, *a.Validation.MaxLength)
		}
		count := 0
		switch {
		case !hasMinimum:
			count = maxlength - (r.Int() % 3)
		case !hasMaximum:
			count = saturatingLengthAdd(minlength, r.Int()%3)
		case minlength < maxlength:
			diff := min(maxlength-minlength, maxLength)
			count = minlength + (r.Int() % diff)
		case minlength == maxlength:
			count = minlength
		default:
			panic("unreachable: MinLength > MaxLength should have been caught by ValidationExpr.Validate")
		}
		if count > limit {
			count = limit
		}
		if count < 0 {
			count = 0
		}
		return count
	}
	return r.ArrayLength()
}

func safeExampleLength(a *AttributeExpr, r *ExampleGenerator) (int, bool) {
	if a.Validation != nil && a.Validation.MinLength != nil && *a.Validation.MinLength > maxGeneratedLength {
		return 0, false
	}
	count := NewLength(a, r)
	return count, count <= maxGeneratedLength
}

func saturatingLengthAdd(base, offset int) int {
	maxInt := int(^uint(0) >> 1)
	if base > maxInt-offset {
		return base
	}
	return base + offset
}

func hasLengthValidation(a *AttributeExpr) bool {
	if a.Validation == nil {
		return false
	}
	return a.Validation.MinLength != nil || a.Validation.MaxLength != nil
}

// byLength generates a random size array of examples based on what's given.
func byLength(a *AttributeExpr, r *ExampleGenerator) any {
	count, ok := safeExampleLength(a, r)
	if !ok {
		return nil
	}
	// Use unaliased type to handle alias types (e.g., Type("Alias", String))
	dt := unalias(a.Type)
	switch dt.Kind() {
	case StringKind:
		return r.Characters(count)
	case BytesKind:
		return []byte(r.Characters(count))
	case MapKind:
		raw := make(map[any]any)
		m := dt.(*Map)
		for attempts := 0; len(raw) < count && attempts < maxAttempts; attempts++ {
			key := m.KeyType.Example(r)
			value := m.ElemType.Example(r)
			if key != nil && value != nil {
				raw[key] = value
			}
		}
		if len(raw) < count {
			return nil
		}
		return m.MakeMap(raw)
	case ArrayKind:
		raw := make([]any, count)
		ar := dt.(*Array)
		for i := range count {
			raw[i] = ar.ElemType.Example(r)
		}
		return ar.MakeSlice(raw)
	default:
		panic("invalid type for length validation: " + a.Type.Name())
	}
}

// byEnum returns a random selected enum value that satisfies the attribute's
// other generated-example constraints.
func byEnum(a *AttributeExpr, r *ExampleGenerator) any {
	if !hasEnumValidation(a) {
		return nil
	}
	values := a.Validation.Values
	count := len(values)
	start := r.Int() % count
	for offset := range count {
		candidate := values[(start+offset)%count]
		if checkLength(a, candidate) && checkPattern(a, candidate) && checkMinMaxValue(a, candidate) {
			return candidate
		}
	}
	return nil
}

func checkLength(a *AttributeExpr, example any) bool {
	if !hasLengthValidation(a) {
		return true
	}
	value := reflect.ValueOf(example)
	if !value.IsValid() {
		return false
	}
	var length int
	switch unalias(a.Type).Kind() {
	case StringKind:
		if value.Kind() != reflect.String {
			return false
		}
		length = utf8.RuneCountInString(value.String())
	case BytesKind, ArrayKind:
		if value.Kind() != reflect.Array && value.Kind() != reflect.Slice {
			return false
		}
		length = value.Len()
	case MapKind:
		if value.Kind() != reflect.Map {
			return false
		}
		length = value.Len()
	default:
		return false
	}
	if a.Validation.MinLength != nil && length < *a.Validation.MinLength {
		return false
	}
	return a.Validation.MaxLength == nil || length <= *a.Validation.MaxLength
}
