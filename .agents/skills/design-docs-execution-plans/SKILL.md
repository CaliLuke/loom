---
name: design-docs-execution-plans
description: Write compact design documents and execution plans for this repo. Use when the user asks for a design doc, implementation plan, milestone plan, execution checklist, or tracking document and wants decisive, low-bloat planning rather than exploratory analysis.
---

# Design Docs And Execution Plans

Use this skill when the user wants a design or execution plan in this repo.

The goal is not to produce a broad analysis memo. The goal is to produce a short working document that can drive implementation, define exit conditions, and be checked off as work progresses.

## Default Shape

Prefer a single document with:

1. A short opening paragraph stating the chosen design
2. Milestones
3. For each milestone:
   - what it is achieving
   - acceptance criteria to exit the milestone
   - a flat checklist

Default to this shape unless the user explicitly asks for something else.

## Non-Negotiables

- Do not pad the document with summary, background, current understanding, frontend note, decisions, risks, open questions, or follow-up sections if the same content can be folded into the main design paragraph or milestone tasks.
- Do not create multiple adjacent sections that restate the same thing in different words.
- Do not leave "open questions" in the document if a reasonable decision has already been made in the conversation.
- Do not create a separate "decisions" section if those decisions can be embedded directly in the design or milestones.
- Do not turn a plan into a changelog or essay.
- Do not use nested bullets.
- Do not treat milestones as loose buckets of tasks. Each milestone must have a goal and exit criteria.
- Do not put tests after implementation when the change can be protected by writing tests first.
- Do not hide process discipline in a final catch-all milestone if it belongs to an earlier milestone.
- Do not write branchy checklist items with `if`, `or`, `and/or`, or fallback phrasing unless the branch is a real explicit decision the user still needs to make.
- Do not write vague checklist items like `verify`, `confirm`, or `investigate` without stating the concrete artifact or outcome they must produce.
- Do not write checklist items that depend on conversation-only context. A fresh agent reading the plan should have enough information to start from the repo and act.
- Do not write review prompts that ask for plan critique without explicitly telling the reviewer to inspect the relevant code and verify that the tasks are actionable against the repo as it exists now.
- Do not name a file, module, package, test, route, or behavior owner unless you inspected it in the current repo or explicitly mark it as unknown.
- Do not include cross-repo work unless the checklist names the repo path and the verification command or concrete proof artifact.
- Do not use placeholder nouns like `queue/store/cache`, `handler/path`, or `module` once code inspection reveals the concrete owner type and file path.
- Do not use `targeted tests`, `verification`, or `checks` without naming the exact command from the correct repo root.

## Writing Rules

- Compress aggressively. The document should read like an execution note, not a review artifact.
- State the chosen design directly. If delete is a hard delete, say that once and move on.
- Turn ambiguity into concrete tasks whenever possible.
- If a technical nuance matters for implementation, encode it in the relevant task.
  Example: "Resolve canonical display ID before cleanup" belongs in the backend milestone, not in a separate decisions appendix.
- Prefer milestone names that describe outcomes, not phases of thinking.
  Good: `Backend Cleanup`
  Bad: `Current Understanding`
- Prefer milestone goals that describe a completed state, not an area of investigation.
- Checklist items should be concrete and markable.
  Good: `Add a graph-level regression test for propose -> delete -> proposal removed`
  Bad: `Think through testing`
- Checklist items should describe one chosen action, not a menu of options.
  Good: `Add frontend store test for evicting proposal rows on node.deleted`
  Bad: `Add a failing repro, targeted test, or otherwise reproducible verification`
- Checklist items should name the concrete locus of change or inspection when it is knowable from the repo.
  Good: `Add a graph regression test in internal/graph/typedb_test.go for propose -> delete -> proposal removed`
  Bad: `Add a backend regression test`
- Checklist items should name the concrete command to run when the step is execution, not design.
  Good: `Run GOFLAGS='-tags=cgo,typedb,typedb_prebuilt' CGO_ENABLED=1 go test -run TestDeleteNodes_... -v ./internal/graph/...`
  Bad: `Run the targeted backend tests`
- If an adjacent test already covers the same path, name that test file explicitly and extend it instead of writing `add or update a test`.
- If code inspection shows the behavior already exists, the plan should switch from presumed implementation to explicit verification.
  Good: `Add a frontend test in <repo path> proving node.deleted invalidates promotions.project`
  Bad: `Implement eviction on node.deleted` when the repo already does that
- Before drafting a non-trivial refactor plan, run a codebase search for the concrete owners, types, or callsites that define the boundary you are changing, record the exact remaining non-test references, and build milestones from that list instead of from memory.
- When a refactor boundary depends on eliminating or preserving specific owners, include a concise inventory of the current non-test hits in the plan and classify each as `remove`, `preserve wrapper`, or `out-of-scope helper`.
- Re-run the same codebase search before finalizing the plan if the draft changed substantially, so the inventory and allowlist still match the current code.
- Reconcile the plan’s inventory against the exact live search output before finalizing. Do not summarize a boundary inventory from memory or from an older search run.
- For every named file in a refactor plan, mark it implicitly or explicitly as `implementation` or `verification-only` based on the inspected current state. Do not leave already-migrated files looking like implementation work.
- Acceptance criteria should be stated as observable truths, not intentions.
  Good: `Deleting an artifact with a pending proposal removes both from backend queries`
  Bad: `Backend seems correct`
- Acceptance criteria should state what makes them true.
  Good: `internal/graph/typedb_test.go contains a regression test where propose -> delete removes both the artifact and its pending proposal from backend queries`
  Bad: `The behavior is covered by tests`
- Acceptance criteria should name the proof artifact when it is knowable.
  Good: `A service-level test in internal/service/... asserts that delete publishes node.deleted`
  Bad: `Event coverage proves delete still emits node.deleted`
- Acceptance criteria should name the exact observable or assertion target, not just the area of behavior.
  Good: `GetNodeDetails returns no node and ListPendingStatusChanges omits the deleted artifact's proposal`
  Bad: `Backend queries are correct`
- Process steps belong where they happen.
  Good: targeted tests immediately after the implementation they verify
  Bad: all validation deferred to the end
- Discovery tasks must name the output they produce.
  Good: `Identify the proposal queue owner and record the concrete module to change`
  Bad: `Identify the proposal queue`
- A fresh agent should not have to guess where to start.
  Include file paths, package names, commands, event names, stores, routes, or tests whenever those are already discoverable.
- If a refactor intentionally allows some remaining usage of a broader type family, state the allowed surviving uses explicitly so the plan does not over-delete legitimate seams.
- If a preserved wrapper remains, state whether it is a thin signature adapter or still owns behavior. Do not call a behavior-owning wrapper an adapter.
- If an exported wrapper is preserved, name at least one current non-test caller that keeps it alive. If there is no current non-test caller, do not justify it as a stable-callsite survivor.
- If a wrapper or helper is removed, name every current caller or test seam that must be updated as part of that removal.
- For route-sensitive work, record both the client callsite and the server route/handler path when both are knowable.
- Do not use conditional checklist items such as `verify unless`, `implement if`, or similar branch points. Choose the action and state the exact observation that justifies a verification-only task.
- Do not defer an architectural choice to a later milestone unless the user must make that choice. A plan should encode one chosen design, not preserve avoidable architecture branches.
- Do not refer to nonexistent artifact sections such as `implementation notes` or `decision appendix` unless the document actually contains them.
- Do not use labels like `IR-owned` or `normalized` as replacement descriptions when downstream code depends on concrete fields. Name the exact replacement fields or symbols.
- Do not name an `existing fixture`, `adjacent test`, or `current helper` unless it is a real reusable symbol, file, or test in the current codebase.
- Do not use non-repo proof sinks such as `commit message`, `acceptance proof note`, or similar narrative artifacts as checklist outputs.

## Milestone Contract

Every milestone should contain exactly these three elements:

1. `Goal`
   - One short sentence describing what the milestone achieves.
2. `Acceptance Criteria`
   - One to three bullets describing the conditions to exit the milestone.
   - Each bullet must be testable or directly checkable.
   - Each bullet should identify the proof artifact when it is already knowable: test file, command, manual check, route, store, event, or output.
   - Avoid vague words like `covered`, `verified`, `handled`, or `works` unless the bullet also states what concrete observation makes that claim true.
3. `Checklist`
   - Flat checkbox list for the work inside the milestone.

If a milestone lacks one of these, the plan is incomplete.

Acceptance criteria are exit conditions, not work items. They should answer:

1. What exact behavior or state must be true?
2. How would a fresh agent prove that it is true?
3. Where does that proof live?

## Ordering Rules

- Put discovery before implementation only when it is truly unresolved.
- If discovery is small, make it an explicit prerequisite milestone or the first checklist items in the relevant milestone.
- Write tests before code when the behavior can be captured in a regression or contract test.
- Run targeted tests immediately after the implementation they verify.
- Put commit, review, and push steps in the milestone where handoff actually happens.
- Do not use a generic final milestone like `Validation` as a dumping ground for unrelated steps.
- If an acceptance criterion depends on a test, the checklist should include writing or updating that test before the implementation step it protects.
- If an acceptance criterion uses `rg`, `grep`, or another structural search as the exit gate, the criterion must also name the explicit allowlist or exclusion pattern for preserved wrappers, tests, generated adapters, or other intentional survivors.
- Structural-search exit criteria must name the intended final survivor files or symbols explicitly, not only categories such as `builder implementation` or `allowed wrappers`.
- Structural-search commands in the plan must themselves be executable and non-duplicative. Prefer one root search or a deduped command/output artifact over overlapping roots that double-count results.
- Do not put an exit criterion in a milestone unless that criterion can be satisfied by the checklist inside that milestone.
- If a branch must exist, make it a named decision outside the checklist before writing the plan. Do not leave execution branches inside checklist items.
- If a fresh reviewer is asked to critique the plan, instruct them to inspect the current code and judge whether each checklist item is executable without hidden context.
- Preserve current public contracts explicitly when a plan changes internal mechanics.
  If IDs, return values, event payloads, or route behavior already have tests, state whether the plan preserves or changes them.
- When preserved public wrappers remain, name every downstream caller or dependent test that keeps those wrappers alive before using a structural-search boundary gate.
- For every preserved wrapper, name every current caller that keeps it alive.
- Signature-preservation criteria should point at the declaration site, not only at caller files.
- If the plan preserves an existing outward contract while changing internals, say that explicitly in the milestone acceptance criteria.
- Verification-only tasks must cite the exact current observation and the exact proof location that keep the work out of implementation scope.

## Default Document Template

```md
# <Title>

<One short paragraph describing the chosen design and any critical constraints.>

## Milestones

### Milestone 1: <Outcome>

Goal: <What this milestone achieves>

Acceptance Criteria

- <Observable exit condition>
- <Observable exit condition>

Checklist

- [ ] <Concrete task>
- [ ] <Concrete task>

### Milestone 2: <Outcome>

Goal: <What this milestone achieves>

Acceptance Criteria

- <Observable exit condition>

Checklist

- [ ] <Concrete task>
- [ ] <Concrete task>
```

## What To Fold Into Tasks

Fold these into the relevant milestone instead of creating separate sections:

- identifier handling details
- event contract details
- backend vs frontend ownership
- validation requirements
- "we are not changing X" constraints
- test-first sequencing
- commit/review/push discipline
- chosen file or module targets when discovery is already complete
- concrete commands or tests when they are already known
- cross-repo repo paths and proof artifacts when work spans repos
- adjacent test files that should be extended instead of invented from scratch

Example:

- Instead of a separate decision section saying "delete remains a hard delete," write:
  `Keep artifact delete as a hard delete`
- Instead of a separate ambiguity section saying "cleanup must use display ID, not IID," write:
  `Resolve each delete input to the canonical display ID before explicit proposal cleanup`
- Instead of putting review and push in a generic footer, write them in the final delivery milestone checklist
- Instead of saying "we should add tests," write:
  `Write failing graph regression test for propose -> delete -> proposal removed`
- Instead of writing `cover both X and Y, or cover Z if canonical`, decide the contract first and write the exact test to add
- Instead of writing `verify whether proposal state is already invalidated`, write:
  `Inspect the proposal store event handler and record whether node.deleted already evicts proposal rows`
- Instead of writing `update the frontend handler`, write:
  `Update <store/module path> to evict proposal rows on node.deleted`
- Instead of asking a reviewer to `check the plan`, ask:
  `Inspect the referenced code paths and judge whether each task names a concrete locus, output, and verification step`
- Instead of writing acceptance criteria like `backend event coverage proves delete still emits node.deleted`, write:
  `internal/service/<test file> contains a test that asserts deleting an artifact publishes node.deleted`
- Instead of writing acceptance criteria like `canonical display ID handling is covered`, write:
  `A delete-by-IID regression test proves DeleteNodes resolves the canonical display ID before proposal cleanup`
- Instead of assuming missing frontend behavior, write:
  `Inspect <frontend repo path> and choose between verification work and implementation work based on the current cache invalidation code`
- Instead of writing `run targeted frontend checks`, write:
  `Run pnpm vitest <exact test file> from <frontend repo root>`

## When To Add Extra Structure

Add one extra section only if it materially changes implementation:

- `Scope` when the user explicitly wants in/out boundaries
- `Acceptance Criteria` when the user needs approval gates
- `Rollout` when deployment sequencing matters

If you add one of these, keep it short.

Do not add a document-level `Acceptance Criteria` section if the same contract is already expressed at the milestone level.

## Repo-Specific Expectations

- Favor milestone checklists over narrative prose.
- Treat milestone structure as mandatory, not optional.
- Use the repo's real terminology from code and routes.
- Keep the document short enough that a human can scan it quickly and start implementation.
- If the user criticizes the format, simplify further instead of defending the original structure.
- A meticulous plan chooses actions. It does not preserve avoidable branches for later.

## Review Prompt Rule

When asking another agent to critique a plan produced with this skill, the prompt should explicitly require all of the following:

1. Inspect the relevant code, not just the plan text.
2. Check whether each checklist item is actionable by a fresh agent with no conversation context.
3. Call out vague tasks that do not name a file, package, module, event, command, or test when that detail is already discoverable.
4. Call out checklist items whose acceptance criteria are not backed by a named verification step.
5. Call out plan items that assume missing behavior when the inspected code already implements that behavior.
6. Compare any structural-search allowlist in the plan against the current codebase search results and flag mismatches.
7. Flag any conditional checklist item or verification-only task that does not name the exact observation keeping it out of implementation scope.
8. Critique only. No delegation, no implementation, no meta-summary.

Use a review pass by default for non-trivial plans. A single planner should not trust themselves to catch all structural defects unaided.

## When To Use Review Prompts

Run an external review prompt when any of these are true:

- the plan spans more than one milestone
- the plan touches more than one layer or repo
- the plan includes backend plus frontend work
- the plan names specific files, tests, routes, stores, or events
- the plan will be used as an execution artifact for someone other than the planner
- the planner had to make architectural or contract decisions while drafting it

You may skip the external review only for very small plans where all of these are true:

- one milestone only
- one repo only
- one obvious implementation locus
- no cross-layer coordination
- no ambiguity about tests, commands, or ownership

If you skip the review, do a local self-check against the same standards before finalizing.

## How To Use Review Prompts

Use this loop:

1. Draft the plan using this skill.
2. Choose the review prompt type that matches the plan shape.
3. Give the reviewer:
   - the plan path
   - the skill path
   - the repo path or repo paths
   - the concrete code paths they must inspect
4. Ask for critique only.
5. Update the plan first to fix plan-local defects.
6. Update the skill if the reviewer found a weakness the skill should have prevented.
7. If the changes were substantial, run one fresh review pass again.

Do not outsource the loop itself to the reviewer. The reviewer critiques; the planner integrates.

## Which Review Prompt To Choose

- Use `Plan Review Prompt` for most non-trivial plans in a single repo.
- Use `Actionability-Focused Review Prompt` when the main risk is that checklist items or acceptance criteria are still too vague for a fresh agent.
- Use `Cross-Repo Review Prompt` when the plan spans multiple repos or names work outside the current repo.

If in doubt:

- start with `Plan Review Prompt`
- switch to `Actionability-Focused Review Prompt` if the first review says the plan is still vague
- use `Cross-Repo Review Prompt` as soon as the plan includes another repo path

## Review Prompt Templates

Use one of these prompts after drafting a plan.

### Plan Review Prompt

```text
Review these files in <repo path>:
1. <skill path>
2. <plan path>

Requirements:
- Inspect the current code where the plan points. Do not review the plan text in isolation.
- Judge whether the plan is executable by a fresh agent with no conversation context.
- Call out vague checklist items, weak acceptance criteria, missing file/module specificity, sequencing defects, wrong behavior loci, and places where the skill still allows weak plans.
- Compare any structural-search boundary allowlist against the current codebase search results.
- Flag any conditional checklist item or verification-only task that does not name the exact observation that keeps it out of implementation scope.
- Critique only. Do not edit files. Do not delegate. Do not spawn sub-agents.

Return exactly these sections:
1. Remaining defects in the plan
2. Remaining ambiguity in the plan
3. Remaining gaps in the skill
4. Specific skill or plan fixes to apply next
```

### Actionability-Focused Review Prompt

Use this when the main risk is that a fresh agent still would not know where to act.

```text
Inspect <plan path> against the current code in <repo path>.

Requirements:
- Inspect the concrete files, tests, commands, routes, stores, and modules referenced by the plan.
- Check whether each checklist item is directly actionable by a fresh agent without hidden conversation context.
- Flag any task that does not name a concrete locus of work or a concrete proof artifact when that detail is already discoverable.
- Flag any acceptance criterion that does not state what makes it true.
- Compare any structural-search boundary allowlist against the current codebase search results.
- Flag any conditional checklist item or verification-only task that does not name the exact observation that keeps it out of implementation scope.
- Critique only. No edits, no delegation, no implementation.

Return exactly these sections:
1. Non-actionable checklist items
2. Weak acceptance criteria
3. Missing repo-specific detail
4. Exact prompt or skill rules needed to prevent those defects next time
```

### Cross-Repo Review Prompt

Use this when a plan spans multiple repos.

```text
Inspect <plan path> against these repos:
- <repo path A>
- <repo path B>

Requirements:
- Verify every cross-repo checklist item names the repo path, concrete module or test locus, and the proof artifact or command that closes it.
- Flag any milestone that cannot be executed from the named repos alone.
- Flag any plan item that assumes missing behavior without checking the current code first.
- Critique only. No edits, no delegation, no implementation.

Return exactly these sections:
1. Broken cross-repo assumptions
2. Missing repo-path or module specificity
3. Missing verification steps
4. Skill rules to add or tighten
```
