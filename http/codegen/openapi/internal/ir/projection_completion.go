package ir

import (
	"fmt"
	"maps"
	"slices"

	"github.com/CaliLuke/loom/expr"
)

type (
	projectionState uint8
	// schemaProjection reserves the result identity before traversing its
	// closed baseline graph. Materialization never acquires a declaration.
	schemaProjection struct {
		baseline     *Schema
		attribute    *expr.AttributeExpr
		plan         expr.ValuePlanNode
		identity     string
		result       *Schema
		ref          string
		component    *componentProjection
		children     []projectionChild
		mapping      map[string]string
		fallback     string
		published    []*schemaProjection
		dependencies []*schemaProjection
		state        projectionState
	}
	projectionChild struct {
		slot schemaSlot
		plan *schemaProjection
	}
	componentProjection struct {
		name      string
		original  string
		body      *schemaProjection
		published []*schemaProjection
		planned   bool
		state     projectionState
	}
)

const (
	projectionReserved projectionState = iota
	projectionMaterializing
	projectionComplete
)

// completeSchemaScope is the sole acquisition-to-projection boundary. The
// caller holds the outer analysis depth until every mapping is installed and
// every projection is materialized, before canonical comparison observes it.
func (a *Analyzer) completeSchemaScope() {
	for _, entry := range a.componentAcquisitions {
		if !entry.complete {
			panic("OpenAPI projection reached an unfinished component acquisition")
		}
	}
	keys := slices.Sorted(maps.Keys(a.contributors))
	plans := make([]*schemaProjection, 0, len(keys))
	for _, key := range keys {
		contributor := a.contributors[key]
		if !contributor.complete || contributor.component == nil {
			panic("OpenAPI projection reached an unfinished contributor result")
		}
		plans = append(plans, a.planBaseline(contributor.schema, contributor.attribute, contributor.plan))
	}
	// All targets above have stable reserved references. Install every walk
	// slot before any projection copies a discriminator or reads body caches.
	for discriminator, mappings := range a.projectionMappings {
		for key, contributor := range mappings {
			plan := a.planBaseline(contributor.schema, contributor.attribute, contributor.plan)
			discriminator.Mapping[key] = plan.ref
		}
	}
	for _, plan := range plans {
		a.materializeProjection(plan)
	}
}

// publicationBaseline exposes the sealed logical mapping edges to identity
// readers. Physical projected references are output slots, not the authority
// for deciding the dependency closure. The captured edge survives installation.
func (a *Analyzer) publicationBaseline(schema *Schema) *Schema {
	return mapSchemaReferences(schema, func(ref, _ string) string {
		return ref
	}, func(source, copy *Schema) {
		for key, contributor := range a.projectionMappings[source.Discriminator] {
			if !contributor.complete {
				panic("OpenAPI publication closure reached an unfinished contributor")
			}
			copy.Discriminator.Mapping[key] = contributor.schema.Ref
		}
	})
}

// projectBaseline uses the same reservation and materialization owner for an
// occurrence root after scope completion as for the scope's deferred targets.
func (a *Analyzer) projectBaseline(baseline *Schema, attribute *expr.AttributeExpr, plan expr.ValuePlanNode) *Schema {
	if baseline == nil {
		return nil
	}
	projection := a.planBaseline(baseline, attribute, plan)
	a.materializeProjection(projection)
	return projection.result
}

func (a *Analyzer) planBaseline(baseline *Schema, attribute *expr.AttributeExpr, plan expr.ValuePlanNode) *schemaProjection {
	if baseline == nil {
		return nil
	}
	identity := a.representationIdentity(attribute, plan)
	if context := a.constructionContext(baseline, plan); context != "" {
		if identity != "" {
			identity += ";"
		}
		identity += "construction:" + context
	}
	key := schemaProjectionKey{baseline, plan, identity}
	if prior := a.projectionPlans[key]; prior != nil {
		return prior
	}
	result := &schemaProjection{baseline: baseline, attribute: attribute, plan: plan, identity: identity, result: &Schema{}, ref: baseline.Ref}
	a.projectionPlans[key] = result
	if identity == "" {
		result.result = baseline
	} else {
		if name, local := schemaComponentName(baseline.Ref); local {
			component := a.reserveProjectionComponent(name, plan, identity)
			result.component, result.ref = component, toRef(component.name)
			a.planProjectionComponent(component, attribute, plan)
		}
		a.planProjectionChildren(result)
	}
	if original, local := schemaComponentName(baseline.Ref); local {
		for _, contributor := range a.projectedChildren[original] {
			result.published = append(result.published, a.planBaseline(contributor.schema, contributor.attribute, contributor.plan))
		}
	}
	return result
}

func (a *Analyzer) reserveProjectionComponent(original string, plan expr.ValuePlanNode, identity string) *componentProjection {
	baseline := a.schemas[original]
	if baseline == nil {
		panic(fmt.Sprintf("OpenAPI baseline component %q is missing", original))
	}
	key := schemaProjectionKey{baseline, plan, identity}
	if prior := a.projectionComponents[key]; prior != nil {
		return prior
	}
	fingerprint := fingerprintString(original + ":" + identity)
	name := a.Uniquify(original, fingerprint)
	result := &componentProjection{name: name, original: original}
	a.projectionComponents[key] = result
	a.schemaFingerprints[name] = fingerprint
	a.schemaTypeIDs[name] = a.schemaTypeIDs[original]
	naming := a.schemaNames[original]
	if naming.desired == "" {
		naming = representationComponentName{desired: original, logical: original}
	}
	naming.json = true
	a.schemaNames[name] = naming
	return result
}

func (a *Analyzer) planProjectionComponent(component *componentProjection, attribute *expr.AttributeExpr, plan expr.ValuePlanNode) {
	if component.planned {
		return
	}
	component.planned = true
	component.body = a.planBaseline(a.schemas[component.original], attribute, plan)
	for _, child := range a.discriminatorFamilies[component.original] {
		if name, ok := a.declarationComponentName(child); ok {
			component.published = append(component.published, a.planDeclarationReference(child, name))
		}
	}
}

func (a *Analyzer) planProjectionChildren(projection *schemaProjection) {
	baseline := projection.baseline
	construction, recorded := a.constructions[baseline]
	if !recorded {
		if baseline.Ref == "" {
			panic("OpenAPI projected baseline has no construction record")
		}
		return
	}
	structural := construction.structuralPlan(projection.plan)
	alternatives := make(map[string]string)
	for _, slot := range construction.slots {
		childPlan, childAttribute := slot.plan(structural), projection.attribute
		switch {
		case slot.contributor != nil:
			childAttribute, childPlan = slot.contributor.attribute, slot.contributor.plan
		case slot.base != nil:
			childAttribute, childPlan = a.declarationRepresentation(slot.base)
		case !slot.positional():
			if !childPlan.Valid() {
				panic("OpenAPI baseline construction has no paired target child")
			}
			childAttribute = childPlan.Attribute()
		}
		source := slot.get(baseline)
		child := a.planBaseline(source, childAttribute, childPlan)
		projection.children = append(projection.children, projectionChild{slot, child})
		if slot.branch != "" && source.Ref != "" {
			if _, found := alternatives[source.Ref]; !found {
				alternatives[source.Ref] = child.ref
			}
		}
	}
	if discriminator := baseline.Discriminator; discriminator != nil {
		if discriminator.Mapping != nil {
			projection.mapping = make(map[string]string, len(discriminator.Mapping))
			for _, key := range slices.Sorted(maps.Keys(discriminator.Mapping)) {
				if contributor := a.projectionMappings[discriminator][key]; contributor != nil {
					target := a.planBaseline(contributor.schema, contributor.attribute, contributor.plan)
					projection.mapping[key] = target.ref
					projection.dependencies = append(projection.dependencies, target)
				} else {
					projection.mapping[key] = a.planDiscriminatorReference(projection, discriminator.Mapping[key], alternatives)
				}
			}
		}
		projection.fallback = a.planDiscriminatorReference(projection, discriminator.DefaultMapping, alternatives)
	}
}

func (a *Analyzer) planDiscriminatorReference(owner *schemaProjection, ref string, alternatives map[string]string) string {
	if projected, found := alternatives[ref]; found {
		return projected
	}
	name, local := schemaComponentName(ref)
	if !local {
		return ref
	}
	typ := a.rootDeclaration(name)
	if typ == nil {
		return ref
	}
	projection := a.planDeclarationReference(typ, name)
	owner.dependencies = append(owner.dependencies, projection)
	return projection.ref
}

func (a *Analyzer) planDeclarationReference(typ expr.UserType, name string) *schemaProjection {
	attribute, plan := a.declarationRepresentation(typ)
	return a.planBaseline(&Schema{Ref: toRef(name)}, attribute, plan)
}

func (a *Analyzer) materializeProjection(projection *schemaProjection) {
	if projection == nil || projection.state != projectionReserved {
		return
	}
	projection.state = projectionMaterializing
	if projection.identity != "" {
		*projection.result = *projection.baseline
		projection.result.Ref = projection.ref
		a.copySchemaConstruction(projection.baseline, projection.result)
		layers, bytes := byteOccurrenceLayers(projection.attribute, projection.plan)
		if bytes {
			projection.result.MinLength, projection.result.MaxLength = nil, nil
			if projection.result.Format == "binary" {
				projection.result.Format = ""
			}
			construction := a.constructions[projection.baseline]
			if construction.kind == expr.BytesKind && len(construction.slots) == 0 {
				projection.result.ContentEncoding = "base64"
				applyByteGate(projection.result, byteConstraintSchema(effectiveByteConstraint(projection.attribute, layers, "$")))
			}
		}
		for _, child := range projection.children {
			a.materializeProjection(child.plan)
			child.slot.set(projection.result, child.plan.result)
		}
		if discriminator := projection.baseline.Discriminator; discriminator != nil {
			copy := *discriminator
			copy.Mapping, copy.DefaultMapping = projection.mapping, projection.fallback
			projection.result.Discriminator = &copy
		}
		if projection.component != nil {
			a.materializeProjectionComponent(projection.component)
		}
	}
	for _, dependency := range projection.dependencies {
		a.materializeProjection(dependency)
	}
	for _, published := range projection.published {
		a.materializeProjection(published)
		if parent, local := schemaComponentName(projection.ref); local {
			if child, childLocal := schemaComponentName(published.ref); childLocal {
				a.addDiscriminatorChild(parent, child)
			}
		}
	}
	projection.state = projectionComplete
}

func (a *Analyzer) materializeProjectionComponent(component *componentProjection) {
	if component.state != projectionReserved {
		return
	}
	component.state = projectionMaterializing
	// Retain the result cell itself, never a value snapshot of an active body.
	a.schemas[component.name] = component.body.result
	a.materializeProjection(component.body)
	for _, child := range component.published {
		a.materializeProjection(child)
		if name, local := schemaComponentName(child.ref); local {
			a.addDiscriminatorChild(component.name, name)
		}
	}
	component.state = projectionComplete
}
