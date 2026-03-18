# Prompt Overrides with Mongo Store

Production prompt management uses:

- baseline specs in `runtime.PromptRegistry`
- scoped overrides persisted via `features/prompt/mongo`

## Wiring

--- CODE ---
import (
    promptmongo "goa.design/goa-ai/features/prompt/mongo"
    clientmongo "goa.design/goa-ai/features/prompt/mongo/clients/mongo"
    "goa.design/goa-ai/runtime/agent/runtime"
)

promptClient, err := clientmongo.New(clientmongo.Options{
    Client:     mongoClient,
    Database:   "aura",
    Collection: "prompt_overrides", // optional (default is prompt_overrides)
})
if err != nil {
    panic(err)
}

promptStore, err := promptmongo.NewStore(promptClient)
if err != nil {
    panic(err)
}

rt := runtime.New(
    runtime.WithEngine(temporalEng),
    runtime.WithPromptStore(promptStore),
)
--- END CODE ---

## Override Resolution and Rollout

Precedence order:

1. session scope
2. facility scope
3. org scope
4. global scope
5. baseline spec

Recommended rollout:

- register baseline specs first
- roll out broad overrides (`org`) then narrow (`facility`, `session`)
- monitor effective versions via `prompt_rendered` events and `model.Request.PromptRefs`
- roll back by writing a newer override or removing scope-specific overrides
