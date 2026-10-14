package ticktock

import (
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	clock "example.com/ticktock/gen/clock"
	client "example.com/ticktock/gen/jsonrpc/clock/client"
	server "example.com/ticktock/gen/jsonrpc/clock/server"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/jsonrpc"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

type (
	contractContextKey struct{}
	contractClock      struct {
		clock.Service
		wait     bool
		fail     bool
		finished chan struct{}
	}
	contractAudit struct {
		t        *testing.T
		calls    int
		contexts []any
	}
)

func (s *contractClock) Tick(ctx context.Context, _ *clock.TickPayload, stream clock.TickServerStream) error {
	defer close(s.finished)
	value := "one"
	if err := stream.Send(ctx, &clock.TickResult{Value: &value}); err != nil {
		return err
	}
	if s.fail {
		return errors.New("private service failure")
	}
	if s.wait {
		<-ctx.Done()
		return ctx.Err()
	}
	return nil
}

func (a *contractAudit) StreamAudit(ctx context.Context, info *clock.StreamAuditInfo, next loom.Endpoint) (any, error) {
	a.calls++
	a.contexts = append(a.contexts, ctx.Value(contractContextKey{}))
	require.NotNil(a.t, ctx.Value(contractContextKey{}), "interception retains the caller's context")
	res, err := next(ctx, nil)
	if err == nil {
		require.Equal(a.t, "one", info.ClientStreamingResult(res).Value())
		info.ClientStreamingResult(res).SetValue("seen")
	}
	return res, err
}

func TestTypedServiceStreamContract(t *testing.T) {
	for _, intercept := range []bool{false, true} {
		for _, mode := range []string{"eof", "protocol-error", "opening-cancel", "call-cancel", "deadline", "close"} {
			t.Run(mode+map[bool]string{false: "/plain", true: "/intercepted"}[intercept], func(t *testing.T) {
				svc := &contractClock{wait: mode != "eof" && mode != "protocol-error", fail: mode == "protocol-error", finished: make(chan struct{})}
				mux := loomhttp.NewMuxer()
				server.Mount(mux, server.New(clock.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
				hs := httptest.NewServer(mux)
				defer hs.Close()
				transport := client.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
				audit := &contractAudit{t: t}
				var ci clock.ClientInterceptors
				if intercept {
					ci = audit
				}
				c := clock.NewClient(transport.Tick(), transport.Tock(), ci)
				ctx, cancel := context.WithCancel(context.WithValue(context.Background(), contractContextKey{}, "opening"))
				defer cancel()
				payload := &clock.TickPayload{}
				if mode == "protocol-error" {
					id := "error-request"
					payload.ID = &id
				}
				stream, err := c.Tick(ctx, payload)
				require.NoError(t, err)
				defer func() { require.NoError(t, stream.Close()) }()
				event, err := stream.Recv()
				require.NoError(t, err)
				want := "one"
				if intercept {
					want = "seen"
				}
				require.Equal(t, want, *event.Value)
				switch mode {
				case "eof":
					_, err = stream.RecvWithContext(context.WithValue(ctx, contractContextKey{}, "operation"))
					require.ErrorIs(t, err, io.EOF)
				case "protocol-error":
					_, err = stream.Recv()
					var protocolError *jsonrpc.RawErrorResponse
					require.ErrorAs(t, err, &protocolError)
					require.Equal(t, int(jsonrpc.InternalError), protocolError.Code)
					require.Equal(t, "internal server error", protocolError.Message)
					require.NotContains(t, err.Error(), "private service failure")
				case "opening-cancel":
					cancel()
					_, err = stream.Recv()
					require.ErrorIs(t, err, context.Canceled)
				case "call-cancel":
					call, stop := context.WithCancel(ctx)
					stop()
					_, err = stream.RecvWithContext(call)
					require.ErrorIs(t, err, context.Canceled)
				case "deadline":
					call, stop := context.WithTimeout(ctx, 30*time.Millisecond)
					defer stop()
					_, err = stream.RecvWithContext(call)
					require.ErrorIs(t, err, context.DeadlineExceeded)
				case "close":
					require.NoError(t, stream.Close())
					_, err = stream.Recv()
					require.True(t, errors.Is(err, io.EOF), "closed receive: %v", err)
				}
				require.NoError(t, stream.Close())
				select {
				case <-svc.finished:
				case <-time.After(3 * time.Second):
					t.Fatal("server did not release the stream")
				}
				if intercept {
					require.Equal(t, 2, audit.calls)
					require.Equal(t, "opening", audit.contexts[0])
					expected := "opening"
					if mode == "eof" {
						expected = "operation"
					}
					require.Equal(t, expected, audit.contexts[1])
				}
			})
		}
	}
}
