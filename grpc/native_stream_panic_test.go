package grpc_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	grpcnative "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// TestNativeWorkerPanicBoundary probes the proposed worker/caller rendezvous,
// not a production Loom implementation. Native user hooks may panic: relay while
// the operation caller exists; contain the value if cancellation already won.
func TestNativeWorkerPanicBoundary(t *testing.T) {
	for _, late := range []bool{false, true} {
		t.Run(map[bool]string{false: "relay", true: "late-containment"}[late], func(t *testing.T) {
			trigger := make(chan struct{})
			panicResult := make(chan any, 1)
			recovered := make(chan any, 1)
			workerDone := make(chan struct{})
			operationReady := make(chan context.CancelFunc, 1)
			stream, cancel := nativeContextProbe(t, func(_ any, server grpcnative.ServerStream) error {
				go func() {
					defer close(workerDone)
					defer func() {
						if value := recover(); value != nil {
							panicResult <- value
						}
					}()
					// A custom native codec/stream override can block before panicking.
					<-trigger
					panic("native hook panic")
				}()
				ctx, stop := context.WithCancel(server.Context())
				defer stop()
				operationReady <- stop
				select {
				case value := <-panicResult:
					panic(value)
				case <-ctx.Done():
					return status.FromContextError(ctx.Err()).Err()
				}
			}, grpcnative.StreamInterceptor(func(service any, stream grpcnative.ServerStream, _ *grpcnative.StreamServerInfo, next grpcnative.StreamHandler) (err error) {
				defer func() {
					if value := recover(); value != nil {
						recovered <- value
						err = status.Error(codes.Internal, "recovered native panic")
					}
				}()
				return next(service, stream)
			}))
			defer cancel()
			select {
			case stop := <-operationReady:
				if late {
					stop()
				} else {
					close(trigger)
				}
			case <-time.After(time.Second):
				t.Fatal("native worker operation was not dispatched")
			}
			err := stream.RecvMsg(new(emptypb.Empty))
			if late {
				require.Equal(t, codes.Canceled, status.Code(err))
				close(trigger)
			} else {
				require.Equal(t, codes.Internal, status.Code(err))
			}
			select {
			case <-workerDone:
			case <-time.After(time.Second):
				t.Fatal("native worker did not stop after its panic")
			}
			if late {
				select {
				case value := <-panicResult:
					require.Equal(t, "native hook panic", value)
				default:
					t.Error("late native panic was not contained")
				}
				require.Empty(t, recovered, "late panic cannot reach an already-returned interceptor")
			} else {
				select {
				case value := <-recovered:
					require.Equal(t, "native hook panic", value)
				default:
					t.Error("active native panic did not reach outer recovery")
				}
			}
		})
	}
}
