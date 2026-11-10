package ir

import (
	"fmt"
	"reflect"
	"slices"

	"github.com/CaliLuke/loom/expr"
)

type canonicalComparison struct {
	name         string
	fingerprint  string
	registration schemaRef
	added        bool
	candidate    *Schema
}

// schemaAnalysisScope owns baseline completion and rollback across recursive
// analysis. Its returned closure must be deferred directly so it can preserve
// an in-flight panic while removing provisional canonical registrations.
func (a *Analyzer) schemaAnalysisScope() func() {
	var rollback func()
	if a.schemaDepth == 0 {
		rollback = a.acquisitionRollback()
	}
	a.schemaDepth++
	return func() {
		if a.schemaDepth != 1 {
			a.schemaDepth--
			return
		}
		defer func() {
			a.schemaDepth--
			a.acquisitionJournal = nil
			if failure := recover(); failure != nil {
				rollback()
				a.finishCanonicalComparisons(false)
				panic(failure)
			}
		}()
		if failure := recover(); failure != nil {
			panic(failure)
		}
		if a.err != nil {
			rollback()
			a.finishCanonicalComparisons(false)
			return
		}
		a.completeSchemaScope()
		a.finishCanonicalComparisons(true)
	}
}

// acquireCanonicalComparison reserves the candidate reference before traversing
// its body. Recursive expression unfoldings can have different preliminary
// fingerprints even when they render the same canonical component graph.
func (a *Analyzer) acquireCanonicalComparison(schema *Schema, attr *expr.AttributeExpr, named expr.UserType, name, fingerprint, metaName string, binding *projectedContributor) bool {
	existing, found := a.schemaFingerprints[name]
	if !found || existing == fingerprint {
		return false
	}
	comparison := &canonicalComparison{
		name:         name,
		fingerprint:  fingerprint,
		registration: schemaRef{ref: toRef(name), explicitName: metaName, canonicalClaim: metaName},
		candidate:    &Schema{},
	}
	comparison.added = !slices.Contains(a.schemasByFingerprint[fingerprint], comparison.registration)
	a.canonicalComparisons = append(a.canonicalComparisons, comparison)
	schema.Ref = toRef(name)
	a.registerComponentReference(fingerprint, schema.Ref, metaName, true)
	a.bindContributor(binding, schema.Ref, fingerprint, metaName, true)
	comparison.candidate = a.analyzeNamedUnderlyingSchema(componentAttribute(attr, named), named, exampleContext("component", name), true)
	return true
}

// finishCanonicalComparisons checks canonical claims after the outer traversal
// completes every baseline body. This is separate from the later representation
// quotient: canonical claims retain their existing exact rendered-body equality.
func (a *Analyzer) finishCanonicalComparisons(complete bool) {
	comparisons := a.canonicalComparisons
	a.canonicalComparisons = nil
	conflict := ""
	if complete && a.err == nil {
		for _, comparison := range comparisons {
			if !reflect.DeepEqual(a.schemas[comparison.name], comparison.candidate) {
				conflict = comparison.name
				break
			}
		}
	}
	if complete && a.err == nil && conflict == "" {
		return
	}
	for index := len(comparisons) - 1; index >= 0; index-- {
		comparison := comparisons[index]
		if !comparison.added {
			continue
		}
		refs := slices.DeleteFunc(slices.Clone(a.schemasByFingerprint[comparison.fingerprint]), func(ref schemaRef) bool {
			return ref == comparison.registration
		})
		if len(refs) == 0 {
			deleteAcquisitionEntry(a, a.schemasByFingerprint, comparison.fingerprint)
		} else {
			setAcquisitionEntry(a, a.schemasByFingerprint, comparison.fingerprint, refs)
		}
	}
	if conflict != "" {
		panic(fmt.Sprintf("openapi: explicit component name %q is claimed by multiple different schemas; use distinct Meta(%q, ...) values", conflict, "openapi:typename"))
	}
}
