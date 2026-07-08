package expr

import (
	"fmt"

	"github.com/CaliLuke/loom/eval"
)

// PrepareValidateFinalize runs prepare, validate, and finalize on the given
// root using an isolated evaluation context and temporary Root binding.
func PrepareValidateFinalize(root *RootExpr) error {
	if root == nil {
		return fmt.Errorf("root cannot be nil")
	}

	restore := installDSLSessionState(dslSessionState{
		root:                 root,
		generatedResultTypes: GeneratedResultTypes,
		context:              eval.NewContext(),
		validated:            make(map[*AttributeExpr]bool),
	})
	defer restore()

	return eval.PrepareValidateFinalize(root)
}
