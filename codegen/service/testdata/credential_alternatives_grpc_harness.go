package testdata

import "strings"

// CredentialAlternativesGRPCHarness exercises metadata extraction and callback
// ordering over a real gRPC connection, with direct codec error controls.
func CredentialAlternativesGRPCHarness() string {
	return strings.ReplaceAll(credentialAlternativesGRPCHarness, "IMPLEMENTATION", credentialAlternativesImplementation)
}

const credentialAlternativesGRPCHarness = `package alternatives_test

import (
	"context"
	"errors"
	"net"
	"reflect"
	"testing"
 "time"

	svc "example.com/alternatives/gen/alternatives"
	pb "example.com/alternatives/gen/grpc/alternatives/pb"
	server "example.com/alternatives/gen/grpc/alternatives/server"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
)

IMPLEMENTATION

func TestCredentialAlternatives(t *testing.T) {
	decoders := map[string]func(context.Context, any, metadata.MD) (any, error){
		"Key":           server.DecodeKeyRequest,
		"And":           server.DecodeAndRequest,
		"Single":        server.DecodeSingleRequest,
		"RequiredToken": server.DecodeRequiredTokenRequest,
		"Anonymous":     server.DecodeAnonymousRequest,
	}
	for _, tc := range []struct {
		method, header, key, decodeError, auth string
		context                                any
		success, tokenPresent                  bool
	}{
		{"Key", "Basic dXNlcjpwYXNz", "key", "", "JK", "K", true, false},
		{"Key", "bAsIc dXNlcjpwYXNz", "key", "", "JK", "K", true, false},
		{"Key", "Bearer good", "key", "", "J", "J", true, true},
		{"Key", "Bearer wrong", "key", "", "JK", "K", true, true},
		{"And", "Bearer good", "key", "", "JK", "JK", true, true},
		{"And", "Bearer wrong", "key", "", "J", nil, false, true},
		{"And", "Basic value", "key", "decode_payload", "", nil, false, false},
		{"Single", "Basic value", "", "decode_payload", "", nil, false, false},
		{"RequiredToken", "Basic value", "key", "decode_payload", "", nil, false, false},
		{"RequiredToken", "", "key", "missing_field", "", nil, false, false},
		{"Anonymous", "Bearer ", "", "decode_payload", "", nil, false, false},
		{"Anonymous", "Basic value", "", "", "J", nil, true, false},
		{"Key", "Bearer", "key", "decode_payload", "", nil, false, false},
		{"Key", "Bearer ", "key", "decode_payload", "", nil, false, false},
		{"Key", "Bearer a b", "key", "decode_payload", "", nil, false, false},
		{"Key", "Bearer UPPER", "key", "invalid_pattern", "", nil, false, false},
		{"Key", "raw-token", "key", "decode_payload", "", nil, false, false},
	} {
		t.Run(tc.method+"/"+tc.header, func(t *testing.T) {
			md := metadata.MD{}
			if tc.header != "" {
				md.Set("authorization", tc.header)
			}
			if tc.key != "" {
				md.Set("x-key", tc.key)
			}
			payload, err := decoders[tc.method](context.Background(), nil, md)
			if tc.decodeError != "" {
				var detail *loom.ServiceError
				require.ErrorAs(t, err, &detail)
				require.Equal(t, tc.decodeError, detail.Name)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.tokenPresent, !reflect.ValueOf(payload).Elem().FieldByName("Token").IsNil())
			}
			impl := &implementation{}
			listener := bufconn.Listen(1 << 20)
			gs := grpc.NewServer()
			pb.RegisterAlternativesServer(gs, server.New(svc.NewEndpoints(impl), nil))
			done := make(chan error, 1)
			go func() {
				done <- gs.Serve(listener)
			}()
			conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return listener.Dial()
			}), grpc.WithTransportCredentials(insecure.NewCredentials()))
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, conn.Close())
				gs.Stop()
				require.NoError(t, <-done)
			})
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			err = conn.Invoke(metadata.NewOutgoingContext(ctx, md), "/"+pb.Alternatives_ServiceDesc.ServiceName+"/"+tc.method, new(emptypb.Empty), new(emptypb.Empty))
			if tc.success {
				require.NoError(t, err)
			} else if tc.decodeError != "" {
				require.Equal(t, codes.InvalidArgument, status.Code(err))
			} else {
				require.Equal(t, codes.Unknown, status.Code(err))
			}
			require.Equal(t, tc.success, impl.calls == 1)
			require.Equal(t, tc.auth, stringsJoin(impl.auth))
			require.Equal(t, tc.context, impl.context)
			if tc.success {
				require.Equal(t, payload, impl.payload)
			}
			for i, previous := range impl.seen {
				if tc.method == "And" && i > 0 {
					require.Equal(t, "J", previous)
				} else {
					require.Nil(t, previous, "failed alternative context leaked")
				}
			}
		})
	}
}
func stringsJoin(values []string) string {
	result := ""
	for _, value := range values {
		result += value
	}
	return result
}
`
