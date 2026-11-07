package loom

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"slices"

	"github.com/CaliLuke/loom/internal/unionmatch"
)

// Evaluated reports the present object members evaluated by a matching shape.
// Nonmatching and non-object values yield no members. Invalid descriptions fail.
func (s *JSONShape) Evaluated(wire jsontext.Value) ([]string, error) {
	result, err := s.evaluate(wire)
	if err != nil || !result.matched || wire.Kind() != '{' {
		return nil, err
	}
	slices.Sort(result.evaluated)
	return result.evaluated, nil
}

// matchClosure consumes annotations retained by unique matching. Re-running
// branch predicates here would multiply work at each recursive union level.
func (s *JSONShape) matchClosure(wire jsontext.Value, evaluated []string) (bool, error) {
	if !s.UnevaluatedClosed || wire.Kind() != '{' {
		return true, nil
	}
	var members map[string]jsontext.Value
	if err := json.Unmarshal(wire, &members); err != nil {
		return false, err
	}
	names := make([]string, 0, len(members))
	for name := range members {
		names = append(names, name)
	}
	return unionmatch.CheckEvaluated("JSON union", names, evaluated) == nil, nil
}
