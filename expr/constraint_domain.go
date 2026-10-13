package expr

import "fmt"

// validationDomainErrors checks authored endpoint eligibility and interval
// consistency without imposing a concrete machine representation on numbers.
// Effective constraints use it both before and after folding inherited rules.
func validationDomainErrors(v *ValidationExpr) []string {
	if v == nil {
		return nil
	}
	var errs []string
	for _, pair := range []struct {
		lower, upper         *NumericBound
		lowerName, upperName string
		exclusive            bool
	}{
		{v.Minimum, v.Maximum, "minimum", "maximum", false},
		{v.Minimum, v.ExclusiveMaximum, "minimum", "exclusive maximum", true},
		{v.ExclusiveMinimum, v.ExclusiveMaximum, "exclusive minimum", "exclusive maximum", true},
		{v.ExclusiveMinimum, v.Maximum, "exclusive minimum", "maximum", true},
	} {
		if pair.lower == nil || pair.upper == nil {
			continue
		}
		comparison := pair.lower.Cmp(*pair.upper)
		if comparison > 0 || (comparison == 0 && pair.exclusive) {
			relation := "greater than"
			if pair.exclusive {
				relation = "greater than or equal to"
			}
			errs = append(errs, fmt.Sprintf("%s is %s %s", pair.lowerName, relation, pair.upperName))
		}
	}
	if v.MinLength != nil && *v.MinLength < 0 {
		errs = append(errs, "min length must be non-negative")
	}
	if v.MaxLength != nil && *v.MaxLength < 0 {
		errs = append(errs, "max length must be non-negative")
	}
	if v.MinLength != nil && v.MaxLength != nil && *v.MinLength > *v.MaxLength {
		errs = append(errs, "min length is greater than max length")
	}
	return errs
}

func validateEffectiveDomain(context *ValueContext, graph *valueOccurrenceGraph, node *valueOccurrenceNode) error {
	// Validate authored layers before a tighter valid minimum hides a negative one.
	for _, layer := range effectiveConstraintLayers(node) {
		if errs := validationDomainErrors(layer.attribute.Validation); len(errs) > 0 {
			source := constraintSource(context, graph, layer)
			return fmt.Errorf("%sinvalid constraint bounds from %q: %s", effectiveConstraintPrefix(node), source.Declaration, errs[0])
		}
	}
	constraints := node.constraints
	if errs := validationDomainErrors(&constraints.validation.rules); len(errs) > 0 {
		lower, upper := constraints.LowerBound(), constraints.UpperBound()
		if lower.Present && upper.Present {
			return fmt.Errorf("%scontradictory bounds from %q and %q: %s", effectiveConstraintPrefix(node), lower.Provenance.Declaration, upper.Provenance.Declaration, errs[0])
		}
		return fmt.Errorf("%scontradictory inherited length bounds: %s", effectiveConstraintPrefix(node), errs[0])
	}
	return nil
}
