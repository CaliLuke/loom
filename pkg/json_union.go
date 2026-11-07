package loom

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"

	"github.com/CaliLuke/loom/internal/jsondecode"
	"github.com/CaliLuke/loom/internal/unionmatch"
)

type (
	// JSONUnionBranch supplies independent schema and typed decoder predicates.
	// Decode must keep its result private until MatchUntaggedJSON succeeds.
	JSONUnionBranch struct {
		// Schema checks the projected JSON contract, independently of Go widths.
		Schema *JSONShape
		// Decode returns true for a valid typed candidate, false for an ordinary
		// mismatch, or an error for a broken matcher or unsupported operation.
		Decode func(jsontext.Value) (bool, error)
	}
)

// MatchUntaggedJSON requires the same unique schema and typed decoder match.
// It visits every candidate, including after ambiguity, so failures cannot be
// hidden by declaration order. It never assigns a caller's destination.
func MatchUntaggedJSON(name string, data []byte, branches []JSONUnionBranch) (int, error) {
	var wire jsontext.Value
	if err := json.Unmarshal(data, &wire); err != nil {
		return -1, err
	}
	candidates := make([]unionmatch.Candidate, len(branches))
	for index, branch := range branches {
		if branch.Schema != nil {
			candidates[index].Schema = func() (bool, error) {
				return branch.Schema.Match(wire)
			}
		}
		if branch.Decode != nil {
			candidates[index].Decode = func() (bool, error) {
				return branch.Decode(wire)
			}
		}
	}
	return unionmatch.Select(name, candidates)
}

// MatchUntaggedJSONClosed selects against the original branches, then rejects
// object members not evaluated by the selected schema. Candidates remain private.
func MatchUntaggedJSONClosed(name string, data []byte, branches []JSONUnionBranch) (int, error) {
	index, err := MatchUntaggedJSON(name, data, branches)
	if err != nil {
		return -1, err
	}
	var wire jsontext.Value
	if err := json.Unmarshal(data, &wire); err != nil {
		return -1, err
	}
	if wire.Kind() != '{' {
		return index, nil
	}
	evaluated, err := branches[index].Schema.Evaluated(wire)
	if err != nil {
		return -1, err
	}
	var object map[string]jsontext.Value
	if err := json.Unmarshal(wire, &object); err != nil {
		return -1, err
	}
	members := make([]string, 0, len(object))
	for name := range object {
		members = append(members, name)
	}
	if err := unionmatch.CheckEvaluated(name, members, evaluated); err != nil {
		return -1, err
	}
	return index, nil
}

// DecodeJSONCandidate decodes an already structurally checked candidate.
// JSON type/conversion and nested union contract failures are mismatches;
// unexpected errors propagate.
// Generated callers must exclude opaque application codecs during analysis.
func DecodeJSONCandidate[T any](wire jsontext.Value, destination *T) (bool, error) {
	err := json.Unmarshal(wire, destination, JSONOptions())
	if err == nil {
		return true, nil
	}
	var nested *unionmatch.Mismatch
	if jsondecode.IsMismatch(err) || errors.As(err, &nested) {
		return false, nil
	}
	return false, err
}
