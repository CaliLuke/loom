package http_test

import (
	"context"
	"math"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"

	loomhttp "github.com/CaliLuke/loom/http"
)

func TestWebSocketConfigValidation(t *testing.T) {
	for _, limits := range []loomhttp.WebSocketLimits{
		{}, {MaxMessageBytes: 1}, {MaxMessageBytes: math.MaxInt64 / 2},
		{MaxMessageBytes: 10, MaxBufferedBytes: 20, MaxBufferedMessages: 1},
	} {
		config, err := loomhttp.NewWebSocketConfig(limits, nil)
		require.NoError(t, err)
		require.Positive(t, config.MaxMessageBytes())
		require.GreaterOrEqual(t, config.MaxBufferedBytes(), 2*config.MaxMessageBytes())
		require.Positive(t, config.MaxBufferedMessages())
	}
	for _, limits := range []loomhttp.WebSocketLimits{
		{MaxMessageBytes: -1}, {MaxMessageBytes: math.MaxInt64/2 + 1},
		{MaxBufferedBytes: -1}, {MaxBufferedMessages: -1},
		{MaxMessageBytes: 10, MaxBufferedBytes: 19},
	} {
		_, err := loomhttp.NewWebSocketConfig(limits, nil)
		require.Error(t, err)
	}
	config := loomhttp.WebSocketConfig{}
	require.EqualValues(t, loomhttp.DefaultMaxRequestBodyBytes, config.MaxMessageBytes())
	require.EqualValues(t, 2*loomhttp.DefaultMaxRequestBodyBytes, config.MaxBufferedBytes())
	require.Equal(t, 256, config.MaxBufferedMessages())
}

func TestWebSocketSetConnOwnsAttachment(t *testing.T) {
	first, cleanupFirst := newIdleWebSocketConn(t)
	defer cleanupFirst()
	second, cleanupSecond := newIdleWebSocketConn(t)
	defer cleanupSecond()
	var calls atomic.Int32
	config, err := loomhttp.NewWebSocketConfig(loomhttp.WebSocketLimits{MaxMessageBytes: 1024}, func(conn *websocket.Conn, cancel context.CancelFunc) *websocket.Conn {
		calls.Add(1)
		require.NotNil(t, cancel)
		return conn
	})
	require.NoError(t, err)
	stream := loomhttp.NewWebSocketStream(config)
	defer func() {
		require.NoError(t, stream.Close())
	}()
	require.Error(t, stream.SetConn(context.Background(), nil, nil))
	_, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, stream.SetConn(context.Background(), first, cancel))
	require.NoError(t, stream.SetConn(context.Background(), first, nil))
	require.EqualValues(t, 1, calls.Load())
	require.Error(t, stream.SetConn(context.Background(), second, nil))
	require.Same(t, first, stream.Conn())
	require.ErrorIs(t, second.NetConn().SetDeadline(time.Now()), net.ErrClosed)
}

func TestWebSocketSetConnHookOwnership(t *testing.T) {
	for _, replacement := range []bool{false, true} {
		name := "nil result closes input"
		if replacement {
			name = "hook releases replaced input"
		}
		t.Run(name, func(t *testing.T) {
			input, cleanup := newIdleWebSocketConn(t)
			defer cleanup()
			var output *websocket.Conn
			if replacement {
				var cleanupOutput func()
				output, cleanupOutput = newIdleWebSocketConn(t)
				defer cleanupOutput()
			}
			config, err := loomhttp.NewWebSocketConfig(loomhttp.WebSocketLimits{MaxMessageBytes: 1024}, func(conn *websocket.Conn, _ context.CancelFunc) *websocket.Conn {
				if replacement {
					require.NoError(t, conn.Close())
				}
				return output
			})
			require.NoError(t, err)
			stream := loomhttp.NewWebSocketStream(config)
			err = stream.SetConn(context.Background(), input, nil)
			if replacement {
				require.NoError(t, err)
				require.Same(t, output, stream.Conn())
				require.NoError(t, output.NetConn().SetDeadline(time.Now().Add(time.Second)))
			} else {
				require.Error(t, err)
				require.Same(t, input, stream.Conn())
			}
			require.ErrorIs(t, input.NetConn().SetDeadline(time.Now()), net.ErrClosed)
			require.NoError(t, stream.Close())
		})
	}
}

func TestWebSocketSetConnCloseDuringHook(t *testing.T) {
	conn, cleanup := newIdleWebSocketConn(t)
	defer cleanup()
	entered := make(chan struct{})
	resume := make(chan struct{})
	config, err := loomhttp.NewWebSocketConfig(loomhttp.WebSocketLimits{MaxMessageBytes: 1024}, func(conn *websocket.Conn, _ context.CancelFunc) *websocket.Conn {
		close(entered)
		<-resume
		return conn
	})
	require.NoError(t, err)
	stream := loomhttp.NewWebSocketStream(config)
	attached := make(chan error, 1)
	go func() {
		attached <- stream.SetConn(context.Background(), conn, nil)
	}()
	<-entered
	require.NoError(t, stream.Close())
	close(resume)
	require.ErrorIs(t, <-attached, loomhttp.ErrWebSocketStreamClosed)
	require.ErrorIs(t, conn.NetConn().SetDeadline(time.Now()), net.ErrClosed)
}
