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
		Validated            map[*AttributeExpr]bool
	}
)

var defaultRuntime = NewRuntime()

// Root is the root object built by the DSL.
var Root = defaultRuntime.Root

// GeneratedResultTypes records the generated result types and is a DSL root
// evaluated after Root.
var GeneratedResultTypes = defaultRuntime.GeneratedResultTypes

// validated keeps track of validated attributes to handle cyclical definitions.
var validated = defaultRuntime.Validated

// NewRuntime creates a fresh DSL runtime.
func NewRuntime() *Runtime {
	return &Runtime{
		Root:                 new(RootExpr),
		GeneratedResultTypes: new(ResultTypesRoot),
		Validated:            make(map[*AttributeExpr]bool),
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
	r.Validated = make(map[*AttributeExpr]bool)
	if r.Root == Root && r.GeneratedResultTypes == GeneratedResultTypes {
		validated = r.Validated
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
	Root = defaultRuntime.Root
	GeneratedResultTypes = defaultRuntime.GeneratedResultTypes
	validated = defaultRuntime.Validated
	return defaultRuntime.RegisterRoots(eval.Context)
}

func registerActiveRoots() error {
	runtime := &Runtime{
		Root:                 Root,
		GeneratedResultTypes: GeneratedResultTypes,
		Validated:            validated,
	}
	return runtime.RegisterRoots(eval.Context)
}
