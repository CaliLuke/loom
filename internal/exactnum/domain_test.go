package exactnum

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDirectedBoundary(t *testing.T) {
	for _, test := range []struct {
		name, source     string
		domain           Domain
		lower, exclusive bool
		want             any
		present          bool
	}{
		{"large signed", "9007199254740993", Domain{Bits: 64}, true, false, int64(9007199254740993), true},
		{"unsigned maximum", "18446744073709551615", Domain{Bits: 64, Unsigned: true}, false, false, uint64(math.MaxUint64), true},
		{"unsigned past maximum", "18446744073709551615", Domain{Bits: 64, Unsigned: true}, true, true, nil, false},
		{"negative tiny integer", "-1e-1000001", Domain{Bits: 64}, false, false, int64(-1), true},
		{"positive tiny integer", "1e-1000001", Domain{Bits: 64}, true, false, int64(1), true},
		{"positive tiny float", "1e-1000001", Domain{Bits: 64, Float: true}, true, false, math.SmallestNonzeroFloat64, true},
		{"positive tiny float upper", "1e-1000001", Domain{Bits: 64, Float: true}, false, false, float64(0), true},
		{"decimal float upper", "0.1", Domain{Bits: 64, Float: true}, false, false, math.Nextafter(0.1, math.Inf(-1)), true},
		{"decimal float lower", "0.1", Domain{Bits: 64, Float: true}, true, false, float64(0.1), true},
		{"decimal float32 upper", "0.1", Domain{Bits: 32, Float: true}, false, false, math.Nextafter32(0.1, float32(math.Inf(-1))), true},
		{"decimal float32 lower", "0.1", Domain{Bits: 32, Float: true}, true, false, float32(0.1), true},
	} {
		t.Run(test.name, func(t *testing.T) {
			number, err := ParseJSON(test.source)
			require.NoError(t, err)
			value, ok := number.Boundary(test.domain, test.lower, test.exclusive)
			require.Equal(t, test.present, ok)
			require.Equal(t, test.want, value)
		})
	}
}
