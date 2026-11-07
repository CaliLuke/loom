package service

import (
	"sort"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// viewHelperSuffixes gives every logical view one shared helper identity.
// Validators and both projection directions consume these suffixes. Natural
// suffixes are claimed before displaced views, independently of authored order.
func viewHelperSuffixes(typ expr.DataType) map[string]string {
	result, ok := typ.(*expr.ResultTypeExpr)
	if !ok {
		return nil
	}
	var names []string
	for _, view := range result.Views {
		if view.Name != expr.DefaultView {
			names = append(names, view.Name)
		}
	}
	sort.Strings(names)
	resolved := make(map[string]string, len(names))
	local := codegen.NewNameScope()
	local.Unique("") // The default view owns the unsuffixed helper.
	var deferred []string
	for _, name := range names {
		preferred := codegen.Goify(name, true)
		if local.PeekUnique(preferred) != preferred {
			deferred = append(deferred, name)
			continue
		}
		resolved[name] = local.Unique(preferred)
	}
	for _, name := range deferred {
		resolved[name] = local.Unique(codegen.Goify(name, true))
	}
	return resolved
}
