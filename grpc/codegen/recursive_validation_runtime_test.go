package codegen

const recursiveValidationRuntime = `package recursive_test

import (
 "context"
 "errors"
 "net"
 "strings"
 "sync/atomic"
 "testing"
 "time"

 client "example.com/recursivegrpc/gen/grpc/trees/client"
 pb "example.com/recursivegrpc/gen/grpc/trees/pb"
 server "example.com/recursivegrpc/gen/grpc/trees/server"
 svc "example.com/recursivegrpc/gen/trees"
 loom "github.com/CaliLuke/loom/pkg"
 "github.com/stretchr/testify/require"
 "google.golang.org/grpc"
 "google.golang.org/grpc/codes"
 "google.golang.org/grpc/credentials/insecure"
 "google.golang.org/grpc/status"
 "google.golang.org/grpc/test/bufconn"
)

type implementation struct {
 output *svc.Node
 failure bool
 calls atomic.Int32
 receives atomic.Int32
}
func (s *implementation) Check(_ context.Context, p *svc.Node) (*svc.Node,error) {
 s.calls.Add(1)
 if s.failure {
  return nil,&svc.Failure{Name:"failure",Tree:s.output}
 }
 return s.output,nil
}
func (s *implementation) Exchange(_ context.Context, stream svc.ExchangeServerStream) error {
 s.calls.Add(1)
 if _,err:=stream.Recv();err!=nil {
  return err
 }
 s.receives.Add(1)
 if s.failure {
  return &svc.Failure{Name:"failure",Tree:s.output}
 }
 return stream.Send(s.output)
}
func (s *implementation) Watch(_ context.Context, stream svc.WatchServerStream) error {
 return stream.Send(&svc.Viewed{Tree:s.output})
}
func tree(invalid bool) *svc.Node {
 value:="good"
 if invalid {
  value="BAD"
 }
 node:=&svc.Node{Value:value}
 for i:=0;i<5;i++ {
  node=&svc.Node{Value:"good",Next:node}
 }
 return node
}
func connect(t *testing.T,s pb.TreesServer) (*client.Client,pb.TreesClient) {
 t.Helper()
 listener:=bufconn.Listen(1024*1024)
 transport:=grpc.NewServer()
 pb.RegisterTreesServer(transport,s)
 go func() {
  if err:=transport.Serve(listener);err!=nil {
   t.Error(err)
  }
 }()
 t.Cleanup(transport.Stop)
 conn,err:=grpc.NewClient("passthrough:///bufconn",grpc.WithContextDialer(func(context.Context,string)(net.Conn,error) {
  return listener.Dial()
 }),grpc.WithTransportCredentials(insecure.NewCredentials()))
 require.NoError(t,err)
 t.Cleanup(func() {
  require.NoError(t,conn.Close())
 })
 return client.NewClient(conn),pb.NewTreesClient(conn)
}
type rawViewedServer struct {
 pb.UnimplementedTreesServer
 output *svc.Node
}
func (s *rawViewedServer) Watch(_ *pb.WatchRequest, stream pb.Trees_WatchServer) error {
 return stream.Send(&pb.WatchResponse{Tree:wireTree(s.output)})
}
func wireTree(node *svc.Node) *pb.Node {
 if node==nil {
  return nil
 }
 return &pb.Node{Value:&node.Value,Next:wireTree(node.Next)}
}
func diagnostic(t *testing.T,err error,root string,invalid bool) {
 t.Helper()
 if !invalid {
  require.NoError(t,err)
  return
 }
 var detail *loom.ServiceError
 require.ErrorAs(t,err,&detail)
 require.NotNil(t,detail.Field)
 require.Equal(t,root+strings.Repeat(".next",5)+".value",*detail.Field)
 require.Contains(t,detail.Error(),"BAD")
}
func TestRecursiveCallForms(t *testing.T) {
 for _,invalid:=range []bool{false,true} {
  t.Run(map[bool]string{false:"valid",true:"invalid"}[invalid],func(t *testing.T) {
   impl:=&implementation{output:tree(false)}
   cli,wire:=connect(t,server.New(svc.NewEndpoints(impl),nil,nil))
   ctx,cancel:=context.WithTimeout(context.Background(),5*time.Second)
   defer cancel()
   request:=client.NewProtoCheckRequest(tree(invalid))
   _,err:=wire.Check(ctx,request)
   if invalid {
    require.Equal(t,codes.InvalidArgument,status.Code(err))
    require.Zero(t,impl.calls.Load())
   } else {
    require.NoError(t,err)
    require.EqualValues(t,1,impl.calls.Load())
   }
   _,err=server.DecodeCheckRequest(ctx,request,nil)
   diagnostic(t,err,"message",invalid)
   result,err:=cli.Exchange()(ctx,nil)
   require.NoError(t,err)
   stream:=result.(svc.ExchangeClientStream)
   require.NoError(t,stream.Send(tree(invalid)))
   _,err=stream.Recv()
   if invalid {
    require.Equal(t,codes.InvalidArgument,status.Code(err))
    require.Zero(t,impl.receives.Load())
   } else {
    require.NoError(t,err)
    require.EqualValues(t,1,impl.receives.Load())
   }
   require.NoError(t,stream.Close())
  })
  for _,failure:=range []bool{false,true} {
   t.Run(map[bool]string{false:"response",true:"designed-error"}[failure]+map[bool]string{false:"-valid",true:"-invalid"}[invalid],func(t *testing.T) {
    impl:=&implementation{output:tree(invalid),failure:failure}
    cli,_:=connect(t,server.New(svc.NewEndpoints(impl),nil,nil))
    ctx,cancel:=context.WithTimeout(context.Background(),5*time.Second)
    defer cancel()
    _,err:=cli.Check()(ctx,tree(false))
    root:="message"
    if failure {
     root="errmsg.tree"
     if !invalid {
      var designed *svc.Failure
      require.ErrorAs(t,err,&designed)
      err=nil
     }
    }
    diagnostic(t,err,root,invalid)
    result,err:=cli.Exchange()(ctx,nil)
    require.NoError(t,err)
    stream:=result.(svc.ExchangeClientStream)
    require.NoError(t,stream.Send(tree(false)))
    _,err=stream.Recv()
    root="stream"
    if failure {
     root="errmsg.tree"
     if !invalid {
      var designed *svc.Failure
      require.True(t,errors.As(err,&designed))
      err=nil
     }
    }
    diagnostic(t,err,root,invalid)
    require.NoError(t,stream.Close())
    if !failure {
     viewClient,_:=connect(t,&rawViewedServer{output:tree(invalid)})
     result,err=viewClient.Watch()(ctx,nil)
     require.NoError(t,err)
     _,err=result.(svc.WatchClientStream).Recv()
     diagnostic(t,err,"result.tree",invalid)
    }
   })
  }
 }
}
`
