package log

import (
	"bytes"
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fakeServerStream implements grpc.ServerStream with a settable context.
type fakeServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *fakeServerStream) Context() context.Context {
	return s.ctx
}

func TestUnaryServerInterceptor(t *testing.T) {
	stubShortID(t, "call-1")
	stubTimeSince(t, 42*time.Millisecond)
	info := &grpc.UnaryServerInfo{FullMethod: "/calc.Calc/Add"}

	cases := []struct {
		name        string
		opts        []GRPCLogOption
		handlerErr  error
		contains    []string
		notContains []string
		empty       bool
	}{
		{
			name: "success logs start and end",
			contains: []string{
				"msg=start",
				"msg=end",
				"grpc.service=calc.Calc",
				"grpc.method=Add",
				"grpc.code=OK",
				"grpc.time_ms=42",
				"request_id=call-1",
			},
		},
		{
			name:       "handler error logs error",
			handlerErr: status.Error(codes.Internal, "boom"),
			contains: []string{
				"level=error",
				"grpc.code=Internal",
				"grpc.status=boom",
			},
			notContains: []string{"msg=end"},
		},
		{
			name:        "disable call id",
			opts:        []GRPCLogOption{WithDisableCallID()},
			contains:    []string{"msg=start", "msg=end"},
			notContains: []string{"request_id="},
		},
		{
			name:  "disable call logging",
			opts:  []GRPCLogOption{WithDisableCallLogging()},
			empty: true,
		},
		{
			name:       "custom error func treats errors as success",
			opts:       []GRPCLogOption{WithErrorFunc(func(codes.Code) bool { return false })},
			handlerErr: status.Error(codes.Internal, "boom"),
			contains:   []string{"msg=end", "grpc.code=Internal"},
			notContains: []string{
				"level=error",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			logCtx := newTestContext(&buf)
			interceptor := UnaryServerInterceptor(logCtx, tc.opts...)

			var handlerCtx context.Context
			res, err := interceptor(context.Background(), "request", info,
				func(ctx context.Context, req any) (any, error) {
					handlerCtx = ctx
					return "response", tc.handlerErr
				})

			require.Equal(t, tc.handlerErr, err)
			require.Equal(t, "response", res)
			require.NotPanics(t, func() {
				MustContainLogger(handlerCtx)
			}, "handler context must contain the logger")
			if tc.empty {
				require.Empty(t, buf.String())
				return
			}
			for _, want := range tc.contains {
				assert.Contains(t, buf.String(), want)
			}
			for _, unwanted := range tc.notContains {
				assert.NotContains(t, buf.String(), unwanted)
			}
		})
	}
}

func TestUnaryServerInterceptorCustomLogFunc(t *testing.T) {
	stubShortID(t, "call-1")
	stubTimeSince(t, time.Millisecond)
	var buf bytes.Buffer
	logCtx := newTestContext(&buf, WithDebug())
	interceptor := UnaryServerInterceptor(logCtx, WithCallLogFunc(Debug))

	_, err := interceptor(context.Background(), nil,
		&grpc.UnaryServerInfo{FullMethod: "/calc.Calc/Add"},
		func(ctx context.Context, req any) (any, error) {
			return nil, nil
		})

	require.NoError(t, err)
	require.Contains(t, buf.String(), "level=debug")
	require.Contains(t, buf.String(), "msg=start")
	require.Contains(t, buf.String(), "msg=end")
}

func TestUnaryServerInterceptorPanicsWithoutLogger(t *testing.T) {
	require.Panics(t, func() {
		UnaryServerInterceptor(context.Background())
	})
}

func TestStreamServerInterceptor(t *testing.T) {
	stubShortID(t, "stream-1")
	stubTimeSince(t, 42*time.Millisecond)
	info := &grpc.StreamServerInfo{FullMethod: "/calc.Calc/Stream"}

	cases := []struct {
		name        string
		opts        []GRPCLogOption
		handlerErr  error
		contains    []string
		notContains []string
		empty       bool
	}{
		{
			name: "success logs start and end",
			contains: []string{
				"msg=start",
				"msg=end",
				"grpc.service=calc.Calc",
				"grpc.method=Stream",
				"grpc.code=OK",
				"grpc.time_ms=42",
				"request_id=stream-1",
			},
		},
		{
			name:       "handler error logs error",
			handlerErr: status.Error(codes.NotFound, "missing"),
			contains: []string{
				"level=error",
				"grpc.code=NotFound",
				"grpc.status=missing",
			},
			notContains: []string{"msg=end"},
		},
		{
			name:  "disable call logging",
			opts:  []GRPCLogOption{WithDisableCallLogging()},
			empty: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			logCtx := newTestContext(&buf)
			interceptor := StreamServerInterceptor(logCtx, tc.opts...)
			stream := &fakeServerStream{ctx: context.Background()}

			var handlerStream grpc.ServerStream
			err := interceptor("service", stream, info,
				func(srv any, s grpc.ServerStream) error {
					handlerStream = s
					return tc.handlerErr
				})

			require.Equal(t, tc.handlerErr, err)
			require.NotPanics(t, func() {
				MustContainLogger(handlerStream.Context())
			}, "stream context must contain the logger")
			if tc.empty {
				require.Empty(t, buf.String())
				return
			}
			for _, want := range tc.contains {
				assert.Contains(t, buf.String(), want)
			}
			for _, unwanted := range tc.notContains {
				assert.NotContains(t, buf.String(), unwanted)
			}
		})
	}
}

func TestStreamServerInterceptorCustomLogFunc(t *testing.T) {
	stubShortID(t, "stream-1")
	stubTimeSince(t, time.Millisecond)
	var buf bytes.Buffer
	logCtx := newTestContext(&buf, WithDebug())
	interceptor := StreamServerInterceptor(logCtx, WithCallLogFunc(Debug))

	err := interceptor("service",
		&fakeServerStream{ctx: context.Background()},
		&grpc.StreamServerInfo{FullMethod: "/calc.Calc/Stream"},
		func(srv any, s grpc.ServerStream) error {
			return nil
		})

	require.NoError(t, err)
	require.Contains(t, buf.String(), "level=debug")
	require.Contains(t, buf.String(), "msg=start")
	require.Contains(t, buf.String(), "msg=end")
}

func TestStreamServerInterceptorPanicsWithoutLogger(t *testing.T) {
	require.Panics(t, func() {
		StreamServerInterceptor(context.Background())
	})
}

func TestUnaryClientInterceptor(t *testing.T) {
	stubTimeSince(t, 42*time.Millisecond)

	cases := []struct {
		name        string
		opts        []GRPCLogOption
		invokerErr  error
		contains    []string
		notContains []string
	}{
		{
			name: "success logs start and end",
			contains: []string{
				"msg=start",
				"msg=end",
				"grpc.service=calc.Calc",
				"grpc.method=Add",
				"grpc.code=OK",
				"grpc.time_ms=42",
			},
		},
		{
			name:       "invoker error logs error",
			invokerErr: status.Error(codes.Unavailable, "down"),
			contains: []string{
				"level=error",
				"grpc.code=Unavailable",
				"grpc.status=down",
			},
			notContains: []string{"msg=end"},
		},
		{
			name:       "custom error func suppresses error",
			opts:       []GRPCLogOption{WithErrorFunc(func(codes.Code) bool { return false })},
			invokerErr: status.Error(codes.Unavailable, "down"),
			contains:   []string{"msg=end"},
			notContains: []string{
				"level=error",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			ctx := newTestContext(&buf)
			interceptor := UnaryClientInterceptor(tc.opts...)

			invoked := false
			err := interceptor(ctx, "/calc.Calc/Add", "req", "reply", nil,
				func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
					invoked = true
					return tc.invokerErr
				})

			require.Equal(t, tc.invokerErr, err)
			require.True(t, invoked)
			for _, want := range tc.contains {
				assert.Contains(t, buf.String(), want)
			}
			for _, unwanted := range tc.notContains {
				assert.NotContains(t, buf.String(), unwanted)
			}
		})
	}
}

func TestUnaryClientInterceptorCustomLogFunc(t *testing.T) {
	stubTimeSince(t, time.Millisecond)
	var buf bytes.Buffer
	ctx := newTestContext(&buf, WithDebug())
	interceptor := UnaryClientInterceptor(WithCallLogFunc(Debug))

	err := interceptor(ctx, "/calc.Calc/Add", "req", "reply", nil,
		func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			return nil
		})

	require.NoError(t, err)
	require.Contains(t, buf.String(), "level=debug")
	require.Contains(t, buf.String(), "msg=start")
	require.Contains(t, buf.String(), "msg=end")
}

func TestStreamClientInterceptorCustomLogFunc(t *testing.T) {
	stubTimeSince(t, time.Millisecond)
	var buf bytes.Buffer
	ctx := newTestContext(&buf, WithDebug())
	interceptor := StreamClientInterceptor(WithCallLogFunc(Debug))

	stream, err := interceptor(ctx, &grpc.StreamDesc{}, nil, "/calc.Calc/Watch",
		func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
			return nil, nil
		})

	require.NoError(t, err)
	require.Nil(t, stream)
	require.Contains(t, buf.String(), "level=debug")
	require.Contains(t, buf.String(), "msg=start")
	require.Contains(t, buf.String(), "msg=end")
}

func TestStreamClientInterceptor(t *testing.T) {
	stubTimeSince(t, 42*time.Millisecond)

	cases := []struct {
		name        string
		streamerErr error
		contains    []string
		notContains []string
	}{
		{
			name: "success logs start and end",
			contains: []string{
				"msg=start",
				"msg=end",
				"grpc.service=calc.Calc",
				"grpc.method=Watch",
				"grpc.code=OK",
			},
		},
		{
			name:        "streamer error logs error",
			streamerErr: status.Error(codes.PermissionDenied, "nope"),
			contains: []string{
				"level=error",
				"grpc.code=PermissionDenied",
				"grpc.status=nope",
			},
			notContains: []string{"msg=end"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			ctx := newTestContext(&buf)
			interceptor := StreamClientInterceptor()

			stream, err := interceptor(ctx, &grpc.StreamDesc{}, nil, "/calc.Calc/Watch",
				func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
					return nil, tc.streamerErr
				})

			require.Equal(t, tc.streamerErr, err)
			require.Nil(t, stream)
			for _, want := range tc.contains {
				assert.Contains(t, buf.String(), want)
			}
			for _, unwanted := range tc.notContains {
				assert.NotContains(t, buf.String(), unwanted)
			}
		})
	}
}

func TestRandShortID(t *testing.T) {
	id := randShortID()
	require.Len(t, id, 8, "6 raw bytes must encode to 8 base64 characters")
	decoded, err := base64.RawURLEncoding.DecodeString(id)
	require.NoError(t, err)
	require.Len(t, decoded, 6)
}
