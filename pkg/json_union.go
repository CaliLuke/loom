package loom

import (
	"encoding/base64"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"strconv"
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
	if len(branches) < 2 {
		return -1, fmt.Errorf("%s: untagged union requires at least two candidates", name)
	}
	schemaCount, decodedCount, schemaIndex, decodedIndex := 0, 0, -1, -1
	for index, branch := range branches {
		if branch.Schema == nil || branch.Decode == nil {
			return -1, fmt.Errorf("%s: untagged union candidate %d is incomplete", name, index)
		}
		matches, err := branch.Schema.Match(wire)
		if err != nil {
			return -1, fmt.Errorf("%s schema candidate %d: %w", name, index, err)
		}
		if matches {
			schemaCount++
			schemaIndex = index
		}
		matches, err = branch.Decode(wire)
		if err != nil {
			return -1, fmt.Errorf("%s decoder candidate %d: %w", name, index, err)
		}
		if matches {
			decodedCount++
			decodedIndex = index
		}
	}
	if schemaCount != 1 {
		return -1, fmt.Errorf("decode %s: untagged union matched %d branches in schema", name, schemaCount)
	}
	if decodedCount != 1 {
		return -1, fmt.Errorf("decode %s: untagged union matched %d branches in decoder", name, decodedCount)
	}
	if schemaIndex != decodedIndex {
		return -1, fmt.Errorf("decode %s: untagged union schema and decoder identities differ", name)
	}
	return decodedIndex, nil
}

// DecodeJSONCandidate decodes an already structurally checked candidate.
// JSON type/conversion failures are mismatches; unexpected errors propagate.
// Generated callers must exclude opaque application codecs during analysis.
func DecodeJSONCandidate[T any](wire jsontext.Value, destination *T) (bool, error) {
	err := json.Unmarshal(wire, destination, JSONOptions())
	if err == nil {
		return true, nil
	}
	var semantic *json.SemanticError
	if errors.As(err, &semantic) {
		var corrupt base64.CorruptInputError
		if semantic.Err == nil || errors.Is(semantic.Err, strconv.ErrRange) || errors.Is(semantic.Err, strconv.ErrSyntax) || errors.As(semantic.Err, &corrupt) {
			return false, nil
		}
	}
	return false, err
}
