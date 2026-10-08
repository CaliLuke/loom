package http

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type responseBodyProbe struct {
	io.Reader
	closes   int
	closeErr error
}

func (b *responseBodyProbe) Close() error {
	b.closes++
	return b.closeErr
}

func TestDecodeResponseOwnership(t *testing.T) {
	decoding, closing := errors.New("decode"), errors.New("close")
	for _, restore := range []bool{false, true} {
		for _, stream := range []bool{false, true} {
			for _, failed := range []bool{false, true} {
				b := &responseBodyProbe{Reader: strings.NewReader("payload"), closeErr: closing}
				resp := &http.Response{Body: b}
				result, err := DecodeResponse(resp, restore, stream, func(resp *http.Response) (any, error) {
					if failed {
						return nil, decoding
					}
					return "ok", nil
				})
				if stream && !failed {
					require.NoError(t, err)
					require.Equal(t, "ok", result)
					require.Zero(t, b.closes)
					require.Same(t, b, resp.Body)
				} else {
					require.Nil(t, result)
					require.Equal(t, 1, b.closes)
					require.ErrorIs(t, err, closing)
					if failed {
						require.ErrorIs(t, err, decoding)
					}
					if restore && !stream {
						data, err := io.ReadAll(resp.Body)
						require.NoError(t, err)
						require.Equal(t, "payload", string(data))
					}
				}
			}
		}
	}
}
