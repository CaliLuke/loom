package expr

import (
	"encoding/json/jsontext"
	"fmt"

	"github.com/CaliLuke/loom/internal/exactnum"
)

// validateHTTPSourceType checks the retained source assertion against the
// normalized non-null domain. Source absence is not an assertion. Null has its
// own schema/presence contract and is deliberately excluded from this check.
func validateHTTPSourceType(attribute *AttributeExpr, source string) error {
	if source == "absent" {
		return nil
	}
	compatible, err := httpSourceTypeCompatible(attribute, source, make(map[*AttributeExpr]bool))
	if err != nil {
		return fmt.Errorf("source type %q: %w", source, err)
	}
	if !compatible {
		return fmt.Errorf("source type %q does not contain the normalized value domain", source)
	}
	return nil
}

func httpSourceTypeCompatible(attribute *AttributeExpr, source string, active map[*AttributeExpr]bool) (bool, error) {
	if attribute == nil || active[attribute] {
		return false, nil
	}
	active[attribute] = true
	defer delete(active, attribute)
	typ := httpParameterBaseType(attribute.Type)
	if typ == nil {
		return false, nil
	}
	kind := jsonValueKind(typ)
	if union, ok := typ.(*Union); ok && !union.Untagged {
		kind = "object"
	}
	if kind == source && source != "integer" {
		return true, nil
	}
	if primitive, ok := typ.(Primitive); ok && primitiveToInlineJSONType(primitive) == source {
		return true, nil
	}
	if union, ok := typ.(*Union); ok && union.Untagged {
		compatible := true
		for _, branch := range union.Values {
			fits, err := httpSourceTypeCompatible(branch.Attribute, source, active)
			if err != nil {
				return false, err
			}
			compatible = compatible && fits
		}
		if compatible {
			return true, nil
		}
	}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	if err != nil {
		return false, err
	}
	constraints := occurrence.EffectiveConstraints()
	if values, present := constraints.EnumCandidates(); present {
		return httpSourceEnumCompatible(context, occurrence, source, values)
	}
	// Exact equal numeric bounds can refine a number to one integer, or to an
	// empty domain. Preserve that valid refinement instead of checking Go kinds.
	if kind == "number" && source == "integer" {
		lower, upper := constraints.LowerBound(), constraints.UpperBound()
		if lower.Present && upper.Present {
			order := lower.Value.Cmp(upper.Value)
			if order > 0 || order == 0 && (lower.Exclusive || upper.Exclusive) {
				return true, nil
			}
			if order == 0 {
				return lower.Value.IsInt(), nil
			}
		}
	}
	return false, nil
}

func httpSourceEnumCompatible(context *ValueContext, occurrence ValueOccurrence, source string, values []any) (bool, error) {
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{TargetOccurrence: occurrence, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	if err != nil {
		return false, err
	}
	for _, value := range values {
		result := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: value, ExplicitNull: value == nil, Origin: "source type refinement"}), ValueRoleEnum)
		projection := context.ProjectJSON(result, plan)
		wire, emitted := projection.JSON()
		if !emitted {
			return false, projection.Err()
		}
		if !httpSourceJSONKind(jsontext.Value(wire), source) {
			return false, nil
		}
	}
	return true, nil
}

func httpSourceJSONKind(value jsontext.Value, source string) bool {
	switch value.Kind() {
	case 'n':
		return true
	case '"':
		return source == "string"
	case 't', 'f':
		return source == "boolean"
	case '[':
		return source == "array"
	case '{':
		return source == "object"
	case '0':
		if source == "number" {
			return true
		}
		if source == "integer" {
			number, err := exactnum.ParseJSON(string(value))
			return err == nil && number.IsInt()
		}
	}
	return false
}
