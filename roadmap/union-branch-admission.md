# Union branch identity admission (#701)

Parent: `a7b09d5a8d2f1900d0a4b50f8af3bb78e245da24`.
Readiness was independently approved before implementation. Final and
integration reviews passed; delivery is recorded below.

## Sequence

1. [x] Read live #701/#702/#703 and current evaluator, DSL, value capture and docs.
2. [x] Reproduce each admission gap and ordinary controls on the current parent.
3. [x] Add direct evaluated-design and actual CLI regressions for #701.
4. [x] Model scope, resolved identity and validation-before-capture alternatives.
5. [x] Freeze exact readiness sources, evidence and bounded acceptance for review.
6. [x] After independent approval, implement only the shared union admission owner.
7. [x] Compare bounded generated artifacts, compile controls, check deterministic
   output and affected lint, then freeze for independent final review.
8. [x] Parent integrates one reviewed atomic commit, pushes and closes #701.

## Reconciliation and chosen boundary

All three reports still reproduce. #701 duplicates reach value occurrence capture
and panic; distinct branch names with the same effective tag do the same. The
same branch name with distinct explicit tags instead reaches generated Go and
produces duplicate fields, constants and methods. Ordinary branches compile.
#702 admits array keys (invalid generated Go) and Bytes keys (capture panic).
#703 accepts zero, negative, nonnumeric, reserved and over-limit field numbers.
These are separate contracts, not one generic validation helper: union identity,
map key representation, and protobuf field-number domain. #702/#703 remain open
and outside this patch; there is no per-transport workaround.

`AttributeExpr.validateChildTypes` is the shared union validation boundary, before
value capture and any generator. Add a private validator for each union's logical
branch names (`AttributeName`) and effective discriminator values
(`UnionVariantTag`). Reuse those existing identity owners; do not copy metadata
precedence. Run it for tagged and untagged unions, in addition to existing wire
key, untagged shape, child and recursive-cycle checks. The same local branch list
must have unique logical names and effective tags. Sibling unions may reuse both.
No authored list, tag, name, source type or generator allocation is rewritten.

A branch occurrence's tag takes precedence over named-type tag metadata; mapped
name suffixes do not distinguish logical names or fallback tags. Exact case and
punctuation are preserved: this admission does not reserve Go identifiers or
reject distinct names merely because a Go naming helper normalizes them alike.
Constructor OneOf already derives unique branch identities; repeated source type
arguments remain admissible when their derived names and effective tags differ.
Forward references must be resolved before this check; tests must establish that
boundary rather than assuming all branches are evaluated in declaration order.

## Alternatives

- Catch the capture panic: too late, misses name-only collisions and other
  consumers; rejected.
- Check raw names in the DSL append operation: misses mapped-name and inherited
  tag conflicts and risks incomplete forward references; rejected.
- Check tags only: misses duplicate logical names with distinct explicit tags;
  demonstrated generated compile failure; rejected.
- Globally reserve names or tags: rejects legal reuse in distinct union owners;
  rejected.
- Chosen: validate both resolved identities within each evaluated union before
  publication to generators, while retaining the existing capture integrity guard.

## Bounded acceptance

Direct evaluated tests cover exact/mapped logical duplicates, explicit and
inherited tag collisions, late named references, repeated occurrences, nested
array/map/recursive graphs, tagged and untagged branches, occurrence overrides,
case distinctions, independent union scope, and constructor-derived identities.
Actual CLI tests must reject invalid branches at `eval.RunDSL`, include a design
location and union/branch diagnostic, produce no generator panic and no gen dir.

After approval, generate a representative ordinary tagged design, mapped/custom
metadata design and untagged named-object design under parent and candidate.
Compile each generated service/HTTP output; byte-compare all files and repeat
candidate generation in another process. Valid output must be unchanged. Invalid
inputs must stop before generation. Existing forward-reference, union-cycle,
promoted-name and untagged validation tests protect adjacent contracts. No full
exported-design corpus or transport state-machine audit is required.

## Evidence

Read-only reconnaissance: `/tmp/loom-admission-reconcile.log` and
`/tmp/loom-admission-compile.log`; exact input
`/tmp/loom-admission-reconcile_test.go` via Go overlay. The compile probe confirms
ordinary success and duplicate logical-name / array-key compilation failures.
These logs classify ownership; readiness regressions assert the intended contract.

The model and its correspondence are in `expr/tla/union_branch_admission/`.
Current, early, name-only, tag-only and global alternatives fail their intended
invariants; the owned alternative passes all 32 bounded states. Parent CLI tests
show two late generator panics and one successful generation of invalid Go.

Parent evidence: 11 direct rejection cases fail because evaluation returns nil;
all 9 valid controls pass. Three CLI cases fail at the intended assertions: two
panic in service value capture after evaluation, while distinct tags sharing one
name incorrectly generate successfully. The added lowercase panic assertion is
unreached on that parent and strengthens final acceptance without changing the
recorded counterexamples. Scoped expr/cmd lint reports zero issues.

Preparation resolves named references recursively (`AttributeExpr.Prepare` /
`prepareTypeRefs`) before validation. Preserve this existing ordering; do not
move identity policy into DSL append or finalization. Constructor-derived name
normalization remains its existing owner and must stay valid after admission.

## Final candidate acceptance

`validateUnionBranchIdentities` now checks both identities per union from
`validateChildTypes`, retaining existing child, cycle, untagged and wire-key
validation. No generator or value-capture path changed. All 20 direct cases pass;
all three actual CLI cases now fail at evaluation with source location and no
panic or generated directory. Complete expr and DSL package tests pass.

Three existing HTTP testdata designs cover ordinary tagged payloads, custom
union wire keys, and an untagged union shared between a request and a projected
response. Their 40 generated service/HTTP/view artifacts are byte-identical
between parent, candidate and a separate candidate generation process. Parent
and candidate generated modules compile, test and vet. Three checked-in union
payload constructor goldens also pass unchanged. There are no intended generated
artifact differences. The parent comparison overlays the exact old admission
source and an empty new-file stub; runtime code is unchanged. No transport codec,
OpenAPI rendering or schema output changed, so no additional transport/model/spec
audit was needed. Scoped expr/cmd lint reports zero issues.

Readiness approval: `/tmp/loom-701-independent-readiness-verdict.json`.
Final evidence and source/input hashes are recorded in the final manifest.
No commit or push has been made by this implementation task.

## Delivery receipt

Delivered/pushed `e631f0a89`; independent final and exact integration reviews passed, ticket #701 confirmed closed. Integrated evaluator and actual CLI race checks passed.
