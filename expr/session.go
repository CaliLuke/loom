package expr

import "github.com/CaliLuke/loom/eval"

type dslSessionState struct {
	root                 *RootExpr
	generatedResultTypes *ResultTypesRoot
	context              *eval.DSLContext
	validated            map[*AttributeExpr]bool
}

func currentDSLSessionState() dslSessionState {
	return dslSessionState{
		root:                 Root,
		generatedResultTypes: GeneratedResultTypes,
		context:              eval.Context,
		validated:            validated,
	}
}

func installDSLSessionState(state dslSessionState) func() {
	previous := currentDSLSessionState()
	if state.validated == nil {
		state.validated = make(map[*AttributeExpr]bool)
	}
	Root = state.root
	GeneratedResultTypes = state.generatedResultTypes
	eval.Context = state.context
	validated = state.validated
	return func() {
		Root = previous.root
		GeneratedResultTypes = previous.generatedResultTypes
		eval.Context = previous.context
		validated = previous.validated
	}
}
