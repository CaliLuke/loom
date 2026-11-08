package ir

import (
	"fmt"
	"maps"
	"slices"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
)

// projectedContributor is a declaration analyzed in the wire context of an
// HTTP body occurrence: its baseline component reference and the projected
// occurrence and plan that project it.
type projectedContributor struct {
	schema      *Schema
	key         string
	fingerprint string
	hint        string
	claim       string
	component   *componentAcquisition
	complete    bool
	attribute   *expr.AttributeExpr
	plan        expr.ValuePlanNode
}

// bodyProjectionAt returns the HTTP body projection recorded on the graph
// attribute that owns the analyzed position: the source paired with the plan
// position of a text, form or multipart occurrence, or attribute itself.
func (a *Analyzer) bodyProjectionAt(attribute *expr.AttributeExpr) (*expr.HTTPBodyProjection, *expr.AttributeExpr) {
	if source := a.parameterSources[a.plan]; source != nil {
		return source.HTTPBodyProjection(), source
	}
	return attribute.HTTPBodyProjection(), attribute
}

// bodyWireContext returns the wire context whose contributors are being
// analyzed, or captures the context of the body occurrence owner.
func (a *Analyzer) bodyWireContext(owner *expr.AttributeExpr) *representation.WireContext {
	if a.projectionWire != nil {
		return a.projectionWire
	}
	wire, err := representation.CaptureWireContext(owner, a.preparedExamples, a.plan)
	if err != nil {
		panic(fmt.Errorf("OpenAPI representation: %w", err))
	}
	return wire
}

// contributor analyzes declaration projected into wire. Equal projections
// share one analysis.
func (a *Analyzer) contributor(wire *representation.WireContext, projection *expr.HTTPBodyProjection, declaration expr.UserType, context string) *projectedContributor {
	projected, err := representation.ProjectContributor(wire, projection, declaration)
	if err != nil {
		panic(fmt.Errorf("OpenAPI representation: %w", err))
	}
	if prior := a.contributors[projected.Key]; prior != nil {
		return prior
	}
	result := &projectedContributor{schema: &Schema{}, key: projected.Key, attribute: projected.Attribute}
	a.contributors[projected.Key] = result
	context = childExampleContext(context, "contributor", declaration.Name())
	if !a.customExampleValue {
		generator := exampleGeneratorForAttribute(a.rand, projected.Attribute, a.closeObjects, context)
		if err := representation.PrepareTargetValues(projected.Target, projected.Attribute, generator, true); err != nil {
			panic(fmt.Errorf("OpenAPI representation: %w", err))
		}
	}
	previous := a.projectionWire
	a.projectionWire = wire
	defer func() {
		a.projectionWire = previous
	}()
	plan := representationRoot(projected.Attribute, projected.Target)
	result.plan = plan
	restoreBinding := a.contributorBindingScope(result)
	defer restoreBinding()
	final := a.analyzePreparedSchemaPositions(projected.Attribute, context, projected.Target, plan)
	if final == nil {
		panic("OpenAPI contributor completed without a result")
	}
	*result.schema = *final
	a.copySchemaConstruction(final, result.schema)
	result.complete = true
	return result
}

// recordProjectedChildren analyzes, for the hinted root component name of a
// projected body or contributor, the direct children of its declaration in the
// same projection. Each representation of the root publishes them once the
// projection phase produces it.
func (a *Analyzer) recordProjectedChildren(attribute *expr.AttributeExpr, owner expr.UserType, name, context string) {
	projection, source := a.bodyProjectionAt(attribute)
	if projection == nil {
		return
	}
	wire := a.bodyWireContext(source)
	id := a.plan.TargetDeclarationID()
	if id == "" {
		id = owner.ID()
	}
	declaration := a.declarationTypes[id]
	if declaration == nil {
		return
	}
	parent, ok := a.declarationComponentName(declaration)
	if !ok {
		return
	}
	for _, child := range a.discriminatorFamilies[parent] {
		a.projectedChildren[name] = append(a.projectedChildren[name], a.contributor(wire, projection, child, context))
	}
}

// publishRecordedChildren publishes the children recorded for every projected
// hinted root on its baseline component, once its occurrence is complete.
func (a *Analyzer) publishRecordedChildren() {
	for _, name := range slices.Sorted(maps.Keys(a.projectedChildren)) {
		a.publishProjectedChildren(toRef(name), toRef(name))
	}
}

// publishProjectedChildren projects the children recorded for the baseline
// root component baselineRef and records them as publication dependencies of
// its representation projectedRef.
func (a *Analyzer) publishProjectedChildren(baselineRef, projectedRef string) {
	baseline, baselineLocal := schemaComponentName(baselineRef)
	projected, projectedLocal := schemaComponentName(projectedRef)
	pair := [2]string{baseline, projected}
	if !baselineLocal || !projectedLocal || len(a.projectedChildren[baseline]) == 0 || a.publishedChildren[pair] {
		return
	}
	a.publishedChildren[pair] = true
	for _, child := range a.projectedChildren[baseline] {
		published := a.projectBaseline(child.schema, child.attribute, child.plan)
		if name, local := schemaComponentName(published.Ref); local {
			a.addDiscriminatorChild(projected, name)
		}
	}
}

// recordProjectedMapping captures the walk-stable discriminator record before
// traversing the target. Nullable wrapping can replace the outer schema while
// retaining this exact record; installation occurs before projection copies it.
func (a *Analyzer) recordProjectedMapping(discriminator *Discriminator, key string, attribute *expr.AttributeExpr, target expr.UserType, context string) {
	discriminator.Mapping[key] = ""
	if projection, source := a.bodyProjectionAt(attribute); projection != nil {
		wire := a.bodyWireContext(source)
		if a.projectionMappings[discriminator] == nil {
			a.projectionMappings[discriminator] = make(map[string]*projectedContributor)
		}
		a.projectionMappings[discriminator][key] = a.contributor(wire, projection, target, context)
		return
	}
	reference := a.analyzeReferenceSchema(target, context)
	if reference != nil {
		discriminator.Mapping[key] = reference.Ref
	}
}

// projectedDiscriminatorProperty returns the property that a hint of the
// projected attribute renders: the physical name of the member the authored
// property binds to. It reports false when the projection removed that member,
// so the hint is suppressed on this occurrence only. An unbound property is
// returned unchanged.
func (a *Analyzer) projectedDiscriminatorProperty(attribute *expr.AttributeExpr, property string) (string, bool) {
	projection, _ := a.bodyProjectionAt(attribute)
	if projection == nil {
		return property, true
	}
	member, bound := expr.DiscriminatorMember(projection.Source, property)
	if !bound {
		return property, true
	}
	if slices.Contains(projection.Excluded, member) {
		return "", false
	}
	if object := expr.AsObject(attribute.Type); object != nil {
		for _, field := range *object {
			if expr.AttributeName(field.Name) == member {
				return expr.JSONFieldName(expr.ElementName(field.Name), field.Attribute), true
			}
		}
	}
	return property, true
}

// componentAcquisitionFingerprint separates provisional traversal contexts
// before their publication dependencies exist. Baseline declaration identity
// stays semantic; the completed graph registry decides final equivalence.
// Same-shaped declarations that author different annotations are different
// contracts, so provisional reuse also keys on the component's annotations.
func (a *Analyzer) componentAcquisitionFingerprint(attribute *expr.AttributeExpr, declaration expr.UserType, fingerprint string) string {
	acquisition := fingerprint
	if annotations := a.componentAnnotationFingerprint(componentAttribute(attribute, declaration), a.examplePlan.Underlying()); annotations != "" {
		acquisition = fingerprintString(acquisition + "\x00annotations:" + annotations)
	}
	projection, source := a.bodyProjectionAt(attribute)
	if projection == nil {
		return acquisition
	}
	wire := a.bodyWireContext(source)
	return fingerprintString(acquisition + "\x00body:" + wire.ProjectionIdentity(projection))
}
