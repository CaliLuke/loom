---
title: gRPC Guide
weight: 5
description: "Complete guide to gRPC transport in Loom - service design, streaming patterns, error handling, and Protocol Buffer integration."
llm_optimized: true
aliases:
---

Loom provides comprehensive support for building gRPC services through its DSL and code generation. This guide covers service design, streaming patterns, error handling, and implementation.

## Overview

Loom's gRPC support includes:

- **Automatic Protocol Buffer Generation**: `.proto` files generated from your design
- **Type Safety**: End-to-end type safety from definition to implementation
- **Code Generation**: Server and client code generated automatically
- **Built-in Validation**: Request validation based on your design
- **Streaming Support**: All gRPC streaming patterns supported
- **Error Handling**: Comprehensive error handling with status code mapping

### Prerequisites and Supported Toolchain

Use Go 1.27 or newer and keep these protobuf tools on `PATH` when generating a
gRPC service:

| Tool | Supported version |
|------|-------------------|
| `protoc` | 35.1 |
| `protoc-gen-go` | v1.36.12 |
| `protoc-gen-go-grpc` | v1.6.2 |

Framework contributors can run `make depend` from the Loom repository to
install these exact versions. Service repositories should pin the same tools
in their own bootstrap or CI instead of installing `@latest`; a changed plugin
can rewrite checked-in generated Go even when the design did not change.

Verify the active binaries before diagnosing generation drift:

```bash
protoc --version
protoc-gen-go --version
protoc-gen-go-grpc --version
```

### Type Mapping

| Loom Type  | Protocol Buffer Type |
|-----------|---------------------|
| Int       | int32              |
| Int32     | int32              |
| Int64     | int64              |
| UInt      | uint32             |
| UInt32    | uint32             |
| UInt64    | uint64             |
| Float32   | float              |
| Float64   | double             |
| String    | string             |
| Boolean   | bool               |
| Bytes     | bytes              |
| ArrayOf   | repeated           |
| MapOf     | map                |

Map keys must be `Boolean`, `String`, or an integer type, including aliases of
these types. `Any`, floating-point, bytes, object, array, and union keys are
rejected during design validation. Map values may use `Any`.

`Any` maps to `google.protobuf.Value`. Generated Go uses `loom.JSONValue`
for direct values and uses the same type for array and map elements. Generated
transforms parse the raw JSON before writing protobuf messages and encode
received protobuf values back to JSON. Conversion errors propagate before Loom
writes a gRPC message. Protobuf represents JSON numbers as `double`, so gRPC
cannot retain number spellings or integer precision beyond that wire type. Prefer
a concrete Loom type whenever the value has a stable contract.

### Scalar Presence and Defaults

All scalar message fields use explicit protobuf presence, including fields
marked `Required` and fields in scalar payload/result wrappers. The generated
Go protobuf fields are pointers for strings, booleans, and numbers; bytes remain
slices. An explicit zero, `false`, empty string, or present empty byte slice is
accepted when the authored constraints allow it. An omitted required field is
rejected before conversion to the service value, in requests, responses, and
stream items. `Required("name")` still requires presence when that field declares a `Default`:
a default does not relax requiredness.

An absent optional field with a default receives that default during decoding.
An explicitly supplied zero overrides the default. Encoding a service value
preserves that value, including zero; it does not replace zero with a default.
Array/map elements remain values. A scalar oneof branch uses its wrapper for
presence rather than another pointer inside it.

Regenerate clients and servers with `loom gen`. Code that constructs protobuf
messages directly must set scalar pointers, for example `proto.String("")`.
Clients generated from the old schema omit zero-valued required scalars and
cannot express their presence; regenerate and deploy those clients with the
corrected schema. There is no compatibility mode for accepting those omissions.

---

## Service Design

### Basic Service Structure

```go
var _ = Service("calculator", func() {
    Description("The Calculator service performs arithmetic operations")

    GRPC(func() {
        Package("calculator.v1")
    })

    Method("add", func() {
        Description("Add two numbers")
        
        Payload(func() {
            Field(1, "a", Int, "First operand")
            Field(2, "b", Int, "Second operand")
            Required("a", "b")
        })
        
        Result(func() {
            Field(1, "sum", Int, "Result of addition")
            Required("sum")
        })
    })
})
```

### Method Definition

Methods define operations with gRPC-specific settings:

```go
Method("add", func() {
    Description("Add two numbers")

    Payload(func() {
        Field(1, "a", Int, "First operand")
        Field(2, "b", Int, "Second operand")
        Required("a", "b")
    })

    Result(func() {
        Field(1, "sum", Int, "Result of addition")
        Required("sum")
    })

    Error("overflow", ErrorResult, "Integer overflow")

    GRPC(func() {
        Response(CodeOK)
        Response("overflow", CodeOutOfRange)
    })
})
```

### Message Types

#### Field Numbering

Use Protocol Buffer best practices:
- Numbers 1-15: Frequently occurring fields (1-byte encoding)
- Numbers 16-2047: Less frequent fields (2-byte encoding)

```go
Method("createUser", func() {
    Payload(func() {
        // Frequently used fields (1-byte encoding)
        Field(1, "id", String)
        Field(2, "name", String)
        Field(3, "email", String)

        // Less frequently used fields (2-byte encoding)
        Field(16, "preferences", func() {
            Field(1, "theme", String)
            Field(2, "language", String)
        })
    })
})
```

Protocol buffer fields cannot have anonymous message types, so Loom generates a
message for each inline object field, including one that only holds a `OneOf`.
The message is named after the enclosing message and the field, for example
`CreateUserRequestPreferences`. Inline object array elements and map values are
handled the same way.

#### Named Arrays and Maps

A `Type` defined as an array, such as `Type("Tags", ArrayOf(String))`, is a
message that holds the array in a repeated field named `field`. A `Type`
defined as a map, such as `Type("Index", MapOf(String, Int))`, is a message
that holds the map in a map field named `field`. This applies wherever the
type appears: as a unary payload, as a result or streaming result, as a
message field, and as an array element or map value. A named array can also be
a union branch. A streaming payload is the exception: the message of
`StreamingPayload(Tags)` is named after the method, such as
`UploadStreamingRequest`, and holds the same `field`. A named array of another
named array, such as `Type("More", Tags)`, holds the array itself in every
other position, but `StreamingPayload(More)` streams `Tags` messages. A named
map of another named map works the same way:

```proto
message EchoRequest {
    Tags labels = 1;               // Field(1, "labels", Tags)
    repeated Tags label_lists = 2; // Field(2, "label_lists", ArrayOf(Tags))
    Index index = 3;               // Field(3, "index", Index)
}

message Tags {
    repeated string field = 1;
}

message Index {
    map<string, sint64> field = 1;
}
```

The service type is still the Go slice or map type, such as
`type Tags []string` or `type Index map[string]int`. Do not rely on the
wrapper message to tell an absent array or map from an empty one: other
transports and JSON encoding treat both as missing.

#### Union Fields

A union is a `oneof` named after the attribute. Each branch of a `OneOf` block
takes the number set with `Field`. When you pass a constructor `OneOf` to
`Field`, the branches take consecutive numbers from the field number, in
declaration order. Leave these numbers free for the branches:

```go
var Envelope = Type("Envelope", func() {
    Field(1, "id", String)
    Field(2, "pick", OneOf(Leaf, Other)) // oneof pick { Leaf leaf = 2; Other other = 3; }
    Field(4, "next", String)
})
```

This gives the same message as the block form
`OneOf("pick", func() { Field(2, "Leaf", Leaf); Field(3, "Other", Other) })`.
Adding a branch at the end of a constructor `OneOf` takes the next number.
Reordering or removing its branches changes the numbers of the other branches,
so use the block form when branches can change. Generation fails when a field
or union branch of a message has no number, or when two of them have the same
number. This applies to every generated message, including nested user types
and inline objects.

The fields, `oneof` names and `oneof` fields of a message share one namespace
in protocol buffers, so Loom gives each of them a unique name in the message:

- A field that is not a union normally keeps its name. Generation fails when two such
  fields have the same protocol buffer name, such as `fooBar` and `foo_bar`.
- A union field is a `oneof` named after the field. Loom adds `_oneof` until
  the name differs from the names of its branches and from the names already
  used, for example `oneof leaf_oneof` for `Field(1, "leaf", OneOf(Leaf, Other))`.
- A branch keeps its name unless an earlier field, `oneof` or branch uses it.
  Loom then adds the union field name and an underscore as a prefix, as many
  times as needed.

Generated protobuf Go fields also avoid generated methods and getters. For
example, `reset` becomes `Reset_`, and a field named `get_label` can become
`GetLabel_` when a preceding field generates `GetLabel()`. Optional fields add
synthetic oneofs that can affect this allocation. A branch wrapper type can
have a different suffix from its field when it collides with a map-entry type.
Use the emitted protobuf declarations when accessing these values directly;
Loom's conversions, validation, and CLI use the same names. Service
`struct:field:name` metadata does not rename protobuf fields.

The Go plugin does not reserve every selector it emits. When that would make
the generated message fail to compile, Loom adds further `_oneof` suffixes to
its oneof names and `_field` suffixes to a field that would clash with
`ProtoReflect`, avoiding names already in use. These repairs preserve service
attribute names and protobuf field numbers. Designs whose generated protobuf
Go selectors already compile keep their wire names.

Fields that are not unions take their names first. The unions then take their
names in declaration order, so the first union to use a name keeps it:

```go
var Envelope = Type("Envelope", func() {
    Field(1, "a", OneOf(String, Int64))  // oneof a { string string_ = 1; sint64 int64_ = 2; }
    Field(3, "b", OneOf(Boolean, Int64)) // oneof b { bool boolean = 3; sint64 b_int64 = 4; }
    Field(5, "pick", OneOf(Leaf, Other)) // oneof pick { Leaf pick_leaf = 5; Other other = 6; }
    Field(7, "leaf", String)             // optional string leaf = 7;
})
```

The names are deterministic, and a message without collisions keeps the names
of its branches. The protocol buffer Go code follows the same names, such as
the `BInt64` field of the `<Message>_BInt64` oneof type, while the service
type keeps the branch names, such as `SetInt64`. A new field that takes
the name of a branch renames that branch. So does a union inserted earlier in
declaration order, or a branch added to an earlier union, that takes the name
of a later branch. This changes the JSON name and the
generated Go names of the branch, but not its field number, so the binary wire
format does not change.

A named union, a `Type` defined as a `OneOf`, works the same way when you pass
it to `Field`. The field is a `oneof` of the enclosing message, and its
branches take their numbers from the field number. Loom does not generate a
separate message for a union field. Each message that uses the union has its
own `oneof`, numbered from the number of its own field:

```go
var Choice = Type("Choice", OneOf(Leaf, Other))

var Holder = Type("Holder", func() {
    Field(1, "label", String)
    Field(2, "choice", Choice) // oneof choice { Leaf leaf = 2; Other other = 3; }
})
```

A protocol buffer `oneof` cannot be repeated or used as a map key or value, so
design validation rejects a union, named or constructor, used as an array
element or as a map key or value in a gRPC method. To send a list of union
values, wrap the union in a type with one field and use that type as the
element:

```go
var Item = Type("Item", func() {
    Field(1, "choice", Choice)
})
// Field(3, "items", ArrayOf(Item))
```

A method payload or result that is a union, such as `Payload(OneOf(Leaf, Other))`
or a `Type` defined as a `OneOf`, is a message that holds one `oneof` named
`field`. When a branch has that name, Loom adds `_oneof` to the oneof name
until it differs from every branch name, for example `field_oneof`. The
branches take the numbers 1, 2 and so on, in declaration order.
This applies to unary and streaming payloads and results:

```proto
message EchoRequest {
    oneof field {
        Leaf leaf = 1;
        Other other = 2;
    }
}
```

The generated server rejects a request message with no branch set with
`InvalidArgument`.

A `oneof` cannot hold another `oneof`, so a union used as a branch of another
union is the message that wraps its `oneof`, as for a union payload. This
applies to a named union used as a branch of a named union, of a constructor
`OneOf` or of a `OneOf` block, and to a constructor `OneOf` inside another
one. The message takes the name of the named union or the derived name of the
constructor `OneOf`, such as `ExtraOrOther` for `OneOf(Extra, Other)`. The
same named union passed to `Field` elsewhere remains a `oneof` of the message
that holds it:

```go
var Choice = Type("Choice", OneOf(Leaf, Other))
var Outer = Type("Outer", OneOf(Choice, Extra))

var Holder = Type("Holder", func() {
    Field(1, "outer", Outer) // oneof outer { Choice choice = 1; Extra extra = 2; }
})
```

```proto
message Choice {
    oneof field {
        Leaf leaf = 1;
        Other other = 2;
    }
}
```

The generated validation rejects a union branch message with no branch set.
A named array can also be a union branch, and so can an array branch of a
`OneOf` block, which Loom names after the union and the branch, for example
`DetailWords`. Design validation rejects a map branch, named or not. Wrap the
map in a type with one field and use that type as the branch.

#### Metadata Handling

Send fields as gRPC metadata instead of message body:

```go
var CreatePayload = Type("CreatePayload", func() {
    Field(1, "name", String, "Name of account")
    TokenField(2, "token", String, "JWT token")
    Field(3, "metadata", String, "Additional info")
})

Method("create", func() {
    Payload(CreatePayload)
    
    GRPC(func() {
        // Send token in metadata
        Metadata(func() {
            Attribute("token")
        })
        // Only include specific fields in message
        Message(func() {
            Attribute("name")
            Attribute("metadata")
        })
        Response(CodeOK)
    })
})
```

#### Response Headers and Trailers

```go
Method("create", func() {
    Result(CreateResult)
    
    GRPC(func() {
        Response(func() {
            Code(CodeOK)
            Headers(func() {
                Attribute("id")
            })
            Trailers(func() {
                Attribute("status")
            })
        })
    })
})
```

---

## Streaming Patterns

> **Design Recap**: Streaming is defined at the design level using `StreamingPayload` and `StreamingResult`. The DSL is transport-agnostic — the same design works for both HTTP and gRPC. See [DSL Reference: Streaming](dsl-reference.md#streaming) for design patterns. This section covers gRPC-specific streaming implementation.

gRPC supports three streaming patterns.

A gRPC server stream sends only streaming messages, so a gRPC method cannot
declare both `Result` and `StreamingResult` with different types. That
combination is available only through Server-Sent Events.

### Initial Payload and Stream Item Envelopes

A client- or bidirectional-streaming method may declare both `Payload` and
`StreamingPayload`. When both shapes need protobuf message fields, Loom emits
a typed `oneof` envelope with two branches:

- `initial_payload` carries the one-shot method payload in the first frame.
- `stream_item` carries every later value sent through the generated stream.

Generated clients send the initial frame before returning the stream wrapper,
then wrap each `Send` value as a stream item. Generated servers require that
ordering: a missing first frame, a stream item in place of the initial payload,
or another initial-payload frame returned from `Recv` is a validation error.
This framing is generated protocol, not an application convention; external
protobuf clients must follow the same ordering.

If the opening send returns `io.EOF`, the generated client still returns the
stream. Read `CloseAndRecv` for client streaming or `Recv` for bidirectional
streaming to obtain the final server status or response. Opening-send EOF alone
does not establish whether the RPC succeeded. Other send errors still fail the
opening call. Regenerate clients to adopt this behavior.

### Server-Side Streaming

Server sends multiple responses to a single client request:

```go
var _ = Service("monitor", func() {
    Method("watch", func() {
        Description("Stream system metrics")
        
        Payload(func() {
            Field(1, "interval", Int, "Sampling interval in seconds")
            Required("interval")
        })
        
        StreamingResult(func() {
            Field(1, "cpu", Float32, "CPU usage percentage")
            Field(2, "memory", Float32, "Memory usage percentage")
            Required("cpu", "memory")
        })
        
        GRPC(func() {
            Response(CodeOK)
        })
    })
})
```

Server implementation:

```go
func (s *monitorService) Watch(ctx context.Context, p *monitor.WatchPayload, stream monitor.WatchServerStream) error {
    ticker := time.NewTicker(time.Duration(p.Interval) * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            metrics := getSystemMetrics()
            if err := stream.Send(&monitor.WatchResult{
                CPU:    metrics.CPU,
                Memory: metrics.Memory,
            }); err != nil {
                return err
            }
        }
    }
}
```

### Client-Side Streaming

Client sends multiple requests, server sends single response:

```go
var _ = Service("analytics", func() {
    Method("process", func() {
        Description("Process stream of analytics events")
        
        StreamingPayload(func() {
            Field(1, "event_type", String, "Type of event")
            Field(2, "timestamp", String, "Event timestamp")
            Field(3, "data", Bytes, "Event data")
            Required("event_type", "timestamp", "data")
        })
        
        Result(func() {
            Field(1, "processed_count", Int64, "Number of events processed")
            Required("processed_count")
        })
        
        GRPC(func() {
            Response(CodeOK)
        })
    })
})
```

Server implementation:

```go
func (s *analyticsService) Process(ctx context.Context, stream analytics.ProcessServerStream) error {
    var count int64
    
    for {
        event, err := stream.Recv()
        if err == io.EOF {
            return stream.SendAndClose(&analytics.ProcessResult{
                ProcessedCount: count,
            })
        }
        if err != nil {
            return err
        }
        
        if err := processEvent(event); err != nil {
            return err
        }
        count++
    }
}
```

### Bidirectional Streaming

Both client and server send streams simultaneously:

```go
var _ = Service("chat", func() {
    Method("connect", func() {
        Description("Establish bidirectional chat connection")
        
        StreamingPayload(func() {
            Field(1, "message", String, "Chat message")
            Field(2, "user_id", String, "User identifier")
            Required("message", "user_id")
        })
        
        StreamingResult(func() {
            Field(1, "message", String, "Chat message")
            Field(2, "user_id", String, "User identifier")
            Field(3, "timestamp", String, "Message timestamp")
            Required("message", "user_id", "timestamp")
        })
        
        GRPC(func() {
            Response(CodeOK)
        })
    })
})
```

Server implementation:

```go
func (s *chatService) Connect(ctx context.Context, stream chat.ConnectServerStream) error {
    for {
        msg, err := stream.Recv()
        if err == io.EOF {
            return nil
        }
        if err != nil {
            return err
        }

        response := &chat.ConnectResult{
            Message:   msg.Message,
            UserID:    msg.UserID,
            Timestamp: time.Now().Format(time.RFC3339),
        }
        
        if err := stream.Send(response); err != nil {
            return err
        }
    }
}
```

---

## Error Handling

> **Design Recap**: Errors are defined at the design level using the `Error` DSL at API, service, or method scope. See [DSL Reference: Error Handling](dsl-reference.md#error-handling-design-level) for design patterns. This section covers gRPC-specific status code mapping.

### Status Codes

Map errors to gRPC status codes:

```go
Method("divide", func() {
    Error("division_by_zero")
    Error("invalid_input")

    GRPC(func() {
        Response(CodeOK)
        Response("division_by_zero", CodeInvalidArgument)
        Response("invalid_input", CodeInvalidArgument)
    })
})
```

Common status code mappings:

| Loom Error | gRPC Status Code | Use Case |
|-----------|-----------------|-----------|
| `not_found` | `CodeNotFound` | Resource doesn't exist |
| `invalid_argument` | `CodeInvalidArgument` | Invalid input |
| `internal_error` | `CodeInternal` | Server error |
| `unauthenticated` | `CodeUnauthenticated` | Missing/invalid credentials |
| `permission_denied` | `CodePermissionDenied` | Insufficient permissions |

Generated clients preserve local request-encoding and response-validation errors,
including their concrete types and validation fields. Use `errors.As` to inspect
`grpc.ClientError` or `loom.ServiceError`. Only failures from the remote call use
the declared or generic gRPC error mappings. Regenerate clients to adopt this
behavior.

When a remote failure carries a generic Loom error detail, generated unary and
stream clients retain the original RPC error as the service error's cause.
`errors.Is`, `errors.As`, and `status.Code` can inspect it while the received
service fields and history stay intact. A remote canceled status does not imply
that the local context was canceled. Custom adapters can use
`grpc.NewServiceErrorWithCause(response, cause)`; the existing
`grpc.NewServiceError(response)` remains available when there is no cause.

### Error Definitions

Define errors at service or method level:

```go
var _ = Service("users", func() {
    // Service-level errors
    Error("not_found", func() {
        Description("User not found")
    })
    Error("invalid_input")

    Method("getUser", func() {
        // Method-specific error
        Error("profile_incomplete")

        GRPC(func() {
            Response(CodeOK)
            Response("not_found", CodeNotFound)
            Response("invalid_input", CodeInvalidArgument)
            Response("profile_incomplete", CodeFailedPrecondition)
        })
    })
})
```

### Returning Errors

Use generated error constructors:

```go
func (s *users) CreateUser(ctx context.Context, p *users.CreateUserPayload) (*users.User, error) {
    exists, err := s.db.EmailExists(ctx, p.Email)
    if err != nil {
        return nil, users.MakeDatabaseError(fmt.Errorf("failed to check email: %w", err))
    }
    if exists {
        return nil, users.MakeDuplicateEmail(fmt.Sprintf("email %s is already registered", p.Email))
    }
    
    user, err := s.db.CreateUser(ctx, p)
    if err != nil {
        return nil, users.MakeDatabaseError(fmt.Errorf("failed to create user: %w", err))
    }
    
    return user, nil
}
```

---

## Implementation

### Server Implementation

```go
package main

import (
    "log"
    "net"

    "google.golang.org/grpc"

    "github.com/yourusername/calc"
    gencalc "github.com/yourusername/calc/gen/calc"
    genpb "github.com/yourusername/calc/gen/grpc/calc/pb"
    gengrpc "github.com/yourusername/calc/gen/grpc/calc/server"
)

func main() {
    svc := calc.New()
    endpoints := gencalc.NewEndpoints(svc)
    svr := grpc.NewServer()
    gensvr := gengrpc.New(endpoints, nil)
    genpb.RegisterCalcServer(svr, gensvr)
    
    lis, err := net.Listen("tcp", ":8080")
    if err != nil {
        log.Fatal(err)
    }
    
    log.Println("gRPC server listening on :8080")
    svr.Serve(lis)
}
```

### Client Implementation

```go
package main

import (
    "context"
    "log"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    
    gencalc "github.com/yourusername/calc/gen/calc"
    genclient "github.com/yourusername/calc/gen/grpc/calc/client"
)

func main() {
    conn, err := grpc.Dial("localhost:8080",
        grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    grpcClient := genclient.NewClient(conn)
    client := gencalc.NewClient(
        grpcClient.Add(),
        grpcClient.Multiply(),
    )

    result, err := client.Add(context.Background(), &gencalc.AddPayload{A: 1, B: 2})
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("1 + 2 = %d", result)
}
```

### Command-Line Client

The generated gRPC command-line client takes the request message in the
`--message` flag as JSON in the
[protocol buffer JSON mapping](https://protobuf.dev/programming-guides/json/),
and decodes it with `protojson`. Fields use their protocol buffer names, such
as `request_id`, or their lowerCamelCase JSON names, such as `requestId`. To
set a `oneof`, use the name of the selected `oneof` field at the level of the
message that holds the `oneof`. Do not use the name of the union attribute:

```go
var Choice = Type("Choice", OneOf(Leaf, Other))

var Envelope = Type("Envelope", func() {
    Field(1, "id", String)
    Field(2, "choice", Choice) // oneof choice { Leaf leaf = 2; Other other = 3; }
})
```

```bash
api-cli svc echo --message '{"id": "a", "leaf": {"name": "n"}}'
```

A union branch that is a union is the message that wraps its `oneof`, so
`{"choice": {"leaf": {"name": "n"}}}` sets the `Choice` branch of
`OneOf(Choice, Extra)`. The client rejects unknown fields and a message that
sets two fields of one `oneof`. The usage examples of the client use the same
mapping.

---

## Protocol Buffer Integration

### Automatic Generation

Loom automatically generates `.proto` files from your design:

```protobuf
syntax = "proto3";
package calc;

service Calc {
    rpc Add (AddRequest) returns (AddResponse);
    rpc Multiply (MultiplyRequest) returns (MultiplyResponse);
}

message AddRequest {
    int64 a = 1;
    int64 b = 2;
}

message AddResponse {
    int64 result = 1;
}
```

The messages generated for a method are named after it: `AddRequest`,
`AddResponse`, `AddStreamingRequest`, `AddStreamItem` and `Add<Error>Error`.
When a design type that a message of the service refers to has one of these
names and different fields, the generated message takes the name followed by
the first free number instead, such as `AddRequest2`, and the type keeps its
name. A type with the same fields shares the message.

Generated gRPC files alias service imports that would collide with framework
imports. For example, a service named `protojson` is imported as `protojsonsvc`.
Service package names, import paths, and protocol buffer names stay unchanged.
You do not need to rename the service to avoid a framework import.

Protocol buffer identifiers are ASCII only. Loom derives service, rpc, message,
field, and `oneof` names from design names by treating every non-ASCII rune as a
word separator: an `añadir` method becomes the `AAdir` rpc. ASCII service and
rpc names keep their Goify form, so a `get3d` method is the `Get3d` rpc, whose
Go method protoc-gen-go names `Get3D`. Generation fails when two methods of a
service map to the same rpc name, such as `añadir` and `a_adir`, or to the same
generated message name with different fields. Generated Go package directories
and names escape non-ASCII runes instead, because Go import paths are ASCII
only: the packages of a `Café` service are under `gen/cafu00e9`.
The same rules apply to metadata: a `struct:name:proto` value such as
`EntréeProto` names the `EntrEProto` message, and a `struct:pkg:path` value such
as `tipos/menü` generates the `menu00fc` package under `gen/tipos/menu00fc`.
Server names are escaped the same way in the example command and client CLI
directories, so a `サーバー` server is under `cmd/valu30b5u30fcu30d0u30fc`, and
the API package of the example files of a `Café` API is `cafu00e9`.

`Meta("struct:name:proto", "MenuProto")` on a `Type` or `ResultType` names
every message generated for that type: payloads, results, errors, stream items,
nested fields, array elements, map values, and union branches. Messages wrapping
named primitives, unions, arrays, and maps also keep the explicit name. A named
primitive used as a field remains a scalar, and a named union used as a field
remains a `oneof`; neither position introduces a separate message.

Generated Go code refers to messages from the service's pb package, independently
of the service type's `struct:pkg:path`. Go references use protoc-gen-go's name,
such as `NodeTree` for the protobuf name `node_tree`.

The metadata must contain exactly one message name. Empty or multi-valued
declarations fail design validation, including repeated names. An explicit name
is authoritative. Loom never substitutes the service type name,
a numbered name, or a compatibility mode when explicit declarations conflict.
Every use sharing a message must have the same fields, numbers, requiredness,
and referenced message names. Compatible customized service types get separate
converters to their shared message. A copy that changes requiredness, or moves a
field to metadata, headers, or trailers, conflicts with an unmodified use under
the same message name and fails generation. Give different contracts distinct named types and explicit names. Distinct protobuf names must also produce distinct Go type names:
`node_tree` and `NodeTree` cannot coexist in one service.

For example, a `Menu` named `MenuProto` used both as a payload and as a nested
field produces one `MenuProto` declaration. If a method removes `Menu.name` from
that message by mapping it to metadata, the reduced shape cannot also be called
`MenuProto` alongside the full nested shape; generation reports the conflict.

Regenerate clients and servers after adopting this naming behavior. Nested
messages and wrappers that previously ignored the metadata now use it, and
redundant declarations disappear. Update direct protobuf Go references,
descriptor lookups, and stored `Any` type URLs that use the old message names.

A client tells the errors of a method apart by the type of the message in the
status details. Two errors of a method can use one type that has a
`struct:name:proto` name, because the error name is a field of the type
(`ErrorName`). Code generation fails when two errors of a method map
different types to one message, because the client could not tell which type
to return.

### Protoc Configuration

The versions above are the supported defaults. Use metadata overrides only
when the deployment has a deliberate custom toolchain or additional imported
schemas:

```go
var _ = API("calculator", func() {
    // Add include paths used when Loom invokes protoc.
    Meta("protoc:include", "/usr/include", "/usr/local/include")
})

var _ = Service("calculator", func() {
    // Override the protoc command for this service.
    Meta("protoc:cmd", "/usr/bin/protoc", "--fatal_warnings")
})
```

---

---

## See Also

- [DSL Reference: Streaming](dsl-reference.md#streaming) — Design-level streaming patterns
- [DSL Reference: Error Handling](dsl-reference.md#error-handling-design-level) — Design-level error definitions
- [HTTP Guide](http-guide.md) — HTTP transport features
- [Error Handling Guide](error-handling.md) — Complete error handling patterns
- [Loom log package](https://pkg.go.dev/github.com/CaliLuke/loom/clue/log) — gRPC logging adapters

---

## Best Practices

### Error Handling
- Use appropriate gRPC status codes
- Include meaningful error messages
- Handle context cancellation and timeouts

### Streaming
- Keep message sizes reasonable
- Implement proper flow control
- Set appropriate timeouts
- Handle EOF and errors gracefully

### Performance
- Use appropriate field types
- Consider message size in design
- Use streaming for large datasets

### Versioning
- Plan for backward compatibility
- Use field numbers strategically
- Consider package versioning

### Resource Management
- Properly manage gRPC connections
- Implement graceful shutdown
- Clean up resources on context cancellation
