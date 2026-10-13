# Imported security credential ownership (#801)

Status: implementation complete and frozen for independent final review.
Approval: checkpoint 38, main commit `0dfab0ccd`.
Parent: `25179e87b8b5d4e9cf5bb233d1818600469d060d` (pushed main).
Managed worktree: `/Users/luca/.codex/worktrees/import-security-credentials/loom`.
The uniform JSON candidate remains frozen in its separate worktree; any review
findings on it take priority over this investigation.

## Recovery sequence

- [x] Read issue801 and establish an isolated managed worktree.
- [x] Reproduce the existing valid Basic/Bearer/OAuth import failure on this parent.
- [x] Trace imported security requirements, credential allocation, DSL tags,
  wire-binding normalization and generated authentication callbacks.
- [x] State a shared credential representation preserving source OR/AND,
  OAuth scopes, Basic framing, API-key locations/names and operation overrides.
- [x] Add direct red and healthy controls for physical credentials versus
  semantic scheme roles. Model any nontrivial grouping/requirement interaction.
- [x] Prove candidate DSL representation evaluates and supports generated
  compilation/export/auth-wire checks without production edits.
- [x] Freeze scope, acceptance and evidence for independent readiness review.
- [x] Implement after readiness release; complete bounded exact-output,
  generated compile/auth-wire and spec checks.
- [ ] Independent final review; root integrates and delivers the atomic ticket.

## Established problem and owner hypothesis

The importer allocates credential fields independently per security scheme in
`internal/openapiimport/security.go:securityParameters`. Payload rendering stores
only one security role on each `renderedParameter`. Bearer and OAuth both use the
same Authorization bearer representation but receive different payload sources.
The shared expr wire-binding owner correctly rejects competing same-codec source
fields. Changing a fixture to use a custom OAuth header would alter the source
contract and is not a repair.

Investigate an importer-owned normalized credential plan, grouping equivalent
physical wire representations while retaining all semantic scheme-role tags.
Security requirements and OAuth scopes remain separate, unchanged semantic
facts. Basic is a different framing with two sources; API keys retain explicit
location/name and raw framing. Do not relax expr collision checks to admit
ambiguous competing sources, and do not coalesce merely by Go names or scheme
kind. Verify that a single attribute can carry Token and AccessToken roles before
adopting this representation. Multiple same-kind schemes and explicit API-key
reuse require evidence rather than assumptions.

## Boundaries

No JSON query/form changes, main edits, shared skill edits, master tracker edits
or commits. Preserve current public security source contracts and reject only
cases genuinely unsupported by Loom's existing semantics. Do not invent new
scope union/intersection rules or weaken AND requirements to make evaluation
pass. Any unresolved semantic admission decision must be identified explicitly.

Resumed after the uniform JSON correction received independent approval. Its worktree remains frozen. Exact parent red is
`/tmp/loom-import-security-801-parent-red.log`, generated design evaluation fails
at conflicting bearerToken/oAuthAccessToken Authorization bindings. No production
source changed. Resume tracing and direct controls after JSON correction freeze.

## Candidate owner and implementation sequence

1. Replace per-scheme `securityCredentialParameters` allocation with one
   importer-owned source-adapter credential plan selected from the operation's
   effective security requirements. The key is physical location + canonical
   wire name + codec + component position. Header names fold case; query/cookie
   names do not. Basic has two positions; Bearer/OAuth share the same bearer
   value; explicit API keys use raw values at their authored location/name.
2. Store all scheme-role annotations on the shared physical field. Emit the
   attribute and HTTP mapping once, with all needed tags. JWT and OAuth reuse
   one attribute with `security:token` and `security:accesstoken`; shared API
   keys retain each `security:apikey:<scheme>` tag. Identical Basic/JWT roles
   need only one DSL role tag, but all scheme identities stay in requirements.
   Preserve first occurrence ordering/names and reserve ordinary payload names.
   Delete the old single-role allocation path; no renderer-local coalescing.
3. Keep `expr.ensureWireBindings` as the sole effective wire occupancy/admission
   owner. The importer translates source slot/encoding facts into its existing
   source-reuse contract; it does not relax collision rules or independently
   decide whether different encodings can share a slot. Basic OR Bearer remains
   supported; Basic AND Bearer and raw Authorization/Bearer collisions remain
   rejected. Ordinary mapped fields are not merged with security fields: doing
   so would require preserving their distinct constraints/requiredness.
4. Preserve the original security list, conjunction membership, scope lists,
   global/operation override, anonymous alternative and explicit empty security.
   Source credential grouping must not deduplicate semantic scheme checks.
   Strengthen importer scope admission before rendering: a Loom conjunction
   cannot represent different required scope lists per scheme. Reject these
   contract-affecting cases, including mixed OAuth/non-OAuth with nonempty OAuth
   scopes and two OAuth schemes where the first has an empty list. Existing
   diagnostics incorrectly use nil to mean both first/uninitialized and empty.
   Keep single-OAuth scopes, OR with differing scopes, equal-scope OAuth AND,
   and empty-scope AND admitted. No scope union/intersection or DSL scope change.
5. Extend direct owner tests to assert retained roles, stable ordering, input
   immutability, one physical mapping, all requirement identities and exact
   diagnostics. Reuse the existing symmetry fixture, updating representation
   assertions without weakening its valid source. Public import coverage should
   explain source sharing, regeneration and the unsupported scope boundary;
   root owns corresponding shared skill updates.
6. Run bounded parent/candidate/isolated-repeat generation and exact-byte
   comparisons, full module build/vet and mounted auth/wire assertions on actual
   imported output. No full design corpus. Freeze exact final diff and evidence
   for independent review before delivery.

This is one #801 source-representation repair with lossless admission. The
runtime DSL's common requirement scopes are its documented contract, not a bug
this change should redesign. The importer falsely claiming to preserve distinct
source scope lists is the mismatch. Non-OAuth role arrays permitted by OAS3.2
remain outside the importer's existing admitted subset; this task does not
expand them.

## Accepted, rejected, and unchanged source examples

| Source contract | Result |
| --- | --- |
| Basic OR Bearer OR OAuth(read) | Three physical fields, original three alternatives |
| Bearer AND OAuth, both scopes empty | One bearer field, two mandatory auth checks |
| OAuthA(read) AND OAuthB(read) | One bearer field, two checks retaining read |
| Bearer OR OAuth(read) | One bearer field, separate scope lists per alternative |
| Two API keys at the same header spelling/case variants | One raw field, both scheme tags/checks |
| Same API-key name at query and cookie, or query key/Key | Distinct physical fields |
| Basic AND Bearer | Existing incompatible framing rejection |
| Raw Authorization API key plus Bearer | Existing ambiguous framing rejection |
| Bearer[] AND OAuth[read] | Unsupported import: cannot preserve per-scheme scopes |
| OAuthA[] AND OAuthB[read], in either order | Unsupported import: cannot preserve per-scheme scopes |
| Explicit operation security: [] | No inherited credentials |
| An empty requirement among alternatives | Preserve anonymous alternative |

The [OAS3.2 Security Requirement Object](https://spec.openapis.org/oas/v3.2.0.html#security-requirement-object)
defines conjunction within an object, alternatives across objects and per-scheme
scope lists. The existing consumer guide's Authentication and Sessions section
and `dsl.Security` document shared-source reuse and Loom requirement scopes.
No new user policy choice is necessary.

## Readiness evidence and remaining acceptance

Parent remains `25179e87b8b5d4e9cf5bb233d1818600469d060d` throughout.
No production file has changed. Original issue red is the existing
`TestImportPreservesRepresentableSecuritySchemes`, retained in
`/tmp/loom-import-security-801-parent-red.log`.

- Direct allocation: eight reds (Bearer/OAuth OR and AND, Basic+both OR,
  repeated Basic/JWT roles, same header/query/cookie API-key roles). Five
  distinct-representation and three operation-scope/name-collision controls pass.
- Scope admission: mixed JWT/OAuth with scoped OAuth in either order and an empty
  first OAuth list are red. Reverse OAuth order is already rejected; equal-scope
  OAuth AND, empty mixed AND and distinct OR scopes are healthy controls.
- Candidate is deliberately hand-authored, not purported imported output.
  `testdata/security_credentials/candidate.go.txt` and `wire.go.txt` exercise
  supported multi-role attributes through full generated module build/vet,
  mounted handler status/service invocation/auth hook values and scope checks.
  Export is parsed with libopenapi and compared for preserved requirement sets.
  `TestCandidateMixedScopeSourceGap` is an expected-red runtime witness: export
  shows Bearer[] + OAuth[read], while both hooks receive RequiredScopes[read].
- The candidate module uses the existing verified pushed pin
  `v1.10.0-alpha.9.0.20261012181439-651372ce6ff0` and an explicit local replacement
  to this worktree. `651372ce6ff05ed1db495369fd51d48df0c0ec75` is an ancestor of
  origin/main. Go1.27.2; no placeholder dependency pin. Readiness only establishes
  representability; production acceptance must generate from actual imports.
- The finite credential allocation model compares current behavior and four
  incomplete repairs against the full key + retained role/requirement proposal.
  See `internal/openapiimport/tla/security_credentials/README.md`.

Before final review after implementation, select: the existing security symmetry
fixture; same-kind Basic/JWT and same-slot API-key aliases; header/query/cookie
case/location controls; global override and anonymous methods; an OAuth
conjunction with equal scopes; mixed/different scope diagnostics; and an
ordinary non-security importer fixture. Compare authored imports, all generated
Go/OpenAPI artifacts and isolated repeat bytes. Build/vet complete modules,
including CLI. Mounted tests must prove fallback, both mandatory checks, auth
failure prevents service invocation, correct scopes and independent locations.
Validate/lint rendered specs with existing libopenapi/Vacuum gates. Intended
changes are fewer duplicate credential attributes/CLI flags/mappings, retained
multi-role metadata, admitted formerly conflicting equivalent representations,
and explicit rejection of lossy scope imports. Normal distinct representations
and ordinary imports should remain byte-identical except reviewed global
security component naming where existing export identity requires it.


## Frozen readiness commands and outputs

From the managed worktree, `LOOM_DIR` explicitly set to its absolute path:

```sh
go test ./internal/openapiimport -run '^TestImportPreservesRepresentableSecuritySchemes$' -count=1
go test ./internal/openapiimport -run '^TestSecurityCredentialReadiness(Allocation|ScopeControls|ScopeAdmission)$' -count=1 -v
go test ./internal/openapiimport -run '^TestSecurityCredentialReadinessCandidateGenerated$' -count=1 -v
go test ./expr -run '^TestHTTPWireBinding(CredentialAlternatives|SharedSource|ExplicitBearerAlternative)$' -count=1 -v
```

The first command fails as originally reported. The direct matrix has eleven
expected failing cases (eight allocation and three scope-admission) and twelve
passing controls. The generated candidate test passes eight mounted cases,
complete build/vet and libopenapi/requirement checks, then separately verifies
the expected failing mixed-scope runtime witness. The selected expr controls
pass unchanged. This evidence is in `/tmp/loom-security-801-direct.log`,
`/tmp/loom-security-801-candidate.log`, `/tmp/loom-security-801-expr-controls.log`
and six `/tmp/loom-security-801-model-<mode>.log` files. The frozen manifest lists
hashes of all durable readiness inputs and these logs. No production changes,
commit, full corpus or unrequested source-policy changes occurred.


## Final implementation and acceptance handoff

The production owner is `security_credentials.go`: one slot/codec/component
record allocates a physical field and retains every semantic role. Payload and
mapping rendering consume it; the old single-role allocator is deleted. Scope
admission moved from parser-specific first-OAuth inference into the normalized
`planDocument` boundary, using `securityRequirementScopes` for Analyze, Render,
partial import and requirement emission. It compares every scheme's list, with
empty lists represented explicitly by their position rather than nil sentinels.
No expr, authentication runtime, transport generator or global naming behavior
changed. Public migration is documented in `docs/openapi-import-coverage.md`.

Final source parent remains `25179e87b8b5d4e9cf5bb233d1818600469d060d`.
The comparison parent is a managed worktree at
`/Users/luca/.codex/worktrees/import-credential-parent/loom`; do not clean it by
manual Git operations. Modules use the verified pushed pin
`v1.10.0-alpha.9.0.20261012223917-25179e87b8b5` plus explicit per-source local
replacement and LOOM_DIR. Tests retain the existing helper's earlier verified
pushed pin plus its explicit current-repository replacement.

Acceptance results:

- Original issue symmetry test passes without changing its source contract.
- The direct owner/scope/partial tests pass, including all original reds, role
  retention across repeated references and conjunctive/alternative requirements,
  input immutability, naming collision control, exact diagnostics, lossless-mode
  rejection and partial import retaining the unaffected public operation.
- `TestImportedSecurityCredentialsGenerated` imports the eleven-operation
  `imported.json` fixture. Its full module builds and vets, including generated
  CLI packages. Eighteen mounted auth/client cases pass: Basic, JWT, OAuth
  fallback with read scope, AND success/denial, two Basic/JWT/API-key roles,
  equal-scope OAuth conjunction, inherited security, explicit public override,
  anonymous fallback, shared query/cookie keys and distinct case/location values.
  Generated-client encoding is exercised for Basic, bearer, bearer+OAuth and
  shared API keys. Spec parsing and exact requirement/location checks pass.
- The full `internal/openapiimport` package ran in 388.116s. Its sole failure was
  the new role test expecting `jwtToken` rather than the existing Goify spelling
  `jWTtoken`. The assertion was corrected without changing production naming;
  the affected owner suite rerun passes. All other package tests passed in that
  run. Reuse their evidence; no unchanged full-package rerun is required.
- The selected expr admission controls remain green. Additional actual import
  comparisons preserve Basic-AND-Bearer and raw-Authorization/Bearer conflicts
  at both parent and candidate; source grouping does not relax admission.
- Scoped golangci-lint reports zero issues; diff whitespace checks pass.
- Eight rendered candidate specs (four designs, JSON and YAML) pass Vacuum
  v0.33.0 with the repository's pinned libopenapi override and contract rules.
  The actual imported fixture also passes libopenapi BuildV3Model and security
  semantic assertions. No new project dependency is introduced.

### Exact output comparison

`/tmp/loom-security-801-comparison/canonical-results.json` records generated
artifact hashes and per-stage command statuses. `canonical-comparison.json`
records exact differences; the first comparison used differently named CLI
binaries and therefore included command-comment differences. Final generations
use argv[0] `loom` consistently. That rerun changes comments only for unchanged
artifacts; prior full build/vet evidence remains valid. Candidate and repeat are
separate generator processes. They match every artifact byte for byte.

| Bounded design | Parent | Candidate | Intended difference |
| --- | --- | --- | --- |
| Existing `supported.yaml` ordinary fixture | Build/vet pass | Build/vet pass | None, every artifact byte-identical |
| Inheritance/public/anonymous/distinct-location subset | Build/vet pass | Build/vet pass | None, every artifact byte-identical |
| Repeated Basic/JWT scheme subset | Build/vet pass | Build/vet pass | 11 artifacts change: remove duplicate credential fields, accidental JSON body/constructors/CLI body flags, body schemas; examples/digest update |
| Eleven-operation mixed credential fixture | Import accepts; generation rejects competing sources | Import/generation/example/build/vet pass | Shared fields and all role tags; 22 artifacts now available or corrected |
| Unequal per-scheme AND scopes | Import accepts, later generation rejects distinct sources | Import rejects with explicit scope diagnostic | No lossy design written |
| Basic AND Bearer; raw Authorization OR Bearer | Generation rejects conflict | Same rejection | None in admission |

Comparisons include `loom example` and complete example server/CLI build/vet.
There are no checked-in generated fixtures owned by this change; the reused
import symmetry assertion changes from two independently allocated fields to one
field plus the access-token tag. Normal existing imports remain unchanged.
The model is unchanged from readiness and its correspondence notes now point to
the implemented owner and actual import tests. Its 728-state checked result and
five negative counterexamples remain valid evidence, not a proof of parsing or
wire implementation.

### Review inputs and integration

Final evidence is under `/tmp/loom-security-801-*`: actual imported/focused log,
final owner log, full package log with the one corrected expectation, final lint,
Vacuum log, comparison scripts/results/diffs and the frozen manifest. Preserve
these until independent review and integration finish. The final manifest lists
exact hashes rather than assuming the earlier readiness manifest still matches.
Root owns commit, issue reconciliation, shared guide updates and cleanup.

Against current origin/main, three existing files overlap delivered work:
`docs/openapi-import-coverage.md`, `internal/openapiimport/plan.go`, and
`internal/openapiimport/symmetry_test.go`. Keep the delivered uniform JSON
contract changes when integrating these hunks. No #709 or #797 source owner was
modified by this repair. No production edits should follow the final freeze
without an affected re-review.

### Final-review correction: inactive root security in partial imports

The independent final review found one medium defect: partial candidates copied
root requirements even when every retained operation explicitly overrides them.
An unsupported root conjunction then rejected valid public and secured overrides.
Strict whole-document admission remains correct and must remain unchanged.

1. [x] Add direct reds for public and scoped secured overrides of an unsupported
   root conjunction; retain an inheriting blocked operation and cover the empty
   retained subset, both lossy modes, and strict rejection.
2. [x] Make the partial operation projection retain root security only while a
   retained operation inherits it. Preserve every effective operation requirement.
3. [x] Check affected partial/source tests and actual partial-import generated
   authentication, OpenAPI, full build/vet and isolated output determinism.
4. [x] Update correspondence and freeze the changed diff for targeted re-review.

This is operation-subset ownership, not a change to supported scope semantics.
The physical credential model is unchanged; root inheritance is a concrete
projection/test obligation outside that model.


Correction evidence and acceptance:

- `go test ./internal/openapiimport -run '^TestSecurityCredentialRootScopePartialImport$' -count=1`
  first failed in all eight table cases at the intended root-scope diagnostic,
  recorded in `/tmp/loom-security-801-partial-correction-red.log`.
- The affected owner, existing partial-import and original security symmetry
  selection passes (`partial-correction-green.log`). The final partial and CLI
  selection suite passes (`partial-correction-selection.log`). Strict Analyze
  and Render continue to reject unequal root scopes, even with lossy admission.
- `TestImportedPartialSecurityCredentialsGenerated` imports the subset, builds
  and vets the entire generated module, validates its spec with libopenapi and
  proves five mounted cases: public access, equal-scope OAuth conjunction,
  second-hook denial, missing-token denial and the skipped inherited route's
  404. Both hook role names and `read` scopes survive. The generated security
  matches each retained source operation exactly (`partial-correction-wire.log`).
- `/tmp/loom-security-801-partial-comparison` compares pushed parent, previously
  reviewed candidate, corrected candidate and separate-process repeat. The
  source module pin remains pushed `25179e87b`, with explicit local replacement.
  For invalid inherited root plus valid overrides, parent import accepts then
  fails generation; reviewed candidate rejects partial import; correction
  imports 2/3 operations (CLI exit 3 intentionally denotes skipped operations)
  and its full example server/CLI build and vet pass. The follow-up resume script
  recognizes that exit status; it does not treat an import error as success.
- Valid active-root partial output is byte-identical at every version. Valid
  inactive-root partial output changes only design, JSON/YAML root security and
  digest relative to the reviewed candidate; generated Go is byte-identical.
  Parent also has the previously proven duplicate OAuth credential differences.
  Each corrected case exactly matches its isolated repeat. Results and paths
  are in `results.json` and `comparison.json`; both scripts and logs are retained.
- Six corrected JSON/YAML specs pass pinned Vacuum (`partial-correction-vacuum.log`).
  Scoped golangci-lint reports zero issues (`partial-correction-lint.log`). The
  physical grouping model and its configuration are unchanged; correspondence
  documents the separate operation-projection obligation. Prior evidence for
  unchanged strict imports, allocation and auth behavior remains valid.

The correction adds `partial.go`, the focused partial test and its wire fixture
and changes only the documentation/correspondence/plan among prior frozen files.
Targeted re-review is pending; no commit or integration is authorized by this
implementer handoff.

### Root integration receipt

The independent targeted re-review approved the revised 25-file diff (manifest
`ab72fe41dde09613849f85883dfd3468e30057ed650606a44d1c14f06d577dbe`).
Root integrated it against `39f9ce17c`, retaining the delivered uniform JSON
changes in the three overlapping files listed above. The consumer/framework
skills and formal-model index now describe the same credential owner.

The integrated credential/direct/generated selection passes (22.230s), as do
the uniform JSON and existing security controls (1.298s). Existing partial
selection checks and scoped lint also pass. Fresh generation of the ordinary,
mixed-role and repeated-scheme inputs matches both the approved candidate and
an independent process repeat byte for byte. Each complete candidate example
server/CLI module builds and vets. The actual partial-import generated auth
regression remains part of the passing integrated selection. Original bounded
parent/candidate and model evidence remains valid for unchanged behavior.

Evidence: `/tmp/loom-801-integrated-{tests,controls,partial-controls,lint}.log`
and `/tmp/loom-801-integrated-comparison/{results,comparison}.json`. Integration
review must authorize the exact final diff before commit.
