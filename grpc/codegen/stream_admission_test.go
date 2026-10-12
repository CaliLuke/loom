package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

func TestStreamAdmissionGenerated(t *testing.T) {
	for _, tc := range []struct {
		name                      string
		dsl                       func()
		bidirectional, authorized bool
	}{
		{"client", testdata.StreamAdmissionDSL, false, false},
		{"bidirectional", testdata.BidirectionalAdmissionDSL, true, false},
		{"authorized", testdata.AuthorizedAdmissionDSL, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := RunGRPCDSL(t, tc.dsl)
			dir := t.TempDir()
			renderGRPCResponseContractModule(t, dir, "example.com/csauth", root)
			harness := streamAdmissionWireTest
			if tc.bidirectional {
				harness = strings.ReplaceAll(harness, "stream.SendAndClose(\"ok\")", "stream.Send(\"ok\")")
			}
			if tc.authorized {
				harness = strings.ReplaceAll(harness, "return up.NewEndpoints(s)", streamAdmissionMiddlewareBinding)
				require.Contains(t, harness, `{name: "wrong-without-frame"`)
				harness = strings.Replace(harness, `{name: "wrong-without-frame"`, streamAdmissionMiddlewareCases+`{name: "wrong-without-frame"`, 1)
				harness += `
func(s *service)AuthorizeUploadWrite(ctx context.Context,p *up.UploadRef)error {
 s.access.Add(1)
 if p.Name!="upload" || ctx.Value(principal{})!="alice" {
  return loom.Fault("incomplete access input")
 }
 return nil
}
`
			}
			require.NoError(t, os.WriteFile(filepath.Join(dir, "admission_test.go"), []byte(harness), 0o600))
			runGRPCGoCommand(t, dir, "mod", "tidy")
			runGRPCGoCommand(t, dir, "test", ".", "-race", "-count=1", "-v")
		})
	}
}

const streamAdmissionWireTest = `package contracttest

import (
	"context"
	"errors"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	pb "example.com/csauth/gen/grpc/up/pb"
	upsvr "example.com/csauth/gen/grpc/up/server"
	"example.com/csauth/gen/up"
	loomgrpc "github.com/CaliLuke/loom/grpc"
	loompb "github.com/CaliLuke/loom/grpc/pb"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type principal struct{}
type service struct {
	auth, invoked, access       atomic.Int32
	mode                        string
	middleware                  atomic.Int32
	replace, revoke, nilContext bool
}

func (s *service) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	count := s.auth.Add(1)
	if key != "secret" || s.revoke && count > 1 {
		return ctx, loom.PermanentError("unauthorized", "bad key")
	}
	if s.nilContext {
		return nil, nil
	}
	if s.replace {
		ctx = context.Background()
	}
	return context.WithValue(ctx, principal{}, "alice"), nil
}
func (s *service) Upload(ctx context.Context, p *up.UploadPayload, stream up.UploadServerStream) error {
	s.invoked.Add(1)
	if ctx.Value(principal{}) != "alice" || p.Name != "upload" {
		return loom.Fault("complete payload or winning context lost")
	}
	return stream.SendAndClose("ok")
}
func endpointsFor(s *service) *up.Endpoints {
	return up.NewEndpoints(s)
}

type observedStream struct {
	grpc.ServerStream
	receiving chan struct{}
}

func (s *observedStream) RecvMsg(message any) error {
	select {
	case s.receiving <- struct{}{}:
	default:
	}
	return s.ServerStream.RecvMsg(message)
}

func TestAdmission(t *testing.T) {
	for _, tc := range []struct {
		name, key, frame, problem, mode   string
		access, middleware                int32
		code                              codes.Code
		wait, replace, revoke, nilContext bool
		rejectBeforeSend                 bool
		auth, service                     int32
		received                          bool
	}{
		{name: "wrong-without-frame", code: codes.Unknown, problem: "unauthorized", key: "wrong", auth: 1},
		{name: "missing-without-frame", code: codes.InvalidArgument, problem: "missing_field"},
		{name: "valid-without-frame", key: "secret", wait: true, auth: 1, received: true},
		{name: "replacement-context-without-frame", key: "secret", wait: true, replace: true, auth: 1, received: true},
		{name: "wrong-with-frame", code: codes.Unknown, problem: "unauthorized", key: "wrong", frame: "valid", auth: 1},
		{name: "rejection-before-send", code: codes.Unknown, problem: "unauthorized", key: "wrong", frame: "valid", auth: 1, rejectBeforeSend: true},
		{name: "valid-with-frame", key: "secret", frame: "valid", auth: 2, service: 1, received: true},
		{name: "replacement-context-with-frame", key: "secret", frame: "valid", replace: true, auth: 2, service: 1, received: true},
		{name: "revocation-while-waiting", code: codes.Unknown, problem: "unauthorized", key: "secret", frame: "valid", revoke: true, auth: 2, received: true},
		{name: "nil-callback-context", code: codes.Internal, problem: "fault", key: "secret", nilContext: true, auth: 1},
		{name: "invalid-payload", code: codes.InvalidArgument, problem: "invalid_length", key: "secret", frame: "invalid", auth: 1, received: true},
		{name: "wrong-envelope-branch", code: codes.InvalidArgument, problem: "missing_field", key: "secret", frame: "wrong-kind", auth: 1, received: true},
		{name: "missing-initial-envelope", code: codes.InvalidArgument, problem: "missing_field", key: "secret", frame: "eof", auth: 1, received: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := &service{mode: tc.mode, replace: tc.replace, revoke: tc.revoke, nilContext: tc.nilContext}
			receiving := make(chan struct{}, 1)
			lis := bufconn.Listen(1 << 20)
			server := grpc.NewServer(grpc.StreamInterceptor(func(srv any, ss grpc.ServerStream, _ *grpc.StreamServerInfo, h grpc.StreamHandler) error {
				return h(srv, &observedStream{ss, receiving})
			}))
			pb.RegisterUpServer(server, upsvr.New(endpointsFor(svc), nil))
			done := make(chan error, 1)
			go func() {
				done <- server.Serve(lis)
			}()
			t.Cleanup(func() {
				server.Stop()
				if err := <-done; err != nil {
					t.Error(err)
				}
			})
			conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return lis.Dial()
			}), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := conn.Close(); err != nil {
					t.Error(err)
				}
			})
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if tc.key != "" {
				ctx = metadata.AppendToOutgoingContext(ctx, "authorization", tc.key)
			}
			stream, err := pb.NewUpClient(conn).Upload(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if tc.rejectBeforeSend {
				if _, err := stream.Header(); err != nil {
					t.Fatal(err)
				}
			}
			if tc.frame != "" {
				if tc.frame == "eof" {
					if err := stream.CloseSend(); err != nil {
						t.Fatal(err)
					}
				} else {
					name := "upload"
					if tc.frame == "invalid" {
						name = ""
					}
					frame := &pb.UploadStreamingRequest{Body: &pb.UploadStreamingRequest_InitialPayload{InitialPayload: &pb.UploadRequest{Name: &name}}}
					if tc.frame == "wrong-kind" {
						frame.Body = nil
					}
					// A server rejection may arrive before the first send. gRPC reports
					// EOF here; RecvMsg below still owns the final status assertions.
					if err := stream.Send(frame); err != nil && !errors.Is(err, io.EOF) {
						t.Fatal(err)
					}
				}
			}
			err = stream.RecvMsg(new(pb.UploadResponse))
			waited := status.Code(err) == codes.DeadlineExceeded
			if waited != tc.wait {
				t.Errorf("waited=%v want %v result=%v", waited, tc.wait, err)
			}
			if tc.service == 1 && err != nil {
				t.Errorf("successful admitted request: %v", err)
			}
			if tc.service == 0 && err == nil {
				t.Error("rejected/incomplete request succeeded")
			}
			if n := svc.auth.Load(); n != tc.auth {
				t.Errorf("auth calls=%d want%d", n, tc.auth)
			}
			if n := svc.invoked.Load(); n != tc.service {
				t.Errorf("service calls=%d want%d", n, tc.service)
			}
			if tc.mode == "" && tc.service == 0 && svc.access.Load() != 0 {
				t.Error("application access ran before complete successful authentication")
			}
			if tc.mode != "" && (svc.access.Load() != tc.access || svc.middleware.Load() != tc.middleware) {
				t.Errorf("access=%d middleware=%d want%d,%d", svc.access.Load(), svc.middleware.Load(), tc.access, tc.middleware)
			}
			select {
			case <-receiving:
				if !tc.received {
					t.Error("server entered Recv before rejecting admission")
				}
			default:
				if tc.received {
					t.Error("valid admission did not enter initial Recv")
				}
			}
			if tc.problem != "" {
				if status.Code(err) != tc.code {
					t.Errorf("status=%s want%s", status.Code(err), tc.code)
				}
				detail, ok := loomgrpc.DecodeError(err).(*loompb.ErrorResponse)
				if !ok {
					t.Errorf("missing Loom problem: %v", err)
				} else if detail.Name != tc.problem || detail.Msg == "" {
					t.Errorf("problem=%+v want%s with public detail", detail, tc.problem)
				}
			}
			if err != nil {
				t.Logf("status=%s detail=%s", status.Code(err), status.Convert(err).Message())
			}
		})
	}
}
`

func TestStreamAdmissionControls(t *testing.T) {
	root := RunGRPCDSL(t, testdata.StreamAdmissionControlsDSL)
	data := CreateGRPCServices(root).Get("controls")
	for _, endpoint := range data.Endpoints {
		require.Nil(t, endpoint.Request.Admission, endpoint.Method.Name)
	}
}

const streamAdmissionMiddlewareBinding = `
 e:=up.NewEndpoints(s,s)
 if s.mode!="" {
  e.Use(func(next loom.Endpoint)loom.Endpoint{
   return func(ctx context.Context,req any)(any,error){
    s.middleware.Add(1)
    p:=req.(*up.UploadEndpointInput).Payload
    if p.Name!="upload" || ctx.Value(principal{})!="alice"{
     return nil,loom.Fault("incomplete middleware input")
    }
    switch s.mode {
    case "stop":return nil,loom.PermanentError("blocked","middleware stopped")
    case "key":p.Key="wrong"
    case "name":p.Name=""
    }
    return next(ctx,req)
   }
  })
 }
 return e
`

const streamAdmissionMiddlewareCases = `
 {name:"middleware short circuit remains protected",key:"secret",frame:"valid",mode:"stop",code:codes.Unknown,problem:"blocked",auth:2,access:1,middleware:1,received:true},
 {name:"middleware credential mutation rechecked",key:"secret",frame:"valid",mode:"key",code:codes.Unknown,problem:"unauthorized",auth:3,access:1,middleware:1,received:true},
 {name:"middleware payload mutation revalidated",key:"secret",frame:"valid",mode:"name",code:codes.InvalidArgument,problem:"invalid_length",auth:2,access:1,middleware:1,received:true},
`
