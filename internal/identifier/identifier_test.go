package identifier

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBase64(t *testing.T) {
	id, err := Base64(strings.NewReader("abcdef"), 6)
	require.NoError(t, err)
	require.Equal(t, "YWJjZGVm", id)
}

func TestHex(t *testing.T) {
	id, err := Hex(strings.NewReader("abcdefgh"), 8)
	require.NoError(t, err)
	require.Equal(t, "6162636465666768", id)
}

func TestEntropyFailure(t *testing.T) {
	sentinel := errors.New("entropy unavailable")
	tests := []struct {
		name   string
		reader io.Reader
	}{
		{name: "immediate failure", reader: failingReader{err: sentinel}},
		{
			name: "partial failure",
			reader: io.MultiReader(
				strings.NewReader("abc"),
				failingReader{err: sentinel},
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := Base64(tt.reader, 6)
			require.Empty(t, id)
			require.ErrorIs(t, err, ErrEntropy)
			require.ErrorIs(t, err, sentinel)
		})
	}
}

func TestMustPanicsOnEntropyFailure(t *testing.T) {
	sentinel := errors.New("entropy unavailable")
	require.PanicsWithError(t, "generate identifier: loom identifier entropy failure: entropy unavailable", func() {
		must(failingReader{err: sentinel}, 6, Base64)
	})
}

type failingReader struct {
	err error
}

func (r failingReader) Read([]byte) (int, error) {
	return 0, r.err
}
