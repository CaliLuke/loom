package rmapstate

import "regexp"

var redisKeyName = regexp.MustCompile(`^[^ \0\*\?\[\]]{1,512}$`)

// ValidName reports whether a replicated-map name is accepted before allocation
// or protocol admission. Constructors that coordinate maps use the same rule.
func ValidName(name string) bool {
	return redisKeyName.MatchString(name)
}
