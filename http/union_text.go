package http

import (
	"fmt"

	"github.com/CaliLuke/loom/internal/unionmatch"
)

type (
	// TextUnionAnnotations describes one matching branch's logical instance.
	// Both sets use decoded member names at the same root location. Scalars and
	// sequences have no members. The codec owner excludes sibling bindings.
	TextUnionAnnotations struct {
		// Members contains the present object members in the branch's raw view.
		Members []string
		// Evaluated contains members evaluated by the matching schema.
		Evaluated []string
	}
	// TextUnionBranch supplies independent predicates over one branch's raw view.
	// Decoded values must remain private until selection succeeds. Captured raw
	// views and annotation slices must remain immutable for this input.
	TextUnionBranch struct {
		// Schema checks codec grammar and schema membership without Go widths.
		Schema func() (bool, error)
		// Decode converts and validates into a private candidate.
		Decode func() (bool, error)
		// Annotations returns the matching schema's evaluated-member annotations
		// together with its instance members. Closed unions require this callback.
		Annotations func() (TextUnionAnnotations, error)
	}
)

// MatchUntaggedText requires one schema and one decoder match at the same index.
// When closed is true, it then rejects members not evaluated by the selected
// schema. Presence and raw decoding belong to the callbacks and their codec owner.
func MatchUntaggedText(name string, branches []TextUnionBranch, closed bool) (int, error) {
	candidates := make([]unionmatch.Candidate, len(branches))
	for index, branch := range branches {
		candidates[index] = unionmatch.Candidate{Schema: branch.Schema, Decode: branch.Decode}
	}
	index, err := unionmatch.Select(name, candidates)
	if err != nil {
		return -1, err
	}
	if closed {
		annotations, err := textUnionAnnotations(name, index, branches[index])
		if err != nil {
			return -1, err
		}
		if err := unionmatch.CheckEvaluated(name, annotations.Members, annotations.Evaluated); err != nil {
			return -1, err
		}
	}
	return index, nil
}

func textUnionAnnotations(name string, index int, branch TextUnionBranch) (TextUnionAnnotations, error) {
	if branch.Annotations == nil {
		return TextUnionAnnotations{}, fmt.Errorf("%s: text union candidate %d has no annotations callback", name, index)
	}
	annotations, err := branch.Annotations()
	if err != nil {
		return TextUnionAnnotations{}, fmt.Errorf("%s annotations candidate %d: %w", name, index, err)
	}
	return annotations, nil
}
