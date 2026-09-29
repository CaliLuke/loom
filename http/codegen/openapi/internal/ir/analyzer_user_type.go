package ir

import (
	"reflect"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func (a *Analyzer) analyzeProjectedResult(attr *expr.AttributeExpr, t expr.UserType, context string, noRef bool) (*Schema, bool) {
	resultType, ok := t.(*expr.ResultTypeExpr)
	if !ok {
		return nil, false
	}
	view, hasView := resultTypeView(attr, resultType)
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
	if len(schema.AllOf) == 1 {
		a.constructions[schema] = schemaConstruction{slots: []schemaSlot{{location: schemaAllOf}}}
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

// resultTypeView returns the view that projects the result type of attr: the
// view of attr, or else the view of the result type. It reports false when
// neither declares a view, so that attr renders the result type as a whole.
func resultTypeView(attr *expr.AttributeExpr, resultType *expr.ResultTypeExpr) (string, bool) {
	if view, ok := attr.Meta.Last(expr.ViewMetaKey); ok {
		return view, true
	}
	return resultType.Meta.Last(expr.ViewMetaKey)
}

// componentFingerprint returns the fingerprint of the component schema of the
// user type t referenced by attr. A user type whose attribute is a result type
// without a view, such as the wrapper of a Body declared with a result type,
// renders the component of that result type. Its fingerprint is therefore the
// fingerprint of the result type's component, as the fingerprint of a user
// type that aliases a plain type already is, and not the fingerprint of a
// reference to the result type.
func (a *Analyzer) componentFingerprint(attr *expr.AttributeExpr, t expr.UserType) string {
	componentAttr := componentAttribute(attr, t)
	if resultType, ok := componentAttr.Type.(*expr.ResultTypeExpr); ok {
		if _, hasView := resultTypeView(componentAttr, resultType); !hasView {
			return a.componentFingerprint(componentAttr, resultType)
		}
	}
	return a.FingerprintAttribute(componentAttr)
}

func (a *Analyzer) reuseEquivalentCanonicalSchema(s *Schema, attr *expr.AttributeExpr, t expr.UserType, typeName, fingerprint, metaName string) bool {
	existingFingerprint, ok := a.schemaFingerprints[typeName]
	if !ok || existingFingerprint == fingerprint {
		return false
	}
	componentContext := exampleContext("component", typeName)
	candidate := a.analyzeUnderlyingSchema(componentAttribute(attr, t), componentContext, true)
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

func (a *Analyzer) componentName(attr *expr.AttributeExpr, t expr.UserType, fingerprint, metaName string, canonical bool, schema *Schema) (string, bool) {
	typeName := codegen.Goify(schemaTypeName(t, metaName), true)
	if canonical {
		if metaName != "" {
			typeName = metaName
		}
		_, schemaExists := a.schemas[typeName]
		existingTypeID, hasTypeID := a.schemaTypeIDs[typeName]
		if schemaExists && hasTypeID && existingTypeID == t.ID() && a.schemaFingerprints[typeName] == fingerprint {
			a.schemaTypeIDs[typeName] = t.ID()
			schema.Ref = toRef(typeName)
			return typeName, true
		}

		if a.reuseEquivalentCanonicalSchema(schema, attr, t, typeName, fingerprint, metaName) {
			return typeName, true
		}
		if a.plan.Valid() && schemaExists && a.schemaNames[typeName].logical == a.logicalDeclaration(t) && a.schemaNames[typeName].baseline == a.componentFingerprint(attr, t) {
			typeName = a.Uniquify(typeName, fingerprint)
		} else {
			typeName = a.ClaimExplicitName(typeName, fingerprint)
		}
	} else {
		typeName = a.Uniquify(typeName, fingerprint)
	}
	return typeName, false
}

func (a *Analyzer) analyzeUserType(attr *expr.AttributeExpr, t expr.UserType, context string, noRef bool) *Schema {
	if schema, projected := a.analyzeProjectedResult(attr, t, context, noRef); projected {
		return schema
	}
	if a.asyncAcquisition != nil {
		return a.analyzeAsyncNamedBaseline(attr, t, context, noRef)
	}
	if schema, overlaid := a.analyzeUserTypeOverlay(attr, t, context, noRef); overlaid {
		return schema
	}
	metaName, canonical := schemaTypeNaming(attr, t)
	if expr.IsAlias(t) && !canonical {
		return a.analyzeUnderlyingSchema(t.Attribute(), context)
	}

	s := &Schema{}
	fingerprint := a.componentFingerprint(attr, t)

	refs, ok := a.schemasByFingerprint[fingerprint]
	if !noRef && ok {
		if ref := findMatchingSchemaRef(refs, metaName, canonical); ref != "" {
			s.Ref = ref
			return s
		}
	}

	typeName, reused := a.componentName(attr, t, fingerprint, metaName, canonical, s)
	if reused {
		return s
	}
	s.Ref = toRef(typeName)
	a.registerSchemaRef(fingerprint, s.Ref, metaName)
	if _, ok := a.schemas[typeName]; !ok {
		a.schemaFingerprints[typeName] = fingerprint
		componentAttr := componentAttribute(attr, t)
		a.schemaTypeIDs[typeName] = t.ID()
		naming := a.componentNaming(attr, t, typeName, canonical)
		componentContext := exampleContext("component", naming.desired)
		a.schemaNames[typeName] = naming
		a.schemas[typeName] = func() *Schema {
			restore := a.componentAnnotationScope(naming.logical, naming.baseline, componentContext)
			defer restore()
			return a.analyzeUnderlyingSchema(componentAttr, componentContext, true)
		}()
	}
	return s
}

func (a *Analyzer) componentNaming(attr *expr.AttributeExpr, t expr.UserType, allocated string, canonical bool) representationComponentName {
	// Allocation owns a newly constructed baseline and its sample context.
	// Matching declaration or shape is not evidence of annotation reuse.
	desired := allocated
	baseName := componentPublicName(attr, t)
	authority := a.logicalDeclaration(t)
	if canonical {
		desired = baseName
	}
	return representationComponentName{desired: desired, logical: authority, baseline: a.componentFingerprint(attr, t), explicit: canonical}
}
