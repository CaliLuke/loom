package cli

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCLIJSONErrorsWrapDecoderCause(t *testing.T) {
	for _, example := range []string{"", `'{"name":"Ada"}'`} {
		t.Run(example, func(t *testing.T) {
			flag := &FlagData{FullName: "body", Example: example}
			for _, code := range []string{
				fmt.Sprintf("%#v", buildSubcommandConversionError(flag, "[]string")),
				fmt.Sprintf("%#v", buildFieldLoadErrorReturn(flag, "body", "[]string", "nil")),
			} {
				require.Contains(t, code, `error: %w`)
				require.NotContains(t, code, `error: %s`)
			}
		})
	}
}
