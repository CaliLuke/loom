package cli

import "github.com/CaliLuke/loom/codegen"

// goifyTerms makes valid go identifiers out of the supplied terms.
func goifyTerms(terms ...string) string {
	res := codegen.Goify(terms[0], false)
	if len(terms) == 1 {
		return res
	}
	for _, t := range terms[1:] {
		res += codegen.Goify(t, true)
	}
	return res
}
