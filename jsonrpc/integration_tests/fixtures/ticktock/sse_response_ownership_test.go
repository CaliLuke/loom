package ticktock_test

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/iotest"

	"example.com/ticktock/gen/clock"
	clockclient "example.com/ticktock/gen/jsonrpc/clock/client"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/require"
)

type (
	sseResponseBody struct {
		reader            io.Reader
		closeErr          error
		closes, bytesRead int
	}
	sseResponseDoer struct {
		response *http.Response
	}
)

func TestGeneratedSSEResponseOwnership(t *testing.T) {
	readErr := errors.New("read failed")
	for _, tc := range []struct {
		name, contentType    string
		status               int
		readFails, oversized bool
	}{
		{"unexpected status", "application/json", http.StatusBadGateway, false, false},
		{"partial read failure", "application/json", http.StatusBadGateway, true, false},
		{"bounded diagnostic", "application/json", http.StatusBadGateway, false, true},
		{"unexpected content type", "application/json", http.StatusOK, false, false},
		{"stream", "text/event-stream", http.StatusOK, false, false},
		{"stream without content type", "", http.StatusOK, false, false},
	} {
		for _, closeFails := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/closeFails=%t", tc.name, closeFails), func(t *testing.T) {
				var reader io.Reader = strings.NewReader("diagnostic")
				if tc.readFails {
					reader = io.MultiReader(reader, iotest.ErrReader(readErr))
				}
				if tc.oversized {
					reader = strings.NewReader(strings.Repeat("x", loomhttp.DefaultMaxErrorBodyBytes+100))
				}
				body := &sseResponseBody{reader: reader}
				if closeFails {
					body.closeErr = errors.New("close failed")
				}
				resp := &http.Response{StatusCode: tc.status, Header: make(http.Header), Body: body}
				resp.Header.Set("Content-Type", tc.contentType)
				client := clockclient.NewClient("http", "example.test", sseResponseDoer{resp}, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
				result, err := client.Tick()(t.Context(), &clock.TickPayload{})
				if tc.status != http.StatusOK || tc.contentType == "application/json" {
					require.Nil(t, result)
					require.Error(t, err)
					if tc.readFails {
						require.ErrorIs(t, err, readErr)
					}
					if closeFails {
						require.ErrorIs(t, err, body.closeErr)
					}
					if tc.oversized {
						require.Equal(t, loomhttp.DefaultMaxErrorBodyBytes, body.bytesRead)
					}
					require.Equal(t, 1, body.closes)
					return
				}
				require.NoError(t, err)
				require.Zero(t, body.closes)
				require.Zero(t, body.bytesRead)
				stream, ok := result.(io.Closer)
				require.True(t, ok)
				err = stream.Close()
				if closeFails {
					require.ErrorIs(t, err, body.closeErr)
				} else {
					require.NoError(t, err)
				}
				require.Equal(t, 1, body.closes)
			})
		}
	}
}

func (b *sseResponseBody) Read(p []byte) (int, error) {
	n, err := b.reader.Read(p)
	b.bytesRead += n
	return n, err
}

func (b *sseResponseBody) Close() error {
	b.closes++
	return b.closeErr
}

func (d sseResponseDoer) Do(*http.Request) (*http.Response, error) {
	return d.response, nil
}
