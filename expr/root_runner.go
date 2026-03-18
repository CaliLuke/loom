package expr

import (
	"fmt"

	"goa.design/goa/v3/eval"
)

// PrepareValidateFinalize runs prepare, validate, and finalize on the given
// root using an isolated evaluation context and temporary Root binding.
func PrepareValidateFinalize(root *RootExpr) error {
	if root == nil {
		return fmt.Errorf("root cannot be nil")
	}

	originalRoot := Root
	originalContext := eval.Context

	Root = root
	eval.Context = eval.NewContext()
	defer func() {
		Root = originalRoot
		eval.Context = originalContext
	}()

	return eval.PrepareValidateFinalize(root)
}
