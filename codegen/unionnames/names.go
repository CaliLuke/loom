// Package unionnames allocates Go members for generated union carriers.
package unionnames

import (
	"slices"
	"strconv"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/naming"
)

// Member separates a branch's public accessor suffix from its storage field.
type Member struct {
	// Method is the suffix of the constructor, As and Set methods and kind constant.
	Method string
	// Field is the storage field, distinct from every generated method and sibling.
	Field string
}

// Resolve allocates members keyed by original branch name. Ordinary names stay
// unchanged. Sanitized collisions receive numeric suffixes in logical-name
// order, independent of branch order. Storage allocation reserves every method
// and every original field candidate before adding suffixes, so a renamed field
// cannot steal another branch's name. JSON-only and form carriers share names.
func Resolve(union *expr.Union) map[string]Member {
	names := make([]string, len(union.Values))
	used := make(map[string]bool, len(names))
	for i, branch := range union.Values {
		names[i] = branch.Name
		used[naming.Goify(branch.Name, true)] = true
	}
	slices.Sort(names)
	result := make(map[string]Member, len(names))
	assigned := make(map[string]bool, len(names))
	reserved := map[string]bool{
		"Kind": true, "Validate": true, "MarshalJSON": true,
		"UnmarshalJSON": true, "MarshalFormValues": true, "UnmarshalFormValues": true,
	}
	for _, name := range names {
		method := naming.Goify(name, true)
		if assigned[method] {
			method = fresh(method, used)
		}
		assigned[method] = true
		result[name] = Member{Method: method}
		reserved["As"+method], reserved["Set"+method] = true, true
	}
	for name := range reserved {
		used[name] = true
	}
	for _, name := range names {
		member := result[name]
		member.Field = member.Method
		if reserved[member.Field] {
			member.Field = fresh(member.Field, used)
		}
		result[name] = member
	}
	return result
}

func fresh(base string, used map[string]bool) string {
	for suffix := 2; ; suffix++ {
		name := base + strconv.Itoa(suffix)
		if !used[name] {
			used[name] = true
			return name
		}
	}
}
