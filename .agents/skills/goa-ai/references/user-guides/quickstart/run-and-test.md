# Goa-AI Quickstart — Step 2: Stub Planner Flow

Create `main.go` and run a fully in-memory stub flow.

```go
package main

import (
    "context"
    "fmt"

    assistant "quickstart/gen/demo/agents/assistant"
    "goa.design/goa-ai/runtime/agent/model"
    "goa.design/goa-ai/runtime/agent/planner"
    "goa.design/goa-ai/runtime/agent/runtime"
)

type StubPlanner struct{}

func (p *StubPlanner) PlanStart(ctx context.Context, in *planner.PlanInput) (*planner.PlanResult, error) {
    return &planner.PlanResult{
        ToolCalls: []*planner.ToolCall{{
            Name:    "weather.get_weather",
            Payload: []byte(`{"city": "Tokyo"}`),
        }},
    }, nil
}

func (p *StubPlanner) PlanResume(ctx context.Context, in *planner.PlanResumeInput) (*planner.PlanResult, error) {
    return &planner.PlanResult{
        FinalResponse: &planner.FinalResponse{
            Message: &model.Message{
                Role:  model.ConversationRoleAssistant,
                Parts: []model.Part{model.TextPart{Text: "Tokyo is 22°C and sunny!"}},
            },
        },
    }, nil
}

type StubExecutor struct{}

func (e *StubExecutor) Execute(ctx context.Context, meta runtime.ToolCallMeta, req *planner.ToolRequest) (*planner.ToolResult, error) {
    return &planner.ToolResult{
        Name:   req.Name,
        Result: map[string]any{"temperature": 22, "conditions": "Sunny"},
    }, nil
}

func main() {
    ctx := context.Background()

    rt := runtime.New()
    sessionID := "demo-session"
    if _, err := rt.CreateSession(ctx, sessionID); err != nil {
        panic(err)
    }

    err := assistant.RegisterAssistantAgent(ctx, rt, assistant.AssistantAgentConfig{
        Planner:  &StubPlanner{},
        Executor: &StubExecutor{},
    })
    if err != nil {
        panic(err)
    }

    client := assistant.NewClient(rt)
    out, err := client.Run(ctx, sessionID, []*model.Message{{
        Role:  model.ConversationRoleUser,
        Parts: []model.Part{model.TextPart{Text: "What's the weather?"}},
    }})
    if err != nil {
        panic(err)
    }

    fmt.Println("RunID:", out.RunID)
    if out.Final != nil {
        for _, p := range out.Final.Parts {
            if tp, ok := p.(model.TextPart); ok {
                fmt.Println("Assistant:", tp.Text)
            }
        }
    }
}
```

Run:

```bash
go mod tidy && go run main.go
```

Expected output:

```bash
RunID: demo.assistant-abc123
Assistant: Tokyo is 22°C and sunny!
```

### Optional prompt store

```go
import (
    promptmongo "goa.design/goa-ai/features/prompt/mongo"
    clientmongo "goa.design/goa-ai/features/prompt/mongo/clients/mongo"
)

promptClient, _ := clientmongo.New(clientmongo.Options{
    Client:   mongoClient,
    Database: "quickstart",
})
promptStore, _ := promptmongo.NewStore(promptClient)

rt := runtime.New(
    runtime.WithPromptStore(promptStore),
)
```
