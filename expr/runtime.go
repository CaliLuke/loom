package expr

import (
	"fmt"

	"github.com/CaliLuke/loom/eval"
)

type (
	// Runtime owns the root expressions that back one Loom DSL session.
	Runtime struct {
		Root                 *RootExpr
		GeneratedResultTypes *ResultTypesRoot
	}
)

var defaultRuntime = NewRuntime()

// Root is the root object built by the DSL.
var Root = defaultRuntime.Root

// GeneratedResultTypes records the generated result types and is a DSL root
// evaluated after Root.
var GeneratedResultTypes = defaultRuntime.GeneratedResultTypes

// NewRuntime creates a fresh DSL runtime.
func NewRuntime() *Runtime {
	return &Runtime{
		Root:                 new(RootExpr),
		GeneratedResultTypes: new(ResultTypesRoot),
	}
}

// DefaultRuntime returns the process-default DSL runtime.
func DefaultRuntime() *Runtime {
	return defaultRuntime
}

// EvalRoots returns the runtime roots in evaluation order.
func (r *Runtime) EvalRoots() []eval.Root {
	return []eval.Root{r.Root, r.GeneratedResultTypes}
}

// RegisterRoots registers the runtime roots with the given context.
func (r *Runtime) RegisterRoots(ctx *eval.DSLContext) error {
	if ctx == nil {
		return fmt.Errorf("eval context cannot be nil")
	}
	for _, root := range r.EvalRoots() {
		if ctx.HasRoot(root.EvalName()) {
			continue
		}
		if err := ctx.Register(root); err != nil {
			return err
		}
	}
	return nil
}

// RegisterDefaultRoots registers the process-default runtime roots with the
// current evaluation context.
func RegisterDefaultRoots() error {
	return defaultRuntime.RegisterRoots(eval.Context)
}

func registerActiveRoots() error {
	runtime := &Runtime{
		Root:                 Root,
		GeneratedResultTypes: GeneratedResultTypes,
	}
	return runtime.RegisterRoots(eval.Context)
}
