package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

func TestStreamAdmissionAlternatives(t *testing.T) {
	root := RunGRPCDSL(t, testdata.StreamAdmissionAlternativesDSL)
	services := CreateGRPCServices(root)
	data := services.Get("up")
	for _, endpoint := range data.Endpoints {
		want := endpoint.Method.Name != "MixedFirst" && endpoint.Method.Name != "MixedLast" && endpoint.Method.Name != "AuthenticateMetaFirst"
		require.Equal(t, want, endpoint.Request.Admission != nil, endpoint.Method.Name)
	}
	require.NotEqual(t, "AuthenticateMetaFirst", data.Endpoint("MetaFirst").Method.Authentication.MethodName)
	dir := t.TempDir()
	renderGRPCResponseContractModule(t, dir, "example.com/csauth", root)
	var methods strings.Builder
	for _, name := range []string{"MetaFirst", "MetaLast", "MixedFirst", "MixedLast", "And", "Anonymous", "Named", "Bearer"} {
		fmt.Fprintf(&methods, "func(s *service)%s(ctx context.Context,p *up.%sPayload,stream up.%sServerStream)error {\n s.invoked.Add(1)\n winner,_:=ctx.Value(principal{}).(string)\n return stream.SendAndClose(winner)\n}\n", name, name, name)
	}
	harness := streamAdmissionAlternativesHarness + methods.String()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "alternatives_test.go"), []byte(harness), 0o600))
	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "test", ".", "-race", "-count=1", "-v")
}

const streamAdmissionAlternativesHarness = `package contracttest

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	pb "example.com/csauth/gen/grpc/up/pb"
	upsvr "example.com/csauth/gen/grpc/up/server"
	"example.com/csauth/gen/up"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type principal struct{}
type failedBranch struct{}
type service struct{ auth, invoked, chained atomic.Int32 }

func (s *service) APIKeyAuth(ctx context.Context, key string, scheme *security.APIKeyScheme) (context.Context, error) {
	s.auth.Add(1)
	if scheme.Name == "secondary" && ctx.Value(principal{}) == "primary" {
		s.chained.Add(1)
	}
	if ctx.Value(failedBranch{}) != nil {
		return ctx, loom.Fault("failed branch context leaked")
	}
	if key != "secret" {
		return context.WithValue(ctx, failedBranch{}, true), loom.PermanentError("unauthorized", "bad key")
	}
	return context.WithValue(ctx, principal{}, scheme.Name), nil
}
func (s *service) JWTAuth(ctx context.Context, token string, scheme *security.JWTScheme) (context.Context, error) {
	return s.APIKeyAuth(ctx, token, &security.APIKeyScheme{Name: scheme.Name})
}
func (s *service) AuthenticateMetaFirst(context.Context) (string, error) {
	return "control", nil
}
func TestAlternatives(t *testing.T) {
	for _, tc := range []struct {
		name, method, first, second, body, winner, authorization string
		frame, wait                                              bool
		calls, chained                                           int32
	}{
		{name: "metadata first winner", method: "MetaFirst", first: "secret", second: "secret", frame: true, winner: "primary", calls: 2},
		{name: "metadata reversed winner", method: "MetaLast", first: "secret", second: "secret", frame: true, winner: "secondary", calls: 2},
		{name: "OR failed context resets", method: "MetaFirst", second: "secret", frame: true, winner: "secondary", calls: 4},
		{name: "metadata rejected before receive", method: "MetaFirst", calls: 2},
		{name: "AND rejects missing component", method: "And", first: "secret", calls: 2, chained: 1},
		{name: "AND accepts both", method: "And", first: "secret", second: "secret", frame: true, winner: "secondary", calls: 4, chained: 2},
		{name: "anonymous admits", method: "Anonymous", frame: true, calls: 2},
		{name: "mixed message first waits", method: "MixedFirst", first: "secret", wait: true},
		{name: "mixed metadata first waits", method: "MixedLast", first: "secret", wait: true},
		{name: "mixed first respects declaration", method: "MixedFirst", first: "secret", body: "secret", frame: true, winner: "secondary", calls: 1},
		{name: "mixed last respects declaration", method: "MixedLast", first: "secret", body: "secret", frame: true, winner: "primary", calls: 1},
		{name: "Bearer accepted", method: "Bearer", authorization: "Bearer secret", frame: true, winner: "jwt", calls: 2},
		{name: "Bearer malformed before receive", method: "Bearer", authorization: "Bearer ", calls: 0},
		{name: "foreign scheme permits alternative", method: "Bearer", authorization: "Basic YTpi", first: "secret", frame: true, winner: "primary", calls: 4},
		{name: "absent Bearer permits alternative", method: "Bearer", first: "secret", frame: true, winner: "primary", calls: 4},
		{name: "named metadata validation precedes authentication", method: "Named", first: "x"},
		{name: "named metadata conversion", method: "Named", first: "secret", frame: true, winner: "primary", calls: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := new(service)
			lis := bufconn.Listen(1 << 20)
			server := grpc.NewServer()
			pb.RegisterUpServer(server, upsvr.New(up.NewEndpoints(svc), nil, nil))
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
			ctx = metadata.AppendToOutgoingContext(ctx, "x-key", tc.first, "y-key", tc.second)
			if tc.authorization != "" {
				ctx = metadata.AppendToOutgoingContext(ctx, "authorization", tc.authorization)
			}
			descriptor := pb.File_loomgen_csauth_up_proto.Services().Get(0)
			method := descriptor.Methods().ByName(protoreflect.Name(tc.method))
			stream, err := conn.NewStream(ctx, &grpc.StreamDesc{ClientStreams: true}, "/"+string(descriptor.FullName())+"/"+tc.method)
			if err != nil {
				t.Fatal(err)
			}
			if tc.frame {
				envelope := dynamicpb.NewMessage(method.Input())
				initial := method.Input().Fields().ByName("initial_payload")
				payload := dynamicpb.NewMessage(initial.Message())
				payload.Set(initial.Message().Fields().ByName("name"), protoreflect.ValueOfString("upload"))
				if tc.body != "" {
					payload.Set(initial.Message().Fields().ByName("second"), protoreflect.ValueOfString(tc.body))
				}
				envelope.Set(initial, protoreflect.ValueOfMessage(payload))
				if err := stream.SendMsg(envelope); err != nil {
					t.Fatal(err)
				}
			}
			response := dynamicpb.NewMessage(method.Output())
			err = stream.RecvMsg(response)
			if (status.Code(err) == codes.DeadlineExceeded) != tc.wait {
				t.Errorf("unexpected wait status: %v", err)
			}
			if tc.frame {
				if err != nil {
					t.Fatalf("valid complete request: %v", err)
				}
				if got := response.Get(method.Output().Fields().Get(0)).String(); got != tc.winner {
					t.Errorf("principal=%q want%q", got, tc.winner)
				}
				if svc.invoked.Load() != 1 {
					t.Errorf("service calls=%d", svc.invoked.Load())
				}
			} else if svc.invoked.Load() != 0 {
				t.Error("service invoked before complete input")
			}
			if svc.chained.Load() != tc.chained {
				t.Errorf("AND contexts=%d want%d", svc.chained.Load(), tc.chained)
			}
			if svc.auth.Load() != tc.calls {
				t.Errorf("auth calls=%d want%d", svc.auth.Load(), tc.calls)
			}
		})
	}
}
`

func TestStreamAdmissionBasicAvailability(t *testing.T) {
	for _, tc := range []struct {
		name               string
		username, password bool
	}{
		{"both", true, true}, {"username only", true, false}, {"password only", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := RunGRPCDSL(t, func() {
				basic := BasicAuthSecurity("basic")
				Service("basic", func() {
					Method("upload", func() {
						Security(basic)
						Payload(func() {
							UsernameField(1, "user", String)
							PasswordField(2, "pass", String)
							Field(3, "name", String)
						})
						StreamingPayload(String)
						Result(String)
						GRPC(func() {
							Metadata(func() {
								if tc.username {
									Attribute("user")
								}
								if tc.password {
									Attribute("pass")
								}
							})
							Message(func() {
								Attribute("name")
								if !tc.username {
									Attribute("user")
								}
								if !tc.password {
									Attribute("pass")
								}
							})
						})
					})
				})
			})
			endpoint := CreateGRPCServices(root).Get("basic").Endpoint("upload")
			require.Equal(t, tc.username && tc.password, endpoint.Request.Admission != nil)
		})
	}
}
