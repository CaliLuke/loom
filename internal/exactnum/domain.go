package exactnum

import "math"

// Domain identifies a finite native numeric carrier. Integer widths are 1..64;
// floating widths are 32 or 64. Unsigned applies only to integers.
type Domain struct {
	// Bits is the carrier width.
	Bits int
	// Float selects an IEEE floating carrier rather than an integer carrier.
	Float bool
	// Unsigned selects an unsigned integer carrier.
	Unsigned bool
}

// Limits returns the finite minimum and maximum native values of the domain,
// using the same scalar representations as Boundary.
func (d Domain) Limits() (any, any) {
	low, high := d.keys()
	return d.value(low), d.value(high)
}

// Boundary returns the first native value satisfying a lower bound, or the last
// satisfying an upper bound. Exclusive removes exact equality. False means no
// value in the domain satisfies the bound. Integers return int64 or uint64;
// floats return their exact native width. It never expands an endpoint's powers.
// An invalid internal carrier description panics.
func (n Number) Boundary(domain Domain, lower, exclusive bool) (any, bool) {
	low, high := domain.keys()
	accepts := func(key uint64) bool {
		value, ok := FromScalar(domain.value(key))
		if !ok {
			panic("exactnum: invalid finite domain value")
		}
		order := value.Cmp(n)
		if lower {
			return order > 0 || order == 0 && !exclusive
		}
		return order < 0 || order == 0 && !exclusive
	}
	if lower {
		if !accepts(high) {
			return nil, false
		}
		for low < high {
			mid := low + (high-low)/2
			if accepts(mid) {
				high = mid
			} else {
				low = mid + 1
			}
		}
	} else {
		if !accepts(low) {
			return nil, false
		}
		for low < high {
			difference := high - low
			mid := low + difference/2 + difference%2
			if accepts(mid) {
				low = mid
			} else {
				high = mid - 1
			}
		}
	}
	return domain.value(low), true
}

func (d Domain) keys() (uint64, uint64) {
	if d.Float {
		if d.Unsigned {
			panic("exactnum: unsigned float domain")
		}
		switch d.Bits {
		case 32:
			return 0x00800000, 0xff7fffff
		case 64:
			return 0x0010000000000000, 0xffefffffffffffff
		default:
			panic("exactnum: invalid floating domain width")
		}
	}
	if d.Bits < 1 || d.Bits > 64 {
		panic("exactnum: invalid integer domain width")
	}
	return 0, math.MaxUint64 >> (64 - d.Bits)
}

func (d Domain) value(key uint64) any {
	if d.Float {
		if d.Bits == 32 {
			bits := uint32(key)
			if bits>>31 == 0 {
				bits = ^bits
			} else {
				bits ^= 1 << 31
			}
			return math.Float32frombits(bits)
		}
		bits := key
		if bits>>63 == 0 {
			bits = ^bits
		} else {
			bits ^= 1 << 63
		}
		return math.Float64frombits(bits)
	}
	if d.Unsigned {
		return key
	}
	return int64(key - uint64(1)<<(d.Bits-1))
}
