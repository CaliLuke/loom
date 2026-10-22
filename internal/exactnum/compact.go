package exactnum

import (
	"math"
	"math/big"
	"math/bits"
)

// compactNumber normalizes a native magnitude without allocating big integers.
func compactNumber(sign int8, magnitude uint64) Number {
	if magnitude == 0 {
		return Number{}
	}
	twos := bits.TrailingZeros64(magnitude)
	magnitude >>= twos
	fives := int32(0)
	for magnitude%5 == 0 {
		magnitude /= 5
		fives++
	}
	return Number{sign: sign, coefficient: magnitude, exponent2: int32(twos), exponent5: fives}
}

func compactFloat(value float64) Number {
	raw := math.Float64bits(value)
	sign := int8(1)
	if raw>>63 != 0 {
		sign = -1
	}
	exponent := int32((raw >> 52) & 0x7ff)
	significand := raw & ((uint64(1) << 52) - 1)
	if exponent == 0 {
		exponent = -1074
	} else {
		significand |= uint64(1) << 52
		exponent -= 1023 + 52
	}
	number := compactNumber(sign, significand)
	if number.sign != 0 {
		number.exponent2 += exponent
	}
	return number
}

func canonicalStorage(sign int8, value *largeNumber) Number {
	if value.c.IsUint64() && value.e2.IsInt64() && value.e5.IsInt64() {
		e2, e5 := value.e2.Int64(), value.e5.Int64()
		if e2 >= math.MinInt32 && e2 <= math.MaxInt32 && e5 >= math.MinInt32 && e5 <= math.MaxInt32 {
			return Number{sign: sign, coefficient: value.c.Uint64(), exponent2: int32(e2), exponent5: int32(e5)}
		}
	}
	return Number{sign: sign, large: value}
}

// parts borrows immutable large storage or materializes a compact number for an
// operation that requires arbitrary precision. Callers must not mutate results.
func (n Number) parts() (*big.Int, *big.Int, *big.Int) {
	if n.large != nil {
		return n.large.c, n.large.e2, n.large.e5
	}
	return new(big.Int).SetUint64(n.coefficient), big.NewInt(int64(n.exponent2)), big.NewInt(int64(n.exponent5))
}

func (n Number) nativeMagnitude() (uint64, bool) {
	if n.large != nil || n.exponent2 < 0 || n.exponent5 < 0 || n.exponent2 >= 64 || n.exponent5 > 27 {
		return 0, false
	}
	magnitude := n.coefficient
	if magnitude > math.MaxUint64>>uint(n.exponent2) {
		return 0, false
	}
	magnitude <<= uint(n.exponent2)
	for range n.exponent5 {
		if magnitude > math.MaxUint64/5 {
			return 0, false
		}
		magnitude *= 5
	}
	return magnitude, true
}
