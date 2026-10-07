package generator

import (
	"strings"
	"testing"
)

func TestGRPCInitialSendPreservesCompletion(t *testing.T) {
	for _, kind := range []string{"client", "bidirectional"} {
		for _, payload := range []bool{false, true} {
			name := kind + "/without-payload"
			if payload {
				name = kind + "/with-payload"
			}
			t.Run(name, func(t *testing.T) {
				receive := "Recv"
				if kind == "client" {
					receive = "CloseAndRecv"
				}
				value, hasPayload, builderCheck := "nil", "false", ""
				if payload {
					value, hasPayload = "&svc.Request{}", "true"
					builderCheck = `md := metadata.MD{}
       request, err := client.EncodeExchangeRequest(context.Background(), payload, &md)
       require.NoError(t, err)
       raw, err := client.BuildExchangeFunc(pb.NewStreamerClient(conn))(context.Background(), request)
       require.Nil(t, raw)
       require.ErrorIs(t, err, tc.sendErr)`
				}
				harness := strings.NewReplacer("RECEIVE", receive, "PAYLOAD_VALUE", value, "HAS_PAYLOAD", hasPayload, "BUILDER_CHECK", builderCheck).Replace(grpcInitialSendHarness)
				if !payload {
					harness = strings.ReplaceAll(harness, `"google.golang.org/grpc/metadata"`, "")
				}
				runDesignHarness(t, "example.com/initialsend", grpcStreamingDSL("streamer", kind, payload), harness)
			})
		}
	}
}

const grpcInitialSendHarness = `package initialsend

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"example.com/initialsend/gen/grpc/streamer/client"
	pb "example.com/initialsend/gen/grpc/streamer/pb"
	svc "example.com/initialsend/gen/streamer"
	"github.com/stretchr/testify/require"
	std "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type initialStream struct {
	std.ClientStream
	sendErr    error
	receiveErr error
	sends      int
	receives   int
}

func (s *initialStream) SendMsg(any) error {
	s.sends++
	return s.sendErr
}

func (s *initialStream) CloseSend() error {
	return nil
}

func (s *initialStream) RecvMsg(v any) error {
	s.receives++
	if s.receiveErr != nil {
		return s.receiveErr
	}
	count := int64(7)
	v.(*pb.ExchangeResponse).Count = &count
	return nil
}

func TestOpeningSendAndCompletion(t *testing.T) {
	denied := status.Error(codes.PermissionDenied, "denied")
	cases := []struct {
		name       string
		sendErr    error
		receiveErr error
	}{
		{"EOF then rejection", io.EOF, denied},
		{"wrapped EOF then rejection", fmt.Errorf("send: %w", io.EOF), denied},
		{"successful send and result", nil, nil},
		{"EOF with result", io.EOF, nil},
		{"empty stream", io.EOF, io.EOF},
		{"canceled stream", io.EOF, status.Error(codes.Canceled, "canceled")},
		{"local send failure", errors.New("send failed"), denied},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stream := &initialStream{sendErr: tc.sendErr, receiveErr: tc.receiveErr}
			conn, err := std.NewClient("passthrough:///unused", std.WithTransportCredentials(insecure.NewCredentials()),
				std.WithStreamInterceptor(func(context.Context, *std.StreamDesc, *std.ClientConn, string, std.Streamer, ...std.CallOption) (std.ClientStream, error) {
					return stream, nil
				}))
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, conn.Close())
			})
			var payload any = PAYLOAD_VALUE
			result, err := client.NewClient(conn).Exchange()(context.Background(), payload)
			if HAS_PAYLOAD {
				require.Equal(t, 1, stream.sends)
			} else {
				require.Zero(t, stream.sends)
			}
			require.Zero(t, stream.receives, "opening must not consume the final status")
			if HAS_PAYLOAD && tc.sendErr != nil && !errors.Is(tc.sendErr, io.EOF) {
				require.ErrorContains(t, err, "send failed")
				require.Nil(t, result)
				BUILDER_CHECK
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			reply, err := result.(svc.ExchangeClientStream).RECEIVE()
			require.Equal(t, 1, stream.receives)
			if tc.receiveErr != nil {
				require.ErrorIs(t, err, tc.receiveErr)
				require.Equal(t, status.Code(tc.receiveErr), status.Code(err))
				require.Nil(t, reply)
			} else {
				require.NoError(t, err)
				require.NotNil(t, reply.Count)
				require.Equal(t, 7, *reply.Count)
			}
		})
	}
}
`
