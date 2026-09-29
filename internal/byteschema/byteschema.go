// Package byteschema projects decoded byte lengths into JSON string constraints
// for the standard padded base64 language accepted by the built-in JSON codec.
package byteschema

import "fmt"

type (
	// Bounds holds inclusive decoded byte lengths. Nil endpoints are unbounded.
	Bounds struct {
		// Minimum is the inclusive lower decoded length, or nil if unbounded.
		Minimum *int
		// Maximum is the inclusive upper decoded length, or nil if unbounded.
		Maximum *int
	}

	// Constraint describes a conjunction of an excluded-character check and a
	// disjunction of residue branches. It applies only to present JSON strings;
	// callers own type, nullability, metadata and other schema constraints.
	Constraint struct {
		// Unsatisfiable means no nonnegative decoded length satisfies the bounds.
		// Emit an unsatisfiable schema such as not:{}, never an empty anyOf.
		Unsatisfiable bool
		// ForbiddenPattern must be negated, for example with JSON Schema not.
		// This excludes line terminators despite ECMA-262 end-anchor semantics.
		ForbiddenPattern string
		// Branches are alternatives ordered by decoded length modulo three.
		Branches []Branch
	}

	// Branch combines an anchored base64 grammar with encoded character bounds.
	// Unused padding bits are unrestricted, matching the actual JSON decoder.
	Branch struct {
		// Pattern uses syntax shared by ECMA-262 and Go regular expressions.
		Pattern string
		// MinLength is the inclusive encoded character lower bound.
		MinLength int
		// MaxLength is the inclusive encoded upper bound, or nil if unbounded.
		MaxLength *int
	}
)

// Intersect conjoins an occurrence's decoded length bounds before projection.
// The returned endpoints are owned copies. Negative and inconsistent endpoints
// are retained so Project can distinguish vacuous lower bounds and empty ranges.
func Intersect(bounds []Bounds) Bounds {
	var result Bounds
	for _, bound := range bounds {
		if bound.Minimum != nil && (result.Minimum == nil || *bound.Minimum > *result.Minimum) {
			value := *bound.Minimum
			result.Minimum = &value
		}
		if bound.Maximum != nil && (result.Maximum == nil || *bound.Maximum < *result.Maximum) {
			value := *bound.Maximum
			result.Maximum = &value
		}
	}
	return result
}

// Project converts decoded byte bounds to a constant-size string constraint.
// A nil minimum means zero; a negative minimum is vacuous. A nil maximum is
// unbounded; a negative maximum or contradictory range is unsatisfiable.
// Returned storage is owned by the caller. Errors identify encoded bounds that
// cannot be represented exactly by both Go int and JSON number validators;
// callers must add the affected schema occurrence's path.
func Project(minimum, maximum *int) (Constraint, error) {
	lower := 0
	if minimum != nil {
		lower = max(0, *minimum)
	}
	var upper *int
	if maximum != nil {
		copy := *maximum
		upper = &copy
		if copy < lower {
			return Constraint{Unsatisfiable: true}, nil
		}
	}
	result := Constraint{ForbiddenPattern: "[^A-Za-z0-9+/=]", Branches: make([]Branch, 0, 3)}
	for residue := range 3 {
		branch, present, err := projectResidue(lower, upper, residue)
		if err != nil {
			return Constraint{}, err
		}
		if present {
			result.Branches = append(result.Branches, branch)
		}
	}
	result.Unsatisfiable = len(result.Branches) == 0
	return result, nil
}

func projectResidue(lower int, upper *int, residue int) (Branch, bool, error) {
	minimumGroups := 0
	if lower > residue {
		distance := lower - residue
		minimumGroups = distance / 3
		if distance%3 != 0 {
			minimumGroups++
		}
	}
	var maximumGroups *int
	if upper != nil {
		if *upper < residue {
			return Branch{}, false, nil
		}
		groups := (*upper - residue) / 3
		if groups < minimumGroups {
			return Branch{}, false, nil
		}
		maximumGroups = &groups
	}
	patterns := [3]string{
		"^([A-Za-z0-9+/]{4})*$",
		"^([A-Za-z0-9+/]{4})*[A-Za-z0-9+/]{2}==$",
		"^([A-Za-z0-9+/]{4})*[A-Za-z0-9+/]{3}=$",
	}
	minimum, err := encodedLength(minimumGroups, residue)
	if err != nil {
		return Branch{}, false, err
	}
	branch := Branch{Pattern: patterns[residue], MinLength: minimum}
	if maximumGroups != nil {
		maximum, err := encodedLength(*maximumGroups, residue)
		if err != nil {
			return Branch{}, false, err
		}
		branch.MaxLength = &maximum
	}
	return branch, true, nil
}

func encodedLength(groups, residue int) (int, error) {
	// JSON validators commonly represent schema keywords using binary64. Both
	// that representation and the adapter's native int must remain exact.
	limit := min(uint64(^uint(0)>>1), uint64(1<<53-1))
	tail := uint64(0)
	if residue != 0 {
		tail = 4
	}
	if uint64(groups) > (limit-tail)/4 {
		return 0, fmt.Errorf("encoded byte length exceeds exact schema integer limit %d", limit)
	}
	return int(uint64(groups)*4 + tail), nil
}
