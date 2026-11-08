package ir

import (
	"reflect"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func (a *Analyzer) analyzeProjectedResult(attr *expr.AttributeExpr, t expr.UserType, context string, noRef bool, binding *projectedContributor) (*Schema, bool) {
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
	restore := a.contributorBindingScope(binding)
	defer restore()
	return a.analyzeSchema(projectedAttr, context, noRef), true
}

func (a *Analyzer) analyzeUserTypeOverlay(attr *expr.AttributeExpr, t expr.UserType, context string, noRef bool, binding *projectedContributor) (*Schema, bool) {
	if !hasUserTypeConstraintOverlay(attr, t) {
		return nil, false
	}
	componentMeta, occurrenceMeta := userTypeOverlayMetadata(attr.Meta)
	baseAttr := &expr.AttributeExpr{Type: t, Meta: componentMeta}
	expr.CopyObjectOpenness(baseAttr, attr)
	var base *Schema
	if negated := negatedDeclarationAnnotations(occurrenceMeta, t); len(negated) > 0 {
		// A reference cannot negate an annotation its declaration asserts. The
		// occurrence owns an inline expansion of the declaration that carries
		// the authored value; the shared component and its name are unchanged.
		declaration := componentAttribute(baseAttr, t)
		for key, values := range negated {
			declaration.Meta[key] = values
		}
		base = a.analyzeNamedUnderlyingSchema(declaration, t, context, true)
	} else {
		base = a.analyzeUserType(baseAttr, t, context, noRef, binding)
	}
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
	occurrence := *attr
	occurrence.Meta = occurrenceMeta
	a.applySchemaAttributeDetails(schema, &occurrence, "", context)
	if base.Ref == "" {
		releaseInheritedAnnotations(base, &occurrence)
	}
	if hasEncodingSourceOverlay(attr, t) {
		schema = a.applyEncodingSourceType(schema, attr)
	}
	return schema, true
}

// userTypeOverlayMetadata separates component-owned identity and discriminator
// policy from reference-local annotations. A discriminator requires the complete
// branch graph and the original named declaration, never a $ref/allOf wrapper.
// Both maps own their values so later occurrence processing cannot alter source
// metadata or the component selected for another occurrence.
func userTypeOverlayMetadata(meta expr.MetaExpr) (component, occurrence expr.MetaExpr) {
	component, occurrence = make(expr.MetaExpr), make(expr.MetaExpr)
	for key, values := range meta {
		target := occurrence
		if key == "openapi:typename" || key == "openapi:typename:canonical" || strings.HasPrefix(key, "openapi:discriminator:") {
			target = component
		}
		target[key] = append([]string(nil), values...)
	}
	return component, occurrence
}

// negatedDeclarationAnnotations returns the boolean annotations the occurrence
// authors as false while the declaration chain asserts them as true.
func negatedDeclarationAnnotations(occurrence expr.MetaExpr, declaration expr.UserType) expr.MetaExpr {
	negated := make(expr.MetaExpr)
	for _, key := range []string{"openapi:deprecated", "openapi:readOnly", "openapi:writeOnly"} {
		value, authored := occurrence.Last(key)
		if !authored || metaBoolValue(value) {
			continue
		}
		if inherited, ok := inheritedDeclarationMeta(declaration, key); ok && metaBoolValue(inherited) {
			negated[key] = append([]string(nil), occurrence[key]...)
		}
	}
	return negated
}

// inheritedDeclarationMeta returns the nearest value of key along a named
// declaration chain.
func inheritedDeclarationMeta(declaration expr.UserType, key string) (string, bool) {
	seen := make(map[expr.UserType]bool)
	for declaration != nil && !seen[declaration] {
		seen[declaration] = true
		attribute := declaration.Attribute()
		if value, ok := attribute.Meta.Last(key); ok {
			return value, true
		}
		next, ok := attribute.Type.(expr.UserType)
		if !ok {
			break
		}
		declaration = next
	}
	return "", false
}

// releaseInheritedAnnotations removes from an inline base the annotations that
// the overlaying occurrence authors. The wrapper owns them, so an authored
// value, including an explicit false, replaces the inherited one instead of
// sitting beside it. Assertions stay on the base.
func releaseInheritedAnnotations(base *Schema, occurrence *expr.AttributeExpr) {
	if occurrence.Title != "" {
		base.Title = ""
	}
	if occurrence.Description != "" {
		base.Description = ""
	}
	if occurrence.DefaultValue != nil {
		base.DefaultValue = nil
	}
	for key, field := range map[string]*bool{"openapi:readOnly": &base.ReadOnly, "openapi:writeOnly": &base.WriteOnly, "openapi:deprecated": &base.Deprecated} {
		if _, authored := occurrence.Meta[key]; authored {
			*field = false
		}
	}
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

func hasUserTypeConstraintOverlay(attr *expr.AttributeExpr, userType expr.UserType) bool {
	if attr == nil {
		return false
	}
	base := userType.Attribute()
	if hasEncodingSourceOverlay(attr, userType) {
		return true
	}
	if attr.Title != "" && attr.Title != base.Title ||
		attr.Description != "" && attr.Description != base.Description ||
		hasUserTypeValidationOverlay(attr, base) ||
		attr.DefaultValue != nil && !reflect.DeepEqual(attr.DefaultValue, base.DefaultValue) ||
		len(attr.UserExamples) > 0 && !reflect.DeepEqual(attr.UserExamples, base.UserExamples) {
		return true
	}
	for key, values := range attr.Meta {
		// An occurrence's openness selects a component variant through
		// componentAttribute; it is never an annotation beside the reference.
		if key == "openapi:typename" || key == "openapi:typename:canonical" || key == expr.ViewMetaKey ||
			key == projectedResultMetaKey || key == "http:encoding:sourceType" || key == "type:generate:force" || strings.HasPrefix(key, "struct:") ||
			expr.IsObjectOpennessMeta(key) || strings.HasPrefix(key, "openapi:discriminator:") {
			continue
		}
		if !reflect.DeepEqual(values, base.Meta[key]) {
			return true
		}
	}
	return false
}

func (a *Analyzer) componentName(attr *expr.AttributeExpr, t expr.UserType, fingerprint, acquisition, metaName string, canonical bool, schema *Schema, binding *projectedContributor) (string, bool) {
	typeName := codegen.Goify(schemaTypeName(t, metaName), true)
	if canonical {
		if metaName != "" {
			typeName = metaName
		}
		_, schemaExists := a.schemas[typeName]
		existingTypeID, hasTypeID := a.schemaTypeIDs[typeName]
		if schemaExists && hasTypeID && existingTypeID == t.ID() && a.schemaFingerprints[typeName] == acquisition {
			a.schemaTypeIDs[typeName] = t.ID()
			schema.Ref = toRef(typeName)
			return typeName, true
		}

		switch {
		case a.plan.Valid() && schemaExists && a.schemaNames[typeName].logical == a.logicalDeclaration(t) && a.schemaNames[typeName].baseline == a.componentBaselineFingerprint(attr, t):
			if owner := a.componentAcquisitions[typeName]; owner != nil && owner.semantic == fingerprint {
				schema.Ref = toRef(typeName)
				a.registerComponentReference(acquisition, schema.Ref, metaName, true)
				return typeName, true
			}
			typeName = a.Uniquify(typeName, fingerprint)
		case a.acquireCanonicalComparison(schema, attr, t, typeName, acquisition, metaName, binding):
			return typeName, true
		default:
			typeName = a.ClaimExplicitName(typeName, acquisition)
		}
	} else {
		typeName = a.Uniquify(typeName, fingerprint)
	}
	return typeName, false
}

func (a *Analyzer) analyzeUserType(attr *expr.AttributeExpr, t expr.UserType, context string, noRef bool, binding *projectedContributor) *Schema {
	if schema, projected := a.analyzeProjectedResult(attr, t, context, noRef, binding); projected {
		return schema
	}
	if a.asyncAcquisition != nil {
		return a.analyzeAsyncNamedBaseline(attr, t, context, noRef)
	}
	if schema, overlaid := a.analyzeUserTypeOverlay(attr, t, context, noRef, binding); overlaid {
		return schema
	}
	metaName, canonical := schemaTypeNaming(attr, t)
	if expr.IsAlias(t) && !canonical {
		return a.analyzeNamedUnderlyingSchema(componentAttribute(attr, t), t, context)
	}

	s := &Schema{}
	fingerprint := a.componentFingerprint(attr, t)
	acquisition := a.componentAcquisitionFingerprint(attr, t, fingerprint)

	refs, ok := a.schemasByFingerprint[acquisition]
	if !noRef && ok {
		if ref := findMatchingSchemaRef(refs, metaName, canonical, preparedComponentBinding{a.preparedExamples, a.examplePlan}); ref != "" {
			s.Ref = ref
			a.bindContributor(binding, ref, acquisition, metaName, canonical)
			return s
		}
	}

	typeName, reused := a.componentName(attr, t, fingerprint, acquisition, metaName, canonical, s, binding)
	if reused {
		a.bindContributor(binding, s.Ref, acquisition, metaName, canonical)
		return s
	}
	s.Ref = toRef(typeName)
	a.registerComponentReference(acquisition, s.Ref, metaName, canonical)
	if _, ok := a.schemas[typeName]; !ok {
		a.schemaFingerprints[typeName] = acquisition
		componentAttr := componentAttribute(attr, t)
		a.schemaTypeIDs[typeName] = t.ID()
		naming := a.componentNaming(attr, t, typeName, fingerprint, canonical)
		componentContext := exampleContext("component", naming.annotationName)
		a.schemaNames[typeName] = naming
		entry := a.reserveComponent(typeName, acquisition, fingerprint, metaName, canonical, attr, t)
		a.bindContributor(binding, s.Ref, acquisition, metaName, canonical)
		a.schemas[typeName] = func() *Schema {
			restore := a.componentAnnotationScope(naming.logical, naming.baseline, componentContext)
			defer restore()
			return a.analyzeNamedUnderlyingSchema(componentAttr, t, componentContext, true)
		}()
		entry.complete = true
		if _, hinted := a.discriminatorHints.parentRoots[a.schemas[typeName]]; hinted {
			a.recordProjectedChildren(attr, t, typeName, componentContext)
		}
	}
	return s
}

func (a *Analyzer) componentNaming(attr *expr.AttributeExpr, t expr.UserType, allocated, semantic string, canonical bool) representationComponentName {
	// Only a fresh baseline allocation reaches this function. Its annotation
	// context belongs to that allocation; logical or public-name equality does
	// not authorize borrowing another component's sample binding. Actual reuse
	// returns earlier, and representation projection copies the acquired record.
	annotationName := allocated
	baseName := componentPublicName(attr, t)
	authority := a.logicalDeclaration(t)
	hint, _ := schemaTypeNaming(attr, t)
	desired := annotationName
	// Contexts and inline declarations may share a preferred family. The final
	// graph comparison still includes retained annotations and publication.
	for _, prior := range a.schemaNames {
		compatible := prior.logical == authority || compatibleSchemaName(schemaRef{ref: toRef(prior.desired), explicitName: prior.hint}, hint, canonical)
		if compatible && prior.semantic == semantic && prior.desired < desired {
			desired = prior.desired
		}
	}
	if canonical {
		desired = baseName
	}
	return representationComponentName{desired: desired, hint: hint, annotationName: annotationName, logical: authority, baseline: a.componentBaselineFingerprint(attr, t), semantic: semantic, explicit: canonical}
}
