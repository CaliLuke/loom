package ir

import (
	"fmt"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

// analyzeReferenceSchema gives a declaration reached outside a captured body
// position its own checked annotations before component acquisition. The cached
// target is annotation-only: reference traversal does not invent example draws
// or reuse another occurrence's default/enum projection. Preparation failure
// returns nil and retains the checked diagnostic in Err; callers must not
// publish the partial analysis or substitute a schema.
func (a *Analyzer) analyzeReferenceSchema(declaration expr.DataType, context string, noRef ...bool) *Schema {
	attribute := &expr.AttributeExpr{Type: declaration}
	target := a.referenceAnnotationTargets[declaration]
	if target == nil {
		var err error
		target, err = representation.PrepareStandaloneValues(attribute, nil, false)
		if err != nil {
			if a.err == nil {
				a.err = fmt.Errorf("%s: prepare referenced schema annotations: %w", context, err)
			}
			return nil
		}
		if a.referenceAnnotationTargets == nil {
			a.referenceAnnotationTargets = make(map[expr.DataType]*transportir.ValueTarget)
		}
		a.referenceAnnotationTargets[declaration] = target
	}
	// A reference has no occurrence-level example publication after traversal.
	// Keep the prepared annotation binding inside the same analysis scope that
	// owned the reference before annotation-only preparation was introduced.
	previous := a.preparedExamples
	a.preparedExamples = target
	defer func() {
		a.preparedExamples = previous
	}()
	return a.analyzeSchemaPositions(attribute, context, expr.ValuePlanNode{}, representationRoot(attribute, target), noRef...)
}
