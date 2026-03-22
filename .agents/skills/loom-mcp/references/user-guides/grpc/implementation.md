# gRPC: Implementation

Use this for generated server/client wiring and protobuf output.

## Server Setup

```go
svc := calc.New()
endpoints := gencalc.NewEndpoints(svc)
svr := grpc.NewServer()
gensvr := gengrpc.New(endpoints, nil)
genpb.RegisterCalcServer(svr, gensvr)
```

## Client Setup

```go
conn, _ := grpc.Dial("localhost:8080",
    grpc.WithTransportCredentials(insecure.NewCredentials()))
grpcClient := genclient.NewClient(conn)
client := gencalc.NewClient(grpcClient.Add(), grpcClient.Multiply())
```

## Protobuf

Goa generates `.proto` definitions from the DSL automatically. Use service/package metadata in the design when package names or protoc behavior need control.
