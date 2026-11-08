package ir

import (
	"maps"
	"slices"
)

// acquisitionRollback snapshots the shared registration indexes at the outer
// scope boundary. Published entries are immutable; mutable registration slices
// and annotation owner maps are copied. Failure retains the original error and
// removes only this scope's provisional acquisition and projection state.
func (a *Analyzer) acquisitionRollback() func() {
	components := maps.Clone(a.componentAcquisitions)
	contributors := maps.Clone(a.contributors)
	schemas := maps.Clone(a.schemas)
	fingerprints := maps.Clone(a.schemaFingerprints)
	names := maps.Clone(a.schemaNames)
	types := maps.Clone(a.schemaTypeIDs)
	refs := maps.Clone(a.schemasByFingerprint)
	for key, values := range refs {
		refs[key] = slices.Clone(values)
	}
	constructions := maps.Clone(a.constructions)
	plans := maps.Clone(a.projectionPlans)
	projections := maps.Clone(a.projectionComponents)
	mappings := maps.Clone(a.projectionMappings)
	children := maps.Clone(a.projectedChildren)
	for key, values := range children {
		children[key] = slices.Clone(values)
	}
	published := maps.Clone(a.publishedChildren)
	discriminatorChildren := maps.Clone(a.discriminatorChildren)
	for key, values := range discriminatorChildren {
		discriminatorChildren[key] = slices.Clone(values)
	}
	opaqueAnnotations := maps.Clone(a.opaqueAnnotationOwners)
	annotations := maps.Clone(a.componentAnnotations)
	for key, owner := range annotations {
		annotations[key] = maps.Clone(owner)
	}
	constructionContexts := maps.Clone(a.constructionContexts)
	publicationContexts := maps.Clone(a.publicationContexts)
	neighbors := maps.Clone(a.declarationNeighborCache)
	dependencies := maps.Clone(a.publicationDependencyCache)
	return func() {
		a.componentAcquisitions, a.contributors = components, contributors
		a.schemas, a.schemaFingerprints, a.schemaNames, a.schemaTypeIDs = schemas, fingerprints, names, types
		a.schemasByFingerprint, a.constructions = refs, constructions
		a.projectionPlans, a.projectionComponents, a.projectionMappings = plans, projections, mappings
		a.projectedChildren, a.publishedChildren = children, published
		a.discriminatorChildren, a.componentAnnotations = discriminatorChildren, annotations
		a.opaqueAnnotationOwners = opaqueAnnotations
		a.constructionContexts, a.publicationContexts = constructionContexts, publicationContexts
		a.declarationNeighborCache, a.publicationDependencyCache = neighbors, dependencies
	}
}
