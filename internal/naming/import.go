package naming

import "strings"

// ImportName returns the explicit import alias, or the conventional package
// name inferred from the last path component and its version suffix. It does
// not discover package declarations; imports whose declared name differs from
// that convention must provide an explicit name. Blank and dot aliases are
// returned unchanged and do not reserve a package identifier.
func ImportName(path, explicit string) string {
	if explicit != "" {
		return explicit
	}
	if _, name, ok := strings.CutLast(path, "/"); ok {
		path = name
	}
	if idx := strings.Index(path, ".v"); idx >= 0 {
		path = path[:idx]
	}
	if idx := strings.Index(path, "."); idx >= 0 {
		suffix := path[idx:]
		if len(suffix) > 1 && (suffix[1] >= '0' && suffix[1] <= '9' || suffix[1] == 'v') {
			path = path[:idx]
		}
	}
	return path
}
