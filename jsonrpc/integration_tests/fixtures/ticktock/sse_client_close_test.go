package ticktock

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	clock "example.com/ticktock/gen/clock"
	clockclient "example.com/ticktock/gen/jsonrpc/clock/client"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/require"
)

type fakeSSEDoer struct {
	body io.ReadCloser
}

func (d fakeSSEDoer) Do(*http.Request) (*http.Response, error) {
	return &http.Response{
		Body:       d.body,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		StatusCode: http.StatusOK,
	}, nil
}

type blockingSSEBody struct {
	readStarted chan struct{}
	closed      chan struct{}
	closeErr    error
	readOnce    sync.Once
	closeOnce   sync.Once
}

func newBlockingSSEBody() *blockingSSEBody {
	return &blockingSSEBody{
		readStarted: make(chan struct{}),
		closed:      make(chan struct{}),
	}
}

func (b *blockingSSEBody) Read([]byte) (int, error) {
	b.readOnce.Do(func() {
		close(b.readStarted)
	})
	<-b.closed
	return 0, io.EOF
}

func (b *blockingSSEBody) Close() error {
	b.closeOnce.Do(func() {
		close(b.closed)
	})
	return b.closeErr
}

func TestJSONRPCGeneratedSSEClientCloseUnblocksBlockedRecv(t *testing.T) {
	body := newBlockingSSEBody()
	stream := newJSONRPCSSETestStream(t, body)

	recvc := make(chan error, 1)
	go func() {
		_, err := stream.Recv(context.Background())
		recvc <- err
	}()
	waitForReadStart(t, body)

	closec := make(chan error, 1)
	go func() {
		closec <- stream.Close()
	}()

	require.NoError(t, receiveSoon(t, closec, "Close"))
	require.ErrorIs(t, receiveSoon(t, recvc, "Recv"), io.EOF)
}

func TestJSONRPCGeneratedSSEClientRecvHonorsContextWhileReadBlocked(t *testing.T) {
	body := newBlockingSSEBody()
	stream := newJSONRPCSSETestStream(t, body)
	ctx, cancel := context.WithCancel(context.Background())

	recvc := make(chan error, 1)
	go func() {
		_, err := stream.Recv(ctx)
		recvc <- err
	}()
	waitForReadStart(t, body)
	cancel()

	require.ErrorIs(t, receiveSoon(t, recvc, "Recv"), context.Canceled)
}

func TestJSONRPCGeneratedSSEClientContextErrorWinsOverCloseError(t *testing.T) {
	body := newBlockingSSEBody()
	body.closeErr = errors.New("close failed")
	stream := newJSONRPCSSETestStream(t, body)
	ctx, cancel := context.WithCancel(context.Background())

	recvc := make(chan error, 1)
	go func() {
		_, err := stream.Recv(ctx)
		recvc <- err
	}()
	waitForReadStart(t, body)
	cancel()

	require.ErrorIs(t, receiveSoon(t, recvc, "Recv"), context.Canceled)
}

func newJSONRPCSSETestStream(t *testing.T, body io.ReadCloser) *clockclient.TickClientStream {
	t.Helper()

	client := clockclient.NewClient(
		"http",
		"example.test",
		fakeSSEDoer{body: body},
		loomhttp.RequestEncoder,
		loomhttp.ResponseDecoder,
		false,
	)
	endpoint := client.Tick()
	raw, err := endpoint(context.Background(), &clock.TickPayload{})
	require.NoError(t, err)
	stream, ok := raw.(*clockclient.TickClientStream)
	require.True(t, ok)
	return stream
}

func waitForReadStart(t *testing.T, body *blockingSSEBody) {
	t.Helper()

	select {
	case <-body.readStarted:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting for generated SSE client to start reading")
	}
}

func receiveSoon(t *testing.T, errc <-chan error, operation string) error {
	t.Helper()

	select {
	case err := <-errc:
		return err
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("timed out waiting for generated SSE client %s", operation)
		return nil
	}
}
