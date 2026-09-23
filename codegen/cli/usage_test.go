package cli

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"

	"github.com/dave/jennifer/jen"
	"github.com/stretchr/testify/require"
)

func TestPrintDescriptionLiteralPrintsDescription(t *testing.T) {
	cases := map[string]struct {
		description string
		printed     string
	}{
		"plain":     {description: "Make requests", printed: "Make requests"},
		"backticks": {description: "Use `loom gen` first", printed: "Use `loom gen` first"},
		"newlines":  {description: "First line\nsecond line", printed: "First line\n\tsecond line"},
		"quotes":    {description: `Say "hi" \ bye`, printed: `Say "hi" \ bye`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			literal := fmt.Sprintf("%#v", jen.Lit(printDescription(tc.description)))
			got, err := strconv.Unquote(literal)
			require.NoError(t, err, literal)
			require.Equal(t, tc.printed, got)
		})
	}
}

func TestUsageExamplesEndEachExampleWithNewline(t *testing.T) {
	cases := map[string]struct {
		examples []string
		literals []string
	}{
		"one example":  {examples: []string{"svc add --a 1"}, literals: []string{" svc add --a 1\n"}},
		"two examples": {examples: []string{"svc add", "svc sub"}, literals: []string{" svc add\n", " svc sub\n"}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			data := make([]*CommandData, len(tc.examples))
			for i, example := range tc.examples {
				data[i] = &CommandData{Example: example}
			}
			var buf bytes.Buffer
			require.NoError(t, UsageExamples(data).Write(&buf))
			source := buf.String()
			for _, want := range tc.literals {
				require.Contains(t, source, strconv.Quote(want), source)
			}
		})
	}
}
