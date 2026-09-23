package cli

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/dave/jennifer/jen"
)

// FuzzPrintDescription checks that the usage literal emitted for any design
// description prints that description byte for byte, with continuation lines
// indented by a tab.
func FuzzPrintDescription(f *testing.F) {
	for _, s := range []string{"", "Make requests", "Use `loom gen`", "a\nb", `"quoted"`, "tab\tand \\ slash", "\xff", "*/ %s %d"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, description string) {
		literal := fmt.Sprintf("%#v", jen.Lit(printDescription(description)))
		got, err := strconv.Unquote(literal)
		if err != nil {
			t.Errorf("printDescription(%q) renders %s, not a Go string literal: %v", description, literal, err)
			return
		}
		var want []byte
		for i := 0; i < len(description); i++ {
			want = append(want, description[i])
			if description[i] == '\n' {
				want = append(want, '\t')
			}
		}
		if got != string(want) {
			t.Errorf("printDescription(%q) prints %q, want %q", description, got, want)
		}
	})
}
