package codegen

const binaryMetadataPeerHarness = `package binarymetadata_test

import (
    "context"
    "net"
    "testing"
    "time"

    "github.com/stretchr/testify/require"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/grpc/metadata"
    "google.golang.org/protobuf/reflect/protoreflect"

    svc "example.com/binarymetadata/gen/binary"
    client "example.com/binarymetadata/gen/grpc/binary/client"
    pb "example.com/binarymetadata/gen/grpc/binary/pb"
    server "example.com/binarymetadata/gen/grpc/binary/server"
)

type metadataPeer struct {
    pb.UnimplementedBinaryServer
    omit bool
}

const peerPlacement = "PLACEMENT"

func TestIndependentBinaryPeers(t *testing.T) {
    t.Run("protobuf-client", func(t *testing.T) {
        conn := binaryPeerConnection(t, server.New(svc.NewEndpoints(implementation{}), nil), nil)
        native := pb.NewBinaryClient(conn)
        for _, raw := range [][]byte{{0, 255, 1}, {}} {
            ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
            request := &pb.EchoRequest{Body: new("peer")}
            if field := request.ProtoReflect().Descriptor().Fields().ByName("raw"); field != nil {
                request.ProtoReflect().Set(field, protoreflect.ValueOfBytes(raw))
            }
            ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("x-raw-bin", string(raw)))
            var header, trailer metadata.MD
            result, err := native.Echo(ctx, request, grpc.Header(&header), grpc.Trailer(&trailer))
            cancel()
            require.NoError(t, err)
            require.Equal(t, "peer", result.GetBody())
            if peerPlacement == "request" {
                field := result.ProtoReflect().Descriptor().Fields().ByName("raw")
                require.NotNil(t, field)
                require.Equal(t, string(raw), string(result.ProtoReflect().Get(field).Bytes()))
            } else {
                observed := header
                if peerPlacement == "trailer" {
                    observed = trailer
                }
                require.Equal(t, []string{string(raw)}, observed.Get("x-raw-bin"))
                require.Empty(t, observed.Get("x-raw"))
            }
        }
    })
    t.Run("protobuf-server", func(t *testing.T) {
        observed := make(chan metadata.MD, 1)
        interceptor := func(ctx context.Context, request any, info *grpc.UnaryServerInfo, next grpc.UnaryHandler) (any, error) {
            incoming, ok := metadata.FromIncomingContext(ctx)
            if ok {
                observed <- incoming.Copy()
            }
            return next(ctx, request)
        }
        conn := binaryPeerConnection(t, metadataPeer{}, interceptor)
        remote := svc.NewClient(client.NewClient(conn).Echo())
        ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
        defer cancel()
        result, err := remote.Echo(ctx, &svc.Value{Body: "peer", Raw: []byte{0, 255, 1}})
        require.NoError(t, err)
        require.Equal(t, []byte{0, 255, 1}, result.Raw)
        select {
        case incoming := <-observed:
            if peerPlacement == "request" {
                require.Equal(t, []string{string([]byte{0, 255, 1})}, incoming.Get("x-raw-bin"))
                require.Empty(t, incoming.Get("x-raw"))
            }
        case <-ctx.Done():
            t.Fatal(ctx.Err())
        }
    })
    t.Run("required-absence", func(t *testing.T) {
        ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
        defer cancel()
        if peerPlacement == "request" {
            conn := binaryPeerConnection(t, server.New(svc.NewEndpoints(implementation{}), nil), nil)
            _, err := pb.NewBinaryClient(conn).Echo(ctx, &pb.EchoRequest{Body: new("peer")})
            require.ErrorContains(t, err, "x-raw-bin")
            return
        }
        conn := binaryPeerConnection(t, metadataPeer{omit: true}, nil)
        remote := svc.NewClient(client.NewClient(conn).Echo())
        _, err := remote.Echo(ctx, &svc.Value{Body: "peer", Raw: []byte{1}})
        require.ErrorContains(t, err, "x-raw-bin")
    })
}
func (p metadataPeer) Echo(ctx context.Context, request *pb.EchoRequest) (*pb.EchoResponse, error) {
    raw := []byte{0, 255, 1}
    response := &pb.EchoResponse{Body: new("peer")}
    if field := response.ProtoReflect().Descriptor().Fields().ByName("raw"); field != nil {
        response.ProtoReflect().Set(field, protoreflect.ValueOfBytes(raw))
    }
    if !p.omit {
        values := metadata.Pairs("x-raw-bin", string(raw))
        if err := grpc.SendHeader(ctx, values); err != nil {
            return nil, err
        }
        if err := grpc.SetTrailer(ctx, values); err != nil {
            return nil, err
        }
    }
    return response, nil
}

func binaryPeerConnection(t *testing.T, implementation pb.BinaryServer, interceptor grpc.UnaryServerInterceptor) *grpc.ClientConn {
    t.Helper()
    listener, err := net.Listen("tcp", "127.0.0.1:0")
    require.NoError(t, err)
    var options []grpc.ServerOption
    if interceptor != nil {
        options = append(options, grpc.UnaryInterceptor(interceptor))
    }
    host := grpc.NewServer(options...)
    pb.RegisterBinaryServer(host, implementation)
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
    return conn
}

`
