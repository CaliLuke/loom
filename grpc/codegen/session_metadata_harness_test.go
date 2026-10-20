package codegen

const sessionMetadataHarness = `package scope_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pb "example.com/scope/gen/grpc/scope/pb"
	server "example.com/scope/gen/grpc/scope/server"
	svc "example.com/scope/gen/scope"
	loomgrpc "github.com/CaliLuke/loom/grpc"
	loompb "github.com/CaliLuke/loom/grpc/pb"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

type metadataImplementation struct {
	endpointImplementation
	observed chan string
}

func TestSelectedMetadata(t *testing.T) {
	for _, tc := range []struct {
		name      string
		values    metadata.MD
		callbacks []string
		accepted  bool
	}{
		{"bearer", metadata.Pairs("authorization", "Bearer bearer"), []string{"jwt:bearer", "service"}, true},
		{"cookie", metadata.Pairs("x-session", "cookie"), []string{"jwt:", "key:cookie", "service"}, true},
		{"separate", metadata.Pairs("authorization", "Bearer wrong", "x-session", "cookie"), []string{"jwt:wrong", "key:cookie", "service"}, true},
		{"invalid bearer", metadata.Pairs("authorization", "Bearer invalid"), []string{"jwt:invalid", "key:"}, false},
		{"invalid cookie", metadata.Pairs("x-session", "invalid"), []string{"jwt:", "key:invalid"}, false},
		{"absent", metadata.MD{}, []string{"jwt:", "key:"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			impl := &metadataImplementation{observed: make(chan string, 3)}
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			host := grpc.NewServer()
			pb.RegisterScopeServer(host, server.New(svc.NewEndpoints(impl), nil))
			done := make(chan error, 1)
			go func() {
				done <- host.Serve(listener)
			}()
			t.Cleanup(func() {
				host.Stop()
				require.NoError(t, <-done)
			})
			conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, conn.Close())
			})
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_, err = pb.NewScopeClient(conn).Read(metadata.NewOutgoingContext(ctx, tc.values), &pb.ReadRequest{})
			if tc.accepted {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Equal(t, codes.Unknown, status.Code(err))
				detail, ok := loomgrpc.DecodeError(err).(*loompb.ErrorResponse)
				require.True(t, ok)
				require.Equal(t, "unauthorized", detail.Name)
				require.Equal(t, "invalid", detail.Msg)
			}
			for _, want := range tc.callbacks {
				select {
				case got := <-impl.observed:
					require.Equal(t, want, got)
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
			}
			require.Empty(t, impl.observed)
		})
	}
}

func (s *metadataImplementation) JWTAuth(ctx context.Context, token string, _ *security.JWTScheme) (context.Context, error) {
	s.observed <- "jwt:" + token
	if token == "bearer" {
		return ctx, nil
	}
	return ctx, loom.PermanentError("unauthorized", "invalid")
}

func (s *metadataImplementation) APIKeyAuth(ctx context.Context, token string, _ *security.APIKeyScheme) (context.Context, error) {
	s.observed <- "key:" + token
	if token == "cookie" {
		return ctx, nil
	}
	return ctx, loom.PermanentError("unauthorized", "invalid")
}

func (s *metadataImplementation) Read(context.Context, *svc.ReadPayload) error {
	s.observed <- "service"
	return nil
}
`
