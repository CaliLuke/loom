package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	servicetestdata "github.com/CaliLuke/loom/codegen/service/testdata"
)

func TestRecursiveProjectionGRPC(t *testing.T) {
	for _, order := range []struct {
		name       string
		unionFirst bool
	}{{"array_first", false}, {"union_first", true}} {
		t.Run(order.name, func(t *testing.T) {
			root := RunGRPCDSL(t, func() {
				servicetestdata.RecursiveProjectionTransportDSL("grpc", order.unionFirst)
			})
			dir := t.TempDir()
			renderGRPCResponseContractModule(t, dir, "example.com/projection", root)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "projection_test.go"), []byte(recursiveProjectionGRPCHarness), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "service_test.go"), []byte(servicetestdata.RecursiveProjectionServiceHarness), 0o600))
			runGRPCGoCommand(t, dir, "mod", "tidy")
			runGRPCGoCommand(t, dir, "vet", "./gen/...")
			runGRPCGoCommand(t, dir, "test", "-race", "-count=1", ".", "./gen/...")
		})
	}
}

const recursiveProjectionGRPCHarness = `package projection_test

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	pb "example.com/projection/gen/grpc/projection/pb"
	server "example.com/projection/gen/grpc/projection/server"
	svc "example.com/projection/gen/projection"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func (s implementation) Watch(ctx context.Context, stream svc.WatchServerStream) error {
	value, err := s.Show(ctx)
	if err != nil {
		return err
	}
	return stream.SendWithContext(ctx, value)
}

func TestSelectedWire(t *testing.T) {
	var prior, priorStream string
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, secret := range []string{"FIRST_SECRET", "SECOND_SECRET"} {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		srv := grpc.NewServer()
		t.Cleanup(srv.Stop)
		pb.RegisterProjectionServer(srv, server.New(svc.NewEndpoints(implementation{secret}), nil, nil))
		done := make(chan error, 1)
		go func() {
			done <- srv.Serve(listener)
		}()
		conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			srv.Stop()
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := conn.Close(); err != nil {
				t.Error(err)
			}
		})
		result, err := pb.NewProjectionClient(conn).Show(ctx, &pb.ShowRequest{})
		if err != nil {
			t.Fatal(err)
		}
		wire, err := proto.MarshalOptions{Deterministic: true}.Marshal(result)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(wire), secret) {
			t.Errorf("excluded secret in protobuf: %v", result)
		}
		if prior != "" && prior != string(wire) {
			t.Errorf("excluded field changed protobuf bytes")
		}
		prior = string(wire)
		jsonWire, err := protojson.Marshal(result)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(jsonWire), "allowed") {
			t.Errorf("explicit full view lost field: %s", jsonWire)
		}
		stream, err := pb.NewProjectionClient(conn).Watch(ctx, &pb.WatchRequest{})
		if err != nil {
			t.Fatal(err)
		}
		streamed, err := stream.Recv()
		if err != nil {
			t.Fatal(err)
		}
		streamWire, err := proto.MarshalOptions{Deterministic: true}.Marshal(streamed)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(streamWire), secret) || !strings.Contains(string(streamWire), "allowed") {
			t.Fatalf("invalid selected stream wire: %v", streamed)
		}
		if priorStream != "" && priorStream != string(streamWire) {
			t.Errorf("excluded field changed streamed protobuf bytes")
		}
		priorStream = string(streamWire)
		srv.Stop()
		if err := <-done; err != nil {
			t.Error(err)
		}
	}
}
`
