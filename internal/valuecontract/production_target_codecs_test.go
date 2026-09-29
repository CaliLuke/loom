package valuecontract

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"math"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	loom "github.com/CaliLuke/loom/pkg"
)

type productionNumericPolicy struct {
	name     string
	bits     int
	unsigned bool
	decimal  bool
}

func productionNumericPolicies() []productionNumericPolicy {
	return []productionNumericPolicy{
		{name: "int32", bits: 32}, {name: "int64", bits: 64},
		{name: "uint32", bits: 32, unsigned: true}, {name: "uint64", bits: 64, unsigned: true},
		{name: "float32", bits: 32, decimal: true}, {name: "float64", bits: 64, decimal: true},
	}
}

func (p productionNumericPolicy) integerFormat() any {
	if p.decimal {
		return "mathematical"
	}
	kind := "signed"
	if p.unsigned {
		kind = "unsigned"
	}
	return referenceConstructor(kind, map[string]any{"bits": p.bits})
}

func (p productionNumericPolicy) decimalFormat() string {
	if !p.decimal {
		return "exact"
	}
	if p.bits == 32 {
		return "binary32"
	}
	return "binary64"
}

func (p productionNumericPolicy) kind() string {
	if p.decimal {
		return "decimal"
	}
	return "integer"
}

func (p productionNumericPolicy) scalar(text string) (any, bool) {
	var pointer any
	switch {
	case p.decimal && p.bits == 32:
		pointer = new(float32)
	case p.decimal:
		pointer = new(float64)
	case p.unsigned && p.bits == 32:
		pointer = new(uint32)
	case p.unsigned:
		pointer = new(uint64)
	case p.bits == 32:
		pointer = new(int32)
	default:
		pointer = new(int64)
	}
	if err := json.Unmarshal([]byte(text), pointer); err != nil {
		return nil, false
	}
	return reflect.ValueOf(pointer).Elem().Interface(), true
}

func (p productionNumericPolicy) key(text string) (any, bool) {
	var pointer any
	switch {
	case p.decimal && p.bits == 32:
		pointer = new(map[float32]bool)
	case p.decimal:
		pointer = new(map[float64]bool)
	case p.unsigned && p.bits == 32:
		pointer = new(map[uint32]bool)
	case p.unsigned:
		pointer = new(map[uint64]bool)
	case p.bits == 32:
		pointer = new(map[int32]bool)
	default:
		pointer = new(map[int64]bool)
	}
	wire, err := json.Marshal(map[string]bool{text: true})
	if err != nil || json.Unmarshal(wire, pointer, loom.JSONOptions()) != nil {
		return nil, false
	}
	keys := reflect.ValueOf(pointer).Elem().MapKeys()
	if len(keys) != 1 {
		return nil, false
	}
	return keys[0].Interface(), true
}

func productionNumericResult(raw any, accepted bool, decimal bool) any {
	if !accepted {
		return nil
	}
	value := reflect.ValueOf(raw)
	if decimal {
		number := value.Float()
		return map[string]any{"value": referenceBinaryDecimal(number), "negativeZero": number == 0 && math.Signbit(number)}
	}
	if value.Kind() >= reflect.Int && value.Kind() <= reflect.Int64 {
		return jsontext.Value(strconv.FormatInt(value.Int(), 10))
	}
	return jsontext.Value(strconv.FormatUint(value.Uint(), 10))
}

func productionSchemaNumber(text string) *referenceDecimal {
	wire := jsontext.Value(text)
	if !wire.IsValid() || wire.Kind() != '0' || strings.TrimSpace(text) != text {
		return nil
	}
	parts := strings.SplitN(strings.ToLower(text), "e", 2)
	var exponent int64
	if len(parts) == 2 {
		parsed, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return nil
		}
		exponent = parsed
	}
	mantissa := parts[0]
	if point := strings.IndexByte(mantissa, '.'); point >= 0 {
		exponent -= int64(len(mantissa) - point - 1)
		mantissa = mantissa[:point] + mantissa[point+1:]
	}
	coefficient, ok := new(big.Int).SetString(mantissa, 10)
	if !ok {
		return nil
	}
	return &referenceDecimal{Coefficient: jsontext.Value(coefficient.String()), Exponent: exponent}
}

func productionTargetCodecs(t *testing.T, texts []string, policies []productionNumericPolicy) map[string]any {
	t.Helper()
	numbers := make([]any, 0, len(texts))
	spellings := make(map[string]any)
	integers, decimals := []any{}, []any{}
	for _, text := range texts {
		numbers = append(numbers, map[string]any{"text": text, "schema": productionSchemaNumber(text)})
		for _, policy := range policies {
			scalar, scalarOK := policy.scalar(text)
			key, keyOK := policy.key(text)
			row := map[string]any{"text": text, "scalar": productionNumericResult(scalar, scalarOK, policy.decimal), "key": productionNumericResult(key, keyOK, policy.decimal)}
			if policy.decimal {
				for _, candidate := range []struct {
					value    any
					accepted bool
				}{{scalar, scalarOK}, {key, keyOK}} {
					if !candidate.accepted {
						continue
					}
					_, identity := referenceNumericScalar(candidate.value, 0)
					wire, err := json.Marshal(candidate.value)
					require.NoError(t, err)
					spellings[productionJSON(t, identity)] = map[string]any{"number": identity, "text": string(wire)}
				}
				row["format"] = policy.decimalFormat()
				decimals = append(decimals, row)
			} else {
				row["format"] = policy.integerFormat()
				integers = append(integers, row)
			}
		}
	}
	return map[string]any{"numberReadings": numbers, "integerReadings": integers, "decimalReadings": decimals, "decimalSpellings": productionSortedRows(spellings), "literalSpellings": []any{}, "literalReadings": []any{}}
}

func productionDecodedScalar(t *testing.T, value any) any {
	t.Helper()
	input := newProductionInput(t)
	return referenceConstructor("scalar", map[string]any{"value": input.scalar(value)})
}

func productionNumericSchemaAccepted(t *testing.T, text string, policy productionNumericPolicy) bool {
	t.Helper()
	decimal := productionSchemaNumber(text)
	if decimal == nil {
		return false
	}
	if policy.decimal {
		return true
	}
	rational, ok := new(big.Rat).SetString(text)
	require.True(t, ok)
	return rational.IsInt()
}
