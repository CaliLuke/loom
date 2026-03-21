package codegen

import (
	"testing"

	"github.com/dave/jennifer/jen"
	"github.com/stretchr/testify/require"
)

func TestJenniferSection(t *testing.T) {
	section, err := JenniferSection("answer", func(stmt *jen.Statement) {
		stmt.Comment(Comment("Answer returns the generated answer.")).Line()
		stmt.Func().Id("Answer").Params().Int().Block(
			jen.Return(jen.Lit(42)),
		)
	})
	require.NoError(t, err)

	code := SectionCode(t, section)
	require.Equal(t, `// Answer returns the generated answer.
func Answer() int {
	return 42
}
`, code)
}
