package expr

import (
	"encoding/json/jsontext"
	"fmt"
	"strconv"

	loom "github.com/CaliLuke/loom/pkg"
)

// MatchJSONSchema checks wire membership against the captured JSON schema
// projection, independently of source selection and concrete Go decoding.
func (p ValuePlan) MatchJSONSchema(wire jsontext.Value) (bool, error) {
	if p.root == nil || p.root.schemaOnly || p.root.codec != ValueCodecJSON {
		return false, fmt.Errorf("JSON schema matching requires a prepared JSON value plan")
	}
	return schemaJSON(p.root, wire), nil
}

// JSONShape lowers the captured JSON representation into a detached matching
// description for generated untagged codecs. It rejects opaque codecs and shapes
// outside the supported object/array/primitive grammar. Enum values are projected
// through this exact plan, retaining effective constraints and wire identity.
func (p ValuePlan) JSONShape() (*loom.JSONShape, error) {
	if p.root == nil || p.root.schemaOnly {
		return nil, fmt.Errorf("JSON matching requires a prepared value plan with enum values")
	}
	return lowerJSONShape(p.root, make(map[*valuePlanNode]*loom.JSONShape))
}

func lowerJSONShape(node *valuePlanNode, seen map[*valuePlanNode]*loom.JSONShape) (*loom.JSONShape, error) {
	if prior := seen[node]; prior != nil {
		return prior, nil
	}
	if node == nil || node.codec != ValueCodecJSON {
		return nil, fmt.Errorf("JSON matching cannot lower an opaque codec")
	}
	shape := &loom.JSONShape{Nullable: node.nullable, Closed: !node.schemaUnknown, NonNullableElements: node.nonNullableElements}
	seen[node] = shape
	rules, err := lowerJSONShapeRules(node)
	if err != nil {
		return nil, err
	}
	shape.Rules = rules
	if node.alias != nil {
		shape.Kind = "alias"
		shape.Child, err = lowerJSONShape(node.alias, seen)
		return shape, err
	}
	switch node.kind {
	case AnyKind:
		shape.Kind = "any"
	case BooleanKind:
		shape.Kind = "boolean"
	case StringKind:
		shape.Kind = "string"
	case BytesKind:
		shape.Kind = "bytes"
	case IntKind, Int32Kind, Int64Kind, UIntKind, UInt32Kind, UInt64Kind:
		shape.Kind = "integer"
	case Float32Kind, Float64Kind:
		shape.Kind = "number"
	case ArrayKind:
		shape.Kind = "array"
		shape.Child, err = lowerJSONShape(node.element, seen)
	case ObjectKind:
		shape.Kind = "object"
		for _, field := range node.members {
			if !field.visible {
				continue
			}
			child, childErr := lowerJSONShape(field.node, seen)
			if childErr != nil {
				return nil, childErr
			}
			shape.Fields = append(shape.Fields, loom.JSONShapeField{Name: field.wire, Required: field.required, Shape: child})
		}
	default:
		err = fmt.Errorf("JSON matching cannot lower %v", node.kind)
	}
	return shape, err
}

func jsonShapeBound(value *float64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatFloat(*value, 'g', -1, 64)
}

func lowerJSONShapeRules(node *valuePlanNode) (loom.JSONShapeRules, error) {
	var result loom.JSONShapeRules
	if node.validation != nil {
		rules := node.validation
		result = loom.JSONShapeRules{
			Minimum: jsonShapeBound(rules.Minimum), Maximum: jsonShapeBound(rules.Maximum),
			ExclusiveMinimum: jsonShapeBound(rules.ExclusiveMinimum), ExclusiveMaximum: jsonShapeBound(rules.ExclusiveMaximum),
			Patterns: rules.Patterns(),
		}
		if rules.MinLength != nil {
			result.MinLength = new(*rules.MinLength)
		}
		if rules.MaxLength != nil {
			result.MaxLength = new(*rules.MaxLength)
		}

		for _, format := range rules.Formats() {
			result.Formats = append(result.Formats, loom.Format(format))
		}
	}
	for _, clause := range node.enumClauses {
		values := make([]jsontext.Value, 0, len(clause))
		for _, member := range clause {
			observed, failure := observeJSON(node, ValueRoleEnum, member)
			if failure != nil {
				return loom.JSONShapeRules{}, fmt.Errorf("JSON matching cannot project enum: %s", failure.message)
			}
			wire, failure := constructJSON(node, observed)
			if failure != nil {
				return loom.JSONShapeRules{}, fmt.Errorf("JSON matching cannot encode enum: %s", failure.message)
			}
			values = append(values, wire)
		}
		result.Enums = append(result.Enums, values)
	}
	return result, nil
}
