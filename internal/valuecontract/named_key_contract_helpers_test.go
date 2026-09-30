package valuecontract

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

type (
	independentKeyContract struct {
		layers    []referenceAliasLayer
		lengths   []byteAliasBound
		raw       []independentKeyRawLayer
		extractor *independentAliasExtractor
	}
	independentKeyRawLayer struct {
		enumeration  *[]independentResolvedValue
		defaultValue *independentResolvedValue
	}
	independentKeyQuery struct {
		contractOK bool
		lengthOK   bool
	}
	independentLengthResult struct {
		Effective byteAliasBound `json:"effective"`
		Accepted  []bool         `json:"accepted"`
	}
)

func captureIndependentKeyContract(
	t *testing.T,
	key *expr.AttributeExpr,
) (independentKeyContract, error) {
	t.Helper()
	chain, err := checkedRawAliasAncestry(key)
	if err != nil {
		return independentKeyContract{}, err
	}
	leaf := chain[len(chain)-1].attribute
	if _, ok := leaf.Type.(expr.Primitive); !ok {
		return independentKeyContract{}, fmt.Errorf("key leaf %T is outside the scalar adapter", leaf.Type)
	}
	semantic := &expr.AttributeExpr{Type: leaf.Type, Nullable: leaf.Nullable}
	for _, layer := range chain {
		if layer.attribute.Nullable {
			return independentKeyContract{}, fmt.Errorf("nullable map key layer %q", layer.name)
		}
		validation := layer.attribute.Validation
		if validation != nil {
			for _, value := range validation.Values {
				if _, err := independentResolve(semantic, value); err != nil {
					return independentKeyContract{}, fmt.Errorf("enum in %q: %w", layer.name, err)
				}
			}
			for _, predicate := range independentRawPredicates(validation) {
				if predicate.Kind == "pattern" {
					if _, err := regexp.Compile(predicate.Value); err != nil {
						return independentKeyContract{}, fmt.Errorf("pattern in %q: %w", layer.name, err)
					}
				}
			}
		}
		if layer.attribute.DefaultValue != nil {
			if _, err := independentResolve(semantic, layer.attribute.DefaultValue); err != nil {
				return independentKeyContract{}, fmt.Errorf("default in %q: %w", layer.name, err)
			}
		}
	}
	layers, extractor := extractIndependentAliasLayers(t, key)
	slices.Reverse(chain)
	lengths := make([]byteAliasBound, len(chain))
	raw := make([]independentKeyRawLayer, len(chain))
	for index, layer := range chain {
		if validation := layer.attribute.Validation; validation != nil {
			lengths[index] = byteAliasBound{
				Minimum: copyIntPointer(validation.MinLength),
				Maximum: copyIntPointer(validation.MaxLength),
			}
			if validation.Values != nil {
				values := make([]independentResolvedValue, len(validation.Values))
				for valueIndex, value := range validation.Values {
					resolved, resolveErr := independentResolve(semantic, value)
					if resolveErr != nil {
						return independentKeyContract{}, fmt.Errorf("enum in %q: %w", layer.name, resolveErr)
					}
					values[valueIndex] = resolved
				}
				raw[index].enumeration = &values
			}
		}
		if layer.attribute.DefaultValue != nil {
			resolved, resolveErr := independentResolve(semantic, layer.attribute.DefaultValue)
			if resolveErr != nil {
				return independentKeyContract{}, fmt.Errorf("default in %q: %w", layer.name, resolveErr)
			}
			raw[index].defaultValue = &resolved
		}
	}
	return independentKeyContract{layers: layers, lengths: lengths, raw: raw, extractor: extractor}, nil
}

func checkedRawAliasAncestry(attribute *expr.AttributeExpr) ([]independentAliasLayer, error) {
	if attribute == nil {
		return nil, fmt.Errorf("missing attribute")
	}
	layers := []independentAliasLayer{{name: "attribute", attribute: attribute}}
	current := attribute
	seen := make(map[expr.UserType]bool)
	for {
		named, ok := current.Type.(expr.UserType)
		if !ok {
			return layers, nil
		}
		if seen[named] {
			return nil, fmt.Errorf("cyclic named key ancestry at %q", named.Name())
		}
		seen[named] = true
		current = named.Attribute()
		if current == nil {
			return nil, fmt.Errorf("named key %q has no declaration", named.Name())
		}
		layers = append(layers, independentAliasLayer{name: named.Name(), attribute: current})
	}
}

func copyIntPointer(value *int) *int {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func runIndependentKeyContract(
	t *testing.T,
	executable string,
	contract independentKeyContract,
) referenceAliasOutcome {
	t.Helper()
	result := runAliasReference(t, executable, contract.layers)
	if result.OK != nil && !independentKeyLengthDeclarationsValid(t, executable, contract) {
		return referenceAliasOutcome{Error: []byte(`"invalidLength"`)}
	}
	return result
}

func queryIndependentKey(
	t *testing.T,
	executable string,
	contract independentKeyContract,
	raw any,
) independentKeyQuery {
	t.Helper()
	if !independentKeyLengthDeclarationsValid(t, executable, contract) {
		return independentKeyQuery{}
	}
	resolved, err := independentResolve(contract.extractor.attribute, raw)
	if err != nil {
		return independentKeyQuery{}
	}
	value := contract.extractor.semantic(raw)
	layers := slices.Clone(contract.layers)
	layers = append(layers, referenceAliasLayer{
		Declaration:  uint64(len(layers) + 1),
		DefaultValue: &value,
		name:         "synthetic-key-query",
	})
	result := runAliasReference(t, executable, layers)
	length := independentKeyLength(resolved)
	return independentKeyQuery{
		contractOK: result.OK != nil,
		lengthOK:   runIndependentLengthQuery(t, executable, contract.lengths, length),
	}
}

// independentKeyLengthDeclarationsValid composes the existing length model
// with raw declaration prefixes. It checks authored enums at their declaration
// prefix and the selected default at every prefix. The synthetic query remains
// separate and never becomes authored provenance.
func independentKeyLengthDeclarationsValid(
	t *testing.T,
	executable string,
	contract independentKeyContract,
) bool {
	t.Helper()
	var selectedDefault *independentResolvedValue
	for index, raw := range contract.raw {
		prefix := contract.lengths[:index+1]
		if raw.enumeration != nil {
			for _, value := range *raw.enumeration {
				if !runIndependentLengthQuery(t, executable, prefix, independentKeyLength(value)) {
					return false
				}
			}
		}
		if raw.defaultValue != nil {
			selectedDefault = raw.defaultValue
		}
		if selectedDefault != nil &&
			!runIndependentLengthQuery(t, executable, prefix, independentKeyLength(*selectedDefault)) {
			return false
		}
	}
	return true
}

func runIndependentLengthQuery(
	t *testing.T,
	executable string,
	layers []byteAliasBound,
	length int,
) bool {
	t.Helper()
	result := runIndependentLengths(t, executable, layers, []int{length})
	require.Len(t, result.Accepted, 1)
	return result.Accepted[0]
}

func runIndependentLengths(
	t *testing.T,
	executable string,
	layers []byteAliasBound,
	lengths []int,
) independentLengthResult {
	t.Helper()
	command := referenceConstructor("aliasLengths", map[string]any{
		"constraints": layers,
		"lengths":     lengths,
	})
	return referenceDecode[independentLengthResult](
		t,
		runReference(t, executable, []any{command})[0],
	)
}

func independentKeyLength(value independentResolvedValue) int {
	if value.presence == "null" || value.presence == "absent" {
		return 0
	}
	switch value.kind {
	case "scalar":
		switch scalar := value.scalar.(type) {
		case string:
			return utf8.RuneCountInString(scalar)
		case []byte:
			return len(scalar)
		}
	case "array":
		return len(value.elements)
	case "map":
		return len(value.entries)
	}
	return 0
}

func resolveWholeMap(
	t *testing.T,
	key *expr.AttributeExpr,
	raw any,
) (expr.ValueOutcome, error) {
	t.Helper()
	attribute := &expr.AttributeExpr{Type: &expr.Map{
		KeyType:  key,
		ElemType: &expr.AttributeExpr{Type: expr.String},
	}}
	context := expr.NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	if err != nil {
		return expr.ValueInvalid, err
	}
	result := context.Resolve(
		occurrence,
		context.SupplyValue(expr.ValueInput{Raw: raw}),
		expr.ValueRoleExample,
	)
	return result.Outcome(), nil
}

func singletonMap(key, value any) (any, error) {
	keyValue := reflect.ValueOf(key)
	valueValue := reflect.ValueOf(value)
	if !keyValue.IsValid() || !keyValue.Type().Comparable() || !valueValue.IsValid() {
		return nil, fmt.Errorf("invalid singleton map types")
	}
	mapping := reflect.MakeMap(reflect.MapOf(keyValue.Type(), valueValue.Type()))
	mapping.SetMapIndex(keyValue, valueValue)
	return mapping.Interface(), nil
}
