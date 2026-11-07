package http

import (
	"errors"
	"fmt"
	"sync"

	"github.com/CaliLuke/loom/internal/unionmatch"
)

// ComposeTextUnion makes a nested union into one outer candidate. Schema
// membership counts only inner schemas; decoding requires agreement in both
// domains before assign receives the selected index. assign must write only to
// a private destination. Ordinary nested union mismatches reject this candidate;
// evaluator and assignment errors propagate. Each predicate runs at most once
// for this input, so nesting does not repeatedly evaluate descendants. Construct
// a fresh candidate for each input; captured raw views must remain immutable.
// Closed unions check annotations after counting the original inner branches.
// The selected annotations are retained for enclosing unions without reevaluation.
func ComposeTextUnion(name string, branches []TextUnionBranch, closed bool, assign func(int) (bool, error)) TextUnionBranch {
	branches = append([]TextUnionBranch(nil), branches...)
	candidates := make([]unionmatch.Candidate, len(branches))
	for index, branch := range branches {
		candidates[index] = unionmatch.Candidate{
			Schema: memoTextPredicate(branch.Schema),
			Decode: memoTextPredicate(branch.Decode),
		}
	}
	schema := sync.OnceValues(func() (int, error) {
		return unionmatch.SelectSchema(name, candidates)
	})
	annotations := sync.OnceValues(func() (TextUnionAnnotations, error) {
		index, err := schema()
		if err != nil {
			return TextUnionAnnotations{}, err
		}
		return textUnionAnnotations(name, index, branches[index])
	})
	closure := sync.OnceValue(func() error {
		if !closed {
			return nil
		}
		value, err := annotations()
		if err != nil {
			return err
		}
		return unionmatch.CheckEvaluated(name, value.Members, value.Evaluated)
	})
	return TextUnionBranch{
		Schema: func() (bool, error) {
			_, err := schema()
			if err == nil {
				err = closure()
			}
			return textUnionCandidateResult(err)
		},
		Decode: sync.OnceValues(func() (bool, error) {
			index, err := unionmatch.Select(name, candidates)
			if err != nil {
				return textUnionCandidateResult(err)
			}
			if err := closure(); err != nil {
				return textUnionCandidateResult(err)
			}
			if assign == nil {
				return false, fmt.Errorf("%s: nested text union has no assignment", name)
			}
			return assign(index)
		}),
		Annotations: annotations,
	}
}

func memoTextPredicate(predicate func() (bool, error)) func() (bool, error) {
	if predicate == nil {
		return nil
	}
	return sync.OnceValues(predicate)
}

func textUnionCandidateResult(err error) (bool, error) {
	if err == nil {
		return true, nil
	}
	var mismatch *unionmatch.Mismatch
	if errors.As(err, &mismatch) {
		return false, nil
	}
	return false, err
}
