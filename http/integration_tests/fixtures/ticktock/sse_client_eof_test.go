package ticktock_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	clockclient "example.com/http-ticktock/gen/http/clock/client"
	"github.com/stretchr/testify/require"
)

func TestGeneratedSSEClientDiscardsIncompleteTail(t *testing.T) {
	for name, eol := range map[string]string{"lf": "\n", "cr": "\r", "crlf": "\r\n"} {
		t.Run(name, func(t *testing.T) {
			data := strings.ReplaceAll("event: tick\ndata: complete", "\n", eol)
			for _, tc := range []struct{ name, tail string }{
				{"empty", ""}, {"unterminated", data}, {"terminated_line", data + eol},
			} {
				t.Run(tc.name, func(t *testing.T) {
					for _, prefix := range []string{"", data + eol + eol} {
						input := prefix + tc.tail
						stream := clockclient.NewTickStream(&http.Response{Body: io.NopCloser(strings.NewReader(input))}, nil)
						if prefix != "" {
							event, err := stream.Recv(context.Background())
							require.NoError(t, err)
							require.Equal(t, "complete", event.Data)
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
