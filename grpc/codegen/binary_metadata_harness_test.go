package codegen

const binaryMetadataHarness = `package binarymetadata_test

import (
    "context"
    "net"
    "testing"
    "time"

    "github.com/stretchr/testify/require"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/grpc/test/bufconn"

    svc "example.com/binarymetadata/gen/binary"
    client "example.com/binarymetadata/gen/grpc/binary/client"
    pb "example.com/binarymetadata/gen/grpc/binary/pb"
    server "example.com/binarymetadata/gen/grpc/binary/server"
)

type implementation struct {
}

func TestBinaryRoundTrip(t *testing.T) {
    listener := bufconn.Listen(1 << 20)
    host := grpc.NewServer()
    pb.RegisterBinaryServer(host, server.New(svc.NewEndpoints(implementation{}), nil))
    done := make(chan error, 1)
    go func() {
        done <- host.Serve(listener)
    }()
    t.Cleanup(func() {
        host.Stop()
        require.NoError(t, <-done)
    })
    conn, err := grpc.NewClient("passthrough:///binary", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
        return listener.Dial()
    }), grpc.WithTransportCredentials(insecure.NewCredentials()))
    require.NoError(t, err)
    t.Cleanup(func() {
        require.NoError(t, conn.Close())
    })
    remote := svc.NewClient(client.NewClient(conn).Echo())
    for _, tc := range []struct {
        name string
        raw []byte
    }{
        {"printable", []byte("hello")},
        {"binary", []byte{0, 255, 1}},
        {"newline", []byte("a b\n")},
    } {
        t.Run(tc.name, func(t *testing.T) {
            ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
            defer cancel()
            input := &svc.Value{Body: "body", Raw: tc.raw}
            result, err := remote.Echo(ctx, input)
            require.NoError(t, err)
            require.Equal(t, input, result)
        })
    }
}
func (implementation) Echo(ctx context.Context, value *svc.Value) (*svc.Value, error) {
    return value, nil
}

`
