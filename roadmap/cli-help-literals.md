# Literal generated CLI help — #698

Parent: pushed `055c2eb0a`. HTTP and JSON-RPC share the Kong command renderer in
`http/codegen/cli_sections.go`. Authored descriptions and displayed defaults are
plain text, but Kong interprets their `${name}` and `$$` sequences as templates.
An unknown variable prevents construction of the entire command parser; a known
variable or default expression silently changes help text.

1. [x] Reproduce all three description scopes and displayed defaults through the
   actual emitted tags and pinned Kong parser, with ordinary text controls.
2. [x] Preserve HTTP and JSON-RPC generated module compiler/parser witnesses.
3. [x] Review the literal-tag boundary and bounded acceptance before production.
4. [x] Encode all authored help text once at the shared Kong renderer boundary;
   keep semantic descriptions and payload defaults unchanged.
5. [x] Compare bounded parent/candidate/independent-repeat artifacts, compile and
   test both generated clients; verify default and explicit-value payloads.
6. [ ] Obtain exact independent review, commit, push, close and preserve receipts.

The owner is the shared renderer's handoff to Kong help tags. Escape literal
Kong interpolation syntax only there, after composing flag description/default
help. Service and method tags must use the same encoding. Do not alter shared
CommandData/FlagData descriptions (other consumers print them directly), define
fake variables, recover parser errors or modify application payload defaults.
Pinned Kong v1.16.1 interprets `$$` as one literal dollar; doubling every dollar
preserves single dollars, repeated dollars and complete/partial interpolation
forms uniformly. Go struct-tag quoting remains Jennifer's responsibility.

Acceptance covers ordinary text, `${unknown}`, `${name=fallback}`, `${default}`,
`$$`, `$${name}`, lone/trailing dollars, quotes, backslashes and line breaks.
Inspect actual Kong model help after initialization and valid parsing, not just
escaped generated strings. Generated HTTP and JSON-RPC tests must verify default
payload text survives exactly and explicit input is not interpreted. Compare
selected ordinary and adversarial designs byte for byte; only help tag literals
may change, and independent repetitions must match. Reuse the owning CLI tests
and existing fixture regeneration controls; no exhaustive design corpus.

This is a synchronous text-encoding boundary without new ordering, ownership or
protocol state. A state model would duplicate the string encoder; real pinned
parser and generated module tests establish the relevant contract.

## Frozen readiness evidence

The 36 direct emitted-tag/pinned-parser cases reproduce 20 failures: unknown or
reserved variables prevent parser construction, default interpolation changes
text, and doubled dollars collapse. Sixteen ordinary/quoting/incomplete-template
controls pass. Four complete generated modules cover HTTP and JSON-RPC with
ordinary and adversarial text. Both ordinary modules pass race tests and vet;
both adversarial modules compile but fail Kong initialization. Exact generated
artifacts are preserved under `/tmp/loom-698-readiness` against clean delivered
`055c2eb0a`, before production edits.

Initial exploratory harness attempts omitted JSON-RPC aggregate CLI generation
and expected attribute descriptions/defaults to appear on a whole-body JSON flag.
Those setup failures are excluded. The frozen specimen maps the same field to a
query parameter in both transports, giving matching named-flag contracts. Direct
renderer cases independently cover descriptions at all three help scopes.

Candidate acceptance adds no new model or dependency: use a private literal-help
encoder after flag help composition and for both command help tags. Do not touch
runtime Kong configuration or any payload values. Regenerate these four modules
in separate processes; only adversarial aggregate CLI help tags may change. The
ordinary modules and all other artifacts must stay exact. Final verification
includes existing owning CLI goldens and the two ticktock fixture comparisons.

Independent readiness and mechanical declaration-order re-review approved the
boundary. The implementation adds `kongHelpLiteral` at the three shared help-tag
emission paths; no runtime, payload/default, semantic description or wire change.

Final acceptance: all 36 emitted-tag/parser cases and four generated modules pass;
module race tests and vet cover both ordinary and adversarial designs. The 56
captured artifacts repeat byte for byte in fresh Go processes. Only the two
adversarial aggregate CLI files change, solely in service, method and default-help
tags; all payload expressions and ordinary artifacts stay exact. The generation
repeat uses a recorded test overlay that omits only redundant module build/test
commands. HTTP and JSON-RPC ticktock fixture regeneration yields the same 39
checked-in artifacts at parent, candidate and independent repeat; parent and
candidate fixture modules build and vet successfully. Existing owning CLI golden
checks pass unchanged, and scoped lint reports zero issues. Final source is
frozen for independent review against `055c2eb0a`.
