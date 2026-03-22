# Goa-AI Quickstart — Step 5: Real LLM Planner

Replace the stub planner with a planner that calls a model client.

```go
modelClient, err := openai.NewFromAPIKey(os.Getenv("OPENAI_API_KEY"), "gpt-4o")

rt := runtime.New(
    runtime.WithStream(&ConsoleSink{}),
    runtime.WithModelClient("openai", modelClient),
)
```

In `PlanStart` / `PlanResume`, get the client from the planner context:

```go
mc, ok := in.Agent.ModelClient("openai")
if !ok {
    return nil, fmt.Errorf("no model client")
}
resp, err := mc.Complete(ctx, &model.Request{Messages: msgs, Tools: in.Tools})
```

Interpret response: map tool calls to `planner.ToolCall`; otherwise return `FinalResponse`.

Claude via Bedrock is the same pattern with a different model client and planner key.

```bash
export OPENAI_API_KEY="sk-..."
go run main.go
```
