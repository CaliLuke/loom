package expr

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/eval"
)

// RunDSL returns the DSL root resulting from running the given DSL.
// Used only in tests.
func RunDSL(t *testing.T, dsl func()) *RootExpr {
	t.Helper()
	root := SetupTestDSL(t)

	// run DSL (first pass)
	require.True(t, eval.Execute(dsl, nil), eval.Context.Error())

	// run DSL (second pass)
	require.NoError(t, eval.RunDSL())

	// return generated root
	return root
}

// RunInvalidDSL returns the error resulting from running the given DSL.
// It is used only in tests.
func RunInvalidDSL(t *testing.T, dsl func()) error {
	t.Helper()
	SetupTestDSL(t)

	// run DSL (first pass)
	if !eval.Execute(dsl, nil) {
		return eval.Context.Errors
	}

	// run DSL (second pass)
	if err := eval.RunDSL(); err != nil {
		return err
	}

	// expected an error - didn't get one
	t.Fatal("expected a DSL evaluation error - got none")

	return nil
}

// CreateTempFile creates a temporary file and writes the given content.
// It is used only for testing.
func CreateTempFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString(content)
	if err != nil {
		require.NoError(t, os.Remove(f.Name()))
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return f.Name()
}

// SetupTestDSL resets the global expression state for testing and initializes
// a default API. This function should be called before running any DSL that
// modifies the global Root or GeneratedResultTypes variables.
//
// Usage in tests:
//
//	func TestMyDSL(t *testing.T) {
//	    // Option 1: Use expr.RunDSL which calls SetupTestDSL automatically
//	    root := expr.RunDSL(t, func() {
//	        Service("my-service", func() { /* ... */ })
//	    })
//
//	    // Option 2: Call SetupTestDSL manually when running DSL directly
//	    expr.SetupTestDSL(t)
//	    eval.Execute(myDSL, nil)
//	    eval.RunDSL()
//	}
//
// Note: RunDSL and RunInvalidDSL automatically call SetupTestDSL, so you
// only need to call it manually when executing DSL code directly.
func SetupTestDSL(t *testing.T) *RootExpr {
	t.Helper()
	root := new(RootExpr)
	installDSLSessionState(dslSessionState{
		root:                 root,
		generatedResultTypes: new(ResultTypesRoot),
		context:              eval.NewContext(),
		validated:            make(map[*AttributeExpr]bool),
	})
	root.API = NewAPIExpr("test api", func() {})
	root.API.Servers = []*ServerExpr{root.API.DefaultServer()}
	require.NoError(t, registerActiveRoots())
	return root
}
