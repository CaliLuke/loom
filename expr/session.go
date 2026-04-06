package expr

import "github.com/CaliLuke/loom/eval"

type dslSessionState struct {
	root                 *RootExpr
	generatedResultTypes *ResultTypesRoot
	context              *eval.DSLContext
}

func currentDSLSessionState() dslSessionState {
	return dslSessionState{
		root:                 Root,
		generatedResultTypes: GeneratedResultTypes,
		context:              eval.Context,
	}
}

func installDSLSessionState(state dslSessionState) func() {
	previous := currentDSLSessionState()
	Root = state.root
	GeneratedResultTypes = state.generatedResultTypes
	eval.Context = state.context
	return func() {
		Root = previous.root
		GeneratedResultTypes = previous.generatedResultTypes
		eval.Context = previous.context
	}
}
