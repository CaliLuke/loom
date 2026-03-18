# Streaming UI

Stream typed Goa-AI execution events to SSE, WebSockets, or message buses.

## Overview

Events are published to a single session stream `session/<session_id>`, with both `run_id` and `session_id`.

- events from nested agent runs are linked with `child_run_linked`
- UI closes the stream on `run_stream_end` for the active run

## Stream Sink Interface

--- CODE ---
type Sink interface {
    Send(ctx context.Context, event stream.Event) error
    Close(ctx context.Context) error
}
--- END CODE ---

## Event Types

| Event Type | Description |
| --- | --- |
| AssistantReply | Assistant chunks (streaming text) |
| PlannerThought | Thinking blocks |
| ToolStart | Tool execution started |
| ToolUpdate | Tool progress |
| ToolEnd | Tool completion (result/error/preview) |
| AwaitClarification | Planner waiting for human clarification |
| AwaitExternalTools | Planner waiting for external tool results |
| Usage | Token usage per model invocation |
| Workflow | Run lifecycle and phase updates |
| ChildRunLinked | Parent tool call → child agent run |
| RunStreamEnd | Terminal marker for one run |

--- CODE ---
switch e := evt.(type) {
case stream.AssistantReply:
    // e.Data.Text
case stream.PlannerThought:
    // e.Data.Note or structured thinking fields
case stream.ToolStart:
    // e.Data.ToolCallID, e.Data.ToolName, e.Data.Payload
case stream.ToolEnd:
    // e.Data.Result, e.Data.Error, e.Data.ResultPreview
case stream.ChildRunLinked:
    // e.Data.ToolName, e.Data.ToolCallID, e.Data.ChildRunID, e.Data.ChildAgentID
case stream.RunStreamEnd:
    // run has no more stream-visible events
}
--- END CODE ---

## Example: SSE Sink

--- CODE ---
type SSESink struct {
    w http.ResponseWriter
}

func (s *SSESink) Send(ctx context.Context, event stream.Event) error {
    switch e := event.(type) {
    case stream.AssistantReply:
        fmt.Fprintf(s.w, "data: assistant: %s\n\n", e.Data.Text)
    case stream.PlannerThought:
        if e.Data.Note != "" {
            fmt.Fprintf(s.w, "data: thinking: %s\n\n", e.Data.Note)
        }
    case stream.ToolStart:
        fmt.Fprintf(s.w, "data: tool_start: %s\n\n", e.Data.ToolName)
    case stream.ToolEnd:
        fmt.Fprintf(s.w, "data: tool_end: %s status=%v\n\n",
            e.Data.ToolName, e.Data.Error == nil)
    case stream.ChildRunLinked:
        fmt.Fprintf(s.w, "data: child_run_linked: %s child=%s\n\n",
            e.Data.ToolName, e.Data.ChildRunID)
    case stream.RunStreamEnd:
        fmt.Fprintf(s.w, "data: run_stream_end: %s\n\n", e.RunID())
    }
    s.w.(http.Flusher).Flush()
    return nil
}

func (s *SSESink) Close(ctx context.Context) error {
    return nil
}
--- END CODE ---

## Session Stream Subscription (Pulse)

Consume `session/<session_id>` and stop when `run_stream_end` for active run arrives.

## Global Stream Sink

Publish all runs by configuring runtime stream sink:

--- CODE ---
rt := runtime.New(
    runtime.WithStream(pulseSink), // or your custom sink
)
--- END CODE ---

## Stream Profiles

| Profile | Use Case | Included |
| --- | --- | --- |
| UserChatProfile() | end-user chat UI | reply, tool start/end, workflow completion |
| AgentDebugProfile() | developer debugging | full visibility |
| MetricsProfile() | observability | usage + workflow events |

Built-in examples:

--- CODE ---
profile := stream.UserChatProfile()
profile := stream.AgentDebugProfile()
profile := stream.MetricsProfile()

sub, _ := stream.NewSubscriberWithProfile(sink, profile)
--- END CODE ---

## Custom Profiles

--- CODE ---
profile := stream.StreamProfile{
    Assistant:  true,
    Thoughts:   false, // Skip planner thinking
    ToolStart:  true,
    ToolUpdate: true,
    ToolEnd:    true,
    Usage:      false, // Skip usage events
    Workflow:   true,
    ChildRuns:  true,  // include parent tool → child links
}

sub, _ := stream.NewSubscriberWithProfile(sink, profile)
--- END CODE ---

## Advanced: Pulse & Stream Bridges

- publish stream events to shared bus (Pulse/Redis Streams)
- use session-owned naming convention
- configure sink stream ID from `SessionID`

--- CODE ---
pulseClient := pulse.NewClient(redisClient)
s, err := pulseSink.NewSink(pulseSink.Options{
    Client: pulseClient,
    StreamID: func(ev stream.Event) (string, error) {
        if ev.SessionID() == "" {
            return "", errors.New("missing session id")
        }
        return fmt.Sprintf("session/%s", ev.SessionID()), nil
    },
})
if err != nil { log.Fatal(err) }

rt := runtime.New(
    runtime.WithEngine(eng),
    runtime.WithStream(s),
)
--- END CODE ---
