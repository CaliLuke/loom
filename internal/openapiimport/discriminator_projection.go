package openapiimport

import (
	"slices"
	"sort"
	"strings"
)

type parentHintPlan struct {
	identity    targetKey
	obligations []NamedSchema
}

// prepareDiscriminatorChildren records base-valid strong closures and their
// typed obligations. The shared document plans are immutable; weak reachability
// belongs to each selection or partial projection's component pruner.
func (a *analyzer) prepareDiscriminatorChildren(document *Document) {
	var hints []*Discriminator
	for _, parent := range document.Components.Schemas {
		hint := parent.Schema.Discriminator
		if hint != nil && hint.validated && hint.Parent {
			hints = append(hints, hint)
		}
	}
	document.parentHintPlans = qualifyParentHintPlans(document, hints, a.diagnostics)
}

// projectWeakChildren computes the greatest eligible set in this operation
// projection and returns the selected-root walk under its weak edges. Hints in
// the strong root graph are emitted without any weak child, so a child answers
// only for obligations it introduces; those can use witnesses in reachable
// siblings or selected operations, including inline schemas. One walk serves
// every reached child. An unreached child cannot be emitted or supply evidence,
// and an unrelated provisional sibling failure cannot poison the child.
func projectWeakChildren(document *Document) *componentPruner {
	children := make(map[string]parentHintPlan)
	for _, named := range document.Components.Schemas {
		if plan, exists := document.parentHintPlans[named.Name]; exists {
			children[named.Name] = plan
		}
	}
	if len(children) == 0 {
		// Without weak children the strong walk is the projection walk.
		return walkSelectedRoots(document, nil)
	}
	forced := walkSelectedRoots(document, nil).closure.schemas
	for {
		pruner := walkSelectedRoots(document, weakChildEdges(document, children))
		retained := retainedIdentities(document.bundle, pruner.schemaOccurrences)
		stable := true
		for name, plan := range children {
			if _, reached := pruner.closure.schemas[name]; !reached {
				// The walk never traversed this edge, so removal leaves it unchanged.
				delete(children, name)
				continue
			}
			if len(parentHintsWithoutWitness(introducedObligations(plan.obligations, forced), retained)) > 0 {
				delete(children, name)
				stable = false
			}
		}
		if stable {
			return pruner
		}
	}
}

// introducedObligations excludes hints that the strong root graph already
// emits. It never mutates the shared base plan.
func introducedObligations(obligations []NamedSchema, forced map[string]struct{}) []NamedSchema {
	introduced := make([]NamedSchema, 0, len(obligations))
	for _, named := range obligations {
		if _, rootOwned := forced[named.Name]; !rootOwned {
			introduced = append(introduced, named)
		}
	}
	return introduced
}

func weakChildEdges(document *Document, children map[string]parentHintPlan) map[*Discriminator][]string {
	edges := make(map[*Discriminator][]string)
	for _, parent := range document.Components.Schemas {
		hint := parent.Schema.Discriminator
		if hint == nil || !hint.validated || !hint.Parent {
			continue
		}
		for name, child := range children {
			if slices.Contains(hint.children, child.identity) {
				edges[hint] = append(edges[hint], name)
			}
		}
		sort.Strings(edges[hint])
	}
	return edges
}

func qualifyParentHintPlans(document *Document, hints []*Discriminator, diagnostics Diagnostics) map[string]parentHintPlan {
	plans := make(map[string]parentHintPlan)
	for _, named := range document.Components.Schemas {
		identity := discriminatorIdentity(document.bundle, named.Schema)
		if !slices.ContainsFunc(hints, func(hint *Discriminator) bool {
			return slices.Contains(hint.children, identity)
		}) {
			continue
		}
		obligations, valid := schemaDeclarationQualification(document, named.Name, diagnostics)
		if valid {
			plans[named.Name] = parentHintPlan{identity: identity, obligations: obligations}
		}
	}
	return plans
}

// schemaDeclarationQualification walks the declaration's strong closure with
// the same retention owner as selection, raw source included, so a blocker that
// normalization dropped still disqualifies the optional child.
func schemaDeclarationQualification(document *Document, name string, diagnostics Diagnostics) ([]NamedSchema, bool) {
	candidate := documentForOperations(document, nil)
	pruner := newDocumentPruner(document)
	pruner.visitSchema(&Schema{Ref: componentDeclaration(name)})
	pruner.retain(candidate)
	candidate.retainedClosure = &pruner.closure
	// Qualification asks whether the retained schema is representable, not
	// whether the caller permits losing annotations. Preserve those diagnostics
	// for the ordinary strict/allow-lossy boundary without dropping the schema.
	fatal, _ := filterSelectionDiagnostics(diagnostics, pruner.closure).Classify(true)
	for _, diagnostic := range fatal {
		if strings.HasPrefix(diagnostic.Path, "#/components/schemas/") || strings.HasPrefix(diagnostic.Path, "#/components/examples/") {
			return nil, false
		}
	}
	plan := planDocumentGraph(candidate)
	return plan.projectionHints, len(plan.diagnostics) == 0
}

func parentHintProjectionDiagnostics(orphaned []NamedSchema) Diagnostics {
	diagnostics := make(Diagnostics, 0, len(orphaned))
	for _, named := range orphaned {
		diagnostics = append(diagnostics, Diagnostic{
			Code: "schema-discriminator-projection", Path: named.Schema.sourcePath + "/discriminator",
			Message: "the original ParentHint has children, but this selection cannot faithfully emit a child schema",
		})
	}
	return diagnostics
}

func unprojectableParentHints(document *Document) []NamedSchema {
	return parentHintsWithoutWitness(document.Components.Schemas, retainedIdentities(document.bundle, documentSchemaOccurrences(document)))
}

// retainedIdentities inventories the canonical source identities of retained
// occurrences once, so every hint checked against one walk shares it.
func retainedIdentities(bundle *bundledDescription, occurrences map[*Schema]struct{}) map[targetKey]bool {
	retained := make(map[targetKey]bool)
	for schema := range occurrences {
		if schema.sourcePath != "" {
			retained[discriminatorIdentity(bundle, schema)] = true
		}
	}
	return retained
}

func parentHintsWithoutWitness(parents []NamedSchema, retained map[targetKey]bool) []NamedSchema {
	var orphaned []NamedSchema
	for _, named := range parents {
		hint := named.Schema.Discriminator
		if hint == nil || !hint.validated || !hint.Parent {
			continue
		}
		if !slices.ContainsFunc(hint.children, func(identity targetKey) bool {
			return retained[identity]
		}) {
			orphaned = append(orphaned, named)
		}
	}
	return orphaned
}

// omitUnprojectableParentHints is an explicit partial-import projection. Every
// valid ParentHint is a canonical declaration: its allOf child references made
// the resolver retain/promote that target. Occurrences resolve those names, so
// copying the affected declarations isolates the hint without cloning values.
func omitUnprojectableParentHints(document *Document) (*Document, Diagnostics) {
	orphaned := unprojectableParentHints(document)
	if len(orphaned) == 0 {
		return document, nil
	}
	result := *document
	result.Components.Schemas = slices.Clone(document.Components.Schemas)
	omitted := make(map[string]bool, len(orphaned))
	var diagnostics Diagnostics
	for _, named := range orphaned {
		omitted[named.Name] = true
		diagnostics = append(diagnostics, Diagnostic{
			Code: "schema-discriminator-omission", Path: named.Schema.sourcePath + "/discriminator",
			Message: "partial import omitted the ParentHint because no child schema can be faithfully emitted; the parent's validation schema is unchanged",
		})
	}
	for index := range result.Components.Schemas {
		named := &result.Components.Schemas[index]
		if omitted[named.Name] {
			projected := *named.Schema
			projected.Discriminator = nil
			named.Schema = &projected
		}
	}
	return &result, diagnostics
}

// reconcileParentHintOmissions reports the emitted projection. Another retained
// operation may supply the inline child absent from one operation's admission
// candidate, in which case the combined document keeps its original hint.
func reconcileParentHintOmissions(operations []SkippedOperation, final Diagnostics) []SkippedOperation {
	omittedPaths := make(map[string]bool, len(final))
	for _, diagnostic := range final {
		omittedPaths[diagnostic.Path] = true
	}
	result := make([]SkippedOperation, 0, len(operations))
	for _, operation := range operations {
		operation.Diagnostics = slices.DeleteFunc(operation.Diagnostics, func(diagnostic Diagnostic) bool {
			return diagnostic.Code == "schema-discriminator-omission" && !omittedPaths[diagnostic.Path]
		})
		if len(operation.Diagnostics) > 0 {
			result = append(result, operation)
		}
	}
	return result
}
