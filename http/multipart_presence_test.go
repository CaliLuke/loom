package http

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMultipartAbsentSourceVersusFailure(t *testing.T) {
	failure := fmt.Errorf("source unavailable: %w", io.EOF)
	for _, tc := range []struct {
		name     string
		body     []byte
		terminal error
		absent   bool
	}{
		{"empty stream", nil, nil, true},
		{"wrapped EOF", nil, failure, false},
		{"truncated boundary", []byte("--incomplete"), nil, false},
		{"unframed bytes", []byte("invalid"), nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := io.MultiReader(bytes.NewReader(tc.body), multipartTerminalReader{tc.terminal})
			request := httptest.NewRequest(http.MethodPost, "/", source)
			request.Header.Set("Content-Type", "multipart/form-data; boundary=test")
			form, err := ReadMultipartFormWithLimit(request, 1024)
			require.Error(t, err)
			require.Nil(t, form)
			require.Equal(t, tc.absent, errors.Is(err, ErrMultipartBodyAbsent))
			if tc.terminal != nil {
				require.ErrorIs(t, err, tc.terminal)
			}
		})
	}
}

type multipartTerminalReader struct{ err error }

func (r multipartTerminalReader) Read([]byte) (int, error) {
	if r.err != nil {
		return 0, r.err
	}
	return 0, io.EOF
}
