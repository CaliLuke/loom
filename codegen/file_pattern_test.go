package codegen

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFinalizeGoSourceCompilesPatternValidations(t *testing.T) {
	source := []byte(`package generated

import loom "github.com/CaliLuke/loom/pkg"

func validate(a, b string) (err error) {
	err = loom.MergeErrors(err, loom.ValidatePattern("a", a, "^[a]+$"))
	err = loom.MergeErrors(err, loom.ValidatePattern("b", b, "^[a]+$"))
	return err
}
`)

	finalized, err := finalizeGoSource("generated.go", source)
	require.NoError(t, err)
	code := string(finalized)

	require.Contains(t, code, `var loomPattern0 = regexp.MustCompile("^[a]+$")`)
	require.Contains(t, code, `loom.ValidatePatternCompiled("a", a, loomPattern0)`)
	require.Contains(t, code, `loom.ValidatePatternCompiled("b", b, loomPattern0)`)
	require.Equal(t, 1, strings.Count(code, "regexp.MustCompile"))
	require.NotContains(t, code, "loom.ValidatePattern(")
}
