// Package exactnum owns exact finite arithmetic shared by design analysis and
// runtime JSON schema matching. Source adapters admit syntax independently of
// this representation; target decoding precision never changes numeric meaning.
package exactnum

import (
	"encoding/json/jsontext"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strconv"
	"strings"
)

// Number is an immutable finite number, sign*C*2^E2*5^E5. The coefficient is
// positive and has no factors of two or five. Exponents have arbitrary magnitude.
// Native-sized canonical components are stored inline; larger components use
// private immutable storage. Its zero value is zero and copies are safe.
type Number struct {
	coefficient          uint64
	exponent2, exponent5 int32
	sign                 int8
	large                *largeNumber
}

type largeNumber struct {
	c, e2, e5 *big.Int
}

// ParseJSON admits a JSON number and preserves its exact mathematical meaning.
// JSON whitespace around the token is allowed; quoted numbers are not numbers.
func ParseJSON(source string) (Number, error) {
	wire := jsontext.Value(source)
	if wire.Kind() != '0' || !wire.IsValid() {
		return Number{}, fmt.Errorf("invalid JSON number %q", source)
	}
	return parse(strings.Trim(source, " \t\r\n")), nil
}

// ParseDSL preserves the existing finite strconv.ParseFloat string admission
// contract, including hexadecimal, separators and underflowing nonzero inputs.
// The rounded parse result is used only to reject nonfinite sources, never as
// the retained numeric value.
func ParseDSL(source string) (Number, error) {
	value, err := strconv.ParseFloat(source, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return Number{}, fmt.Errorf("invalid finite number %q", source)
	}
	return parse(source), nil
}

// FromInt64 preserves an integer exactly.
func FromInt64(value int64) Number {
	if value < 0 {
		return compactNumber(-1, uint64(-(value+1))+1)
	}
	return compactNumber(1, uint64(value))
}

// FromUint64 preserves an unsigned integer exactly.
func FromUint64(value uint64) Number {
	return compactNumber(1, value)
}

// FromInteger copies an arbitrary-size integer into immutable numeric storage.
func FromInteger(value *big.Int) Number {
	return canonical(value, new(big.Int), new(big.Int))
}

// FromFloat64 preserves the actual finite binary value, not its short decimal
// spelling. Converting a float32 to float64 before calling this is exact.
func FromFloat64(value float64) (Number, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return Number{}, fmt.Errorf("nonfinite numeric value %v", value)
	}
	return compactFloat(value), nil
}

// FromScalar reads the exact value of an already admitted Go numeric scalar,
// including named numeric types. It does not decide source-type eligibility.
func FromScalar(value any) (Number, bool) {
	actual := reflect.ValueOf(value)
	if !actual.IsValid() {
		return Number{}, false
	}
	switch actual.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return FromInt64(actual.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return FromUint64(actual.Uint()), true
	case reflect.Float32, reflect.Float64:
		number, err := FromFloat64(actual.Float())
		return number, err == nil
	default:
		return Number{}, false
	}
}

// Sign returns -1, 0 or 1 according to the sign of n.
func (n Number) Sign() int {
	return int(n.sign)
}

// IsInt reports whether n is mathematically integral without expanding powers.
func (n Number) IsInt() bool {
	if n.sign == 0 {
		return true
	}
	if n.large != nil {
		return n.large.e2.Sign() >= 0 && n.large.e5.Sign() >= 0
	}
	return n.exponent2 >= 0 && n.exponent5 >= 0
}

// Key returns a canonical exact identity without expanding exponent magnitudes.
// It is an internal identity spelling, not JSON or DSL syntax.
func (n Number) Key() string {
	if n.sign == 0 {
		return "0"
	}
	if n.large == nil {
		return strconv.Itoa(int(n.sign)) + ":" + strconv.FormatUint(n.coefficient, 10) + ":" + strconv.FormatInt(int64(n.exponent2), 10) + ":" + strconv.FormatInt(int64(n.exponent5), 10)
	}
	return strconv.Itoa(int(n.sign)) + ":" + n.large.c.String() + ":" + n.large.e2.String() + ":" + n.large.e5.String()
}

// String returns a canonical exact JSON numeric token. A binary-radix input
// can intrinsically require output proportional to its decimal expansion.
func (n Number) String() string {
	if n.sign == 0 {
		return "0"
	}
	c, e2, e5 := n.parts()
	exponent := new(big.Int).Set(e2)
	if e5.Cmp(exponent) < 0 {
		exponent.Set(e5)
	}
	coefficient := new(big.Int).Set(c)
	for _, factor := range []struct {
		base     int64
		exponent *big.Int
	}{{2, e2}, {5, e5}} {
		difference := new(big.Int).Sub(factor.exponent, exponent)
		coefficient.Mul(coefficient, new(big.Int).Exp(big.NewInt(factor.base), difference, nil))
	}
	digits := coefficient.String()
	token := digits
	if exponent.Sign() != 0 {
		token += "e" + exponent.String()
		// Choose plain notation only when it is no longer than scientific form.
		// This comparison bounds allocation by the existing coefficient spelling.
		if exponent.IsInt64() {
			e := exponent.Int64()
			switch {
			case e > 0 && e <= int64(len(token)-len(digits)):
				token = digits + strings.Repeat("0", int(e))
			case e < 0 && e > -int64(len(digits)):
				at := len(digits) + int(e)
				token = digits[:at] + "." + digits[at:]
			case e < 0 && e >= -int64(len(token)-2):
				token = "0." + strings.Repeat("0", int(-e)-len(digits)) + digits
			}
		}
	}
	if n.sign < 0 {
		return "-" + token
	}
	return token
}

// MarshalJSON emits an unquoted exact numeric token.
func (n Number) MarshalJSON() ([]byte, error) {
	return []byte(n.String()), nil
}

// MarshalJSONTo emits an unquoted exact numeric token through JSON v2.
func (n Number) MarshalJSONTo(encoder *jsontext.Encoder) error {
	return encoder.WriteValue(jsontext.Value(n.String()))
}

func canonical(coefficient, exponent2, exponent5 *big.Int) Number {
	sign := int8(coefficient.Sign())
	n := &largeNumber{c: new(big.Int).Abs(coefficient), e2: new(big.Int).Set(exponent2), e5: new(big.Int).Set(exponent5)}
	if sign == 0 {
		return Number{}
	}
	zeros := n.c.TrailingZeroBits()
	n.c.Rsh(n.c, zeros)
	n.e2.Add(n.e2, new(big.Int).SetUint64(uint64(zeros)))
	// Extract powers of five in logarithmically many divisions, each bounded by
	// coefficient size. This never allocates a power derived from source exponent.
	powers := []*big.Int{}
	for power := big.NewInt(5); power.Cmp(n.c) <= 0; power = new(big.Int).Mul(power, power) {
		powers = append(powers, power)
	}
	for index := len(powers) - 1; index >= 0; index-- {
		quotient, remainder := new(big.Int), new(big.Int)
		quotient.QuoRem(n.c, powers[index], remainder)
		if remainder.Sign() == 0 {
			n.c = quotient
			n.e5.Add(n.e5, new(big.Int).Lsh(big.NewInt(1), uint(index)))
		}
	}
	return canonicalStorage(sign, n)
}

// parse consumes a spelling already admitted by a source adapter.
func parse(source string) Number {
	source = strings.ReplaceAll(source, "_", "")
	sign := 1
	if strings.HasPrefix(source, "-") {
		sign, source = -1, source[1:]
	} else {
		source = strings.TrimPrefix(source, "+")
	}
	base, symbols, places := 10, "eE", int64(1)
	if strings.HasPrefix(strings.ToLower(source), "0x") {
		base, symbols, places, source = 16, "pP", 4, source[2:]
	}
	exponent := new(big.Int)
	if at := strings.IndexAny(source, symbols); at >= 0 {
		if _, ok := exponent.SetString(source[at+1:], 10); !ok {
			panic("exactnum: source adapter admitted invalid exponent")
		}
		source = source[:at]
	}
	if at := strings.IndexByte(source, '.'); at >= 0 {
		exponent.Sub(exponent, big.NewInt(int64(len(source)-at-1)*places))
		source = source[:at] + source[at+1:]
	}
	if base == 10 {
		trimmed := strings.TrimRight(source, "0")
		exponent.Add(exponent, big.NewInt(int64(len(source)-len(trimmed))))
		source = trimmed
	}
	if source == "" {
		return Number{}
	}
	coefficient, ok := new(big.Int).SetString(source, base)
	if !ok {
		panic("exactnum: source adapter admitted invalid coefficient")
	}
	if sign < 0 {
		coefficient.Neg(coefficient)
	}
	if base == 16 {
		return canonical(coefficient, exponent, new(big.Int))
	}
	return canonical(coefficient, exponent, exponent)
}
