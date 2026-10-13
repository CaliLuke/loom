package codegen

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

type (
	// metadataRepresentation binds primitive transport storage to the service
	// occurrence without changing the service declaration or its constraints.
	metadataRepresentation struct {
		attribute      *expr.AttributeExpr
		elementCast    string
		defaultValue   any
		defaultLiteral string
	}
)

func metadataStorage(attribute *expr.AttributeExpr, required bool, scope *codegen.NameScope) (*metadataRepresentation, error) {
	physical, err := metadataStorageAttribute(attribute)
	if err != nil {
		return nil, err
	}
	result := &metadataRepresentation{attribute: physical}
	if source := expr.AsArray(attribute.Type); source != nil && expr.IsAlias(source.ElemType.Type) {
		result.elementCast = scope.GoTypeRef(expr.AsArray(physical.Type).ElemType)
	}
	constraints, err := expr.EffectiveConstraintsFor(attribute)
	if err != nil {
		return nil, err
	}
	var hasDefault bool
	result.defaultValue, hasDefault = constraints.Default()
	if hasDefault && !required {
		result.defaultLiteral, err = serviceTypeContext("", scope).DefaultValue(physical, result.defaultValue)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func metadataStorageAttribute(attribute *expr.AttributeExpr) (*expr.AttributeExpr, error) {
	constraints, err := expr.EffectiveConstraintsFor(attribute)
	if err != nil {
		return nil, err
	}
	physical := expr.DupAtt(attribute)
	physical.Type = unalias(attribute).Type
	physical.Validation = constraints.Validation().Lowered()
	if array := expr.AsArray(physical.Type); array != nil {
		element, err := metadataStorageAttribute(array.ElemType)
		if err != nil {
			return nil, err
		}
		copy := *array
		copy.ElemType = element
		physical.Type = &copy
	}
	return physical, nil
}
