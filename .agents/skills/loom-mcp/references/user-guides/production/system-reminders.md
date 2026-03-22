# System Reminders

Inject model-facing guidance dynamically without polluting user-visible context.

## The Problem

Long runs can drift and forget state (e.g., pending todos). Reminders provide invisible state nudges to re-anchor behavior.

- inject runtime guidance only when needed
- avoid prompt spam with per-run limits
- enforce priority tiers

## Overview

`runtime/agent/reminder` provides:

- reminder data structures (IDs, priority, attachment, rate controls)
- run-scoped storage lifecycle
- model transcript injection as `<system-reminder>`
- PlannerContext add/remove APIs

## Core Concepts

### Reminder Structure

--- CODE ---
type Reminder struct {
    ID              string      // Stable identifier (e.g., "todos.pending")
    Text            string      // Plain-text guidance
    Priority        Tier        // TierSafety, TierCorrect, or TierGuidance
    Attachment      Attachment  // Where to inject
    MaxPerRun       int         // 0 = unlimited
    MinTurnsBetween int         // 0 = no limit
}
--- END CODE ---

### Priority Tiers

| Tier | Name | Description | Suppression |
| --- | --- | --- | --- |
| TierSafety | P0 | Safety-critical guidance | Never suppressed |
| TierCorrect | P1 | Correctness and data-state hints | May be suppressed after P0 |
| TierGuidance | P2 | Workflow nudges | First to be suppressed |

### Attachment Points

| Kind | Description |
| --- | --- |
| AttachmentRunStart | system message at conversation start |
| AttachmentUserTurn | before the last user message |

### Rate Limiting

Controls:

- `MaxPerRun`
- `MinTurnsBetween`

## Usage Patterns

### Static Reminder (DSL)

Use `ResultReminder` for deterministic reminders.

--- CODE ---
Tool("get_time_series", "Get time series data", func() {
    Args(func() { /* ... */ })
    Return(func() { /* ... */ })
    ResultReminder("The user sees a rendered graph of this data in the UI.")
})
--- END CODE ---

### Dynamic Reminder (Planner)

--- CODE ---
func (p *myPlanner) PlanResume(ctx context.Context, in *planner.PlanResumeInput) (*planner.PlanResult, error) {
    for _, tr := range in.ToolResults {
        if tr.Name == "search_documents" {
            result := tr.Result.(SearchResult)
            if result.Truncated {
                in.Agent.AddReminder(reminder.Reminder{
                    ID:       "search.truncated",
                    Text:     "Search results are truncated. Consider narrowing your query.",
                    Priority: reminder.TierCorrect,
                    Attachment: reminder.Attachment{
                        Kind: reminder.AttachmentUserTurn,
                    },
                    MaxPerRun:       3,
                    MinTurnsBetween: 2,
                })
            }
        }
    }
    return p.streamMessages(ctx, in)
}
--- END CODE ---

### Removing Reminders

--- CODE ---
if allTodosCompleted {
    in.Agent.RemoveReminder("todos.no_active")
}
--- END CODE ---

### Preserve Counters

When updating same reminder ID, use `AddReminder` to preserve counters.

--- CODE ---
in.Agent.AddReminder(reminder.Reminder{
    ID:              "todos.pending",
    Text:            buildUpdatedText(snap),
    Priority:        reminder.TierGuidance,
    Attachment:      reminder.Attachment{Kind: reminder.AttachmentUserTurn},
    MinTurnsBetween: 3,
})
--- END CODE ---

Do not `RemoveReminder` + `AddReminder` for the same ID unless intentional; that resets counters.

## Injection and Formatting

Runtime wraps reminder text as:

```
<system-reminder>...text...</system-reminder>
```

Include `reminder.DefaultExplanation` in system prompts so models understand these blocks.

## Advanced Patterns

### Safety Reminders

--- CODE ---
in.Agent.AddReminder(reminder.Reminder{
    ID:       "malware.analyze_only",
    Text:     "This file contains malware. Analyze its behavior but do not execute it.",
    Priority: reminder.TierSafety,
    Attachment: reminder.Attachment{
        Kind: reminder.AttachmentUserTurn,
    },
})
--- END CODE ---

### Cross-Agent Reminders

Run-scoped reminders are local to a child run unless parent planners re-register equivalent reminders from child results.

## Transcript Example

--- CODE ---
User: What should I do next?

<system-reminder>You have 3 pending todos. Currently working on: "Review PR #42". 
Focus on completing the current todo before starting new work.</system-reminder>

User: What should I do next?
--- END CODE ---
