package exactnum

import (
	"math"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNativeNumbersDoNotAllocate(t *testing.T) {
	var number Number
	for _, test := range []struct {
		name string
		make func() Number
	}{
		{"integer", func() Number {
			return FromInt64(-1234567890)
		}},
		{"unsigned", func() Number {
			return FromUint64(math.MaxUint64)
		}},
		{"float", func() Number {
			n, err := FromFloat64(0.1)
			require.NoError(t, err)
			return n
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			allocations := testing.AllocsPerRun(100, func() {
				number = test.make()
			})
			require.Equal(t, float64(0), allocations)
			require.NotZero(t, number.Sign())
		})
	}
	left, right := FromInt64(math.MinInt64), FromUint64(math.MaxUint64)
	var order int
	allocations := testing.AllocsPerRun(100, func() {
		order = left.Cmp(right)
	})
	require.Equal(t, float64(0), allocations)
	require.Equal(t, -1, order)
	left, right = FromUint64(math.MaxUint64-1), FromUint64(math.MaxUint64)
	allocations = testing.AllocsPerRun(100, func() {
		order = left.Cmp(right)
	})
	require.Equal(t, float64(0), allocations)
	require.Equal(t, -1, order)
}

func TestNativeFloatBitsPreserveExactMeaning(t *testing.T) {
	values := []float64{0, math.Copysign(0, -1), 1, -1, 0.1, math.SmallestNonzeroFloat64, -math.SmallestNonzeroFloat64, math.MaxFloat64, -math.MaxFloat64, math.Float64frombits(0x0010000000000000), math.Float64frombits(0x000fffffffffffff)}
	random := rand.New(rand.NewPCG(902, 718))
	for range 1000 {
		value := math.Float64frombits(random.Uint64())
		if !math.IsNaN(value) && !math.IsInf(value, 0) {
			values = append(values, value)
		}
	}
	for _, value := range values {
		number, err := FromFloat64(value)
		require.NoError(t, err)
		actual, ok := new(big.Rat).SetString(number.String())
		require.True(t, ok)
		require.Zero(t, actual.Cmp(new(big.Rat).SetFloat64(value)), "bits %x", math.Float64bits(value))
	}
}

func TestNativeIntegerExtremesAndComparison(t *testing.T) {
	values := make([]*big.Int, 0, 207)
	values = append(values, big.NewInt(0), big.NewInt(1), big.NewInt(-1), big.NewInt(math.MinInt64), big.NewInt(math.MaxInt64), new(big.Int).SetUint64(math.MaxUint64), new(big.Int).SetUint64(1<<63))
	random := rand.New(rand.NewPCG(611, 812))
	for range 100 {
		values = append(values, new(big.Int).SetUint64(random.Uint64()), big.NewInt(int64(random.Uint64())))
	}
	for _, value := range values {
		var number Number
		if value.IsInt64() {
			number = FromInt64(value.Int64())
		} else {
			number = FromUint64(value.Uint64())
		}
		expected, err := ParseJSON(value.String())
		require.NoError(t, err)
		require.Equal(t, expected.Key(), number.Key())
		require.Equal(t, expected.String(), number.String())
		require.True(t, number.IsInt())
		for _, other := range values {
			otherNumber := FromInteger(other)
			require.Equal(t, value.Cmp(other), number.Cmp(otherNumber))
		}
	}
}

func BenchmarkNativeInteger(b *testing.B) {
	var number Number
	b.ReportAllocs()
	for b.Loop() {
		number = FromInt64(1234567890)
	}
	if number.Sign() != 1 {
		b.Fatal("invalid number")
	}
}

func TestCompactAndLargeStorageAreEquivalent(t *testing.T) {
	for _, test := range []struct {
		coefficient uint64
		e2, e5      int64
	}{
		{1, 0, 0}, {3, 63, 0}, {1, 64, 0}, {7, 0, 27}, {7, 0, 28}, {3, -400, 200},
		{3, math.MaxInt32, math.MaxInt32}, {3, math.MinInt32, math.MinInt32},
	} {
		for _, signum := range []int64{-1, 1} {
			coefficient := new(big.Int).SetUint64(test.coefficient)
			coefficient.Mul(coefficient, big.NewInt(signum))
			compact := canonical(coefficient, big.NewInt(test.e2), big.NewInt(test.e5))
			require.Nil(t, compact.large)
			c, e2, e5 := compact.parts()
			large := Number{sign: compact.sign, large: &largeNumber{c: c, e2: e2, e5: e5}}
			require.Equal(t, compact.Key(), large.Key())
			require.Equal(t, compact.String(), large.String())
			require.Equal(t, compact.IsInt(), large.IsInt())
			require.Zero(t, compact.Cmp(large))
			require.Zero(t, large.Cmp(compact))
			for _, other := range []Number{FromInt64(math.MinInt64), FromInt64(-1), {}, FromInt64(1), FromUint64(math.MaxUint64)} {
				require.Equal(t, large.Cmp(other), compact.Cmp(other))
				require.Equal(t, other.Cmp(large), other.Cmp(compact))
			}
		}
	}
}

func TestCompactStoragePromotesWithoutTruncation(t *testing.T) {
	tooWide := new(big.Int).Lsh(big.NewInt(1), 64)
	tooWide.Add(tooWide, big.NewInt(1))
	for _, test := range []struct {
		coefficient *big.Int
		e2, e5      int64
	}{
		{tooWide, 0, 0},
		{big.NewInt(3), math.MaxInt32 + 1, math.MaxInt32 + 1},
		{big.NewInt(3), math.MinInt32 - 1, math.MinInt32 - 1},
	} {
		number := canonical(test.coefficient, big.NewInt(test.e2), big.NewInt(test.e5))
		require.NotNil(t, number.large)
		round, err := ParseJSON(number.String())
		require.NoError(t, err)
		require.Equal(t, number.Key(), round.Key())
		require.Zero(t, number.Cmp(round))
	}
	for _, test := range []struct {
		left, right string
		order       int
	}{
		{"18446744073709551616", "18446744073709551615", 1},
		{"27670116110564327424", "18446744073709551615", 1},
		{"-27670116110564327424", "-9223372036854775808", -1},
		{"37252902984619140625", "18446744073709551615", 1},
		{"0.0001", "0", 1},
	} {
		left, err := ParseJSON(test.left)
		require.NoError(t, err)
		right, err := ParseJSON(test.right)
		require.NoError(t, err)
		require.Equal(t, test.order, left.Cmp(right))
	}
}
