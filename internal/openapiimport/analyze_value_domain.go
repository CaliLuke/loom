package openapiimport

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/CaliLuke/libopenapi/datamodel/high/base"
	"github.com/pb33f/go-yaml"

	"github.com/CaliLuke/loom/internal/exactnum"
)

func hasSchemaEnum(source *base.Schema) bool {
	return source.Enum != nil || source.GoLow() != nil && source.GoLow().Enum.ValueNode != nil
}

func numericDomainValue(schema *Schema, number exactnum.Number) (any, error) {
	if schema.Type == "integer" {
		bits := 64
		if schema.Format == "int32" {
			bits = 32
		}
		maximum := int64(1<<(bits-1) - 1)
		if number.Cmp(exactnum.FromInt64(-maximum-1)) < 0 || number.Cmp(exactnum.FromInt64(maximum)) > 0 {
			return nil, fmt.Errorf("integer enum value %s is outside its DSL carrier", number.String())
		}
		// Expansion is bounded by the carrier check above, even for exponent
		// notation. Emitting an integer avoids Go's floating constant default.
		integer, ok := new(big.Rat).SetString(number.String())
		if !ok {
			return nil, fmt.Errorf("cannot render integer enum value %s", number.String())
		}
		return int(integer.Num().Int64()), nil
	}
	bits := 64
	if schema.Format == "float" {
		bits = 32
	}
	value, err := strconv.ParseFloat(number.String(), bits)
	if err != nil {
		return nil, fmt.Errorf("numeric enum value %s is outside its DSL carrier: %w", number.String(), err)
	}
	wire, err := exactnum.ParseJSON(strconv.FormatFloat(value, 'g', -1, bits))
	if err != nil || wire.Cmp(number) != 0 {
		return nil, fmt.Errorf("numeric enum value %s cannot retain its value through its DSL carrier", number.String())
	}
	// Keep the source decimal spelling when Float32 normalization happens in
	// the DSL; a widened binary float32 value has a different decimal spelling.
	return strconv.ParseFloat(number.String(), 64)
}

// schemaValueDomain intersects authored membership with the admitted source
// type before splitting non-null values and nullability for DSL rendering.
func (a *analyzer) schemaValueDomain(schema *Schema, source *base.Schema, path string) {
	enum := hasSchemaEnum(source)
	if !enum && source.Const == nil {
		// An outer bare-null composition may have copied an inner enum. It is
		// not an outer membership assertion and must not remove that null branch.
		return
	}
	keyword := "enum"
	values := source.Enum
	if source.Const != nil {
		keyword = "const"
		values = []*yaml.Node{source.Const}
	}
	schema.valueDomainKeyword = keyword
	schema.Enum = nil
	fail := func(location, message string) {
		schema.valueDomainError = &Diagnostic{Code: "schema-" + keyword, Path: location, Message: message}
	}
	if enum && source.Const != nil {
		fail(path+"/const", "const combined with enum is not in the strict import subset")
		return
	}
	nullMember := false
	for index, node := range values {
		location := path + "/" + keyword
		if keyword == "enum" {
			location += "/" + strconv.Itoa(index)
		}
		value, included, err := a.valueDomainMember(schema, node)
		if err != nil {
			fail(location, err.Error())
			return
		}
		if !included {
			continue
		}
		if value == nil {
			nullMember = true
		} else {
			schema.Enum = append(schema.Enum, value)
		}
	}
	schema.Nullable = schema.Nullable && nullMember
	if len(schema.Enum) == 0 {
		fail(path+"/"+keyword, "empty or null-only value domains have no representation in the strict import subset")
	}
}

// valueDomainMember first classifies the exact JSON value against the source
// type. Representability failures for included values reject the whole domain.
func (a *analyzer) valueDomainMember(schema *Schema, node *yaml.Node) (any, bool, error) {
	if raw, present := a.nodeNumericSource(node); present {
		if number, err := raw.number(); err == nil {
			if schema.Type != "number" && schema.Type != "integer" || schema.Type == "integer" && !number.IsInt() {
				return nil, false, nil
			}
			value, err := numericDomainValue(schema, number)
			return value, true, err
		}
	}
	var value any
	if err := node.Decode(&value); err != nil {
		return nil, false, fmt.Errorf("value cannot be decoded: %w", err)
	}
	if value == nil {
		return nil, schema.Nullable, nil
	}
	compatible := false
	switch value.(type) {
	case string:
		compatible = schema.Type == "string"
	case bool:
		compatible = schema.Type == "boolean"
	case []any:
		compatible = schema.Type == "array"
	case map[string]any:
		compatible = schema.Type == "object"
	}
	if !compatible {
		return nil, false, nil
	}
	if _, err := scalarLiteral(value); err != nil {
		return nil, true, err
	}
	return value, true, nil
}
