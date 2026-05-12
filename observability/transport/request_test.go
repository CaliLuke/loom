package transport_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CaliLuke/loom/observability/transport"
	"github.com/stretchr/testify/require"
)

func TestRequestObserverEmitsStartAndFinishOnSuccess(t *testing.T) {
	t.Parallel()
	rec := &recordingObserver{}
	r := httptest.NewRequest(http.MethodGet, "/foo", nil)
	w := httptest.NewRecorder()
	ctx := transport.WithObserver(r.Context(), rec)
	obs, ww := transport.BeginHTTPRequest(ctx, w, "Service", "Method", r)
	ww.WriteHeader(http.StatusOK)
	_, _ = ww.Write([]byte("ok"))
	obs.End()

	events := rec.snapshot()
	require.Len(t, events, 2)
	require.Equal(t, transport.EventKindRequestStart, events[0].Kind)
	require.Equal(t, "Service", events[0].Service)
	require.Equal(t, "Method", events[0].Method)
	require.Equal(t, transport.TransportHTTP, events[0].Transport)
	require.Equal(t, transport.EventKindRequestFinish, events[1].Kind)
	require.Equal(t, transport.ReasonOK, events[1].Reason)
	require.Equal(t, http.StatusOK, events[1].StatusCode)
	require.EqualValues(t, 2, events[1].BytesWritten)
}

func TestRequestObserverEmitsFailureWithRecordedReason(t *testing.T) {
	t.Parallel()
	rec := &recordingObserver{}
	ctx := transport.WithObserver(context.Background(), rec)
	obs := transport.BeginRequest(ctx, transport.TransportJSONRPC, "Svc", "Method")
	obs.Fail(transport.ReasonRequestDecodeFailed)
	obs.End()

	events := rec.snapshot()
	require.Len(t, events, 2)
	require.Equal(t, transport.EventKindRequestFailure, events[1].Kind)
	require.Equal(t, transport.ReasonRequestDecodeFailed, events[1].Reason)
	require.Equal(t, transport.TransportJSONRPC, events[1].Transport)
}

func TestRequestObserverFirstFailWins(t *testing.T) {
	t.Parallel()
	rec := &recordingObserver{}
	ctx := transport.WithObserver(context.Background(), rec)
	obs := transport.BeginRequest(ctx, transport.TransportHTTP, "S", "M")
	obs.Fail(transport.ReasonHandlerError)
	obs.Fail(transport.ReasonResponseWriteFailed) // ignored
	obs.End()

	events := rec.snapshot()
	require.Equal(t, transport.ReasonHandlerError, events[1].Reason)
}

func TestRequestObserverEmitsPanicAndRepanics(t *testing.T) {
	t.Parallel()
	rec := &recordingObserver{}
	ctx := transport.WithObserver(context.Background(), rec)

	require.PanicsWithValue(t, "boom", func() {
		obs := transport.BeginRequest(ctx, transport.TransportHTTP, "S", "M")
		defer obs.End()
		panic("boom")
	})

	events := rec.snapshot()
	require.Len(t, events, 2)
	require.Equal(t, transport.EventKindRequestFailure, events[1].Kind)
	require.Equal(t, transport.ReasonPanic, events[1].Reason)
}

func TestRequestObserverStreamEvents(t *testing.T) {
	t.Parallel()
	rec := &recordingObserver{}
	ctx := transport.WithObserver(context.Background(), rec)
	obs := transport.BeginRequest(ctx, transport.TransportHTTP, "S", "M")
	obs.EmitStreamOpen()
	obs.EmitStreamFailure(transport.ReasonStreamWriteFailed)
	obs.EmitStreamClose()
	obs.End()

	events := rec.snapshot()
	require.Equal(t, transport.EventKindStreamOpen, events[1].Kind)
	require.Equal(t, transport.EventKindStreamFailure, events[2].Kind)
	require.Equal(t, transport.ReasonStreamWriteFailed, events[2].Reason)
	require.Equal(t, transport.EventKindStreamClose, events[3].Kind)
	require.Equal(t, transport.EventKindRequestFinish, events[4].Kind)
}

func TestRequestObserverNilSafe(t *testing.T) {
	t.Parallel()
	var obs *transport.RequestObserver
	obs.Fail(transport.ReasonHandlerError)
	obs.EmitStreamOpen()
	obs.EmitStreamClose()
	obs.EmitStreamFailure(transport.ReasonStreamWriteFailed)
	obs.SetJSONRPC("m", "1", 0, false)
	obs.SetSession("s")
	obs.End()
}

func TestRequestObserverSetsJSONRPCAndSessionFields(t *testing.T) {
	t.Parallel()
	rec := &recordingObserver{}
	ctx := transport.WithObserver(context.Background(), rec)
	obs := transport.BeginRequest(ctx, transport.TransportJSONRPC, "S", "M")
	obs.SetJSONRPC("rpcMethod", "42", 3, false)
	obs.SetSession("session-x")
	obs.End()

	events := rec.snapshot()
	finish := events[len(events)-1]
	require.Equal(t, "rpcMethod", finish.JSONRPCMethod)
	require.Equal(t, "42", finish.JSONRPCID)
	require.Equal(t, 3, finish.BatchCount)
	require.Equal(t, "session-x", finish.SessionID)
}
