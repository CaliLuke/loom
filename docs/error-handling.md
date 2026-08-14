---
title: Error Handling
weight: 6
description: "Complete guide to error handling in Loom - defining errors, transport mapping, custom types, and best practices."
llm_optimized: true
aliases:
---

Loom provides a robust error handling system that enables you to define, manage, and communicate errors effectively across your services. This guide covers everything from basic error definitions to advanced customization.

## Overview

Loom takes a "batteries included" approach to error handling where errors can be defined with minimal information (just a name) while also supporting completely custom error types when needed.

Key features:
- Service-level and method-level error definitions
- Default and custom error types
- Transport-specific mapping for HTTP, JSON-RPC, and gRPC
- Generated helper functions for error creation
- RFC 9457 `application/problem+json` responses for default HTTP errors
- Automatic contract generation

---

## Defining Errors

### API-Level Errors

Define reusable errors at the API level with transport mappings:

```go
var _ = API("calc", func() {
    Error("invalid_argument")
    HTTP(func() {
        Response("invalid_argument", StatusBadRequest)
    })
})

var _ = Service("divider", func() {
    Error("invalid_argument")  // Reuses API-level definition
    
    Method("divide", func() {
        Error("div_by_zero", DivByZero, "Division by zero")
    })
})
```

### Service-Level Errors

Service-level errors are available to all methods within a service:

```go
var _ = Service("calc", func() {
    Error("invalid_arguments", ErrorResult, "Invalid arguments provided")
    
    Method("divide", func() {
        // Can return invalid_arguments without explicitly declaring it
    })

    Method("multiply", func() {
        // Can also return invalid_arguments
    })
})
```

### Method-Level Errors

Method-specific errors are scoped to a particular method:

```go
var _ = Service("calc", func() {
    Method("divide", func() {
        Payload(func() {
            Field(1, "dividend", Int)
            Field(2, "divisor", Int)
            Required("dividend", "divisor")
        })
        Result(func() {
            Field(1, "quotient", Int)
            Required("quotient")
        })
        Error("div_by_zero")  // Only available to this method
    })
})
```

An error declared with `Error(...)` but omitted from the method's JSON-RPC
`Response(...)` mappings has no typed transport contract. Generated clients
therefore return the raw `*jsonrpc.Error` and do not decode or validate its
`data` field. Add a JSON-RPC `Response` mapping whenever callers need a typed,
validated service error.

---

## Error Types

### Default ErrorResult

The default `ErrorResult` type includes standard fields:

- **Name**: The error name as defined in the DSL
- **ID**: Unique identifier for the error instance
- **Message**: Descriptive error message
- **Temporary**: Whether the error is transient
- **Timeout**: Whether the error was caused by a timeout
- **Fault**: Whether the error was a server-side fault

```go
var _ = Service("divider", func() {
    Error("DivByZero", ErrorResult, "Division by zero")
    Error("ServiceUnavailable", ErrorResult, "Service temporarily unavailable", func() {
        Temporary()
    })
})
```

Generated helper functions:

```go
// MakeDivByZero builds a loom.ServiceError from an error
func MakeDivByZero(err error) *loom.ServiceError {
    return loom.NewServiceError(err, "DivByZero", false, false, false)
}

// MakeServiceUnavailable builds a loom.ServiceError from an error
func MakeServiceUnavailable(err error) *loom.ServiceError {
    return loom.NewServiceError(err, "ServiceUnavailable", true, false, false)
}
```

### Custom Error Types

For more detailed error information, define custom error types:

```go
var DivByZero = Type("DivByZero", func() {
    Description("DivByZero is the error returned when using value 0 as divisor.")
    Field(1, "message", String, "Error message")
    Field(2, "dividend", Int, "Dividend that was used")
    ErrorName(3, "name", String, "Error name")
    Required("message", "dividend", "name")
})

var _ = Service("divider", func() {
    Method("divide", func() {
        Error("DivByZero", DivByZero, "Division by zero")
    })
})
```

When one custom type represents multiple errors in the same method, use
`ErrorName` to identify the routing field. The field is part of the response
body by default. To keep routing metadata exclusively in the `Loom-Error`
header, map it as a response header for every named response:

```go
Response("forbidden", StatusForbidden, func() {
    Header("name:loom-error")
})
```

The generated server writes the header, the generated client restores the
field from it, and the field is omitted from the response body and OpenAPI
schema. This supports several named errors with one unchanged payload shape,
including multiple errors that use the same HTTP status.

The generated `Error()` method returns the first non-empty string field with one of these names:

1. `message`
2. `detail`
3. `title`
4. `error`
5. `reason`

If these fields are empty or absent, `Error()` returns the custom type description.

### Error Properties

Error properties inform clients about error characteristics (only available with `ErrorResult`):

```go
var _ = Service("calc", func() {
    Error("service_unavailable", ErrorResult, func() {
        Description("Service is temporarily unavailable")
        Temporary()  // Client should retry
    })

    Error("request_timeout", ErrorResult, func() {
        Description("Request timed out")
        Timeout()    // Deadline exceeded
    })

    Error("internal_error", ErrorResult, func() {
        Description("Internal server error")
        Fault()      // Server-side issue
    })
})
```

Client-side handling:

```go
res, err := client.Divide(ctx, payload)
if err != nil {
    if e, ok := err.(*loom.ServiceError); ok {
        if e.Temporary {
            return retry(ctx, func() error {
                res, err = client.Divide(ctx, payload)
                return err
            })
        }
        if e.Fault {
            log.Error("server fault detected", "error", e)
            alertAdmins(e)
        }
    }
}
```

---

## Transport Mapping

### HTTP Status Codes

```go
var _ = Service("divider", func() {
    Error("DivByZero", func() {
        Description("Division by zero error")
    })

    HTTP(func() {
        Response("DivByZero", StatusBadRequest)
    })

    Method("integral_divide", func() {
        Error("HasRemainder", func() {
            Description("Integer division has a remainder")
        })

        HTTP(func() {
            POST("/divide/integral")
            Response("HasRemainder", StatusExpectationFailed)
        })
    })
})
```

### gRPC Status Codes

```go
var _ = Service("divider", func() {
    Error("DivByZero", func() {
        Description("Division by zero error")
    })

    GRPC(func() {
        Response("DivByZero", CodeInvalidArgument)
    })

    Method("integral_divide", func() {
        Error("HasRemainder")

        GRPC(func() {
            Response("HasRemainder", CodeUnknown)
        })
    })
})
```

### Combined HTTP and gRPC

```go
var _ = Service("divider", func() {
    Error("DivByZero")

    Method("divide", func() {
        HTTP(func() {
            POST("/divide")
            Response("DivByZero", StatusUnprocessableEntity)
        })

        GRPC(func() {
            Response("DivByZero", CodeInvalidArgument)
        })
    })
})
```

### JSON-RPC Error Codes

JSON-RPC errors are mapped with `JSONRPC` response blocks. Loom includes
standard JSON-RPC constants such as `RPCInvalidParams` and `RPCInternalError`,
and custom application errors can use implementation-defined codes.

```go
var _ = Service("divider", func() {
    Error("DivByZero")

    JSONRPC(func() {
        POST("/jsonrpc")
        Response("DivByZero", RPCInvalidParams)
    })

    Method("divide", func() {
        JSONRPC(func() {})
    })
})
```

---

## Producing and Consuming Errors

### Producing Errors

Using generated helper functions:

```go
func (s *dividerSvc) IntegralDivide(ctx context.Context, p *divider.IntOperands) (int, error) {
    if p.Divisor == 0 {
        return 0, gendivider.MakeDivByZero(fmt.Errorf("divisor cannot be zero"))
    }
    if p.Dividend%p.Divisor != 0 {
        return 0, gendivider.MakeHasRemainder(fmt.Errorf("remainder is %d", p.Dividend%p.Divisor))
    }
    return p.Dividend / p.Divisor, nil
}
```

Using custom error types:

```go
func (s *dividerSvc) IntegralDivide(ctx context.Context, p *divider.IntOperands) (int, error) {
    if p.Divisor == 0 {
        return 0, &gendivider.DivByZero{
            Name:     "DivByZero",
            Message:  "divisor cannot be zero",
            Dividend: p.Dividend,
        }
    }
    return p.Dividend / p.Divisor, nil
}
```

### Consuming Errors

Handling default errors:

```go
res, err := client.Divide(ctx, payload)
if err != nil {
    if serr, ok := err.(*loom.ServiceError); ok {
        switch serr.Name {
        case "HasRemainder":
            // Handle remainder error
        case "DivByZero":
            // Handle division by zero
        default:
            // Handle unknown errors
        }
    }
}
```

Handling custom errors:

```go
res, err := client.Divide(ctx, payload)
if err != nil {
    if dbz, ok := err.(*gendivider.DivByZero); ok {
        fmt.Printf("Division by zero: %s (dividend was %d)\n", dbz.Message, dbz.Dividend)
    }
}
```

---

## Structured Remediation

Use `Remedy` when a client or operator needs a stable next action in addition
to the error classification:

```go
Error("expired_session", ErrorResult, func() {
    Description("The session can no longer authorize requests.")
    Remedy(func() {
        RemedyCode("session.reauthenticate")
        SafeMessage("Your session expired. Sign in again.")
        RetryHint("Obtain a new session before retrying the request.")
    })
})
```

A remedy must declare at least one field:

- `RemedyCode` is a stable machine-facing action or classification. Do not use
  human prose as the code.
- `SafeMessage` is suitable for an end user and replaces internal error detail
  on default transport error paths.
- `RetryHint` tells the caller how or when a retry can succeed; it should not
  merely repeat that the operation failed.

Generated default-error constructors attach the remedy to the returned
`*loom.ServiceError`. Generated custom error types expose the same metadata
through `LoomErrorRemedy`. Runtime code can consume either shape with
`loom.ExtractErrorRemedy`, `loom.ErrorRemedyCode`, `loom.ErrorSafeMessage`, and
`loom.ErrorRetryHint` instead of type-switching on generated errors.

The default HTTP problem document uses `SafeMessage` as `detail` and publishes
`RetryHint` as `retry_hint`. JSON-RPC error `data` carries the full nested
remedy object. Keep internal causes in wrapped errors and logs; do not place
credentials, queries, or stack traces in any remediation field.

---

## Default HTTP Problem Documents

When an HTTP endpoint returns a default Loom service error, Loom serializes it
as an RFC 9457 problem document using `application/problem+json`.

The default problem body contains:

- `type`: a URI identifying the problem category
- `title`: a short human-readable title
- `status`: the HTTP status code
- `detail`: a safe message for this occurrence
- `instance`: a stable `urn:loom:error:<id>` occurrence identifier
- `code`: the stable Loom error name
- `retry_hint`: optional retry or correction guidance

Generic HTTP problems use `about:blank` when the error code matches the status.
Other errors get deterministic Loom problem type URIs such as
`https://github.com/CaliLuke/loom/problems/div-by-zero`.

Override the generated problem type or title in the error DSL:

```go
Error("wrong_token_type", ErrorResult, func() {
    ProblemType("https://example.com/problems/wrong-token-type")
    ProblemTitle("Wrong Token Type")
})
```

### Identifier Entropy Failures

Loom never substitutes partial bytes or a zero identifier when the operating
system's cryptographic entropy source fails. APIs that already return only a
string — including framework-generated service-error, request-log, and trace
identifiers — fail loudly because they cannot report the failure in their
signature. Treat such a panic as a process-level environment failure, not a
recoverable request condition.

Generated JSON-RPC clients have an error-returning encoding path. If generation
of an automatic request ID fails, the client returns a wrapped encoding error
and does not send a request with a missing, partial, or zero ID.

## Custom Error Serialization

Customize error serialization by providing a custom formatter:

```go
type CustomErrorResponse struct {
    Code    string            `json:"code"`
    Message string            `json:"message"`
    Details map[string]string `json:"details,omitempty"`
}

func (r *CustomErrorResponse) StatusCode() int {
    switch r.Code {
    case "VALIDATION_ERROR":
        return http.StatusBadRequest
    case "NOT_FOUND":
        return http.StatusNotFound
    default:
        return http.StatusInternalServerError
    }
}

func customErrorFormatter(ctx context.Context, err error) loomhttp.Statuser {
    if serr, ok := err.(*loom.ServiceError); ok {
        switch serr.Name {
        case loom.MissingField:
            return &CustomErrorResponse{
                Code:    "MISSING_FIELD",
                Message: fmt.Sprintf("The field '%s' is required", *serr.Field),
                Details: map[string]string{"field": *serr.Field},
            }
        default:
            return &CustomErrorResponse{
                Code:    "VALIDATION_ERROR",
                Message: serr.Message,
            }
        }
    }
    return &CustomErrorResponse{
        Code:    "INTERNAL_ERROR",
        Message: err.Error(),
    }
}

// Use when creating the server
server = calcsvr.New(endpoints, mux, dec, enc, eh, customErrorFormatter)
```

---

## Best Practices

### 1. Consistent Error Naming

Use clear, descriptive names:

```go
// Good
Error("DivByZero", func() {
    Description("DivByZero is returned when the divisor is zero.")
})

// Bad
Error("Error1", func() {
    Description("An unspecified error occurred.")
})
```

### 2. Prefer ErrorResult Over Custom Types

Use the default `ErrorResult` for most errors. Reserve custom types for scenarios requiring additional context:

```go
// Simple errors - use ErrorResult
Error("InvalidInput", ErrorResult, "Invalid input provided.")

// Complex errors needing extra context - use custom types
Error("InvalidOperation", InvalidOperation, "Unsupported operation.")
```

### 3. Use Error Properties

Leverage `Temporary()`, `Timeout()`, and `Fault()` to provide metadata:

```go
Error("ServiceUnavailable", ErrorResult, func() {
    Description("Service is temporarily unavailable")
    Temporary()
})
```

### 4. Document Errors Thoroughly

Provide clear descriptions:

```go
Error("AuthenticationFailed", ErrorResult, func() {
    Description("AuthenticationFailed is returned when user credentials are invalid.")
})
```

### 5. Implement Proper Error Mapping

Map errors consistently across transports:

```go
var _ = Service("auth", func() {
    Error("InvalidToken", func() {
        Description("InvalidToken is returned when the provided token is invalid.")
    })

    HTTP(func() {
        Response("InvalidToken", StatusUnauthorized)
    })

    GRPC(func() {
        Response("InvalidToken", CodeUnauthenticated)
    })
})
```

### 6. Test Error Handling

Write tests to verify error behavior:

```go
func TestDivideByZero(t *testing.T) {
    svc := internal.NewDividerService()
    _, err := svc.Divide(context.Background(), &divider.DividePayload{A: 10, B: 0})
    if err == nil {
        t.Fatalf("expected error, got nil")
    }
    if serr, ok := err.(*loom.ServiceError); !ok || serr.Name != "DivByZero" {
        t.Fatalf("expected DivByZero error, got %v", err)
    }
}
```

### 7. Security Considerations

- Never expose internal system details in errors
- Sanitize all error messages
- Log detailed errors internally but return safe messages to clients

```go
func secureErrorFormatter(ctx context.Context, err error) loomhttp.Statuser {
    log.Printf("Error: %+v", err)  // Log full details
    
    if serr, ok := err.(*loom.ServiceError); ok && serr.Fault {
        // Return generic message for server faults
        return &CustomErrorResponse{
            Code:    "INTERNAL_ERROR",
            Message: "An internal error occurred",
        }
    }
    // Return specific message for validation errors
    return formatValidationError(err)
}
```

---

## See Also

- [DSL Reference: Error Handling](dsl-reference.md#error-handling-design-level) — Design-level error definitions
- [HTTP Guide](http-guide.md) — HTTP status code mapping and error responses
- [gRPC Guide](grpc-guide.md#error-handling) — gRPC status code mapping
- [Loom log package](https://pkg.go.dev/github.com/CaliLuke/loom/clue/log) — error logging helpers
