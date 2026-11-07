// Package unionmatch owns exact-one selection across schema and carrier domains.
package unionmatch

import "fmt"

type (
	// Candidate supplies independent predicates over one private candidate.
	Candidate struct {
		// Schema checks membership without carrier-width restrictions.
		Schema func() (bool, error)
		// Decode converts and validates without publishing the candidate.
		Decode func() (bool, error)
	}
	// Mismatch is an ordinary union contract failure, including nested failures.
	// Configuration, evaluator and application-codec errors retain their causes.
	Mismatch struct {
		// Message describes the rejected union contract.
		Message string
	}
)

// Select requires a nonempty candidate set and a unique match in both domains
// on the same candidate. A singleton still runs both independent predicates.
// It visits every candidate unless a callback fails and never assigns a value.
func Select(name string, candidates []Candidate) (int, error) {
	if len(candidates) == 0 {
		return -1, fmt.Errorf("%s: untagged union requires at least one candidate", name)
	}
	schemaCount, decodedCount, schemaIndex, decodedIndex := 0, 0, -1, -1
	for index, branch := range candidates {
		if branch.Schema == nil || branch.Decode == nil {
			return -1, fmt.Errorf("%s: untagged union candidate %d is incomplete", name, index)
		}
		matches, err := branch.Schema()
		if err != nil {
			return -1, fmt.Errorf("%s schema candidate %d: %w", name, index, err)
		}
		if matches {
			schemaCount++
			schemaIndex = index
		}
		matches, err = branch.Decode()
		if err != nil {
			return -1, fmt.Errorf("%s decoder candidate %d: %w", name, index, err)
		}
		if matches {
			decodedCount++
			decodedIndex = index
		}
	}
	if schemaCount != 1 {
		return -1, &Mismatch{Message: fmt.Sprintf("decode %s: untagged union matched %d branches in schema", name, schemaCount)}
	}
	if decodedCount != 1 {
		return -1, &Mismatch{Message: fmt.Sprintf("decode %s: untagged union matched %d branches in decoder", name, decodedCount)}
	}
	if schemaIndex != decodedIndex {
		return -1, &Mismatch{Message: fmt.Sprintf("decode %s: untagged union schema and decoder identities differ", name)}
	}
	return decodedIndex, nil
}

// Error returns the contract failure without hiding its classification.
func (e *Mismatch) Error() string {
	return e.Message
}
