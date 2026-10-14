# Value pipeline adoption plan

The shared value resolver and primary OpenAPI/CLI consumers are delivered.
The [inventory](value-contract-inventory.md) records verified remaining
ownership and evidence gaps. The accepted framework contract is maintained in
[internal/valuecontract/DESIGN.md](../internal/valuecontract/DESIGN.md); completed
delivery and proof evidence lives beside the implementation.

## Sequence

1. Establish R1's JSON CLI body default representation with bytes, unions and
   mapped keys, retaining the existing HTTP/JSON-RPC and plain-byte controls.
2. Check R2's material location/metadata example precedence at shared HTTP and
   gRPC seams, preserving the location codecs.
3. Resolve R4's inline-schema semantic-failure versus raw-compatibility boundary
   with the existing ambiguous-example and emitted-schema controls.
4. Establish R7's OpenAPI enum/default branch-retention boundary before choosing
   a repair; preserve prepared examples and explicit custom callbacks.
5. Assess R3's unused sampling effects for one carrier class at a time. Remove
   calls only after checking later material examples and public carrier use.
6. Extend R5's independent reference adapter only for a selected supported graph
   with independently encoded inputs and negative controls. Retain guards for
   every unsupported shape.

R1, R2, R4 and R7 are source-level adoption questions. R3 is cleanup and sampling
order evidence; R5 is proof/test coverage, not a demonstrated production defect.
The completed R6 compiler repairs are recorded in the
[comparison harness](../internal/valuecontract/README.md#historical-compiler-repairs).

## Acceptance

For each selected follow-up, define its owner, concrete contract and bounded
regression before production changes. Correct the shared owner and check its
actual consumers. Use direct tests, representative generated output/compilation
and affected transport checks; preserve known-correct codecs and source outcomes.
Update the owning model when its assumptions or correspondence change.

Freeze the candidate and reuse evidence for unchanged source and inputs. Compare
independent generation processes where determinism is relevant. A full corpus
audit needs a separate purpose and cost estimate; it is not a default ticket gate.
Obtain independent review of the exact final diff, then deliver and remove the
completed item from this roadmap. Follow [AGENTS.md](../AGENTS.md).
