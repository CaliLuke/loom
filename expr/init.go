package expr

import (
	"github.com/CaliLuke/loom/v3/eval"
)

// Register DSL roots.
func init() {
	if err := eval.Register(Root); err != nil {
		panic(err) // bug
	}
	if err := eval.Register(GeneratedResultTypes); err != nil {
		panic(err) // bug
	}
}
