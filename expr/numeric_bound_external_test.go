package expr_test

import "github.com/CaliLuke/loom/expr"

func boundForTest(value any) *expr.NumericBound {
	bound, err := expr.NewNumericBound(value)
	if err != nil {
		panic(err)
	}
	return &bound
}
