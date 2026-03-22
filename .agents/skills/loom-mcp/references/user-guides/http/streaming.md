# HTTP: Streaming

Use this for HTTP-specific WebSocket and SSE behavior.

## WebSocket

Goa maps streaming HTTP endpoints to WebSocket by default.

```go
Method("echo", func() {
    StreamingPayload(MessageIn)
    StreamingResult(MessageOut)
    HTTP(func() { GET("/echo") })
})
```

Implementation rule:

- Handle `ctx.Done()`
- Clean up connections explicitly
- Keep send/receive loops isolated so one blocked side does not leak goroutines

## Server-Sent Events

Use `ServerSentEvents()` for one-way server streaming over HTTP:

```go
Method("stream", func() {
    StreamingResult(Event)
    HTTP(func() {
        GET("/events/stream")
        ServerSentEvents(func() {
            SSEEventData("message")
            SSEEventType("type")
            SSEEventID("id")
            SSEEventRetry("retry")
        })
    })
})
```

Use `SSERequestID("startID")` if resumability via `Last-Event-Id` matters.
