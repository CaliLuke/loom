package expr

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/internal/headerlist"
	loom "github.com/CaliLuke/loom/pkg"
)

// validateHeaderCollections checks only mapped occurrences, leaving body values
// and scalar header grammars independent of simple collection framing.
func validateHeaderCollections(headers *MappedAttributeExpr, source *AttributeExpr, parent eval.Expression, verr *eval.ValidationErrors) {
	if headers == nil || headers.IsEmpty() {
		return
	}
	prepared := DupMappedAtt(headers)
	initAttr(prepared, source)
	for _, field := range *AsObject(prepared.Type) {
		array := AsArray(field.Attribute.Type)
		if array == nil {
			continue
		}
		name := prepared.ElemName(field.Name)
		if strings.EqualFold(name, "Set-Cookie") {
			verr.Add(parent, "array header %q cannot use simple list framing; use the cookie DSL", name)
			continue
		}
		validateHeaderAuthoredValues(field.Attribute, name, parent, verr)
		validateHeaderAuthoredValues(array.ElemType, name, parent, verr)
	}
}

func validateHeaderAuthoredValues(attribute *AttributeExpr, name string, parent eval.Expression, verr *eval.ValidationErrors) {
	constraints, err := EffectiveConstraintsFor(attribute)
	if err != nil {
		verr.Add(parent, "header %q: %s", name, err)
		return
	}
	values, _ := constraints.EnumCandidates()
	if value, ok := constraints.Default(); ok {
		values = append(values, value)
	}
	for _, example := range attribute.ExtractUserExamples() {
		values = append(values, example.Value)
	}
	for _, value := range values {
		if err := validateHeaderValue(attribute.Type, value); err != nil {
			verr.Add(parent, "header %q has an unrepresentable authored value: %s", name, err)
		}
	}
}

func validateHeaderValue(dt DataType, value any) error {
	if array := AsArray(dt); array != nil {
		v := reflect.ValueOf(value)
		if !v.IsValid() || (v.Kind() != reflect.Slice && v.Kind() != reflect.Array) {
			// Ordinary attribute validation owns type mismatches.
			return nil
		}
		for i := range v.Len() {
			if err := validateHeaderValue(array.ElemType.Type, v.Index(i).Interface()); err != nil {
				return fmt.Errorf("item %d: %w", i, err)
			}
		}
		return nil
	}
	for {
		named, ok := dt.(UserType)
		if !ok {
			break
		}
		dt = named.Attribute().Type
	}
	var text string
	switch dt {
	case String:
		var ok bool
		text, ok = value.(string)
		if !ok {
			return nil
		}
	case Bytes:
		switch v := value.(type) {
		case []byte:
			text = string(v)
		case string:
			text = v
		default:
			return nil
		}
	case Any:
		encoded, err := loom.JSONValueFrom(value)
		if err != nil {
			return errors.New("value cannot be encoded as JSON")
		}
		text = loom.JSONValueString(encoded)
	default:
		// Numeric and boolean conversion cannot produce ambiguous delimiters.
		return nil
	}
	return headerlist.ValidateItem(text)
}
