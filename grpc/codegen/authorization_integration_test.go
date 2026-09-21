package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestAuthorizationGRPCIntegration(t *testing.T) {
	root := RunGRPCDSL(t, func() {
		ref := Type("ResourceRef", func() {
			Attribute("id", String)
			Required("id")
		})
		event := Type("Event", func() {
			Field(1, "id", String)
			Required("id")
		})
		access := Authorization("resource.edit", ref)
		Service("resources", func() {
			StrictAuthorization()
			Method("watch", func() {
				Payload(func() {
					Field(1, "id", String)
					Required("id")
				})
				Authorize(access, func() {
					Bind("id", "id")
				})
				StreamingResult(event)
				Error("forbidden")
				GRPC(func() {
					Response(CodeOK)
					Response("forbidden", CodePermissionDenied)
				})
			})
			Method("edit", func() {
				Payload(func() {
					Field(1, "id", String)
					Required("id")
				})
				Authorize(access, func() {
					Bind("id", "id")
				})
				Result(String)
				Error("forbidden")
				GRPC(func() {
					Response(CodeOK)
					Response("forbidden", CodePermissionDenied)
				})
			})
		})
	})
	dir := t.TempDir()
	renderGRPCResponseContractModule(t, dir, "example.com/accessgrpc", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "authorization_test.go"), []byte(authorizationGRPCHarness), 0o600))
	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "test", ".")
}

const authorizationGRPCHarness = `package integration

import (
	"context"
	resourcespb "example.com/accessgrpc/gen/grpc/resources/pb"
	resourcessvr "example.com/accessgrpc/gen/grpc/resources/server"
	resources "example.com/accessgrpc/gen/resources"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"testing"
)

type resourceService struct {
	calls int
}

func (s *resourceService) Edit(_ context.Context, p *resources.EditPayload) (string, error) {
	s.calls++
	return p.ID, nil
}

func (s *resourceService) Watch(_ context.Context, p *resources.WatchPayload, stream resources.WatchServerStream) error {
	s.calls++
	return stream.Send(&resources.Event{ID: p.ID})
}

type testStream struct {
	grpc.ServerStream
	sent int
}
func (*testStream) Context() context.Context {
	return context.Background()
}
func (s *testStream) Send(*resourcespb.WatchResponse) error {
	s.sent++
	return nil
}

type accessEvaluator struct {
}

func (*accessEvaluator) AuthorizeResourceEdit(_ context.Context, r *resources.ResourceRef) error {
	if r.ID != "allowed" {
		return loom.PermanentError("forbidden", "Access denied")
	}
	return nil
}
func TestGeneratedGRPCStreamAccess(t *testing.T) {
	s := &resourceService{}
	e := resources.NewEndpoints(s, &accessEvaluator{})
	server := resourcessvr.New(e, nil, nil)
	stream := &testStream{}
	err := server.Watch(&resourcespb.WatchRequest{Id: "denied"}, stream)
	require.Equal(t, codes.PermissionDenied, status.Code(err))
	require.Zero(t, s.calls)
	require.Zero(t, stream.sent)
	for _, payload := range []any{nil, (*resources.WatchEndpointInput)(nil), &resources.WatchEndpointInput{}} {
		_, err = e.Watch(context.Background(), payload)
		require.Error(t, err)
	}
	require.Zero(t, s.calls)
	err = server.Watch(&resourcespb.WatchRequest{Id: "allowed"}, stream)
	require.NoError(t, err)
	require.Equal(t, 1, s.calls)
	require.Equal(t, 1, stream.sent)
}

func TestGeneratedGRPCAccess(t *testing.T) {
	s := &resourceService{}
	server := resourcessvr.New(resources.NewEndpoints(s, &accessEvaluator{}), nil, nil)
	_, err := server.Edit(context.Background(), &resourcespb.EditRequest{Id: "denied"})
	require.Equal(t, codes.PermissionDenied, status.Code(err))
	require.Zero(t, s.calls)
	_, err = server.Edit(context.Background(), &resourcespb.EditRequest{Id: "allowed"})
	require.NoError(t, err)
	require.Equal(t, 1, s.calls)
}
`
