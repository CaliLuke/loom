package ticktock_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	clockclient "example.com/http-ticktock/gen/http/clock/client"
)

type blockingBody struct {
	readStarted chan struct{}
	closed      chan struct{}
	startOnce   sync.Once
	closeOnce   sync.Once
	mu          sync.Mutex
	chunks      [][]byte
	afterChunk  func()
	closeErr    error
}

func newBlockingBody(chunks ...[]byte) *blockingBody {
	return &blockingBody{
		readStarted: make(chan struct{}),
		closed:      make(chan struct{}),
		chunks:      chunks,
	}
}

func (b *blockingBody) Read(p []byte) (int, error) {
	b.startOnce.Do(func() {
		close(b.readStarted)
	})
	b.mu.Lock()
	if len(b.chunks) > 0 {
		chunk := b.chunks[0]
		b.chunks = b.chunks[1:]
		afterChunk := b.afterChunk
		b.mu.Unlock()
		n := copy(p, chunk)
		if afterChunk != nil {
			afterChunk()
		}
		return n, nil
	}
	b.mu.Unlock()
	<-b.closed
	return 0, io.EOF
}

func (b *blockingBody) Close() error {
	b.closeOnce.Do(func() {
		close(b.closed)
	})
	return b.closeErr
}

func TestGeneratedSSEClientCloseUnblocksBlockedRecv(t *testing.T) {
	body := newBlockingBody()
	stream := clockclient.NewTickStream(&http.Response{Body: body}, nil)

	recvc := make(chan error, 1)
	go func() {
		_, err := stream.Recv(context.Background())
		recvc <- err
	}()

	requireReadStarted(t, body)

	closec := make(chan error, 1)
	go func() {
		closec <- stream.Close()
	}()

	require.NoError(t, receiveSoon(t, closec, "Close"))
	require.NoError(t, receiveSoon(t, recvc, "Recv"))
}

func TestGeneratedSSEClientRecvHonorsContextWhileReadBlocked(t *testing.T) {
	body := newBlockingBody()
	stream := clockclient.NewTickStream(&http.Response{Body: body}, nil)
	ctx, cancel := context.WithCancel(context.Background())

	recvc := make(chan error, 1)
	go func() {
		_, err := stream.Recv(ctx)
		recvc <- err
	}()

	requireReadStarted(t, body)
	cancel()

	err := receiveSoon(t, recvc, "Recv")
	require.ErrorIs(t, err, context.Canceled)
}

func TestGeneratedSSEClientContextErrorWinsOverCloseError(t *testing.T) {
	body := newBlockingBody()
	body.closeErr = errors.New("close failed")
	stream := clockclient.NewTickStream(&http.Response{Body: body}, nil)
	ctx, cancel := context.WithCancel(context.Background())

	recvc := make(chan error, 1)
	go func() {
		_, err := stream.Recv(ctx)
		recvc <- err
	}()

	requireReadStarted(t, body)
	cancel()

	err := receiveSoon(t, recvc, "Recv")
	require.ErrorIs(t, err, context.Canceled)
	require.NotErrorIs(t, err, body.closeErr)
}

func TestGeneratedSSEClientCanceledRecvDoesNotReturnPartialEvent(t *testing.T) {
	body := newBlockingBody([]byte("data: partial"))
	stream := clockclient.NewTickStream(&http.Response{Body: body}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	body.afterChunk = cancel

	recvc := make(chan error, 1)
	go func() {
		_, err := stream.Recv(ctx)
		recvc <- err
	}()

	err := receiveSoon(t, recvc, "Recv")
	require.ErrorIs(t, err, context.Canceled)
}

func TestGeneratedSSEClientCancelAfterSuccessfulRecvDoesNotCloseStream(t *testing.T) {
	body := newBlockingBody(
		[]byte("event: tick\ndata: one\n\n"),
		[]byte("event: tick\ndata: two\n\n"),
	)
	stream := clockclient.NewTickStream(&http.Response{Body: body}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	first, err := stream.Recv(ctx)
	require.NoError(t, err)
	require.Equal(t, "one", first.Data)
	cancel()

	second, err := stream.Recv(context.Background())
	require.NoError(t, err)
	require.Equal(t, "two", second.Data)
}

func TestGeneratedSSEClientCancelAfterCompleteReadDoesNotDropEvent(t *testing.T) {
	body := newBlockingBody([]byte("event: tick\ndata: one\n\n"))
	stream := clockclient.NewTickStream(&http.Response{Body: body}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	body.afterChunk = cancel

	event, err := stream.Recv(ctx)
	require.NoError(t, err)
	require.Equal(t, "one", event.Data)
}

func requireReadStarted(t *testing.T, body *blockingBody) {
	t.Helper()
	select {
	case <-body.readStarted:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for generated SSE client to start reading")
	}
}

func receiveSoon(t *testing.T, errc <-chan error, operation string) error {
	t.Helper()
	select {
	case err := <-errc:
		return err
	case <-time.After(500 * time.Millisecond):
		return errors.New(operation + " did not return")
	}
}
