package valuecontract

import "github.com/CaliLuke/loom/expr"

func boundForTest(value any) *expr.NumericBound {
	bound, err := expr.NewNumericBound(value)
	if err != nil {
		panic(err)
	}
	return &bound
}

// boundDecimal adapts exact physical endpoints to the independent Lean decimal
// protocol. That finite test protocol uses Int64 exponents; unsupported cases
// must fail explicitly instead of disappearing from the conformance corpus.
func boundDecimal(value *expr.NumericBound) *referenceDecimal {
	if value == nil {
		return nil
	}
	number := productionSchemaNumber(value.String())
	if number == nil {
		panic("numeric bound exceeds the reference adapter's exponent domain")
	}
	return number
}
