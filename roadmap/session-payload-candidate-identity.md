# Session payload candidate identity — #909

## Contract and owner

Session injection owns a detached candidate until all transport fields validate.
Nominal naming must follow that candidate's representation. An absent or Empty
payload becomes an anonymous object; a cloned named payload retains its nominal
role with a method-specific name. Unchanged payload reuse and conflict rollback
must preserve the original declaration. Auth scope and transport selection remain
with their existing owners.

## Sequence

- [x] Reproduce the Empty-sentinel panic directly; nil payload/type controls pass.
- [x] Reproduce with a real HTTP DSL method declaring SessionSecurity and no Payload.
- [x] Add direct candidate identity and original-declaration controls, plus bounded
      generated empty/anonymous/named payload compilation cases.
- [x] Name the actual candidate at the shared injection owner; delete the assertion
      based on the source payload's type.
- [x] Compare supported controls at parent and candidate byte for byte, repeat
      generation in fresh processes, and compile the selected generated modules.
- [ ] Obtain independent exact-final review, commit/push, close #909 and reconcile
      the master tracker. Preserve evidence and remove inactive temporary output.

Acceptance covers the generated injected field, original declaration isolation,
unchanged reuse and failure rollback, not merely absence of a panic. This is a
representation correction without a new public policy or dependency.

Parent: `b9e51df28`. Reproduction inputs and logs are
`/tmp/loom-session-empty-{repro_test.go,overlay.json,current.log}` and
`/tmp/loom-session-empty-dsl{_test.go,-overlay.json,-current.log}`.

## Implemented evidence

The rename operation now tests and renames the actual detached candidate. It no
longer derives candidate identity from the source Empty sentinel. Direct cases
cover nil, nil type, Empty, anonymous, named and unchanged named payloads; the
existing conflicting-field test still proves atomic rejection without mutation.

The real DSL absent/Empty cases fail on the parent; anonymous/named controls
already compile. Candidate cases all compile and vet. The selected expression
and HTTP session/cookie-publication races pass (1.506/13.107s); scoped lint reports
zero issues. The artifact ledger `/tmp/loom-909-output-comparison.json` proves
all 28 parent-control files unchanged and all 56 candidate files repeat exactly
across independent processes. Only previously rejected designs gain output.

No new concurrency or ordering mechanism is introduced; the existing candidate
validation-before-publication boundary is preserved. No auth-scope change or
consumer migration is required. Parent/candidate captures and direct test inputs
remain under `/tmp/loom-909-*` pending final review and cleanup.
