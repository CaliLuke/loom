package ir

import (
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

type preparedComponentBinding struct {
	target   *transportir.ValueTarget
	position expr.ValuePlanNode
}

type componentAcquisition struct {
	name        string
	fingerprint string
	semantic    string
	hint        string
	claim       string
	attribute   *expr.AttributeExpr
	declaration expr.UserType
	complete    bool
}

// registerComponentReference records the selected claim association separately
// from the public hint. Preferred-only suffixed registrations cannot satisfy a
// canonical request, while preferred requests retain their existing hint rule.
func (a *Analyzer) registerComponentReference(fingerprint, ref, hint string, canonical bool) {
	claim := ""
	binding := preparedComponentBinding{}
	if canonical {
		claim = hint
		if hint != "" && ref != toRef(hint) {
			binding = preparedComponentBinding{a.preparedExamples, a.examplePlan}
			if binding.target == nil || !binding.position.Valid() {
				panic("OpenAPI canonical suffix requires a prepared example binding")
			}
		}
	}
	for index, prior := range a.schemasByFingerprint[fingerprint] {
		if prior.ref == ref && prior.explicitName == hint {
			if claim != "" {
				a.schemasByFingerprint[fingerprint][index].canonicalClaim = claim
				a.schemasByFingerprint[fingerprint][index].binding = binding
			}
			return
		}
	}
	a.schemasByFingerprint[fingerprint] = append(a.schemasByFingerprint[fingerprint], schemaRef{ref: ref, explicitName: hint, canonicalClaim: claim, binding: binding})
}

// reserveComponent records visiting ownership without inserting an incomplete
// body into schemas. The fingerprint bucket and selected reference still own
// reuse; declaration identity is provenance rather than a new partition.
func (a *Analyzer) reserveComponent(name, fingerprint, semantic, hint string, canonical bool, attribute *expr.AttributeExpr, declaration expr.UserType) *componentAcquisition {
	entry := &componentAcquisition{name: name, fingerprint: fingerprint, semantic: semantic, hint: hint, attribute: attribute, declaration: declaration}
	if canonical {
		entry.claim = hint
	}
	a.componentAcquisitions[name] = entry
	return entry
}

// contributorBindingScope carries a one-shot root request through the projected
// result route. Its first named selection consumes the token before traversal.
func (a *Analyzer) contributorBindingScope(contributor *projectedContributor) func() {
	previous, depth := a.contributorRoot, a.contributorRootDepth
	a.contributorRoot, a.contributorRootDepth = contributor, a.schemaDepth+1
	return func() {
		a.contributorRoot, a.contributorRootDepth = previous, depth
	}
}

func (a *Analyzer) takeContributorBinding() *projectedContributor {
	if a.contributorRoot == nil || a.schemaDepth != a.contributorRootDepth {
		return nil
	}
	binding := a.contributorRoot
	a.contributorRoot = nil
	return binding
}

func (a *Analyzer) bindContributor(contributor *projectedContributor, ref, fingerprint, hint string, canonical bool) {
	if contributor == nil {
		return
	}
	name, local := schemaComponentName(ref)
	if !local {
		panic("OpenAPI contributor root did not acquire a named component")
	}
	if contributor.component != nil && contributor.component != a.componentAcquisitions[name] {
		panic("OpenAPI contributor root acquired more than one component")
	}
	contributor.fingerprint, contributor.hint = fingerprint, hint
	if canonical {
		contributor.claim = hint
	}
	contributor.component = a.componentAcquisitions[name]
	if contributor.component == nil {
		panic("OpenAPI contributor root has no component acquisition")
	}
}
