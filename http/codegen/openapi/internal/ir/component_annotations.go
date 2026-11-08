package ir

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/expr"
)

type (
	componentExamples struct {
		representative any
		values         []any
	}

	componentAnnotationIdentity struct {
		declaration string
		baseline    string
		context     string
		examplePlan expr.ValuePlanNode
	}

	componentAnnotationPosition struct {
		context     string
		fingerprint string
		examplePlan expr.ValuePlanNode
	}
)

// componentAnnotationScope preserves the authority formerly held by the first
// component schema. Representation variants change constraints, not the legacy
// component's example source or sampling context. This is annotation reuse, not
// a target-value projection cache; direct occurrence overlays remain outside it.
func (a *Analyzer) componentAnnotationScope(declaration, baseline, context string) func() {
	identity := componentAnnotationIdentity{
		declaration: declaration, baseline: baseline, context: context, examplePlan: a.examplePlan,
	}
	if a.componentAnnotations == nil {
		a.componentAnnotations = make(map[componentAnnotationIdentity]map[componentAnnotationPosition]componentExamples)
	}
	owner, ok := a.componentAnnotations[identity]
	if !ok {
		owner = make(map[componentAnnotationPosition]componentExamples)
		a.componentAnnotations[identity] = owner
	}
	prior := a.annotationOwner
	a.annotationOwner = owner
	return func() {
		a.annotationOwner = prior
	}
}

// copyComponentAnnotation reuses expr's callback-free builtin value snapshot.
// The synthetic primitive attribute has no named graph to register. Opaque
// custom values retain the same explicit host boundary as authored examples.
func copyComponentAnnotation(value any) any {
	if value == nil {
		return nil
	}
	return expr.DupAtt(&expr.AttributeExpr{Type: expr.Any, DefaultValue: value}).DefaultValue
}

// componentAnnotationFingerprint identifies the authored schema annotations of
// a component graph that the shape fingerprint deliberately omits: descriptions,
// defaults and rendered openapi metadata at every reachable position. Examples
// keep their per-position owner and identity metadata keeps its naming owner.
// The result is empty for an annotation-free graph.
func (a *Analyzer) componentAnnotationFingerprint(attribute *expr.AttributeExpr, position expr.ValuePlanNode) string {
	encoder := newCanonicalSchemaEncoder()
	annotated := false
	active := make(map[expr.DataType]bool)
	var visit func(*expr.AttributeExpr, expr.ValuePlanNode)
	visit = func(attr *expr.AttributeExpr, node expr.ValuePlanNode) {
		if attr == nil {
			encoder.writeString("nil")
			return
		}
		if a.writeComponentAnnotations(encoder, attr, node) {
			annotated = true
		}

		visit(attr.AdditionalAttributes, node.Additional())
		switch actual := attr.Type.(type) {
		case expr.UserType:
			if active[actual] {
				encoder.writeString("cycle:" + actual.ID())
				return
			}
			active[actual] = true
			visit(actual.Attribute(), node.Underlying())
			delete(active, actual)
		case *expr.Array:
			visit(actual.ElemType, node.Element())
		case *expr.Map:
			visit(actual.KeyType, node.Key())
			visit(actual.ElemType, node.Element())
		case *expr.Object:
			children := make(map[string]expr.ValuePlanNode, len(*actual))
			for _, member := range node.Members() {
				children[member.Name] = member.Node
			}
			for _, member := range *actual {
				encoder.writeString(member.Name)
				visit(member.Attribute, children[member.Name])
			}
		case *expr.Union:
			children := make(map[string]expr.ValuePlanNode, len(actual.Values))
			for _, branch := range node.Branches() {
				children[branch.Tag] = branch.Node
			}
			for _, branch := range actual.Values {
				encoder.writeString(branch.Name)
				visit(branch.Attribute, children[expr.UnionVariantTag(branch)])
			}
		}
	}
	visit(attribute, position)
	if !annotated {
		return ""
	}
	return fingerprintString(encoder.String())
}

// schemaAnnotationMeta reports whether key is rendered schema annotation
// metadata rather than component identity, discriminator policy or example
// ownership. Every rendered schema annotation key uses the openapi prefix.
func schemaAnnotationMeta(key string) bool {
	return strings.HasPrefix(key, "openapi:") && key != "openapi:typename" && key != "openapi:typename:canonical" &&
		!strings.HasPrefix(key, "openapi:discriminator:") && !strings.HasPrefix(key, "openapi:example")
}

// snapshot detaches all builtin containers without invoking custom codecs.
func (e componentExamples) snapshot() componentExamples {
	result := componentExamples{representative: copyComponentAnnotation(e.representative)}
	if e.values != nil {
		result.values = copyComponentAnnotation(e.values).([]any)
	}
	return result
}

// componentDefaultIdentity consumes only the checked projection. An opaque
// codec boundary is comparable only with the same captured owner; deterministic
// encounter numbers distinguish owners without serializing addresses or values.
func (a *Analyzer) componentDefaultIdentity(attribute *expr.AttributeExpr, position expr.ValuePlanNode) string {
	if attribute.DefaultValue == nil {
		return ""
	}
	if a.preparedExamples != nil {
		if annotation, found := a.preparedExamples.Annotations[position]; found && annotation.HasDefault {
			if wire, emitted := annotation.Default.Projection().JSON(); emitted {
				return "wire:" + string(wire)
			}
			if annotation.Default.Projection().Outcome() == expr.ProjectionUnsupported {
				binding := annotation.Default.DefaultOwner()
				if binding == (expr.DefaultValueOwner{}) {
					if a.err == nil {
						a.err = fmt.Errorf("OpenAPI opaque default has no authored checked owner")
					}
					return "unavailable"
				}
				if a.opaqueAnnotationOwners == nil {
					a.opaqueAnnotationOwners = make(map[expr.DefaultValueOwner]int)
				}
				owner, found := a.opaqueAnnotationOwners[binding]
				if !found {
					owner = len(a.opaqueAnnotationOwners) + 1
					a.opaqueAnnotationOwners[binding] = owner
				}
				return "opaque:" + strconv.Itoa(owner)
			}
		}
	}
	if a.err == nil {
		a.err = fmt.Errorf("OpenAPI component default has no checked annotation at its captured position")
	}
	return "unavailable"
}

// writeComponentAnnotations writes annotations without invoking value codecs.
func (a *Analyzer) writeComponentAnnotations(encoder *canonicalSchemaEncoder, attr *expr.AttributeExpr, node expr.ValuePlanNode) bool {
	keys := slices.Sorted(func(yield func(string) bool) {
		for key := range attr.Meta {
			if schemaAnnotationMeta(key) && !yield(key) {
				return
			}
		}
	})
	encoder.writeString(attr.Description)
	encoder.writeString(a.componentDefaultIdentity(attr, node))
	for _, key := range keys {
		encoder.writeString(key)
		encoder.writeString(strings.Join(attr.Meta[key], "\x00"))
	}
	return attr.Description != "" || attr.DefaultValue != nil || len(keys) > 0
}
