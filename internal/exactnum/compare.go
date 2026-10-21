package exactnum

import (
	"cmp"
	"math/big"
)

// interval encloses a positive value between lo*2^scale and hi*2^scale.
type interval struct {
	lo, hi, scale *big.Int
}

// Cmp returns -1, 0 or 1 according to the exact ordering of n and other.
func (n Number) Cmp(other Number) int {
	left, right := n, other
	if left.sign != right.sign {
		return sign(int(left.sign - right.sign))
	}
	if left.sign == 0 || left == right {
		return 0
	}
	if lm, ok := left.nativeMagnitude(); ok {
		if rm, ok := right.nativeMagnitude(); ok {
			return cmp.Compare(lm, rm) * int(left.sign)
		}
	}
	lc, l2, l5 := left.parts()
	rc, r2, r5 := right.parts()
	if lc.Cmp(rc) == 0 && l2.Cmp(r2) == 0 && l5.Cmp(r5) == 0 {
		return 0
	}
	d2 := new(big.Int).Sub(l2, r2)
	d5 := new(big.Int).Sub(l5, r5)
	precision := 64 + max(d2.BitLen(), d5.BitLen())
	for {
		l, r := exact(lc, precision), exact(rc, precision)
		if d5.Sign() > 0 {
			l = multiply(l, power5(d5, precision), precision)
		} else if d5.Sign() < 0 {
			r = multiply(r, power5(new(big.Int).Neg(d5), precision), precision)
		}
		l.scale.Add(l.scale, d2)
		order := 0
		if scaledCmp(l.hi, l.scale, r.lo, r.scale) < 0 {
			order = -1
		} else if scaledCmp(l.lo, l.scale, r.hi, r.scale) > 0 {
			order = 1
		}
		if order != 0 {
			return order * int(left.sign)
		}
		precision *= 2
	}
}

func exact(value *big.Int, precision int) interval {
	return normalize(interval{new(big.Int).Set(value), new(big.Int).Set(value), big.NewInt(0)}, precision)
}

func normalize(value interval, precision int) interval {
	for value.hi.BitLen() > precision {
		shift := value.hi.BitLen() - precision
		divisor := new(big.Int).Lsh(big.NewInt(1), uint(shift))
		quotient, remainder := new(big.Int), new(big.Int)
		quotient.QuoRem(value.hi, divisor, remainder)
		if remainder.Sign() != 0 {
			quotient.Add(quotient, big.NewInt(1))
		}
		value.hi = quotient
		value.lo.Rsh(value.lo, uint(shift))
		value.scale.Add(value.scale, big.NewInt(int64(shift)))
	}
	return value
}

func multiply(left, right interval, precision int) interval {
	return normalize(interval{
		new(big.Int).Mul(left.lo, right.lo), new(big.Int).Mul(left.hi, right.hi),
		new(big.Int).Add(left.scale, right.scale),
	}, precision)
}

func power5(exponent *big.Int, precision int) interval {
	result, base := exact(big.NewInt(1), precision), exact(big.NewInt(5), precision)
	remaining := new(big.Int).Set(exponent)
	for remaining.Sign() != 0 {
		if remaining.Bit(0) != 0 {
			result = multiply(result, base, precision)
		}
		remaining.Rsh(remaining, 1)
		if remaining.Sign() != 0 {
			base = multiply(base, base, precision)
		}
	}
	return result
}

func scaledCmp(left, leftScale, right, rightScale *big.Int) int {
	if left.Sign() == 0 || right.Sign() == 0 {
		return sign(left.Sign() - right.Sign())
	}
	leftMagnitude := new(big.Int).Add(leftScale, big.NewInt(int64(left.BitLen())))
	rightMagnitude := new(big.Int).Add(rightScale, big.NewInt(int64(right.BitLen())))
	if result := leftMagnitude.Cmp(rightMagnitude); result != 0 {
		return result
	}
	// Equal magnitudes imply alignment depends only on bounded mantissa lengths.
	l, r := new(big.Int).Set(left), new(big.Int).Set(right)
	if left.BitLen() < right.BitLen() {
		l.Lsh(l, uint(right.BitLen()-left.BitLen()))
	} else {
		r.Lsh(r, uint(left.BitLen()-right.BitLen()))
	}
	return l.Cmp(r)
}

func sign(value int) int {
	if value < 0 {
		return -1
	}
	if value > 0 {
		return 1
	}
	return 0
}
