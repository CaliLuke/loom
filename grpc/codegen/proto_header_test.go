package codegen

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
)

func TestProtoHeaderTitleStaysInComments(t *testing.T) {
	cases := map[string]struct {
		title string
		want  string
	}{
		"plain":   {title: "calc protocol buffer definition", want: "// calc protocol buffer definition\n"},
		"newline": {title: "calc\nservice Injected {}\n protocol buffer definition", want: "// calc\n// service Injected {}\n// protocol buffer definition\n"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			section := codegen.NewTextTemplateSection("proto-header", grpcTemplates.Read(grpcProtoHeaderT), nil, map[string]any{"Title": tc.title})
			var buf bytes.Buffer
			require.NoError(t, section.Write(&buf))
			require.Contains(t, buf.String(), tc.want)
			for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
				require.True(t, strings.HasPrefix(line, "//"), "line %q escapes the header comment:\n%s", line, buf.String())
			}
		})
	}
}
