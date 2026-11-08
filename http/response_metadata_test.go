package http

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResponseMetadataAdmission(t *testing.T) {
	t.Run("replacement and snapshot", func(t *testing.T) {
		metadata := NewResponseMetadata(nil)
		header := http.Header{"X-Trace": {"first"}}
		require.NoError(t, metadata.Set(func() (SSEHandshake, error) {
			return SSEHandshake{Status: 201, Header: header}, nil
		}))
		header.Set("X-Trace", "mutated")
		invalid := errors.New("invalid metadata")
		require.ErrorIs(t, metadata.Set(func() (SSEHandshake, error) {
			return SSEHandshake{Status: 202}, invalid
		}), invalid)
		value, err := metadata.Capture(context.Background())
		require.NoError(t, err)
		require.Equal(t, 201, value.Status)
		require.Equal(t, "first", value.Header.Get("X-Trace"))
		require.ErrorIs(t, metadata.Set(func() (SSEHandshake, error) {
			t.Error("sealed setter executed")
			return SSEHandshake{}, nil
		}), ErrSSEResponseCommitted)
	})
	t.Run("failed capture seals", func(t *testing.T) {
		missing := errors.New("missing metadata")
		metadata := NewResponseMetadata(func(context.Context) (SSEHandshake, error) {
			return SSEHandshake{}, missing
		})
		_, err := metadata.Capture(context.Background())
		require.ErrorIs(t, err, missing)
		require.ErrorIs(t, metadata.Set(func() (SSEHandshake, error) {
			return SSEHandshake{Status: 201}, nil
		}), ErrSSEResponseCommitted)
	})
	for _, terminal := range []string{"handler error", "SSE failure", "SSE suppression", "SSE close"} {
		t.Run(terminal, func(t *testing.T) {
			metadata := NewResponseMetadata(func(context.Context) (SSEHandshake, error) {
				return SSEHandshake{Status: 201}, nil
			})
			writer := newHandshakeWriter(&deadlineResponseWriter{header: make(http.Header)}, WithSSEResponseMetadata(metadata))
			switch terminal {
			case "handler error":
				metadata.Close()
			case "SSE failure":
				require.False(t, writer.ClaimFailure())
			case "SSE suppression":
				require.NoError(t, writer.Suppress())
			case "SSE close":
				require.NoError(t, writer.Close())
			}
			require.ErrorIs(t, metadata.Set(func() (SSEHandshake, error) {
				t.Error("terminal setter executed")
				return SSEHandshake{Status: 202}, nil
			}), ErrSSEResponseCommitted)
		})
	}
}

func TestResponseMetadataCaptureWinsSetter(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	metadata := NewResponseMetadata(func(context.Context) (SSEHandshake, error) {
		close(entered)
		<-release
		return SSEHandshake{Status: 201}, nil
	})
	captured, set := make(chan error, 1), make(chan error, 1)
	go func() {
		_, err := metadata.Capture(context.Background())
		captured <- err
	}()
	<-entered
	go func() {
		set <- metadata.Set(func() (SSEHandshake, error) {
			t.Error("setter ran after capture admission")
			return SSEHandshake{Status: 202}, nil
		})
	}()
	close(release)
	require.NoError(t, <-captured)
	require.ErrorIs(t, <-set, ErrSSEResponseCommitted)
}
