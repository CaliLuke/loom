package unionmatch

import (
	"fmt"
	"slices"
)

// CheckEvaluated rejects instance members absent from the selected branch's
// evaluated annotations. Call only after Select succeeds on original branches.
func CheckEvaluated(name string, members, evaluated []string) error {
	allowed := make(map[string]bool, len(evaluated))
	for _, member := range evaluated {
		allowed[member] = true
	}
	var rejected []string
	for _, member := range members {
		if !allowed[member] {
			rejected = append(rejected, member)
		}
	}
	if len(rejected) == 0 {
		return nil
	}
	slices.Sort(rejected)
	return &Mismatch{Message: fmt.Sprintf("decode %s: untagged union has unevaluated member %q", name, rejected[0])}
}
