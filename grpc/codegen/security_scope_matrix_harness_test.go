package codegen

const grpcSecurityScopeMatrixHarness = `package scope_test

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	api "example.com/scopematrix/gen/api_scope"
	apipb "example.com/scopematrix/gen/grpc/api_scope/pb"
	apisvr "example.com/scopematrix/gen/grpc/api_scope/server"
	scopepb "example.com/scopematrix/gen/grpc/scope/pb"
	scopesvr "example.com/scopematrix/gen/grpc/scope/server"
	scope "example.com/scopematrix/gen/scope"
	loomgrpc "github.com/CaliLuke/loom/grpc"
	loompb "github.com/CaliLuke/loom/grpc/pb"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type counters struct {
	basic, jwt, key                                                      atomic.Int32
	apiCalls, inheritedCalls, overrideCalls, alternativeCalls, openCalls atomic.Int32
}

func (c *counters) reset() {
	c.basic.Store(0)
	c.jwt.Store(0)
	c.key.Store(0)
	c.apiCalls.Store(0)
	c.inheritedCalls.Store(0)
	c.overrideCalls.Store(0)
	c.alternativeCalls.Store(0)
	c.openCalls.Store(0)
}

type apiImplementation struct {
	api.Service
	counts *counters
}

type scopeImplementation struct {
	scope.Service
	counts *counters
}

func TestSecurityScopeMatrix(t *testing.T) {
	counts := &counters{}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	host := grpc.NewServer()
	apipb.RegisterAPIScopeServer(host, apisvr.New(api.NewEndpoints(&apiImplementation{counts: counts}), nil))
	scopepb.RegisterScopeServer(host, scopesvr.New(scope.NewEndpoints(&scopeImplementation{counts: counts}), nil))
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
	client := apipb.NewAPIScopeClient(conn)
	scopeClient := scopepb.NewScopeClient(conn)

	for _, tc := range []struct {
		name                                                            string
		call                                                            func(context.Context) error
		metadata                                                        metadata.MD
		accept                                                          bool
		wantBasic, wantJWT, wantKey                                     int
		wantAPI, wantInherited, wantOverride, wantAlternative, wantOpen int
	}{
		{
			name: "api inherited accepts basic",
			call: func(ctx context.Context) error {
				_, err := client.Read(ctx, &apipb.ReadRequest{})
				return err
			},
			metadata: metadata.Pairs("user", "user", "password", "pass"), accept: true,
			wantBasic: 1, wantAPI: 1,
		},
		{
			name: "api inherited rejects basic",
			call: func(ctx context.Context) error {
				_, err := client.Read(ctx, &apipb.ReadRequest{})
				return err
			},
			metadata:  metadata.Pairs("user", "bad", "password", "bad"),
			wantBasic: 1,
		},
		{
			name: "service inherited accepts jwt",
			call: func(ctx context.Context) error {
				_, err := scopeClient.Inherited(ctx, &scopepb.InheritedRequest{})
				return err
			},
			metadata: metadata.Pairs("authorization", "Bearer jwt-ok"), accept: true,
			wantJWT: 1, wantInherited: 1,
		},
		{
			name: "service inherited rejects jwt",
			call: func(ctx context.Context) error {
				_, err := scopeClient.Inherited(ctx, &scopepb.InheritedRequest{})
				return err
			},
			metadata: metadata.Pairs("authorization", "Bearer bad"),
			wantJWT:  1,
		},
		{
			name: "method override accepts api key",
			call: func(ctx context.Context) error {
				_, err := scopeClient.Override(ctx, &scopepb.OverrideRequest{})
				return err
			},
			metadata: metadata.Pairs("authorization", "key-ok"), accept: true,
			wantKey: 1, wantOverride: 1,
		},
		{
			name: "method override rejects api key",
			call: func(ctx context.Context) error {
				_, err := scopeClient.Override(ctx, &scopepb.OverrideRequest{})
				return err
			},
			metadata: metadata.Pairs("authorization", "bad"),
			wantKey:  1,
		},
		{
			name: "alternatives accept basic",
			call: func(ctx context.Context) error {
				_, err := scopeClient.Alternatives(ctx, &scopepb.AlternativesRequest{})
				return err
			},
			metadata: metadata.Pairs("user", "user", "password", "pass"), accept: true,
			wantBasic: 1, wantAlternative: 1,
		},
		{
			name: "alternatives accept jwt",
			call: func(ctx context.Context) error {
				_, err := scopeClient.Alternatives(ctx, &scopepb.AlternativesRequest{})
				return err
			},
			metadata: metadata.Pairs("authorization", "Bearer jwt-ok"), accept: true,
			wantBasic: 1, wantJWT: 1, wantAlternative: 1,
		},
		{
			name: "alternatives reject both",
			call: func(ctx context.Context) error {
				_, err := scopeClient.Alternatives(ctx, &scopepb.AlternativesRequest{})
				return err
			},
			metadata:  metadata.Pairs("authorization", "Bearer bad", "user", "bad", "password", "bad"),
			wantBasic: 1, wantJWT: 1,
		},
		{
			name: "no security ignores bad credentials",
			call: func(ctx context.Context) error {
				_, err := scopeClient.Open(ctx, &scopepb.OpenRequest{})
				return err
			},
			metadata: metadata.Pairs("authorization", "Bearer bad", "user", "bad", "password", "bad"), accept: true,
			wantOpen: 1,
		},
		{
			name: "no security accepts absent credentials",
			call: func(ctx context.Context) error {
				_, err := scopeClient.Open(ctx, &scopepb.OpenRequest{})
				return err
			},
			metadata: metadata.MD{}, accept: true,
			wantOpen: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			counts.reset()
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			err := tc.call(metadata.NewOutgoingContext(ctx, tc.metadata))
			if tc.accept {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Equal(t, codes.Unknown, status.Code(err))
				detail, ok := loomgrpc.DecodeError(err).(*loompb.ErrorResponse)
				require.True(t, ok)
				require.Equal(t, "unauthorized", detail.Name)
				require.Equal(t, "invalid credentials", detail.Msg)
			}
			require.Equal(t, int32(tc.wantBasic), counts.basic.Load())
			require.Equal(t, int32(tc.wantJWT), counts.jwt.Load())
			require.Equal(t, int32(tc.wantKey), counts.key.Load())
			require.Equal(t, int32(tc.wantAPI), counts.apiCalls.Load())
			require.Equal(t, int32(tc.wantInherited), counts.inheritedCalls.Load())
			require.Equal(t, int32(tc.wantOverride), counts.overrideCalls.Load())
			require.Equal(t, int32(tc.wantAlternative), counts.alternativeCalls.Load())
			require.Equal(t, int32(tc.wantOpen), counts.openCalls.Load())
		})
	}
}

func (s *apiImplementation) BasicAuth(ctx context.Context, user, pass string, _ *security.BasicScheme) (context.Context, error) {
	s.counts.basic.Add(1)
	if user == "user" && pass == "pass" {
		return ctx, nil
	}
	return ctx, loom.PermanentError("unauthorized", "invalid credentials")
}

func (s *apiImplementation) Read(context.Context, *api.ReadPayload) error {
	s.counts.apiCalls.Add(1)
	return nil
}

func (s *scopeImplementation) BasicAuth(ctx context.Context, user, pass string, _ *security.BasicScheme) (context.Context, error) {
	s.counts.basic.Add(1)
	if user == "user" && pass == "pass" {
		return ctx, nil
	}
	return ctx, loom.PermanentError("unauthorized", "invalid credentials")
}

func (s *scopeImplementation) JWTAuth(ctx context.Context, token string, _ *security.JWTScheme) (context.Context, error) {
	s.counts.jwt.Add(1)
	if token == "jwt-ok" {
		return ctx, nil
	}
	return ctx, loom.PermanentError("unauthorized", "invalid credentials")
}

func (s *scopeImplementation) APIKeyAuth(ctx context.Context, token string, _ *security.APIKeyScheme) (context.Context, error) {
	s.counts.key.Add(1)
	if token == "key-ok" {
		return ctx, nil
	}
	return ctx, loom.PermanentError("unauthorized", "invalid credentials")
}

func (s *scopeImplementation) Inherited(context.Context, *scope.InheritedPayload) error {
	s.counts.inheritedCalls.Add(1)
	return nil
}

func (s *scopeImplementation) Override(context.Context, *scope.OverridePayload) error {
	s.counts.overrideCalls.Add(1)
	return nil
}

func (s *scopeImplementation) Alternatives(context.Context, *scope.AlternativesPayload) error {
	s.counts.alternativeCalls.Add(1)
	return nil
}

func (s *scopeImplementation) Open(context.Context, *scope.OpenPayload) error {
	s.counts.openCalls.Add(1)
	return nil
}
`
