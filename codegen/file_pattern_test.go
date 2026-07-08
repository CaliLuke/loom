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

	require.Contains(t, code, `var loomPatternGenerated0 = regexp.MustCompile("^[a]+$")`)
	require.Contains(t, code, `loom.ValidatePatternCompiled("a", a, loomPatternGenerated0)`)
	require.Contains(t, code, `loom.ValidatePatternCompiled("b", b, loomPatternGenerated0)`)
	require.Equal(t, 1, strings.Count(code, "regexp.MustCompile"))
	require.NotContains(t, code, "loom.ValidatePattern(")
}

func TestFinalizeGoSourcePatternVarsUniquePerFile(t *testing.T) {
	// Two files in the same package must not hoist patterns into identically
	// named package-level variables (redeclaration compile error).
	source := []byte(`package generated

import loom "github.com/CaliLuke/loom/pkg"

func validate(a string) (err error) {
	err = loom.MergeErrors(err, loom.ValidatePattern("a", a, ".*\\S.*"))
	return err
}
`)

	cli, err := finalizeGoSource("cli.go", source)
	require.NoError(t, err)
	validation, err := finalizeGoSource("types_validation.go", source)
	require.NoError(t, err)

	require.Contains(t, string(cli), "var loomPatternCli0 = regexp.MustCompile")
	require.Contains(t, string(validation), "var loomPatternTypesValidation0 = regexp.MustCompile")
	require.NotContains(t, string(validation), "loomPatternCli0")
}

func TestPatternVarPrefix(t *testing.T) {
	cases := map[string]string{
		"cli.go":                          "loomPatternCli",
		"types_validation.go":             "loomPatternTypesValidation",
		"/abs/path/to/encode_decode.go":   "loomPatternEncodeDecode",
		"gen/http/svc/client/types.go":    "loomPatternTypes",
		"weird-name.v2.go":                "loomPatternWeirdNameV2",
	}
	for path, want := range cases {
		require.Equal(t, want, patternVarPrefix(path), "path %q", path)
	}
}
