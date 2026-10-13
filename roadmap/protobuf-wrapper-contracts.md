# Anonymous protobuf wrapper contracts: readiness

Parent `66c61745673d797d4d15c860210fc082afc80fed`; managed worktree
`/Users/luca/.codex/worktrees/protobuf-wrapper-contracts/loom`.
This is a separate prerequisite for #684. Its typed-contents candidate remains
frozen in `protobuf-collection-presence`. Root independently approved readiness
and released this owner for implementation. The candidate was safely advanced
to `e64a14d1d` after #695 delivery, retaining the original parent evidence.

1. [x] Confirm duplicate search, exact-parent reproduction and file focused issue.
2. [x] Reproduce permissive/strict contracts in both traversal orders for array
   and map contents, named boundaries and equal-contract sharing controls.
3. [x] Establish normalized declaration/validation identity ownership; compare
   wire-only, merged, positional and contract-aware candidates in a finite model.
4. [x] Freeze a bounded proposal, consumer impact, tests/model and acceptance
   manifest; request root's independent readiness review.
5. [x] After release, implement only this owner, check generated/wire controls
   and parent/candidate/process-isolated-repeat bytes/build/vet.
6. [ ] Freeze exact final atomic diff for fresh independent review; root owns
   integration, commit, push, issue closure and release of dependent #684.

Observed cause: `wrapNestedCollection` hashes `protoAttributeShape` for generated
name/UID. That shape represents protobuf fields but omits contents predicates.
`collectValidationsR` and `addUserTypeValidation` treat that identity as one
complete named validation contract. Distinct occurrence contracts collide; the
first visited contract wins. A parent binary-protobuf witness accepts one item
under MinLength2. The #684 removal of invented requiredness additionally exposes
a call with no emitted helper; #819 does not solve this declaration collision.

Do not merge predicates, emit stubs, or change explicit named-message sharing
as an incidental side effect. Public MinLength semantics and retained constraints
already settle the required outcome; no new user policy is needed.

## Filed owner ticket and current evidence

[#831](https://github.com/CaliLuke/loom/issues/831) was filed after duplicate
search and root's independent editorial approval. The original standalone
generated probe is included in the issue. At readiness, the committed-source candidate was unchanged; additions were
tests, model/configs and this plan. Minimal parent generated/vet/binary reproduction fails in 2.407s in
`/tmp/loom-wrapper-contract-minimal-red.log`.

The matrix uses four shapes (array/array, array/map, map/array, map/map), each
with permissive-first and strict-first order. Permissive-first silently accepts
a one-member strict value; strict-first rejects a valid permissive value.
Distinct named collection declarations pass in both orders. Direct allocation
checks expose five contents-contract collisions (min/max length and element
enum/pattern/format), while unconstrained/equal-length/equal-clause reuse passes.
Map constraints are attached with `Elem` inside `MapOf`; the initial temporary
map-level constraint attempt was corrected and is not accepted evidence.

## Proposed shared owner

Keep `protoAttributeShape` as the existing wire-layout/explicit-name contract.
It must not become a general predicate hash or change explicit named-message
sharing incidentally. Anonymous collection allocation instead registers the pair
of that wire shape and a detached contents validation contract. Use the existing
name reservation allocator to assign a generated name/UID to each distinct pair;
both normalized references and complete named validators consume that identity.
Equal pairs continue sharing. Distinct contracts cannot silently share one
validator, even when their binary protobuf fields happen to be identical.

The private contract snapshot derives complete lowered predicates from
`expr.EffectiveConstraintsFor`; retain ordered clause kind/value (without source
provenance) as well as the complete lowered predicate carrier. Traverse owned
inline contents in deterministic member order. Named boundaries retain their
allocated declaration identity and occurrence predicates, then stop: their
intrinsic child graph belongs to the named declaration and may still be under
normalization. Snapshot matching may use structural equality over detached facts;
it must not stringify only selected fields, duplicate precedence rules, depend
on pointer addresses, or reuse the unrelated OpenAPI schema fingerprint.
No attempt to decide semantic equivalence of arbitrary regexes/custom values is
required. Identical effective fact sequences share; conservative distinction of
different fact sequences is preferable to a false merge.

This registry is the anonymous collection declaration owner, not a second
validator collector. Keep one existing collector/call path. Compare alternatives:
wire-only identity loses or strengthens predicates; global merging strengthens
permissive occurrences; per-position allocation avoids loss but discards useful
sharing; complete occurrence-validator identities could retain one wire message
but would require a broader rewrite of named call/definition ownership. The
bounded declaration-contract correction preserves the current complete named
validator architecture and confines new schema names to actual conflicting
anonymous contracts. It is the recommended implementation.

## Models and remaining acceptance

`grpc/codegen/tla/wrapper_contracts` explores registration orders. Current,
presence-only, merged and positional modes fail their intended invariants;
contract identity passes 14/10 states. README maps concrete owners and states
limits. Models do not prove the proposed snapshot implementation.

After scoped approval:

1. Add direct snapshot detachment, reordered/copied equivalent graphs,
   multi-predicate combinations and named-boundary/cycle tests before production.
   Confirm different logical contracts get different declaration identities;
   identical detached facts still share. Preserve metadata/physical references.
2. Implement one private anonymous collection registry and delete wire-only
   coalescing at that seam. Keep #684's current contents-requiredness unchanged
   in this atomic commit, so #831 proves its independent parent bug.
3. Extend binary tests to generated response validators and mounted unary calls:
   valid free+strict payload reaches service once and round-trips; strict-invalid
   rejects before invocation with a length problem; permissive input remains
   accepted in both traversal orders. Preserve populated bytes and field numbers.
4. Compare bounded parent/candidate/process-isolated-repeat output for the two
   order specimens with all four shapes, existing MessageDeclarationsDSL and a
   representative recursive named collection. Build/vet distinct outputs.
   Expected changes: formerly colliding anonymous contracts gain distinct normal
   allocated wrapper names, descriptors/references/converters/validators follow
   those identities. Equal-contract and named controls stay unchanged. Binary
   protobuf payloads remain field-compatible; direct generated Go references may
   require regeneration. No compatibility mode or explicit-name rewrite.
5. Check #684 compatibility with its frozen typed-contents production overlay:
   both predicates and complete helpers remain correct without invented Required.
   Do not include that overlay's production changes in #831.
6. Add minimal consumer regeneration guidance and framework owner/model notes;
   scoped lint, exact manifest and fresh independent final review before root
   delivery. No full corpus or unrelated default/text-format changes.

## Frozen readiness receipt

Final source command:
`LOOM_DIR="$PWD" go test ./grpc/codegen -run
'^Test(WrapperContract|AnonymousCollectionWrapperContractIdentity)' -count=1 -v`.
Expected reds and positive controls complete in 22.247s in
`/tmp/loom-831-final-readiness-red.log`: five direct contract-identity reds,
three sharing controls, eight generated order/shape reds, two named generated
controls, and the independent minimal strict-bound witness. Every generated
module is vetted before binary tests. The final pinned golangci-lint 2.14.0 /
Go 1.27.2 check reports 0 issues (`/tmp/loom-831-final-readiness-lint.log`).

Exact source/evidence manifest: `/tmp/loom-831-readiness-manifest.json`.
Root independently approved this exact readiness manifest and released step 5.
The historical readiness evidence excludes production changes. No changes to
the frozen #684 worktree are part of this repair.


## Implementation and final acceptance

The private `ServiceData.collectionMessages` registry now pairs wire shape with
`collectionContract`: complete detached effective rules, ordered clause kinds
and values, effective defaults, nullability, array storage facts, logical member
order and named declaration identity. Named edges stop after occurrence facts;
inline children recurse. `wrapNestedCollection` consumes the allocated name;
existing validator definition/call collectors are unchanged. There is no
predicate merge, no per-position identity, and no change to explicit-name owners.
Opaque custom values have the same borrowed-value limitation documented by
`EffectiveValidation.Lowered`/`Default`; admitted built-in values are detached.

Additional red source: `/tmp/loom-831-snapshot-red.log` confirms mutation/copy,
traversal-order and required-element collisions before implementation. The new
snapshot tests cover all numeric endpoint forms, multiple enum/pattern/format
clauses, named occurrence required/nullability facts and recursive copied named
boundaries. Parent traversal and minimal red evidence above remain valid.

The candidate safely fast-forwarded to `e64a14d1d016efa88a31af9930fa2ee159def5f3`,
which includes #695 effective bounds admission. Focused owner, explicit-name,
recursive, generated binary request/result and mounted tests pass in 46.388s in
`/tmp/loom-831-final-contracts.log`. Every new generated probe runs build, vet and
race. The ten mounted shape/order/named cases assert InvalidArgument,
`invalid_length`, the existing sanitized `validation error` message, zero service
invocations for invalid input, one for valid input, and exact service payload and
result. Binary request/result field contents match under deterministic protobuf
marshaling. The initial mounted assertion expected an unsanitized length message;
it was corrected to the already accepted public error contract, without changing
production error behavior.

Four comparison specimens: all four collection shapes permissive-first and
strict-first, existing `MessageDeclarationsDSL`, and a named recursive graph
containing nested arrays. Run independently with:

```sh
LOOM_DIR="$PWD" LOOM_WRAPPER_CONTRACT_OUTPUT=/tmp/loom-831-comparison/current go test ./grpc/codegen -run '^TestWrapperContractArtifacts$' -count=1 -v
```

The managed parent worktree is `wrapper-contract-parent` at the original
`66c61745673d797d4d15c860210fc082afc80fed`, with only identical comparison test
inputs added. All temporary modules pin the verified pushed
`v1.10.0-alpha.9.0.20261013020843-063198a2f427` and explicitly replace Loom with
the source under test. Parent/candidate/repeat each contain 121 generated
artifacts. Candidate and independent-process repeat are byte-identical. A fresh
current generation after #695 is also identical to all 121 candidate artifacts;
existing build/vet evidence remains valid for the unchanged outputs, while the
mounted modules were rebuilt against the new owner.

The 32 intended differences are four files per conflicting shape/order:
client and server `types.go`, `.proto`, and generated `.pb.go`. The second
conflicting contract receives `ArrayOfSint642` or `MapOfStringSint642`; converter
references, validator definitions/calls and protobuf descriptors follow it.
The earlier contract retains its original name. All field numbers, scalar wire
types and other messages remain unchanged. The declaration and recursive
specimens are byte-identical to parent. Complete hashes and exact output diff:
`/tmp/loom-831-comparison/hashes.json`, `differences.json`, `output.patch`.
Eight parent/candidate modules build/vet (`/tmp/loom-831-build-vet.log`);
repeat compilation is unnecessary for identical source/toolchain/dependency
inputs. Scoped final lint reports zero issues (`/tmp/loom-831-final-lint.log`).

The frozen #684 production changes were applied only through
`/tmp/loom-831-with684/overlay.json`, including its two changed direct-test call
signatures and a corresponding signature adaptation in the new registry.
`TestCollectionWrapper*` and `TestPresenceNestedCollectionReadiness` pass in 2.908s
against current #695 (`/tmp/loom-831-final-with684.log`): scalar/union requiredness,
containing message presence, all collection contents, positive MinLength and
binary empty round trips. No #684 production changes are included here.

Model source/configurations did not change after readiness approval; the pinned
five model results remain valid. Documentation explains regeneration for direct
generated-wrapper references. This ticket does not claim to solve ordinary
protobuf collection absence, repeated text formats, defaults application, or
#684 contents requiredness. Final review and delivery remain root-owned.
