package codegen

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/representation"
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
		var defaultPlan codegen.DefaultValueCode
		defaultPlan, err = serviceTypeContext("", scope).DefaultValue(physical, result.defaultValue)
		result.defaultLiteral = defaultPlan.Expression
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
	if representation.CustomString(attribute) {
		delete(physical.Meta, "struct:field:type")
	}
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

// metadataServiceField binds a logical mapping to its actual service occurrence.
func metadataServiceField(service *expr.AttributeExpr, name string) (string, *expr.AttributeExpr, bool) {
	if !expr.IsObject(service.Type) {
		return "", service, false
	}
	key, attribute := service.FindAttribute(name)
	return codegen.GoifyAtt(attribute, key, true), attribute, service.IsPrimitivePointer(key, true)
}

// metadataPublicationPlan derives custom physical-to-text conversion from the
// shared transform owner; native metadata keeps its existing primitive codec.
func metadataPublicationPlan(source, target *expr.AttributeExpr, ctx *codegen.AttributeContext) (*codegen.TransformPlan, error) {
	if !representation.ContainsCustomString(source) {
		return nil, nil
	}
	plan, err := codegen.GoTransform(source, target, "metadataSource", "metadataText", ctx, ctx, "", true)
	if err != nil {
		return nil, err
	}
	return &plan, nil
}
