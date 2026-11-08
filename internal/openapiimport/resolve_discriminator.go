package openapiimport

import "sort"

type mappingOutcome struct {
	source                   targetKey
	entry, bundled, external string
	failure                  *Diagnostic
}

func (r *resolution) discriminatorEdges() []mappingOutcome {
	var outcomes []mappingOutcome
	for _, edge := range r.edges {
		if edge.class == "discriminatorMapping" {
			outcomes = append(outcomes, mappingOutcome{source: edge.at, entry: edge.entry, bundled: edge.bundled, external: edge.external, failure: edge.failure})
		}
	}
	sort.Slice(outcomes, func(i, j int) bool {
		x, y := outcomes[i].source, outcomes[j].source
		if r.registry.docs[x.doc].portable != r.registry.docs[y.doc].portable {
			return r.registry.docs[x.doc].portable < r.registry.docs[y.doc].portable
		}
		return x.pointer < y.pointer
	})
	return outcomes
}
