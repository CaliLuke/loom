package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	loompb "github.com/CaliLuke/loom/grpc/pb"
	loom "github.com/CaliLuke/loom/pkg"
)

// testServerTransportStream is a fake grpc.ServerTransportStream that records
// the header and trailer metadata sent by the handler.
type testServerTransportStream struct {
	sendHeaderErr error
	setTrailerErr error
	header        metadata.MD
	trailer       metadata.MD
}

// Method returns a fixed full method name.
func (s *testServerTransportStream) Method() string {
	return "loom.test.Service/Method"
}

// SetHeader records the given header metadata.
func (s *testServerTransportStream) SetHeader(md metadata.MD) error {
	s.header = metadata.Join(s.header, md)
	return nil
}

// SendHeader records the given header metadata or fails with sendHeaderErr.
func (s *testServerTransportStream) SendHeader(md metadata.MD) error {
	if s.sendHeaderErr != nil {
		return s.sendHeaderErr
	}
	s.header = metadata.Join(s.header, md)
	return nil
}

// SetTrailer records the given trailer metadata or fails with setTrailerErr.
func (s *testServerTransportStream) SetTrailer(md metadata.MD) error {
	if s.setTrailerErr != nil {
		return s.setTrailerErr
	}
	s.trailer = metadata.Join(s.trailer, md)
	return nil
}

func TestUnaryHandlerHandle(t *testing.T) {
	var (
		errDecode          = errors.New("decode failed")
		errEncode          = errors.New("encode failed")
		svcErrDecode       = loom.PermanentError("bad_request", "decode service error")
		svcErrEncode       = loom.PermanentError("bad_response", "encode service error")
		errEndpoint        = errors.New("endpoint failed")
		statusErrEndpoint  = NewStatusError(codes.PermissionDenied, loom.PermanentError("forbidden", "nope"))
		failingEndpointErr = func(err error) loom.Endpoint {
			return func(_ context.Context, _ any) (any, error) {
				return nil, err
			}
		}
		okEndpoint = func(_ context.Context, req any) (any, error) {
			return req, nil
		}
	)

	cases := []struct {
		name           string
		decoder        RequestDecoder
		encoder        ResponseEncoder
		endpoint       loom.Endpoint
		stream         *testServerTransportStream
		wantResp       any
		wantErr        error
		wantCode       codes.Code
		wantDetailName string
	}{
		{
			name: "happy path",
			decoder: func(_ context.Context, pb any, _ metadata.MD) (any, error) {
				assert.Equal(t, "pb-request", pb)
				return "loom-request", nil
			},
			endpoint: func(_ context.Context, req any) (any, error) {
				assert.Equal(t, "loom-request", req)
				return "loom-response", nil
			},
			encoder: func(_ context.Context, v any, _, _ *metadata.MD) (any, error) {
				assert.Equal(t, "loom-response", v)
				return "pb-response", nil
			},
			wantResp: "pb-response",
		},
		{
			name: "nil decoder and encoder",
			endpoint: func(_ context.Context, req any) (any, error) {
				assert.Nil(t, req)
				return "loom-response", nil
			},
			wantResp: nil,
		},
		{
			name: "decode error maps to invalid argument status",
			decoder: func(_ context.Context, _ any, _ metadata.MD) (any, error) {
				return nil, errDecode
			},
			endpoint: func(_ context.Context, _ any) (any, error) {
				t.Errorf("endpoint must not be called when decoding fails")
				return nil, nil
			},
			wantCode:       codes.InvalidArgument,
			wantDetailName: loom.DecodePayload,
		},
		{
			name: "decode service error propagates unchanged",
			decoder: func(_ context.Context, _ any, _ metadata.MD) (any, error) {
				return nil, svcErrDecode
			},
			endpoint: okEndpoint,
			wantErr:  svcErrDecode,
		},
		{
			name:     "endpoint error propagates unchanged",
			endpoint: failingEndpointErr(errEndpoint),
			wantErr:  errEndpoint,
		},
		{
			name:     "endpoint status error propagates unchanged",
			endpoint: failingEndpointErr(statusErrEndpoint),
			wantErr:  statusErrEndpoint,
		},
		{
			name:     "encode error maps to unknown status",
			endpoint: okEndpoint,
			encoder: func(_ context.Context, _ any, _, _ *metadata.MD) (any, error) {
				return nil, errEncode
			},
			wantCode:       codes.Unknown,
			wantDetailName: "fault",
		},
		{
			name:     "encode service error propagates unchanged",
			endpoint: okEndpoint,
			encoder: func(_ context.Context, _ any, _, _ *metadata.MD) (any, error) {
				return nil, svcErrEncode
			},
			wantErr: svcErrEncode,
		},
		{
			name:     "sends headers and trailers through transport stream",
			endpoint: okEndpoint,
			encoder: func(_ context.Context, _ any, hdr, trlr *metadata.MD) (any, error) {
				hdr.Set("resp-header", "header-value")
				trlr.Set("resp-trailer", "trailer-value")
				return "pb-response", nil
			},
			stream:   &testServerTransportStream{},
			wantResp: "pb-response",
		},
		{
			name:     "header send failure maps to unknown status",
			endpoint: okEndpoint,
			encoder: func(_ context.Context, _ any, hdr, _ *metadata.MD) (any, error) {
				hdr.Set("resp-header", "header-value")
				return "pb-response", nil
			},
			stream:         &testServerTransportStream{sendHeaderErr: errors.New("send header failed")},
			wantCode:       codes.Unknown,
			wantDetailName: "fault",
		},
		{
			name:     "trailer send failure maps to unknown status",
			endpoint: okEndpoint,
			encoder: func(_ context.Context, _ any, _, trlr *metadata.MD) (any, error) {
				trlr.Set("resp-trailer", "trailer-value")
				return "pb-response", nil
			},
			stream:         &testServerTransportStream{setTrailerErr: errors.New("set trailer failed")},
			wantCode:       codes.Unknown,
			wantDetailName: "fault",
		},
		{
			// Without a server transport stream in the context, sending
			// headers fails and the handler returns an unknown status error.
			name:     "header send without transport stream maps to unknown status",
			endpoint: okEndpoint,
			encoder: func(_ context.Context, _ any, hdr, _ *metadata.MD) (any, error) {
				hdr.Set("resp-header", "header-value")
				return "pb-response", nil
			},
			wantCode:       codes.Unknown,
			wantDetailName: "fault",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			handler := NewUnaryHandler(c.endpoint, c.decoder, c.encoder)
			ctx := context.Background()
			if c.stream != nil {
				ctx = grpc.NewContextWithServerTransportStream(ctx, c.stream)
			}
			respb, err := handler.Handle(ctx, "pb-request")
			switch {
			case c.wantErr != nil:
				require.ErrorIs(t, err, c.wantErr)
				assert.Nil(t, respb)
			case c.wantDetailName != "":
				requireStatusWithDetail(t, err, c.wantCode, c.wantDetailName)
				assert.Nil(t, respb)
			default:
				require.NoError(t, err)
				assert.Equal(t, c.wantResp, respb)
			}
			if c.stream != nil && err == nil {
				assert.Equal(t, []string{"header-value"}, c.stream.header.Get("resp-header"))
				assert.Equal(t, []string{"trailer-value"}, c.stream.trailer.Get("resp-trailer"))
			}
		})
	}
}

func TestUnaryHandlerDecoderReceivesIncomingMetadata(t *testing.T) {
	decoder := func(_ context.Context, pb any, md metadata.MD) (any, error) {
		assert.Equal(t, "pb-request", pb)
		assert.Equal(t, []string{"meta-value"}, md.Get("meta-key"), "decoder must receive incoming metadata")
		return "loom-request", nil
	}
	endpoint := func(_ context.Context, req any) (any, error) {
		return req, nil
	}

	handler := NewUnaryHandler(endpoint, decoder, nil)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("meta-key", "meta-value"))
	respb, err := handler.Handle(ctx, "pb-request")
	require.NoError(t, err)
	assert.Nil(t, respb)
}

func TestStreamHandlerDecode(t *testing.T) {
	var (
		errDecode    = errors.New("stream decode failed")
		svcErrDecode = loom.PermanentError("bad_stream", "stream decode service error")
	)

	cases := []struct {
		name           string
		decoder        RequestDecoder
		wantReq        any
		wantErr        error
		wantCode       codes.Code
		wantDetailName string
	}{
		{
			name: "happy path with incoming metadata",
			decoder: func(_ context.Context, pb any, md metadata.MD) (any, error) {
				assert.Equal(t, "pb-request", pb)
				assert.Equal(t, []string{"meta-value"}, md.Get("meta-key"))
				return "loom-request", nil
			},
			wantReq: "loom-request",
		},
		{
			name:    "nil decoder returns nil request",
			wantReq: nil,
		},
		{
			name: "decode error maps to invalid argument status",
			decoder: func(_ context.Context, _ any, _ metadata.MD) (any, error) {
				return nil, errDecode
			},
			wantCode:       codes.InvalidArgument,
			wantDetailName: loom.DecodePayload,
		},
		{
			name: "decode service error propagates unchanged",
			decoder: func(_ context.Context, _ any, _ metadata.MD) (any, error) {
				return nil, svcErrDecode
			},
			wantErr: svcErrDecode,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			endpoint := func(_ context.Context, _ any) (any, error) {
				t.Errorf("endpoint must not be called by Decode")
				return nil, nil
			}
			handler := NewStreamHandler(endpoint, c.decoder)
			ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("meta-key", "meta-value"))
			req, err := handler.Decode(ctx, "pb-request")
			switch {
			case c.wantErr != nil:
				require.ErrorIs(t, err, c.wantErr)
				assert.Nil(t, req)
			case c.wantDetailName != "":
				requireStatusWithDetail(t, err, c.wantCode, c.wantDetailName)
				assert.Nil(t, req)
			default:
				require.NoError(t, err)
				assert.Equal(t, c.wantReq, req)
			}
		})
	}
}

func TestStreamHandlerHandle(t *testing.T) {
	errEndpoint := errors.New("stream endpoint failed")

	cases := []struct {
		name    string
		wantErr error
	}{
		{
			name: "endpoint success",
		},
		{
			name:    "endpoint error propagates unchanged",
			wantErr: errEndpoint,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var gotStream any
			endpoint := func(_ context.Context, stream any) (any, error) {
				gotStream = stream
				return nil, c.wantErr
			}
			handler := NewStreamHandler(endpoint, nil)
			err := handler.Handle(context.Background(), "stream-value")
			if c.wantErr != nil {
				require.ErrorIs(t, err, c.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, "stream-value", gotStream, "endpoint must receive the stream value")
		})
	}
}

func TestNewStatusError(t *testing.T) {
	t.Run("attaches details", func(t *testing.T) {
		serr := loom.PermanentError("boom", "it broke")
		err := NewStatusError(codes.FailedPrecondition, serr, NewErrorResponse(serr))
		requireStatusWithDetail(t, err, codes.FailedPrecondition, "boom")
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, serr.Error(), st.Message())
	})

	t.Run("no details", func(t *testing.T) {
		err := NewStatusError(codes.NotFound, errors.New("missing"))
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
		assert.Equal(t, "missing", st.Message())
		assert.Empty(t, st.Details())
	})

	t.Run("ok code returns nil", func(t *testing.T) {
		// Current behavior: status.WithDetails rejects codes.OK, so the
		// fallback st.Err() is used which returns nil for an OK status.
		err := NewStatusError(codes.OK, errors.New("ignored"), NewErrorResponse(errors.New("ignored")))
		assert.NoError(t, err)
	})
}

func TestErrInvalidType(t *testing.T) {
	err := ErrInvalidType("calc", "add", "*calcpb.AddRequest", 42)

	var cerr *ClientError
	require.ErrorAs(t, err, &cerr)
	assert.Equal(t, "invalid_type", cerr.Name)
	assert.Equal(t, "calc", cerr.Service)
	assert.Equal(t, "add", cerr.Method)
	assert.Equal(t, "invalid value expected *calcpb.AddRequest, got 42", cerr.Message)
	assert.False(t, cerr.Temporary)
	assert.False(t, cerr.Timeout)
	assert.False(t, cerr.Fault)
	assert.Equal(t, "[calc add]: invalid value expected *calcpb.AddRequest, got 42", err.Error())
}

// requireStatusWithDetail asserts that err is a gRPC status error with the
// given code carrying an ErrorResponse detail with the given error name.
func requireStatusWithDetail(t *testing.T, err error, code codes.Code, detailName string) {
	t.Helper()
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok, "error must be a gRPC status error")
	assert.Equal(t, code, st.Code())
	details := st.Details()
	require.Len(t, details, 1)
	resp, ok := details[0].(*loompb.ErrorResponse)
	require.True(t, ok, "status detail must be an ErrorResponse")
	assert.Equal(t, detailName, resp.Name)
}
