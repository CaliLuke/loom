package ir

import "github.com/CaliLuke/loom/expr"

type (
	componentAnnotationIdentity struct {
		declaration string
		baseline    string
		context     string
	}

	componentAnnotationPosition struct {
		context     string
		fingerprint string
	}
)

// componentAnnotationScope preserves the authority formerly held by the first
// component schema. Representation variants change constraints, not the legacy
// component's example source or sampling context. This is annotation reuse, not
// a target-value projection cache; direct occurrence overlays remain outside it.
func (a *Analyzer) componentAnnotationScope(declaration, baseline, context string) func() {
	identity := componentAnnotationIdentity{declaration, baseline, context}
	if a.componentAnnotations == nil {
		a.componentAnnotations = make(map[componentAnnotationIdentity]map[componentAnnotationPosition]any)
	}
	owner, ok := a.componentAnnotations[identity]
	if !ok {
		owner = make(map[componentAnnotationPosition]any)
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
