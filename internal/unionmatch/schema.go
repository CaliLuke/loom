package unionmatch

import "fmt"

// SelectSchema requires a nonempty candidate set and exactly one schema match
// without consulting decoders.
// A nested union uses this result as its outer schema-domain predicate.
func SelectSchema(name string, candidates []Candidate) (int, error) {
	if len(candidates) == 0 {
		return -1, fmt.Errorf("%s: untagged union requires at least one candidate", name)
	}
	count, selected := 0, -1
	for index, candidate := range candidates {
		if candidate.Schema == nil {
			return -1, fmt.Errorf("%s: untagged union candidate %d has no schema", name, index)
		}
		match, err := candidate.Schema()
		if err != nil {
			return -1, fmt.Errorf("%s schema candidate %d: %w", name, index, err)
		}
		if match {
			count++
			selected = index
		}
	}
	if count != 1 {
		return -1, &Mismatch{Message: fmt.Sprintf("decode %s: untagged union matched %d branches in schema", name, count)}
	}
	return selected, nil
}
