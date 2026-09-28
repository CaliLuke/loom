package jsonkey

import (
	"fmt"
	"math"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNameRejectsNonJSONKeys(t *testing.T) {
	for _, value := range []any{nil, math.NaN(), math.Inf(1), math.Inf(-1), complex(1, 2), new(int), struct{}{}, [1]int{1}} {
		t.Run(fmt.Sprintf("%T", value), func(t *testing.T) {
			_, ok := Name(reflect.ValueOf(value))
			require.False(t, ok)
		})
	}
}
