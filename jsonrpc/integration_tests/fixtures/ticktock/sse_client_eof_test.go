package ticktock

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeneratedSSEClientDiscardsIncompleteTail(t *testing.T) {
	for name, eol := range map[string]string{"lf": "\n", "cr": "\r", "crlf": "\r\n"} {
		t.Run(name, func(t *testing.T) {
			data := strings.ReplaceAll(`data: {"jsonrpc":"2.0","method":"Tick","params":{"value":"complete"}}`, "\n", eol)
			for _, tc := range []struct{ name, tail string }{
				{"empty", ""}, {"unterminated", data}, {"terminated_line", data + eol},
			} {
				t.Run(tc.name, func(t *testing.T) {
					for _, prefix := range []string{"", data + eol + eol} {
						input := prefix + tc.tail
						stream := newJSONRPCSSETestStream(t, io.NopCloser(strings.NewReader(input)))
						if prefix != "" {
							event, err := stream.Recv(context.Background())
							require.NoError(t, err)
							require.NotNil(t, event.Value)
							require.Equal(t, "complete", *event.Value)
						}
						for range 2 {
							event, err := stream.Recv(context.Background())
							require.ErrorIs(t, err, io.EOF)
							require.Nil(t, event)
						}
						require.NoError(t, stream.Close())
					}
				})
			}
		})
	}
}
