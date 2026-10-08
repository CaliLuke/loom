package http

import (
	"context"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSSEStreamReaderDiscardsIncompleteTail(t *testing.T) {
	for name, eol := range map[string]string{"lf": "\n", "cr": "\r", "crlf": "\r\n"} {
		t.Run(name, func(t *testing.T) {
			complete := "data: complete" + eol + eol
			for _, tc := range []struct {
				name  string
				input string
				count int
			}{
				{"empty", "", 0},
				{"unterminated", "data: partial", 0},
				{"terminated_line", "data: partial" + eol, 0},
				{"complete", complete, 1},
				{"complete_then_unterminated", complete + "data: partial", 1},
				{"complete_then_terminated_line", complete + "data: partial" + eol, 1},
				{"two_complete_then_partial", complete + complete + "data: partial", 2},
			} {
				t.Run(tc.name, func(t *testing.T) {
					chunks := [][]string{strings.Split(tc.input, "")}
					for split := 0; split <= len(tc.input); split++ {
						chunks = append(chunks, []string{tc.input[:split], tc.input[split:]})
					}
					for i, parts := range chunks {
						t.Run(strconv.Itoa(i), func(t *testing.T) {
							reader := NewSSEStreamReader(io.NopCloser(&chunkedReader{chunks: parts}))
							var events []SSEEvent
							for range tc.count {
								event, err := reader.ReadEvent(context.Background())
								require.NoError(t, err)
								events = append(events, event)
								require.Equal(t, "complete", event.Data)
							}
							for range 2 {
								event, err := reader.ReadEvent(context.Background())
								require.ErrorIs(t, err, io.EOF)
								require.Empty(t, event)
							}
							for _, event := range events {
								require.Equal(t, "complete", event.Data)
							}
							require.NoError(t, reader.Close())
						})
					}
				})
			}
		})
	}
}
