package ir

import (
	"reflect"
	"strings"

	"github.com/CaliLuke/loom/expr"
)

func (a *Analyzer) analyzeProjectedResult(attr *expr.AttributeExpr, t expr.UserType, context string, noRef bool) (*Schema, bool) {
	resultType, ok := t.(*expr.ResultTypeExpr)
	if !ok {
		return nil, false
	}
	view, hasView := attr.Meta.Last(expr.ViewMetaKey)
	if !hasView {
		view, hasView = resultType.Meta.Last(expr.ViewMetaKey)
	}
	if !hasView {
		return nil, false
	}
	projected, err := expr.Project(resultType, view)
	if err != nil {
		panic(err)
	}
	projectedAttr := expr.DupAtt(attr)
	projectedAttr.Type = projected
	projectedAttr.Validation = projected.Validation
	if projectedAttr.Meta == nil {
		projectedAttr.Meta = make(expr.MetaExpr)
	}
	projectedAttr.Meta[projectedResultMetaKey] = []string{"true"}
	delete(projectedAttr.Meta, expr.ViewMetaKey)
	return a.analyzeSchema(projectedAttr, context, noRef), true
}

func (a *Analyzer) analyzeUserTypeOverlay(attr *expr.AttributeExpr, t expr.UserType, context string, noRef bool) (*Schema, bool) {
	if !hasUserTypeConstraintOverlay(attr, t) {
		return nil, false
	}
	baseAttr := &expr.AttributeExpr{Type: t, Meta: userTypeNamingMeta(attr.Meta)}
	base := a.analyzeUserType(baseAttr, t, context, noRef)
	schema := &Schema{AllOf: []*Schema{base}}
	if base.Ref != "" {
		schema = &Schema{Ref: base.Ref}
		if value, ok := attr.Meta.Last("openapi:allOf:reference"); ok && metaBoolValue(value) {
			schema.Ref = ""
			schema.AllOf = []*Schema{{Ref: base.Ref}}
		}
	}
	a.applySchemaAttributeDetails(schema, attr, "", context)
	return schema, true
}

func userTypeNamingMeta(meta expr.MetaExpr) expr.MetaExpr {
	naming := make(expr.MetaExpr)
	for _, key := range []string{"openapi:typename", "openapi:typename:canonical"} {
		if values, ok := meta[key]; ok {
			naming[key] = append([]string(nil), values...)
		}
	}
	return naming
}

func componentAttribute(attr *expr.AttributeExpr, t expr.UserType) *expr.AttributeExpr {
	componentAttr := expr.DupAtt(t.Attribute())
	if attr == nil {
		return componentAttr
	}
	if componentAttr.Meta == nil {
		componentAttr.Meta = make(expr.MetaExpr)
	}
	_, aliasesUserType := componentAttr.Type.(expr.UserType)
	if aliasesUserType {
		delete(componentAttr.Meta, "openapi:typename")
		delete(componentAttr.Meta, "openapi:typename:canonical")
	}
	for key, values := range attr.Meta {
		if aliasesUserType && (key == "openapi:typename" || key == "openapi:typename:canonical") {
			continue
		}
		componentAttr.Meta[key] = append([]string(nil), values...)
	}
	if attr.Title != "" {
		componentAttr.Title = attr.Title
	}
	if attr.Description != "" {
		componentAttr.Description = attr.Description
	}
	if attr.Validation != nil {
		componentAttr.Validation = attr.Validation
	}
	if attr.DefaultValue != nil {
		componentAttr.DefaultValue = attr.DefaultValue
	}
	if len(attr.UserExamples) > 0 && !expr.AllowsNull(attr) {
		componentAttr.UserExamples = attr.UserExamples
	}
	return componentAttr
}

func (a *Analyzer) reuseEquivalentCanonicalSchema(s *Schema, attr *expr.AttributeExpr, t expr.UserType, typeName, fingerprint, metaName string) bool {
	existingFingerprint, ok := a.schemaFingerprints[typeName]
	if !ok || existingFingerprint == fingerprint {
		return false
	}
	componentContext := exampleContext("component", typeName)
	candidate := a.analyzeSchema(componentAttribute(attr, t), componentContext, true)
	if !reflect.DeepEqual(a.schemas[typeName], candidate) {
		return false
	}
	s.Ref = toRef(typeName)
	a.registerSchemaRef(fingerprint, s.Ref, metaName)
	return true
}

func hasUserTypeConstraintOverlay(attr *expr.AttributeExpr, userType expr.UserType) bool {
	if attr == nil {
		return false
	}
	base := userType.Attribute()
	if attr.Title != "" && attr.Title != base.Title ||
		attr.Description != "" && attr.Description != base.Description ||
		hasUserTypeValidationOverlay(attr, base) ||
		attr.DefaultValue != nil && !reflect.DeepEqual(attr.DefaultValue, base.DefaultValue) ||
		len(attr.UserExamples) > 0 && !reflect.DeepEqual(attr.UserExamples, base.UserExamples) {
		return true
	}
	for key, values := range attr.Meta {
		if key == "openapi:typename" || key == "openapi:typename:canonical" || key == expr.ViewMetaKey ||
			key == projectedResultMetaKey || key == "type:generate:force" || strings.HasPrefix(key, "struct:") {
			continue
		}
		if !reflect.DeepEqual(values, base.Meta[key]) {
			return true
		}
	}
	return false
}
