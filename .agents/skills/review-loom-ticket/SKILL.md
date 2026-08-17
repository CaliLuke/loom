---
name: review-loom-ticket
description: Evaluate Loom GitHub bug reports and feature requests for factual validity, architectural ownership, and coherence with Loom framework principles. Use when reviewing new or existing Loom issues, deciding whether a reported bug is real, assessing whether a proposed capability belongs in Loom, comparing multiple tickets, or recommending accept, reframe, duplicate, defer, or close.
---

# Review a Loom Ticket

Review tickets as evidence-backed design proposals. Keep validity and framework
fit separate: a report can be true but propose the wrong solution.

## Workflow

1. Read the repository `AGENTS.md` and `../loom-framework/SKILL.md` completely.
2. Fetch the issue, labels, and discussion. For example:

   ```bash
   gh issue view <number> --repo CaliLuke/loom \
     --json number,title,body,labels,comments,state,url
   ```

   Search closed issues and pull requests for prior decisions or duplicates.
3. Locate the claimed behavior in canonical `origin/main`, plus any supplied
   fix branch, without changing the user's checkout. Read the owning code,
   direct tests, public docs, consumer skill, and active roadmap items. Search
   before inferring intent.
4. For a bug, reproduce the smallest supported contract or identify the exact
   missing regression. Distinguish a framework defect from misuse, an explicit
   limitation, stale behavior, or a generated-code symptom whose cause lives
   elsewhere.
5. For a feature, identify:
   - the repeated application workaround or concrete risk
   - the generic framework behavior that would remove it
   - a real consumer or contract that proves the need
   - the layer that owns the behavior
   - policy that must remain application-owned
6. Check the proposal against Loom's boundaries:
   - The design DSL is the source of truth.
   - Evaluated semantics belong in `expr`; shared transport decisions belong in
     shared IR; stable protocol execution belongs in handwritten runtimes.
   - Generated code supplies typed adapters and declarations, not duplicated
     policy or protocol state machines.
   - HTTP, gRPC, and JSON-RPC semantics remain distinct unless a deliberate
     shared core owns the behavior.
   - Fail closed when Loom cannot preserve the authored contract.
   - Preserve type safety, backward compatibility, and authored contract
     semantics; prefer an explicit limitation to lossy support.
   - Prefer a narrow generic capability over an application-specific switch,
     compatibility shim, or generated-file workaround.
7. State what evidence would change the verdict. Do not edit code, close issues,
   or post comments unless the user separately asks for those actions.

## Verdicts

Assign one validity verdict and one design-fit verdict.

- Validity: `confirmed`, `likely`, `not reproduced`, `duplicate`, `outdated`, or
  `not a framework bug`.
- Design fit: `fits`, `fits if reframed`, `needs a product decision`, or `does
  not fit`.

Use `confirmed` only with a reproduction, failing test, or direct code-path
proof. Do not label a feature request false merely because it is not a bug.

## Output

For each ticket, report:

1. **Verdict** — validity, design fit, and confidence.
2. **Evidence** — issue claim, current behavior, and precise code/docs/tests.
3. **Design assessment** — correct ownership layer and any conflict with Loom
   invariants.
4. **Recommendation** — accept, reframe, merge as duplicate, defer, or close.
5. **Acceptance criteria** — smallest direct regression plus generated,
   compile, contract, or integration coverage required by the risk.
6. **Open questions** — only facts that materially affect the disposition.

When reviewing several tickets, lead with a compact comparison table, then
expand only tickets whose verdict needs explanation. Separate confirmed facts
from inferences and cite GitHub URLs and local file paths.
