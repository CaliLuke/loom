# Goa-AI Quickstart — Step 3: Streaming

Goa-AI emits typed events for planning, tools, workflow phases, and assistant streaming.

```go
type ConsoleSink struct{}

func (s *ConsoleSink) Send(ctx context.Context, event stream.Event) error {
    switch e := event.(type) {
    case stream.ToolStart:
        fmt.Printf("🔧 Tool: %s\n", e.Data.ToolName)
    case stream.ToolEnd:
        fmt.Printf("✅ Done: %s\n", e.Data.ToolName)
    case stream.Workflow:
        fmt.Printf("📋 %s\n", e.Data.Phase)
    }
    return nil
}

func (s *ConsoleSink) Close(ctx context.Context) error { return nil }

rt := runtime.New(runtime.WithStream(&ConsoleSink{}))
```

Expected stream sample:

```text
📋 started
🔧 Tool: weather.get_weather
✅ Done: weather.get_weather
📋 completed
```
