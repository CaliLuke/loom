TESTING & TROUBLESHOOTING
=========================

Learn how to test agents, planners, and tools, and troubleshoot common issues.

Testing & Troubleshooting
=========================

This guide covers testing strategies for Goa-AI agents and solutions to common issues.


Testing Agents
==============

Testing with the In-Memory Engine
---------------------------------

The in-memory engine is ideal for testing because it:

- Requires no external dependencies (no Temporal)
- Executes synchronously for predictable test behavior
- Provides fast feedback during development

--- CODE ---
func TestChatAgent(t *testing.T) {
    // Create runtime with in-memory engine (default)
    rt := runtime.New()
    ctx := context.Background()
    
    // Register agent with test planner
    err := chat.RegisterChatAgent(ctx, rt, chat.ChatAgentConfig{
        Planner: &TestPlanner{},
    })
    require.NoError(t, err)
    
    // Run agent
    client := chat.NewClient(rt)
    out, err := client.Run(
        ctx,
        "test-session",
        []*model.Message{{
            Role:  model.ConversationRoleUser,
            Parts: []model.Part{model.TextPart{Text: "Hello"}},
        }},
    )
    require.NoError(t, err)
    
    // Assert on output
    assert.NotEmpty(t, out.RunID)
    assert.NotNil(t, out.Final)
}
--- END CODE ---


Testing Planners with Mock Model Clients
----------------------------------------

Isolate planner logic by mocking the model client:

--- CODE ---
type MockModelClient struct {
    responses []*model.Message
    callCount int
}

func (m *MockModelClient) Generate(ctx context.Context, req *model.Request) (*model.Response, error) {
    if m.callCount >= len(m.responses) {
        return nil, fmt.Errorf("no more mock responses")
    }
    resp := &model.Response{
        Message: m.responses[m.callCount],
    }
    m.callCount++
    return resp, nil
}

func (m *MockModelClient) Stream(ctx context.Context, req *model.Request) (model.Streamer, error) {
    // Return a mock streamer for streaming tests
    return &MockStreamer{response: m.responses[m.callCount]}, nil
}

func TestPlannerWithMockClient(t *testing.T) {
    mockClient := &MockModelClient{
        responses: []*model.Message{
            {
                Role: model.ConversationRoleAssistant,
                Parts: []model.Part{
                    model.TextPart{Text: "I'll search for that."},
                    model.ToolUsePart{
                        ID:    "call-1",
                        Name:  "search",
                        Input: json.RawMessage(`{"query": "test"}`),
                    },
                },
            },
        },
    }
    
    planner := &MyPlanner{client: mockClient}
    
    input := &planner.PlanInput{
        Messages: []*model.Message{{
            Role:  model.ConversationRoleUser,
            Parts: []model.Part{model.TextPart{Text: "Search for test"}},
        }},
        Tools: []tools.ToolSpec{/* ... */},
    }
    
    result, err := planner.PlanStart(context.Background(), input)
    require.NoError(t, err)
    
    // Assert planner returned tool calls
    assert.NotNil(t, result.ToolCalls)
    assert.Len(t, result.ToolCalls, 1)
    assert.Equal(t, "search", string(result.ToolCalls[0].Name))
}
--- END CODE ---


Testing Tools in Isolation
--------------------------

Test tool executors independently from the agent:

--- CODE ---
func TestSearchToolExecutor(t *testing.T) {
    // Create executor with mock dependencies
    mockSearchService := &MockSearchService{
        results: []string{"doc1", "doc2", "doc3"},
    }
    executor := &SearchExecutor{searchService: mockSearchService}
    
    // Create test tool call
    meta := &runtime.ToolCallMeta{
        RunID:      "test-run",
        SessionID:  "test-session",
        TurnID:     "test-turn",
        ToolCallID: "call-1",
    }
    
    call := &planner.ToolRequest{
        Name:    specs.Search,
        Payload: json.RawMessage(`{"query": "test", "limit": 5}`),
    }
    
    // Execute tool
    result, err := executor.Execute(context.Background(), meta, call)
    require.NoError(t, err)
    
    // Assert on result
    assert.Nil(t, result.Error)
    assert.NotNil(t, result.Result)
    
    // Unmarshal and verify typed result
    searchResult, ok := result.Result.(*specs.SearchResult)
    require.True(t, ok)
    assert.Len(t, searchResult.Documents, 3)
}
--- END CODE ---


Testing Tool Validation and Retry Hints
---------------------------------------

Verify that tools return proper errors and hints for invalid input:

--- CODE ---
func TestToolValidationReturnsHint(t *testing.T) {
    executor := &SearchExecutor{}
    
    // Invalid payload - missing required field
    call := &planner.ToolRequest{
        Name:    specs.Search,
        Payload: json.RawMessage(`{"limit": 5}`), // missing "query"
    }
    
    result, err := executor.Execute(context.Background(), &runtime.ToolCallMeta{}, call)
    require.NoError(t, err) // Executor should not return error
    
    // Should return ToolError with RetryHint
    assert.NotNil(t, result.Error)
    assert.NotNil(t, result.RetryHint)
    assert.Equal(t, planner.RetryReasonMissingFields, result.RetryHint.Reason)
    assert.Contains(t, result.RetryHint.MissingFields, "query")
}
--- END CODE ---


Testing Agent Composition
-------------------------

Test agent-as-tool scenarios:

--- CODE ---
func TestAgentComposition(t *testing.T) {
    rt := runtime.New()
    ctx := context.Background()
    
    // Register provider agent
    err := planner.RegisterPlannerAgent(ctx, rt, planner.PlannerAgentConfig{
        Planner: &PlanningPlanner{},
    })
    require.NoError(t, err)
    
    // Register consumer agent that uses provider's tools
    err = orchestrator.RegisterOrchestratorAgent(ctx, rt, orchestrator.OrchestratorAgentConfig{
        Planner: &OrchestratorPlanner{},
    })
    require.NoError(t, err)
    
    // Run orchestrator - it should invoke planner agent as a tool
    client := orchestrator.NewClient(rt)
    out, err := client.Run(
        ctx,
        "test-session",
        []*model.Message{{
            Role:  model.ConversationRoleUser,
            Parts: []model.Part{model.TextPart{Text: "Create a plan for X"}},
        }},
    )
    require.NoError(t, err)
    
    // Verify child run was created
    assert.Greater(t, out.ChildrenCount, 0)
}
--- END CODE ---


Troubleshooting
===============


Common Errors
-------------


“registration closed” Error
---------------------------

Symptom:

--- CODE ---
error: registration closed: cannot register agent after runtime start
--- END CODE ---

Cause: Attempting to register an agent after the runtime has started processing runs.

Solution: Register all agents before starting any runs:

--- CODE ---
rt := runtime.New()

// ✓ Register all agents first
chat.RegisterChatAgent(ctx, rt, chatConfig)
planner.RegisterPlannerAgent(ctx, rt, plannerConfig)

// ✓ Then start runs
client := chat.NewClient(rt)
out, err := client.Run(ctx, messages, opts...)
--- END CODE ---


“missing session ID” Error
--------------------------

Symptom:

--- CODE ---
error: missing session ID: session ID is required for run
--- END CODE ---

Cause: Starting a run without providing a session ID.

Solution: Always provide a session ID as the required positional argument:

--- CODE ---
// ✗ Wrong - no session ID
out, err := client.Run(ctx, "", messages)

// ✓ Correct - session ID provided
out, err := client.Run(ctx, "session-123", messages)
--- END CODE ---

Tip: For testing, use a fixed session ID. For production, generate unique session IDs per conversation.


Policy Violation Errors
-----------------------

Symptom:

--- CODE ---
error: policy violation: max tool calls exceeded (10/10)
--- END CODE ---

Cause: The agent exceeded the configured MaxToolCalls limit.

Solutions:

1. Increase the limit if the use case legitimately requires more tool calls:

--- CODE ---
RunPolicy(func() {
    DefaultCaps(MaxToolCalls(20)) // Increase from default
})
--- END CODE ---

2. Improve planner efficiency to use fewer tool calls:
   - Batch operations where possible
   - Use more specific tool calls
   - Improve prompt engineering
3. Check for infinite loops in planner logic that repeatedly calls the same tool.

Symptom:

--- CODE ---
error: policy violation: max consecutive failed tool calls exceeded (3/3)
--- END CODE ---

Cause: Multiple consecutive tool calls failed.

Solutions:

1. Fix the underlying tool errors - check tool executor logs
2. Improve retry hints so the planner can self-correct
3. Increase the limit if transient failures are expected:

--- CODE ---
RunPolicy(func() {
    DefaultCaps(MaxConsecutiveFailedToolCalls(5))
})
--- END CODE ---

Symptom:

--- CODE ---
error: policy violation: time budget exceeded (2m0s)
--- END CODE ---

Cause: The agent run exceeded the configured TimeBudget.

Solutions:

1. Increase the budget for long-running operations:

--- CODE ---
RunPolicy(func() {
    TimeBudget("10m")
})
--- END CODE ---

2. Use Timing for fine-grained control:

--- CODE ---
RunPolicy(func() {
    Timing(func() {
        Budget("10m")  // Overall budget
        Plan("1m")     // Per-plan timeout
        Tools("2m")    // Per-tool timeout
    })
})
--- END CODE ---

3. Optimize tool execution to complete faster.

“unknown tool” Error
--------------------

Symptom:

--- CODE ---
error: unknown tool: orchestrator.helpers.search
--- END CODE ---

Cause: The planner requested a tool that isn’t registered.

Solutions:

1. Verify toolset registration - ensure the toolset is registered with the agent:

--- CODE ---
Agent("chat", "Chat agent", func() {
    Use(HelpersToolset) // Make sure this is included
})
--- END CODE ---

2. Check tool name spelling - tool names are case-sensitive and use qualified names.
3. Regenerate code after DSL changes:

--- CODE ---
goa gen example.com/project/design
--- END CODE ---


“invalid payload” Error
-----------------------

Symptom:

--- CODE ---
error: invalid payload: json: cannot unmarshal string into Go struct field SearchPayload.limit of type int
--- END CODE ---

Cause: The LLM provided a payload that doesn’t match the tool’s schema.

Solutions:

1. Return a RetryHint from the executor so the planner can self-correct:

--- CODE ---
if err != nil {
    return &planner.ToolResult{
        Error: planner.NewToolError("invalid payload"),
        RetryHint: &planner.RetryHint{
            Reason:       planner.RetryReasonInvalidArguments,
            Tool:         call.Name,
            ExampleInput: map[string]any{"query": "example", "limit": 10},
            Message:      "limit must be an integer",
        },
    }, nil
}
--- END CODE ---

2. Improve tool descriptions to clarify expected types.
3. Add examples to the DSL:

--- CODE ---
Args(func() {
    Attribute("limit", Int, "Maximum results", func() {
        Example(10)
        Minimum(1)
        Maximum(100)
    })
})
--- END CODE ---


Debugging Tips
--------------

Enable Debug Logging
--------------------

--- CODE ---
import "goa.design/goa-ai/runtime/agent/runtime"

rt := runtime.New(
    runtime.WithLogger(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelDebug,
    }))),
)
--- END CODE ---


Subscribe to Events for Debugging
---------------------------------

--- CODE ---
type DebugSink struct{}

func (s *DebugSink) Send(ctx context.Context, event stream.Event) error {
    fmt.Printf("[%s] %s run=%s session=%s payload=%v\n",
        time.Now().Format(time.RFC3339),
        event.Type(),
        event.RunID(),
        event.SessionID(),
        event.Payload(),
    )
    return nil
}

func (s *DebugSink) Close(ctx context.Context) error { return nil }

// Wire the sink into the runtime to observe all stream events.
rt := runtime.New(runtime.WithStream(&DebugSink{}))
--- END CODE ---


Inspect Tool Specs at Runtime
-----------------------------

--- CODE ---
// List all registered tools
for _, spec := range rt.ToolSpecsForAgent(chat.AgentID) {
    fmt.Printf("Tool: %s\n", spec.Name)
    fmt.Printf("  Description: %s\n", spec.Description)
    fmt.Printf("  Payload Schema: %s\n", spec.Payload.Schema)
}
--- END CODE ---


Next Steps
==========

- DSL Reference - Complete DSL function reference
- Runtime - Understand runtime architecture
- Production - Deploy with Temporal and streaming UI
