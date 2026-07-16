package jsonrpc

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewRequestID(t *testing.T) {
	id, err := newRequestID(zeroReader{})
	require.NoError(t, err)
	require.Equal(t, "00000000-0000-4000-8000-000000000000", id)
}

func TestNewRequestIDPropagatesEntropyFailure(t *testing.T) {
	sentinel := errors.New("entropy unavailable")
	id, err := newRequestID(failingIDReader{err: sentinel})
	require.Empty(t, id)
	require.ErrorIs(t, err, sentinel)
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

type failingIDReader struct {
	err error
}

func (r failingIDReader) Read([]byte) (int, error) {
	return 0, r.err
}
