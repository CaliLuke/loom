package http

import "fmt"

// Annotations returns the decoded instance members and schema annotations for a
// matching branch. It uses the same immutable raw view and JSON projection as
// Schema, so sibling parameter keys cannot enter the logical instance. Scalars
// and sequences bypass object closure. Call only after schema selection succeeds.
func (b TextBranch) Annotations() (TextUnionAnnotations, error) {
	if b.View == nil {
		return TextUnionAnnotations{}, fmt.Errorf("text branch requires a view")
	}
	if err := validateParamBranchShape(b.Shape); err != nil {
		return TextUnionAnnotations{}, err
	}
	value, err := b.View()
	if err != nil {
		return TextUnionAnnotations{}, err
	}
	raw, matched, err := ParamValueJSON(value, b.Shape)
	if err != nil {
		return TextUnionAnnotations{}, err
	}
	if !matched {
		return TextUnionAnnotations{}, fmt.Errorf("selected text branch has no matching raw view")
	}
	if value.Kind != ParamPairs {
		return TextUnionAnnotations{}, nil
	}
	if b.Evaluate == nil {
		return TextUnionAnnotations{}, fmt.Errorf("text object branch requires an annotation evaluator")
	}
	evaluated, err := b.Evaluate(raw)
	if err != nil {
		return TextUnionAnnotations{}, err
	}
	members := make([]string, len(value.Pairs))
	for index, pair := range value.Pairs {
		members[index] = pair.Key
	}
	return TextUnionAnnotations{Members: members, Evaluated: evaluated}, nil
}
