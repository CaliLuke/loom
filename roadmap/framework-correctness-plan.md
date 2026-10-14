# Framework correctness recovery plan

Implement the architectural direction accepted in the Codex discussion on 2026-10-12. Repair shared contracts and ownership boundaries; use review findings as regression witnesses. Authentication is resolved before transport execution, and results are recursively projected before transport encoding. This file is the recovery entry point and progress source of truth.

## Current state

- **221 of 227 tracked reports are delivered, pushed and confirmed closed.**
  The [inventory](framework-correctness-inventory.md) records each report;
  `/tmp/loom-tracker-221-issues.json` is the frozen GitHub reconciliation input.
- **User-delegated technical decisions:** choose the best coherent framework
  contract, document tradeoffs and migration effects, and continue work. The user
  explicitly superseded the repository rule requiring questions for unresolved
  public semantics within this backlog. Ask only for genuinely missing user
  information or authorization, explaining concrete consequences plainly.
- Automatic collection examples #884 (`ba0609ed2`) are reviewed, pushed and
  closed. One logical representation boundary prevents invalid reflection and
  hashing; one map candidate owner covers ordinary and constrained examples.
  Custom-key admission and authored examples survive; ordinary artifacts stay
  exact and supported custom designs compile. Cleanup is complete.
- Consumer admission #772 (`6f82fb52f`) is reviewed, pushed and closed. Shared
  buffer validation and sink grace admission precede Redis publication; both
  stale-consumer paths share an overflow-safe comparison. Race/boundary controls
  and independent review pass. Cleanup is complete.
- CI fixture adoption #904 (`b6c7e169c`) and #906 (`d88ff756a`) is reviewed,
  pushed and closed. Ticktock asserts the shared typed protocol error. Pool
  guard setup uses existing revision acknowledgment instead of waiting for a
  transient value that cleanup may supersede. Production semantics are unchanged.
- Required reconstruction #780 (`5eea445b9`) is reviewed, pushed and closed.
  One expression coverage query serves request/result/error admission; existing
  response order now has an expression owner consumed by IR. Valid generated
  output stays exact; invalid designs reject without replacing prior output.
- Request media #787 (`b1a142df2`) is reviewed, pushed and closed. One authored
  request identity feeds importer, IR, client publication and OpenAPI; the shared
  decoder accepts valid concrete +json media. Presence/error/limit controls pass.
- Imported fields #789 (`f7378d514`) is reviewed, pushed and closed. One object
  allocator covers normal, inherited and error names while preserving wire keys.
  Mounted HTTP/protobuf, inherited constraints and deterministic outputs pass.
- Static file targets #769 (`91ca31168`) is reviewed, pushed and closed. All
  handlers delegate to the existing runtime owner; the generated prefix adapter
  and shared basename table are deleted. Index redirects and alias corruption
  are fixed; protocol and directory/SPA controls pass.
- Health status #764 (`7685ae527`) is reviewed, pushed and closed. The existing
  shared pinger accepts only final 2xx statuses; parent300 regression, seven
  boundary cases, full health race and independent review pass.
- Sink group recovery #765 (`883a19f84`) is reviewed, pushed and closed.
  One active registration retains its start selector and pending recovery; read,
  rotation and existing keepalive share locked reconciliation. Native Redis,
  lifecycle/uncertain-reply controls and bounded models pass. Cleanup is complete.
- Route fixtures #902 (`8375ce604`) and header expectations #903 (`19cf194ad`)
  are reviewed, pushed and closed. Naming intent and runtime assertions survive;
  production contracts are unchanged. Generated module and artifact checks pass.
- Logr normalization #747 (`92e232503`) is reviewed, pushed and closed.
  One adapter owner replaces three unguarded pair loops. All public entry paths
  preserve ordered fields and tolerate missing values; full log races and lint
  pass, with independent exact review and adapter races.
- Protobuf service symbols #736 (`2de11550d`) is reviewed, pushed and closed.
  One pinned-plugin declaration inventory reserves inferred message names and
  rejects explicit collisions. Actual plugin inventories, eleven compiled modules
  and exact artifact comparisons pass; cleanup and safe worktree reuse are complete.
- Typed response tags #760 (`6eb4159e5`) is reviewed, pushed and closed.
  One validated logical field flows through IR to typed comparison/reconstruction.
  Eleven carriers and 35 mounted states pass; integration preserves the newer
  JSON-RPC error decoder. Independent final and integration reviews approve.
- Typed error discrimination #774 (`a24801920`) is reviewed, pushed and closed.
  Expression-owned selectors and one method decoder feed unary, SSE and WebSocket
  clients. Nine generated modules and affected fixtures establish exact output,
  compilation and wire behavior. All three review findings are corrected; private
  trial metadata and unexpected close failures retain their ownership guarantees.
- Sink delivery activation #901 (`4db060dfc`) is reviewed, pushed and closed.
  First admitted subscription starts both readers and claimers atomically under
  the lifecycle lock. Native Redis, concurrency and model checks pass. Explicitly
  subscribed #768/#800 probes retain their distinct failures; cleanup is complete.
- Reader membership #773 (`b40eb24e6`) is reviewed, pushed and closed.
  One locked stream-key lookup serves duplicate admission and removal, while
  admitted-read completion retains registration-instance identity. Full streaming
  races and unchanged registration model pass; independent review finds no gap.
- Complete wire names #710 (`5876ada4e`) is reviewed, pushed and closed.
  Shared token admission covers effective HTTP/JSON-RPC header/cookie bindings;
  the mapping parser preserves the entire suffix for valid query/body names.
  Sixty invalid cases reject, generated round trips and specs pass, and ordinary
  output stays exact. Invalid CLI generation preserves prior files.
- Legal package identity #856 (`481bdf788`) is reviewed, pushed and closed.
  One package-base rule feeds service, transport and API example consumers;
  six modules build/vet and all 123 artifacts repeat exactly. Protocol and
  directory identities remain unchanged; cleanup and reuse are complete.
- Imported encoded strings #686 (`227ea92be`) is reviewed, pushed and closed.
  Shared representation selection feeds import rendering, examples and admission;
  shared Base64 validation preserves encoded bounds and exact literal identity.
  The accepted migration changes imported byte fields to strings. Mounted wire,
  bounded output/build, independent-process repeat and spec checks pass.
- Request-cookie publication #715 (`9a4f1c2f3`) is reviewed, pushed and closed.
  One runtime owner validates before mutating headers; HTTP/JSON-RPC ordinary,
  security, session and streaming consumers fail before sending invalid values.
  Presence and normal output controls pass; the exact change was reviewed.
- Physical route ownership #781 (`47db83e78`) is reviewed, pushed and closed.
  Shared expression records drive admission, HTTP/static mounts, JSON-RPC dispatch
  and CORS. Preparation and early parent resolution are stable; actual generated
  dispatch and separate-server controls pass, with only intended static output
  changes and identical checked-in fixtures.
- Filesystem publication #751 (`2537b97ac`) is reviewed, pushed and closed.
  Output-local rendering preserves detached module context; all external files
  prepare on their destination filesystem before publication. Four actual mounted
  cases pass and generated bytes remain exact. Failed restoration exposes a
  separate original-backup lifetime defect, #899, now delivered below.
- Recoverable originals #899 (`18ae28c9e`) is reviewed, pushed and closed.
  One transaction backup owner protects original files and trees until confirmed
  restoration or complete installation. Failed rollback and repeated cleanup
  retain original bytes and report recovery paths; independent focused races pass.
- CI expectation adoption #897 (`41d2d578b`), #898 (`271784d3c`) and #900
  (`177658d22`) is reviewed, pushed and closed. HTTP collection, JSON-RPC stream
  and importer metadata assertions now consume delivered contracts; no production
  behavior changed in those three tickets.
- Service public symbols #770 (`ebf2c4ea9`) and complete evaluated provenance
  #753 (`1b9833038`) are reviewed, pushed and closed. The original #742 two-design
  witness is independently reconciled against that same provenance repair.
  Three remaining collection snapshots now adopt the shared #889 contract in
  #896 (`f97d9dd38`); production and exact assertions are unchanged.
- Recent deliveries complete the inspected receiver-name contracts: HTTP/JSON-RPC
  client #879 (`64fcddc25`), native gRPC server #880 (`cc3cc5ee2`) and service
  client #883 (`814a3a2e3`). Each uses a complete role allocation in its actual
  receiver scope; declarations and references consume recorded names.
- Shared map-key representation admission #702 (`d3cb9ad26`) rejects unsupported
  native keys before generation across every design root. The separate example
  synthesis owner is now repaired by #884 without rejecting supported custom keys.
- Metadata fixture/peer adoption #885 is delivered as `c01bd5447`. All affected
  gRPC/transport-IR packages, native binary controls and the exact failing CI
  security compile case pass. Receiver wrapper report #786 is independently
  reconciled against the delivered service symbol owner; `35ff54729` preserves
  all three reported collision forms without another production naming patch.
- Multipart source accounting #759 (`6a549820f`) and interceptor logical contracts
  #767 (`a5aad7c72`) are independently reviewed, pushed and confirmed closed.
  Built-in/custom multipart consumers share one raw-source budget; interceptor
  interfaces share logical signatures while each method owns its storage binding.
- Ticker lifetime #738 (`07401d22c`) and #707 (`abca0b975`) are reviewed,
  pushed and confirmed closed. Constructor failure, Close/Stop and every remote
  exit share local teardown; timer admission prevents revival after termination.
  Peer/shared state remains outside local cleanup ownership.
- Server stream contracts #785 (`05567e2b0`) are reviewed, pushed and confirmed
  closed. Service interfaces and interceptor adapters share operation signatures
  and endpoint carriers across native, JSON-RPC SSE and WebSocket methods.
- Imported scalar domains #777 (`6edea4932`) are reviewed and pushed; #778 is
  independently reconciled against the same owner. Both reports are closed.
  Type, nullability and enum/const membership normalize once before publication;
  the shared planner rejects unrepresentable domains in Analyze and Render.
- WebSocket completion #739 (`ec2536d3c`) is reviewed, pushed and closed. One
  connection-lock transition owns removal, timer release and result publication;
  the obsolete stream wake-up path is removed. Held results and timeout origin
  remain intact; final integrated WebSocket race checks pass.
- Per-scheme required scopes #752 (`24a277291`) are reviewed, pushed and closed.
  One expression projection feeds authorizers and OpenAPI; per-requirement data
  preserves OR independence and the obsolete duplicate projection is removed.
  Corrected parent regressions fail, generated callbacks pass and all 51 selected
  artifacts repeat exactly; ordinary generated output is unchanged.
- Worker admission #704 (`f1d1368fb`) is reviewed, pushed and closed. One
  owner covers preparation, publication, guarded startup and rollback; Close joins
  in-flight attempts. Independent native Redis race checks pass (18.298s);
  failed attempts cannot invoke handlers or renew their initial lease.
- Explicit response-body presence #886 (`720d8a5f7`) is reviewed, pushed and
  closed. Shared expr preparation preserves authored empty objects across HTTP,
  JSON-RPC and OpenAPI; inferred bodyless responses and Empty remain unchanged.
  This unblocks the importer representation contract without transport patches.
- Collection header framing #689 (`7ed1bfc63`) is reviewed, pushed and closed.
  One internal framing/admissibility owner feeds Expr and HTTP/JSON-RPC codecs;
  repeated/combined fields, empty presence and rejected lossy values agree.
  The separate projected named-collection carrier defect is tracked as #887.
- Imported object representation #776 (`3476cc239`) is reviewed, pushed and
  closed. Analysis, constructor preparation, admission and rendering share one
  carrier decision. Source-declared structured response content stays explicit;
  direct and mounted integration race checks pass (20.000s), with bounded
  generated/schema/determinism evidence and ordinary controls retained.
- Physical primitive-field queries #887 (`ee1d653cf`) are reviewed, pushed and
  closed. The shared query derives from canonical declarations; selected-body
  duplication is removed. One generated dereference changes, 55 fixture outputs
  stay identical, and mounted integration race checks pass. The independently
  reproduced named-collection conversion is now delivered as #889 below.
- Scheduler intent #737 (`f9ee29706`) is reviewed, pushed and closed. Producer
  responsibility precedes dispatch; operation identity and stop cutoffs persist
  across uncertain replies. Later authored starts remain new commands. Native
  integration race checks pass; protocol 3 requires the documented drain/shutdown.
- Collection conversion #889 (`051da453a`) is reviewed, pushed and closed.
  One array/map conversion owner preserves nil, empty and supplied values and
  clears stale targets. Seven selected generated files change; 55 fixture
  artifacts remain identical. Required/default materialization remains separate.
- Imported binding admission #791 (`43edbf9fe`) is reviewed, pushed and closed.
  One location/shape planner rule covers request parameters and response headers,
  including direct Render. Supported outputs remain identical; all four generated
  modules compile and all eight spec checks pass.
- Local reference identity #891 (`e67f359e2`) is reviewed, pushed and closed.
  One URI-fragment/JSON Pointer parser feeds importer and OpenAPI consumers;
  ordinary/encoded reference equivalence and actual generated consumers pass.
- Concrete presence #892 (`590897d3f`) is reviewed, pushed and closed. The shared
  helper consumes the entire root alias spine while preserving nullable children.
  Both real CI regressions and the codegen race suite pass; 16 ordinary artifacts
  are identical and all 32 candidate artifacts repeat across processes.
- Schema annotations #783 (`066b68aab`) are reviewed, pushed and closed. One
  emission owner and detached shared OpenAPI location preparation preserve schema
  and parameter/header descriptions separately; generated and spec checks pass.
- Imported name occupancy #688 (`dc55bff2c`) is reviewed, pushed and closed.
  One importer allocator reserves every emitted candidate; actual collision
  designs compile, 16 ordinary artifacts stay identical and 32 repeat exactly.
- Stream observation #749 (`be89c93b5`) is reviewed, pushed and closed. Immutable
  operation identity and finite lifecycle delivery feed shared SSE/WebSocket
  owners; generated SSE terminal flags and write locks are removed. Runtime
  races and actual generated integration pass on the delivered parent.
- Required CI is not confirmed green. Nullable alias failures are fixed by #892.
  The explicit alias error-body failure and recursive declaration mutation are
  fixed by #893 (`222f0a269`), with the original pointer assertion unchanged.
  The existing #820 dependency race also remains open. Source `2a98e4123` run
  `38473286562` passes Redis, generated-code-quality, OpenAPI, value-contract and
  all generated-design jobs; its `ci` and race jobs failed. The `ci` log records
  collection/query snapshots and structural expectations,
  importer parameter assertions, JSON-RPC lifecycle assertions, and a worker-event
  delivery timeout. Service/HTTP collection, JSON-RPC lifecycle and importer
  assertions are now reconciled by #896/#897/#898/#900 with focused current-source
  checks. The worker-delivery timeout did not recur in 15 current-source race
  repetitions; its cause remains unresolved. Newer required runs are queued.
- #703 parent parity evidence uses an exact archived parent, superseding the
  invalid CLI-overlay claim. All 30 artifacts match and modules build/vet;
  its owning plan preserves the independently checked correction.

## Execution focus after architecture check

The delivery log and report count do not establish that a bug family is closed.
Organize active work by the owning contract and its consumers; use individual
reports as regression witnesses. Older milestone checkboxes below require
reconciliation with the inventory before they can establish remaining work.

- Receiver-name repairs #700/#879/#880/#883 have one reconciled declaration and
  reference inventory in [the service-client plan](service-client-member-identity.md).
  Existing NameScope owns allocation within each actual scope. Do not infer
  closure of unrelated package namespaces from these receiver checks.
- #702 and #884 repair separate admission and automatic-example owners; preserve
  their distinction and valid custom representations when extending either.
- Select subsequent work by unresolved correctness contracts and impact, not
  ticket closure throughput. State the owner, superseded paths, affected
  consumers and family exit condition before implementation.
- Reuse existing focused tests, models and comparison evidence. A newly filed
  witness alone does not justify another harness, model or broad audit. Retain
  independent exact-diff review and atomic commits; rerun only evidence affected
  by a change. Record recovery state concisely instead of appending another full
  historical checkpoint.

## Current execution queue

1. **Filtered sink disposition (#768):** omission's implementation is frozen in
   `numeric-comparison`; root owns independent exact-final review. The user accepted one enforced
   effective filter per named group, conflicting joins rejected, separate groups
   for independent subscriptions and terminal acknowledgment of filtered events.
   Existing groups require coordinated migration. One typed membership record
   must retain filter identity through empty membership and metadata lifetime;
   no independent registry. Models, full affected races and bounded native Redis
   checks pass. Packet: `/tmp/loom-768-final-manifest.json`; not delivered yet.
2. **Request framing (#766):** projection owns implementation in
   `interceptor-accessor-union`. A shared finalizer publishes
   mapped Content-Length after body encoding, across typed/raw/form consumers.
   Zero EOF verification and close-once cancellation follow the bounded model.
   Actual HTTP/1 and HTTP/2 checks expose zero-header omission outside
   POST/PUT/PATCH; explicit unsupported zero fails before publication, while
   absent and positive values remain supported. Final review is pending.
3. **JSON-RPC response representation (#905):** diagnostic implements the
   user-accepted uniform result/error.data contract in `grpc-message-context`.
   Shared admission rejects authored response Header/Cookie mappings across
   unary, batch, SSE and WebSocket; delete unreachable metadata paths. Ordinary
   HTTP response mappings, JSON-RPC request metadata and framework-owned transport
   metadata retain their owners. Migrate affected fixtures and document usage.
4. **CI reconciliation:** the retained ticktock and pool guard witnesses are
   delivered as #904/#906. Latest required CI at `6f82fb52f` remained queued
   (`38496558589`); no overall green-CI claim. Reconcile actual new failures by
   owner without timeout increases or an automatic corpus sweep.
5. **Dependency gates:** #820 libopenapi
   [PR #658](https://github.com/pb33f/libopenapi/pull/658) and #800 go-redis
   [PR #4095](https://github.com/redis/go-redis/pull/4095) remain open at fresh read,
   heads `3180fde1` and `8df7cf3`; their checks pass and releases remain v0.41.6
   and v9.23.0. The #800 packet stays in `bounded-streaming-shutdown`;
   original-module fetchability gates delivery. No fork/cache workaround.
   Pool subordinate shutdown #687 follows #800; no closure by inference.

The latest required CI snapshot is `/tmp/loom-ci-218-runs.json`; queued runs do
not establish a green suite. Delivered #902/#903 address earlier expectation
failures; #904/#906 address the retained fixture witnesses. The dependency race
remains open.

Latest independent delivery records:

- #769/#764: exact root approvals in their `independent-final-verdict.json` files;
  #769 preservation/cleanup: `/tmp/loom-769-cleanup-receipt.json`.
- #789/#787/#780: exact final and composed integration approvals in their
  `independent-final-verdict.json` and `independent-integration-verdict.json`
  files; integration source hashes preserve all preceding owners. Cleanup:
  `/tmp/loom-789-cleanup-receipt.json`,
  `/tmp/loom-787-cleanup-reuse-receipt.json`,
  `/tmp/loom-780-cleanup-receipt.json`; all three cleanups are complete.

- #765: `/tmp/loom-765-independent-final-verdict.json` and integration verdict;
  preservation/cleanup: `/tmp/loom-765-cleanup-receipt.json`.
- #902/#903: `/tmp/loom-902-independent-final-verdict.json` and
  `/tmp/loom-903-independent-final-verdict.json`; exact unchanged integrations
  recorded in their `integration-sources.json` manifests.
- #747: `/tmp/loom-747-independent-final-verdict.json`; exact root manifest
  `/tmp/loom-747-final-manifest.json`, full log race and independent adapter race.
- #736: `/tmp/loom-736-independent-final-verdict.json` and
  `/tmp/loom-736-independent-integration-verdict.json`; cleanup receipt
  `/tmp/loom-736-cleanup-receipt.json`.
- #760: `/tmp/loom-760-independent-final-verdict.json` and
  `/tmp/loom-760-independent-integration-verdict.json`; exact integrated sources
  `/tmp/loom-760-integration-sources.json`.
- #774 final re-review and integration authorization:
  `/tmp/loom-774-independent-final-rereview.json` and
  `/tmp/loom-774-independent-integration-verdict.json`. Preservation and reuse:
  `/tmp/loom-774-cleanup-receipt.json`, `/tmp/loom-736-reuse-receipt.json`.
  The mapped JSON-RPC error-header emission residual remains preserved and
  unconfirmed as a separate ticket; it is not claimed repaired under #774.
- #901 final and integration approvals:
  `/tmp/loom-901-independent-final-verdict.json` and
  `/tmp/loom-901-independent-integration-verdict.json`; preservation/reuse:
  `/tmp/loom-901-cleanup-receipt.json`, `/tmp/loom-901-worktree-reuse.json`.
- #773 exact independent review:
  `/tmp/loom-773-independent-final-verdict.json`; preservation:
  `/tmp/loom-773-preservation.json`. The original direct negative and both
  held-reply/fanout controls establish key and registration identity separately.
- #710 final re-review and integrated authorization are
  `/tmp/loom-710-independent-final-rereview-verdict.json` and
  `/tmp/loom-710-independent-integration-verdict.json`. Eleven non-skill files
  match the approved candidate; two skills preserve newer package guidance.
- #856 final/integration authorization is
  `/tmp/loom-856-independent-final-verdict.json`. Direct races and six actual
  module builds/vet passed independently. Evidence preservation and clean reuse:
  `/tmp/loom-856-cleanup-receipt.json`, `/tmp/loom-856-worktree-reuse.json`.
- #901 read and claim assessments are
  `/tmp/loom-sink-startup-independent-assessment.json` and
  `/tmp/loom-sink-startup-claim-independent-assessment.json`; the latter also
  independently checks the bounded model and its explicit limitations.

- #899 exact final/integration approval is
  `/tmp/loom-899-independent-final-verdict.json`; all 15 integrated source hashes
  match the candidate, and independent focused race checks pass (2.012s).
  Evidence preservation/cleanup and clean worktree reuse are complete in
  `/tmp/loom-899-cleanup-receipt.json` and `/tmp/loom-899-worktree-reuse.json`.
- #686 final and integrated approvals are
  `/tmp/loom-686-independent-final-verdict.json` and
  `/tmp/loom-686-independent-integration-verdict.json`. All 28 non-skill source
  files match the approved candidate; two skills retain newer publication guidance.
  Exact inputs are archived in `/tmp/loom-686-final-inputs.tar.gz`; cleanup and
  clean worktree reuse are complete in `/tmp/loom-686-cleanup-receipt.json` and
  `/tmp/loom-686-reuse-receipt.json`.
- #715 final/integration approval is `/tmp/loom-715-independent-final-verdict.json`.
  All 19 source files matched the candidate; 45 artifacts repeated exactly.
  Cleanup and clean worktree reuse are recorded in
  `/tmp/loom-715-cleanup-receipt.json` and `/tmp/loom-715-worktree-reuse.json`.

- #781 final and integration verdicts are
  `/tmp/loom-781-independent-final-verdict.json` and
  `/tmp/loom-781-independent-integration-verdict.json`. Integration direct and
  actual generated-module races pass. Cleanup is complete in
  `/tmp/loom-781-cleanup-receipt.json`; #899 exact clean-parent transfer is recorded
  in `/tmp/loom-899-worktree-transfer.json`.
- #751 final and integration verdicts are
  `/tmp/loom-751-independent-final-verdict.json` and
  `/tmp/loom-751-independent-integration-verdict.json`. The latter also approves
  the one stale staging-location assertion found during integration; failed and
  corrected evidence are retained separately. Cleanup/reuse is complete in
  `/tmp/loom-751-cleanup-receipt.json` and `/tmp/loom-774-reuse-receipt.json`.
- #897/#898/#900 each have an independent final verdict and a cleanup receipt
  under `/tmp/loom-<number>-independent-final-verdict.json` and
  `/tmp/loom-<number>-cleanup-receipt.json`. Exact sources, parent expectations and
  compact verification inputs are archived; owned inactive raw logs are removed.

- #688 final verdict is `/tmp/loom-688-independent-final-verdict.json`;
  `/tmp/loom-688-cleanup-receipt.json` preserves delivered source/input archives
  and compact evidence after removal of five inactive temporary trees.
- #749 final and integration verdicts are
  `/tmp/loom-749-independent-final-verdict.json` and
  `/tmp/loom-749-independent-integration-verdict.json`. Integration runtime races
  pass (HTTP 9.638s, JSON-RPC 2.446s, observation 1.356s); both generated modules
  pass (5.343s and 5.827s). Cleanup is complete; the exact-source receipt is
  `/tmp/loom-749-cleanup-receipt.json`.
- #893 final and integration verdicts are
  `/tmp/loom-893-independent-final-verdict.json` and
  `/tmp/loom-893-independent-integration-verdict.json`. Actual current-root
  shared-error/reference integration passes in 89.157s. Cleanup/reuse is complete:
  `/tmp/loom-893-cleanup-receipt.json` preserves 103 exact source/input/evidence files.
- #894 final and integration verdicts are
  `/tmp/loom-894-independent-final-verdict.json` and
  `/tmp/loom-894-independent-integration-verdict.json`. Integration races pass;
  `/tmp/loom-894-cleanup-receipt.json` and `/tmp/loom-770-resume-receipt.json`
  preserve delivery and exact candidate restoration.
- #770 final/integration verdicts are `/tmp/loom-770-independent-final-verdict.json`
  and `/tmp/loom-770-independent-integration-verdict.json`. The exact candidate
  integrated unchanged; only checkpoint documents differed in its parent.
  Cleanup/reuse is complete in `/tmp/loom-770-cleanup-receipt.json` and
  `/tmp/loom-771-reuse-receipt.json`.
- #753 final doc re-review and integration verdicts are
  `/tmp/loom-provenance-independent-final-rereview-verdict.json` and
  `/tmp/loom-provenance-independent-integration-verdict.json`. Actual two-design
  and three-fixture comparisons retain only intended manifest changes; current
  root interceptor compilation and provenance/vet/CLI seams pass. #742 has its
  own original-witness reconciliation in the closing comment.
- #896 exact four-file snapshot adoption was independently approved in
  `/tmp/loom-896-independent-final-verdict.json`; three comparisons and compiled
  root/nested/named collection controls pass. No production change.

- #895 final/integration verdicts are `/tmp/loom-895-independent-final-verdict.json`
  and `/tmp/loom-895-independent-integration-verdict.json`. Nine original generated
  compiler witnesses and mounted HTTP/JSON-RPC controls pass. Exact input and
  source preservation, cleanup and reuse are recorded in
  `/tmp/loom-895-delivered-preservation.json`, `/tmp/loom-895-cleanup-receipt.json`
  and `/tmp/loom-895-reuse-receipt.json`.
- #771 final/integration verdicts are `/tmp/loom-771-independent-final-verdict.json`
  and `/tmp/loom-771-independent-integration-verdict.json`. Full affected runtime
  race suites pass; root independent focused races pass (1.387s/1.367s). The model
  exposes drop/overwrite failures and checks strongest-event retention. Cleanup
  and worktree reuse completed before #751; the compact cleanup receipt is
  `/tmp/loom-771-cleanup-receipt.json`.


Completed cleanup receipts preserve compact evidence and exact reproduction
inputs after activity checks: #872 `/tmp/loom-872-cleanup-receipt.json`, #790
worktree `/tmp/loom-790-cleanup-receipt.json` and root integration
`/tmp/loom-790-root-integration-cleanup-receipt.json`, #699
`/tmp/loom-699-cleanup-receipt.json`, and #682
`/tmp/loom-682-cleanup-receipt.json`. Cleanup is also complete for #697, #723,
#700, #698 and #877, with `/tmp/loom-<issue>-cleanup-receipt.json` preserving
source archives, exact inputs and compact validation records. #701 cleanup also
completed without disturbing active #703 inputs. #881/#882 and their CI triage
temporary outputs were removed after inactivity checks and completed reviews.
`/tmp/loom-881-882-cleanup-receipt.json` preserves exact archived inputs, artifact
ledgers, reviews and compact validation results.

#703 implementer and root temporary outputs were removed after independent review
and activity checks. Exact source/input archives, corrected parent Git archive,
ledgers, verdicts and compact results remain. Receipts:
`/tmp/loom-703-cleanup-receipt.json` and
`/tmp/loom-703-root-cleanup-receipt.json`. Its worktree was safely reused for #702.

#782 temporary generated/model directories and raw logs were removed after review
and inactivity checks. Exact reproduction inputs, compact results, hashes and
review records remain: `/tmp/loom-782-cleanup-receipt.json`, SHA
`a3cb6d73fc9fd8a458c69a5cf0f3fecb51b2abe152bef1b338f441072d90d44b`.

#871 temporary checker trees and raw logs were removed after independent review
and activity checks. Committed regression/model inputs, manifests, patches,
verdicts and condensed results remain. Receipt:
`/tmp/loom-871-cleanup-receipt.json`, SHA
`97bdec7f539a0eccb8116a6c848fbccd66436e2f122b6842bb069d2ef9fd8a6f`.

#734/#758 temporary generated/build/model trees and raw logs were removed after
review and activity checks. Source archives, exact reproduction inputs, compact
results and review/artifact records remain. Receipt:
`/tmp/loom-734-758-cleanup-receipt.json`, SHA
`7d23025229e6c782e93e23edd4604a7ab037354db76073cfa4da39ff92e79321`.

#726 and #867 temporary generated/build/checker trees and raw logs were removed
following review and inactivity checks. Durable inputs, manifests, artifact
ledgers and integration logs remain. Receipts:
`/tmp/loom-726-cleanup-receipt.json` (SHA
`cd5229c0a81908b343abcbcfefc109dcc580748466bb5aa46fbd5a0f93c8fc87`)
and `/tmp/loom-867-cleanup-receipt.json` (SHA
`21194a49dde6555b6e88f2a892daec2be7d991bc29b8c0d3a02e6946e1ef9c12`).

#870 temporary generated trees and raw command logs were removed after independent
review and inactivity checks. Exact fixture inputs, artifact hashes and compact
results remain: `/tmp/loom-870-cleanup-receipt.json`, SHA
`e603d82602ff90d4762629d331a98c5b0288a12ca41be652f53fab51e7038187`.

#735 and #859 temporary outputs were cleaned after review and activity checks;
compact evidence and exact reproduction inputs remain. Receipts:
`/tmp/loom-735-cleanup-receipt.json` and `/tmp/loom-859-cleanup-receipt.json`.
The #859 receipt SHA is
`759468134a8138bf1215715265ab68488e740193408be66ca4ba085d2368c72c`.
#865/#866 raw logs were also condensed after delivery; see
`/tmp/loom-865-866-cleanup-receipt.json`. Source worktrees and shared tools remain.

Recent temporary cleanup completed for #847, #854, #720/#861/#862, #740, #863
and #864 after process checks and preserving compact evidence plus exact inputs.
Receipts are `/tmp/loom-N-cleanup-receipt.json`, with #720's receipt covering
#861/#862 too. The #863 receipt SHA is
`7df59a9b6cb9acc1ea16d1fec21c10c93eccdddd74e97aa5f763c17dd45c71a1`.
#853 cleanup also completed after #735 confirmed no remaining dependency; its
receipt is `/tmp/loom-853-cleanup-receipt.json` (SHA
`3c7b1359e797b4f653eb1171dac321b64be5a285e398d75f8d698dac105b449b`).
Source worktrees and reusable toolchains remain intact.
Earlier #858/#860 receipts remain available at the same per-ticket paths.

Delivered #693/#801 temporary cleanup removed 19 inactive directories after
process checks, preserving compact evidence and installed tools. Receipt:
`/tmp/loom-693-801-cleanup-receipt.json`, SHA
`d0ce5a6b205b0cf897256dc92ac9ae4aa41c23960f3953ea42a7c261fcdd6b4b`.
#803 cleanup already removed 30 paths; receipt
`/tmp/loom-803-cleanup-receipt.json`, SHA
`a3133a7f4c26f7666bee343a167e7cbda60000c9da8e11fafec761b58a949469`.
Managed protected worktrees remain intact. Prior cleanup receipts stay in the
owning delivery history; retain active #800 artifacts. Delivered #799/#804/#808 cleanup removed 86 inactive
paths (~316 MiB) after process/lsof checks. Receipt
`/tmp/loom-delivered-799-804-808-cleanup-receipt.json`, SHA
`a307fbf7b3c17e74fd5fab00db8cef67ea5df9fa45e11861364811052fb829b7`.
Comparison JSONs and copied command logs remain in the comparison roots. The
condensed #804 evidence is `/tmp/loom-804-validation-summary.json`, SHA
`b8e7b6b285ba8c2461c6f574771cabc0dd8fd9d7da2c4dbfa20197988adef3a8`.
Archival was refused for protected/other-task worktrees; source checkouts remain.
#810 temporary comparison/model outputs and raw logs were subsequently removed
after process checks; compact evidence is retained in
`/tmp/loom-810-cleanup-receipt.json`. Its protected worktree remains attached.

Delivered #679/#680 cleanup removed 39 inactive model/log paths after process
checks, preserving compact evidence in `/tmp/loom-debug-delivery-cleanup-receipt.json`
(SHA `0bc1bc6e8d73bb664df9f93cb4c4ccc30f5e62a53001ba9692f75985901a4fdd`).
#823/#819/#824 cleanup removed 6,908 inactive generated/model/log files after
all dependent comparisons completed; compact receipt is
`/tmp/loom-completed-union-receipts/completion-receipt.json`, SHA
`c2c8ab0ca9202a5d0de80e665553b462767afd09550953bcb6bf5127165c1e1d`.
Source worktrees and active #829 reproduction inputs remain intact.

## Current CI repairs

- [x] #696 builtin uint bounds: `90aaa5f89`, independently reviewed, pushed and closed.
- [x] #805 JSON-RPC shared required validator test: `a9285e5ec`, reviewed, pushed and closed; generated build/vet and 22 mounted wire cases pass.
- [x] #806 checked numeric design errors: `6f1297e7a`, reviewed, pushed and closed; nine invalid cases pass and complete DSL/expr lint is clean.
- [x] #807 exact numeric bounds in the test-only OpenAPI contract clone: `4b459a121`, reviewed, pushed and closed; precision/copy controls, both supported semantic roundtrips and shared-status checks pass.
- [x] #808 schema annotation helper split: `baf64781d`, reviewed, pushed and closed; size/IR/lint gates pass and 33 generated artifacts match parent/repeat exactly.
- [x] #809 distinct JWT/API-key fixture headers: `55cd4ca77`, reviewed, pushed and closed; AND semantics retained, selected corpus/build/vet pass, all 47 artifacts repeat exactly.
- [x] #810 shared validator physical naming: `60fe5ec9a`, reviewed, pushed and closed; independent shared/protobuf regressions, three models and 91 repeated generated artifacts pass.
- [x] #811 variadic validator callback harness: `285da6b0e`, reviewed, pushed and closed; existing generated compile/vet/runtime assertions pass unchanged.
- [x] #813 HTTP alias validator assertions: `0a1d0a928`, reviewed, pushed and closed; diagnostic path prologue and value predicates are checked separately, generated integration passes.
- [x] #814 preallocated numeric conformance cases: `110d19521`, reviewed, pushed and closed; all 15 cases and order preserved, existing source extraction regression and lint pass.
- [x] #812 test-only OpenAPI semantic reference comparison: `2f6f3aabe`, reviewed, pushed and closed. Lossless reference occurrence overlays, immutable registry and exact literal cloning; generated/schema/lint controls pass.
- [x] #727 selected SSE body ownership: `00fc9ba8b`, independently reviewed, pushed and closed. Shared selected-view constructor, mapped omitted-field semantics, 207 deterministic artifacts and full fixture CLI checks pass with prerequisite #818.

- [x] #815 fixture lint: `0b5657422`, reviewed, pushed and closed; three mechanical fixture extractions preserve all 112 compared artifacts. Full main lint now passes.
- [x] #733 collection identity: `91d126f38`, reviewed, pushed and closed; shared physical reference owner, 64 repeated artifacts, original six compiler failures corrected and mounted round trips pass.
- [x] #816 proof-gate fake inventory: `b756c06c3`, reviewed, pushed and closed; all 26 orchestration cases pass without changing production proofs/scripts.
- [x] #728 effective JSON-RPC WebSocket auth admission: `408caf2a2`, independently reviewed, pushed and closed. Resolved binding/source-role admission rejects 42 invalid designs, preserves mounted body authentication and all 95 supported artifacts; explicit fixture lint and integrated owner/mounted checks pass.
- [x] #817 shared JSON-RPC mapped error source: `68299ad7b`, independently reviewed, pushed and closed. Three emitters and SSE endpoint failures share the prepared mapping; 83 deterministic artifacts, mounted race checks and regenerated ticktock controls pass.
- [x] #818 shared annotation codec boundary: `d172c4620`, independently reviewed, pushed and closed. Retained declared defaults/enums survive opaque target codecs; invalid builtin/source controls, models and 84 deterministic artifacts pass.
- [x] #821 occurrence-path assertions: `0edf714f8`, independently reviewed, pushed and confirmed closed. No production change; 119 artifacts unchanged and mounted validator controls pass.
- [x] #822 JSON query carrier assertions: `5bb31fd19`, independently reviewed, pushed and confirmed closed. No production change; 48 artifacts unchanged and mounted collision/null/malformed controls pass.
- [x] #825 socket-close observation: `bf47b5d43`, independently reviewed, pushed and confirmed closed; deterministic gated witness, two isolated 100-iteration race runs, unchanged generated bytes.
- [x] #679 Debug response transparency: `fa5897578`, independently reviewed, pushed and confirmed closed; shared bounded capture and response capabilities, accepted-byte and real-network SSE checks pass.
- [x] #823 union branch carrier: `73451e4da`, independently reviewed, pushed and confirmed closed; review found and corrected predicate-free pointer admission, 78 artifacts repeat exactly and integrated regressions pass.
- [ ] Active CI repair: #820 YAML merge race; #819 is delivered. Aggregate release failures remain consequences, not independent tickets.

CI audit update: completed job `115077869674` at `abeb200e9` contains exactly the
three lint failures repaired by #815. Full local lint also passes on the exact integrated source delivered as `68299ad7b`
(`/tmp/loom-main-lint-817-integration.log`). Newer main CI remains running/queued;
active union and YAML race repairs prevent an overall green
claim. The proof harness is fixed in #816. No runs were canceled.

Observed CI `38328082038` on `55cd4ca77` passes OpenAPI contract, generated
code quality, service/expr/gRPC/JSON-RPC compile partitions and real Redis.
Its `ci` job fails the preallocation lint now repaired by #814. HTTP compile,
race and value-contract jobs were still running at this checkpoint; newer main
runs are queued. No overall CI success is claimed and no runs were canceled.


Delivered #812/#815/#816/#733/#728 cleanup removed 136 inactive paths (~211 MiB)
after process checks. Compact evidence: `/tmp/loom-delivered-cleanup-receipt.json`,
SHA `6cf99fd004f808a24736009347d3cc3dbf6a19fa11008c63cb98a5a3453c94c8`.
Protected managed worktrees remain intact; active evidence was preserved.


CI run `38335487607` at `063198a2f` now passes OpenAPI contract, all five exported-design compile partitions, generated-code quality, value-contract and real Redis. Main and race jobs have now failed, with aggregate Release eligibility also failed. Newer main checks remain queued; this is not an overall green claim.

Delivered #818/#727/#817 cleanup removed 108 inactive roots (22,601 files) after process/lsof checks. Compact receipt: `/tmp/loom-delivered-818-727-817-cleanup-receipt.json`, SHA `773e1df8a4fcfbeb6a417db5bf88adb1fde3cda0bd3015b12ec21e82d543a5e3`. Protected managed worktrees remain; active evidence and shared toolchains were preserved.

Delivered #825 cleanup removed 26 inactive temporary roots after open-file checks.
Compact receipt: `/tmp/loom-825-cleanup-receipt.json`, SHA
`567434480f1908998ec11677214bb09f417fda513d175ac1a6f8de9fd0780946`.
Managed archive refused the protected worktree; its source remains intact.

- [x] #824 tagged union null admission: `e90a9e45f`, reviewed, pushed and closed; shared semantic branch fact, 114 deterministic artifacts and generated wire matrix pass. #829 projected declaration limitation remains open.
- [x] #830 HTTP contract goldens: `66c617456`, reviewed, pushed and closed; three stale expectations aligned, 40 generated artifacts unchanged, targeted decoder/WebSocket families pass.
- [x] #828 span completion observation: `0d5736d64`, reviewed, pushed and closed; recorder publication precedes completion signal, exact snapshot assertion retained, integrated race count10 passes.

CI run `38338504807` at `73451e4da` failed the three stale goldens now repaired
by #830 and the unresolved #820 libopenapi race. Its five exported-design compile
jobs, generated quality, OpenAPI, value-contract and real-Redis checks passed.
The earlier span observation failure is repaired by #828. No overall green CI
claim is made; later commits require their own CI results.

## Delivery log

- 2026-10-12 — Delivered #824 `e90a9e45f`, #830 `66c617456`, and #828 `0d5736d64`, independently reviewed, pushed and confirmed closed. Inventory contains 152 reports through #831, with 73 delivered. Integrated patches: #824 `c4986911ad69a75377711c8e7232294bef2a82da69ed4038b1bf623deb7ef0c0`; #830 `069f0bc7e394d40862367b26318499ea559e1e396d8dd4bc7050251bfa50bd74`; #828 `d5a70ea2523d9abe76fdbf703416e95f1c48a38467a06abacd5b9c7efb2a77df`. User accepted rejection of contradictory inherited bounds; #695 resumes. #684 exposed a parent validation-identity prerequisite; #829 and #827 readiness are assigned. #820 and #800 remain separate dependency-owner work.


- 2026-10-12 — Delivered #825 `bf47b5d43`, #679 `fa5897578` and #823 `73451e4da`, each independently reviewed, pushed and confirmed closed. Exact integrated patches: `38497f7dedd3b0785811c097a9d7f43478f01a5c43da66aa70a575b7b44ead01`, `2aacc33664cc85310517169755db4a0798b0fa2771e1e6cd5a72d4024b73c5b4` and `b9f5203c956da00a1478509af5c278b4d4a3d5cce855e8ed76bbd72143e7ee34`. All review findings were corrected and confirmed, including #823’s medium predicate-free pointer gap. Inventory remains 147 reports, now 67 delivered/closed. #819 integration follows its delivered prerequisite. #695 inherited admission awaits the user’s answer; independent request observation and metadata defaults continue.


- 2026-10-12 — Delivered #821 `0edf714f8` and #822 `5bb31fd19`, separately reviewed, pushed and confirmed closed. Their exact integrated patches are `aadecd1a5568df4b49564337a8e8dcaf577ae9169f44e472de69bcd1b5a2765d` and `f9a7c37424ff0ab50b19be0d96cec72da47bf945e60a00e681735a79004b7ca0`. #822 LOW harness assertion finding was corrected and confirmed. Inventory has 147 reports through #825, 64 delivered/closed. The user authorized libopenapi report publication and delegated Debug capture policy; root published upstream #657 and adopted 64 KiB with explicit omission. Implementation/final review remains active for the queued repairs.

- 2026-10-12 — Delivered #818 `d172c4620`, #727 `00fc9ba8b` and #817 `68299ad7b`, independently reviewed, pushed and confirmed closed. Their exact integrated patches are respectively `62ff55c93bcb0191a0a4280955892fb8a7ff32d698ffac45dd81023e9c6fa38f`, `ed57456e980a00f4802b1330f4758815ea89d94c0ea0e54b7ce47533a2392f9a` and `9251d3068ebd173efed44276c41c63994cb2bf665b4558e5958609b0262267c9`. #818/#727 review LOW findings were corrected and confirmed. Full integrated main lint passes. Inventory has 145 reports through #823, with 62 delivered/closed. #821/#822 are frozen test-only candidates; #823 precedes #819; #820 upstream report and #679 body-capture policy await user answers. No overall CI green or dependency delivery is claimed.


- 2026-10-12 — Delivered #810 `60fe5ec9a`, #811 `285da6b0e`, #813 `0a1d0a928` and #814 `110d19521`, each independently reviewed, pushed and confirmed closed. Inventory now contains 136 reports, 54 closed. #810 integrated patch `6c149e72e8b4188a1dc1d3a50f9c5ae9303e6ddb5911d3d997c230f0bdad6f0a`; #811 patch `693d3f301784214e0e89a8daabd7f2c65b96b3155f024f2d5615b0c5f35469e4`; #813 patch `9dd6003eae518c1d22e82aa18172547d2fcdffc4a5861ff51983205e5368bce5`; #814 patch `8386c0f60516277b5ef6e70c41ff406b6999c2ed2efb9380d3ec3c570cfb20fe`. `/root/review_object_default_readiness` authorized each exact diff separately. #812 readiness is independently approved and implementation is active; #727 readiness is assigned. #800 remains dependency-gated.

- 2026-10-12 — Baseline: `4535a403048510f055fc59ec3dfed41e3769ad58` on `main`; initial checkout clean. Plan drafted; independent readiness review pending. No implementation is complete.
- 2026-10-12 — Root orchestrates and owns this tracker. Authentication and projection agents may inventory isolated worktrees while readiness review runs. Implementation starts only after their milestone is ready.

- 2026-10-12 — Auth owner: agent `/root/auth`, worktree `/Users/luca/.codex/worktrees/auth-contract/loom`. Projection owner: agent `/root/projection`, worktree `/Users/luca/.codex/worktrees/recursive-projection/loom`. Both are inventory-only pending slice readiness. Root recited and adopted the test-first, model-first, bounded-evidence, independent-review and atomic-delivery workflow.
- 2026-10-12 — Inventory contains all 114 open reports in #679–#793 plus #670. Independent reviewer requested exact per-slice readiness records and full unary/stream error-disclosure inventory; corrections are being applied.

- 2026-10-12 — Planning artifacts independently reviewed by `/root/plan_review`, committed as `eb5a169b`, and pushed to `origin/main`. Auth readiness: `/Users/luca/.codex/worktrees/auth-contract/loom/roadmap/auth-contract-readiness.md`; projection readiness: `/Users/luca/.codex/worktrees/recursive-projection/loom/roadmap/recursive-projection-readiness.md`. Both await final scoped readiness approval.

- 2026-10-12 — Authentication #724 readiness approved independently; implementation released. #725 is its declaration-resolution witness. Decoder alternatives, constraints and stream admission remain separate tasks. Pristine baseline worktree: `/Users/luca/.codex/worktrees/correctness-baseline/loom`; do not mutate it. Error-disclosure preparation assigned to `/root/plan_review` in `/Users/luca/.codex/worktrees/safe-error-disclosure/loom`.

- 2026-10-12 — Projection #788 readiness approved independently and implementation released. Auth baseline tests and two legacy TLA controls reproduced #724/#725; corrected model checks 2048 states and direct expr/security/service suites pass during implementation. Final generated evidence and exact-diff review are still pending. Public decisions on Extend redeclaration and collection presence were requested asynchronously; dependent changes remain gated.

- 2026-10-12 — Error disclosure #713 readiness independently approved by root; implementation released. Legacy public-message model and direct tests reproduce leaks; corrected model and direct regressions pass during implementation. Generated unary/SSE/WebSocket evidence and final review remain pending. Auth generated security checks and bounded output/determinism comparisons pass; its final evidence record is being prepared.

- 2026-10-12 — **Delivered #724 / #725 witness:** commit `544c57f3` on parent `eb5a169b`, pushed to `origin/main`. `/root/review_auth` approved the exact 19-file integrated diff (SHA-256 `c40b9ae4c7279e9013b35811702995c441a6cfe28ce787dcccace4e89f79571e`) with no findings. Reviewer independently reran expr/security/service, focused three-transport security tests, both failing legacy models and the passing 2,048-state model. Paths, failing baseline, intended generated differences and commands are recorded in `roadmap/auth-contract-readiness.md` and `expr/tla/authentication_scope/README.md`. Regenerate consuming adapters. Remaining authentication tasks stay open.

- 2026-10-12 — Projection #788 passes generated HTTP/gRPC wire checks during implementation. #784 reproduces after the shared projection fix: HTTP client/server union declaration registries need separate ownership; it is excluded from #788, as is SSE body selection #727. Error #713 passes direct runtime and generated gRPC/HTTP/JSON-RPC unary/SSE/WebSocket regressions; bounded baseline/output comparisons remain in progress.

- 2026-10-12 — Priority refinement: promote P1 file-overwrite #670 from residual reconciliation to the next integration slot after the first wave, before Milestone 5. Agent `/root/artifact_ownership` owns readiness preparation in `/Users/luca/.codex/worktrees/artifact-write-intent/loom` at `eb5a169b`. The scope must preserve write intent through rendering and transactional installation; #741 is being reconciled as a consumer-policy witness, while cross-device #751 stays separate. No source implementation is released until readiness approval.

- 2026-10-12 — #724 validation scratch/model output was removed after review and delivery. Managed archive of `/Users/luca/.codex/worktrees/auth-contract/loom` was refused because the checkout is protected by a pinned task or workspace; it remains attached and is no longer needed for implementation. Durable evidence is committed.

- 2026-10-12 — Recovery update committed/pushed as `7d736078`, independently reviewed by `/root/review_auth`. #713 frozen 22-file candidate plus three root skill edits integrated/staged on that parent. Fresh reviewer `/root/review_errors` found one high-severity history disclosure: a nil/code-only outer remedy hides an intermediate SafeMessage from the separate history traversal. No commit authorized. Implementer `/root/plan_review` is repairing the common owner and adding the counterexample; re-review is required.

- 2026-10-12 — #670 readiness approved for external create-only artifacts and typed intent propagation. Root requested the public choice for create-only files inside wholly regenerated `gen/` (reject vs preserve); that dependent behavior remains gated. Agent may model/reproduce and implement independent external-file behavior. #741 is confirmed a separate declaration fix and will get its own atomic follow-up.

- 2026-10-12 — **Delivered #713:** commit `63e96099` on parent `7d736078`, pushed to `origin/main`. `/root/review_errors` confirmed the high-severity history finding fixed and authorized the exact 26-file diff (SHA-256 `182bd56beffe229363f32a882f173370f35da2e741b9e2a829e76a316a9b9974`). `pkg.ResolveErrorMessage` now supplies message and explicit-safe-override identity; gRPC history uses the same owner. Evidence/commands: `pkg/tla/error_disclosure/README.md`. Generated bytes unchanged; runtime upgrade suffices. Task temporary evidence was removed after review. Managed archive was refused as protected/pinned, so the completed error worktree remains attached.

- 2026-10-12 — **User accepted three public contracts explicitly:** Extend refines inherited constraints without weakening them; explicit empty collections remain distinct from absence; create-only declarations inside `gen/` are rejected and remain supported outside it. #670 agent notified; gen-tree rejection is unblocked. These are target contracts, not claims that later Extend/presence implementations have shipped.

- 2026-10-12 — #788 frozen 32-file candidate plus 3 shared skill edits integrated/staged on `63e96099`; `/root/review_projection` owns fresh final review. Six baseline wire probes fail and six candidate probes pass with `-race`; 30 bounded CLI generation/build/vet receipts pass, all 124 generated files match independent repeats. Exact differences and scope: `codegen/service/tla/recursive_projection/README.md`. No projection commit authorized yet.

- 2026-10-12 — Milestone 5 readiness assigned to `/root/websocket_lifetime` in `/Users/luca/.codex/worktrees/websocket-lifetime/loom` at `7d736078`. Source untouched pending root review. Existing runtime owner is incomplete; connection lifecycle, single-reader ownership and full-inbox behavior need explicit invariants.

- 2026-10-12 — **Delivered #788 / #730 and #779 witnesses:** commit `ce3f6a73` on parent `63e96099`, pushed to `origin/main`. `/root/review_projection` approved the exact 35-file diff (SHA-256 `e5843f8923c78f3f61bc284e3fb4482b25491b0b1eb476082376e4acac06fd67`) after a low stale-skill reference was corrected and re-reviewed. Reviewer reran full expr/codegen/service and HTTP projection/presence tests, verified 124 byte-identical repeats and 30 passing receipts. Regenerate adapters. #784 and #727 remain separate unresolved tasks.

- 2026-10-12 — WebSocket readiness approved for models/regressions only. Root requested the public overload contract (explicit close/cancel at configurable buffer bound, recommended, versus backpressure with delayed disconnect detection). #716 is a separately scoped proposed prerequisite using the existing HTTP 32 MiB individual-message bound and configurable policy; no lifecycle source edits until contract/dependency readiness is settled.

- 2026-10-12 — Recovery checkpoint committed and pushed as `096a793b`, independently reviewed by `/root/review_auth`. Completed projection scratch outputs were removed; its managed worktree could not be archived because it is protected by a pinned task or workspace.

- 2026-10-12 — #670 candidate integrated and staged on `096a793b`. `/root/review_artifacts` found a P1 in the exact 28-file diff: a stable live symlink such as `alias -> gen` bypasses lexical create-only admission, so replacing `gen/` can destroy the supposedly protected file. **No commit authorized.** `/root/artifact_ownership` is repairing shared live-path admission and adding the counterexample in its existing worktree. Re-review is required; root must preserve its integrated shared documentation deltas when copying repairs.

- 2026-10-12 — Milestone 6 readiness assigned to `/root/effective_design` in `/Users/luca/.codex/worktrees/effective-design-validation/loom` at `ce3f6a73`. Proposed owner #755 with #757 as a shared composition-order witness. Existing effective-constraint analysis must own refinement; multiple-base precedence and phase dependencies require concrete readiness review. No implementation released.

- 2026-10-12 — #716 prerequisite exploration now covers the decompressed message size as well as Gorilla's framing limit. Root accepted a configurable, symmetric 32 MiB default concept for HTTP/JSON-RPC clients and servers, with protocol close 1009 for oversized messages. Exact attachment/configuration API readiness remains pending before production edits. #722 overload semantics still await the user's answer; legacy models and direct/generated red regressions are available in the WebSocket worktree.

- 2026-10-12 — #716 concrete API approved: immutable `http.WebSocketConfig` owns the message cap and existing configuration hook; an unattached `WebSocketStream` configures and adopts the socket through one `SetConn` path. HTTP per-endpoint configuration and JSON-RPC configuration pass the same value. Context-carried limits were rejected during design because JSON-RPC shares connections across calls. Agent may implement this separate prerequisite after red tests/models, updating its readiness record first. #722 pump/overload behavior remains gated.

- 2026-10-12 — #755/#757 readiness independently reviewed by root; models and failing regressions released. Explicit nullable overlay provenance and identity/codec metadata classifications must be recorded before production changes. #741 follow-up readiness is drafted in `roadmap/example-interceptor-ownership-readiness.md`; root reproduced both server/client overwrites and cwd-only suppression with an overlay probe. That follow-up still awaits independent readiness approval and a post-#670 parent.

- 2026-10-12 — #755/#757 production implementation released after root reviewed the completed nullable/metadata readiness delta and model correspondence. Ten red cases reproduce the intended defects; three preservation controls pass. Four model controls fail as expected and the corrected bounded model passes five invariants across 246 states. These are design/reproduction results, not final implementation acceptance. Agent `/root/effective_design` continues in its recorded worktree.

- 2026-10-12 — **User accepted the WebSocket overload contract:** close the connection and cancel its context when the configurable buffer limit is exceeded. Agent `/root/websocket_lifetime` notified. #716 remains the separate message-size prerequisite; #722 follows with bounded raw-message accounting and explicit overload termination. The previous pending public-policy question is resolved; implementation and final validation are still outstanding.

- 2026-10-12 — #670 alias repair integrated and staged on `096a793b`; `/root/review_artifacts` owns re-review of the 31-file final diff (SHA-256 `5ae6031981d106005daa9579ae499a3e77212e99dde68ebff2b4f1349b2e78b6`). Original reproduction now passes; live parent resolution, absent descendants, filesystem identity and reserved case variants are covered. Agent validation and refined model pass, but commit remains gated on re-review. #741 readiness is being reviewed separately and is excluded from this staged candidate.

- 2026-10-12 — **Delivered #670:** commit `41b4a8a3` on parent `096a793b`, pushed to `origin/main`. `/root/review_artifacts` confirmed the P1 alias finding fixed and authorized the exact 31-file diff above with no unresolved findings. Independent original-reproduction, policy/renderer/generator/CLI, real-plugin, vet and whitespace checks pass; filesystem-identity tests ran without skips. Durable evidence: `cmd/loom/tla/artifact_write_intent/README.md`. Update the CLI and design-module dependency together. #741 and cross-device #751 remain separate.

- 2026-10-12 — #741 readiness independently approved by `/root/review_artifacts`. Follow-up worktree created at `41b4a8a3`: `/Users/luca/.codex/worktrees/example-interceptor-ownership/loom`. Its declaration-only correction reuses #670's ownership machinery; source implementation is next after this recovery checkpoint. Root retains the red overlay probe until its regression is incorporated.

- 2026-10-12 — Recovery checkpoint independently reviewed, committed and pushed as `6183551e`. Agent `/root/artifact_ownership` is implementing #741 in the new worktree above; the old #670 worktree is frozen. Completed #670 comparison, checker, integration and reviewer scratch directories were removed after checking active processes. Managed archival was refused because the old worktree is protected by a pinned task or workspace; its source remains attached.

- 2026-10-12 — #741 is frozen: direct/service tests and unchanged goldens pass; 26 generated/example files match its immediate parent and an independent repeat; real CLI probes preserve edited files and no longer suppress fresh output because of cwd files. Validation exposed a separate inherited JSON v1 lint violation in the recursive projection wire-test harness. Root reproduced the lint failure, switched only that harness to JSON v2 with explicit envelope tags and raw-value retention, and reran lint plus both generated wire cases successfully. `/root/review_artifacts` is reviewing that one-file maintenance change before #741 integration. #708 proxy-trust readiness is separately drafted for review; source is untouched.

- 2026-10-12 — Test-maintenance correction independently reviewed with no findings and delivered as `93f513be`, pushed to `origin/main`. `/root/review_artifacts` approved exact one-file diff `c057e2e3727be1c85cc3d640161625d6f6e1e491e68a23be646cc9645b5a72a2`; JSON v2 lint and both recursive projection wire cases pass. No production behavior or generated output changed.

- 2026-10-12 — **Delivered #741:** commit `23f2af5d` on parent `93f513be`, pushed to `origin/main`. `/root/review_artifacts` approved the exact four-file diff (SHA-256 `2ce00171da55e3602d5f42a029ad2dced8c0d8c0b4bccee1997a3500e1b641db`) with no findings, independently checked regressions and all 26 output comparisons/CLI receipts. Evidence: `roadmap/example-interceptor-ownership-readiness.md`. Example interceptor declarations now use the shared create-only writer. #708 readiness was also independently approved; its implementation is the next available assignment.

- 2026-10-12 — #708 implementation assigned to `/root/forwarded_client` in `/Users/luca/.codex/worktrees/forwarded-client-trust/loom`, parent `23f2af5d2e4ad9c9b659309839af4f3b25b3f8e3`. Independently approved readiness is copied into that worktree. The shared request-metadata owner must stop at invalid forwarding hops; test/model-first implementation is released. No generator sweep is required for this handwritten runtime correction.

- 2026-10-12 — Completed #741 comparison and readiness-probe scratch was removed after independent review, delivery and an active-process check. Managed archive of `/Users/luca/.codex/worktrees/example-interceptor-ownership/loom` was refused as protected by a pinned task or workspace; its source stays attached. Durable reproduction and results remain in the committed readiness record.

- 2026-10-12 — User explicitly set the goal to work through the entire tracked backlog without stopping at status updates. Root verified GitHub state: owning #724, #713, #788, #670 and #741 were already closed by their pushed commits; root closed resolved witnesses #725, #730 and #779 as completed. Recovery checkpoint `4c7bd8d7` is pushed. All eight delivered reports are now closed. #716 and #708 candidates are frozen for integration/review; #755 continues final validation. A historical JSON-package reference in #741 evidence triggered the Markdown lint scan; its wording is corrected without changing the evidence, and delivery/cleanup status is made current.

- 2026-10-12 — **Delivered #708:** commit `9880865a` on parent `26f252ef`, pushed to `origin/main`; GitHub confirms closure. `/root/review_artifacts` authorized the exact 12-file integrated diff (SHA-256 `20cfa227f31d7ebc41599a83bac44be89b1e8f0ca3d0e3a554dbd2915a2a566e`) with no findings. Root and reviewer independently passed affected race suites; vet, JSON lint and whitespace checks pass. Evidence: `http/tla/forwarded_client/README.md`. Runtime upgrade suffices. Task-owned scratch was removed after review/process check; managed worktree archival was refused as protected/pinned.

- 2026-10-12 — #716 frozen 124-file patch integrated onto `9880865a` without conflicts. Source hashes match the agent candidate except the HTTP guide retains newer #708 text. Agent `/root/websocket_lifetime` is validating the three selected designs against this immediate parent before fresh independent review. Root owns shared guidance and final integration; the 19 pending #722 files remain excluded. Next authentication readiness is prepared by `/root/auth_readiness`; no production work is released.

- 2026-10-12 — #732 readiness independently accepted by root; implementation released to `/root/auth_readiness` in `/Users/luca/.codex/worktrees/auth-credential-alternatives/loom`, parent `9880865a1c2873cf455fbae979dc7de3a533071f`. Shared alternative-aware extraction must preserve payload-wide Required and malformed-matching-credential rejection; #731, #743, #729, #752 and #728 remain separate. Readiness: `roadmap/auth-credential-readiness.md`.

- 2026-10-12 - #716 current-parent comparison passes for all three selected designs (68 output files) and generated limit/error/projection controls. `/root/review_websocket` reviewed staged 128-file hash `2dd9446272195b7bb1ee85884864d8537bf954ae5cfb7933f55b5bcb02649e66` and found one medium cancellation-cause regression: later reads returned a low-level socket error instead of the documented closed sentinel. No commit authorized. `/root/websocket_lifetime` is repairing the common terminal transition in its worktree, with model and deterministic regressions before re-review.

- 2026-10-12 - #755/#757 candidate frozen in its recorded worktree: 43 files, manifest `/tmp/loom-effective-final-manifest.json` SHA-256 `dd83997cce82eb51e72d4d818379a82aaf2d0997a7913256a08271d840541bcc`, base `ce3f6a73`. Its six specimens compile/vet and 151 output files match independent repeats. Integration and current-parent comparison follow #716 and its required CI-maintenance follow-up; no delivery claimed.

- 2026-10-12 - Remote CI for `9880865a` failed its `ci` job with three maintainability-index violations (`codegen/generator.Generate`, `codegen/service.projectionConversion.typ`, `internal/artifactpolicy.Validate`) and two test-slice preallocation warnings. These are required follow-up repairs to earlier deliveries. Managed worktree `/Users/luca/.codex/worktrees/correctness-ci-gates/loom` at `9880865a` is owned by `/root/ci_gates`; private helper extraction and test preallocation are implemented and validating. Root full golangci on integrated #716 reproduced exactly those five baseline diagnostics and no new #716 diagnostics. #743 admission readiness investigation is assigned to `/root/grpc_admission`; source implementation remains unreleased; its readiness checkpoint is saved in `roadmap/grpc-stream-admission-readiness.md` and will resume after CI repair capacity is available.

- 2026-10-12 - #716 medium terminal-cause finding fixed and independently re-reviewed by `/root/review_websocket`; exact final 133-file diff SHA-256 `369ec8cbdd0c36340a251f8a3039abbdc41281cc66428d299299b929be204b8a` approved with no unresolved findings. Committed locally as `46b627ef`; push follows CI cleanup. Main runtime race tests and vet pass. CI cleanup seven-file candidate is staged for `/root/review_ci_cleanup` on this parent; integrated `make lint` is running. #732 frozen 23-file candidate is under independent review by `/root/review_auth_alternatives`.

- 2026-10-12 - #722 refreshed readiness assigned to `/root/websocket_lifecycle` in `/Users/luca/.codex/worktrees/websocket-lifecycle/loom` at `46b627ef`. Carry only the old lifecycle models/red witnesses from the previous worktree. Concrete buffer API and accounting must handle zero-length messages without unbounded queue metadata; production remains gated on refreshed readiness.

- 2026-10-12 - **Delivered #716:** `46b627ef` pushed with CI cleanup `651372ce`; GitHub confirms closure. CI cleanup independently approved by `/root/review_ci_cleanup`, exact seven-file staged hash `34102545cdf15df78a45cb10563c5c611e7761ddd1694edf52e53332b79923f8`. Full lint in the isolated source and bounded byte comparison passed; root integrated lint also exited zero, but overlapped the next integration and is not independent acceptance evidence. #755 integrated 43 frozen files with exact parent/candidate hash verification; `/root/effective_integration` owns six-specimen validation against `651372ce`.

- 2026-10-12 - #722 refreshed readiness approved for production after current-parent red witnesses and refined models. Explicit message count bounds empty frames; shared terminal owner covers attachment, handler completion/panic and bounded final writes. `/root/websocket_lifecycle` implements in its recorded new worktree. #755 integration exposed three maintainability-index failures; `/root/effective_integration` is performing cohesive private extractions and revalidating unchanged output before final review.

- 2026-10-12 - Completed remote race job at `9880865a` additionally failed custom-package projected-carrier references (`common.CardView` and framework-name variants). Root reproduced the existing generator witness on `651372ce` and prepared a separate ordering repair in `/Users/luca/.codex/worktrees/projection-carrier-packages/loom`; readiness and four-state failing/passing carrier-location model are saved there. Production remains gated on independent readiness review. This is separate from #733 and #784. CI maintenance scratch was cleaned after delivery/review; worktree archival was refused as protected/pinned.

- 2026-10-12 - #755 final staged 46-file diff `970388e34ca9eaa56ec482d65c54b2d225ff593977f49d77e62304fda2664f9d` is under `/root/review_effective_design`. Reviewer reproduced a medium blocker: mixed recursive field/inheritance paths are mistaken for inheritance cycles. No commit authorized; full findings and repair/re-review follow. Root custom-package follow-up is frozen as seven files, manifest `/tmp/loom-projection-packages-final-manifest.json`, SHA-256 `71a015da22ecda29218ca83a45c724aaa342c4495868c2e1f36077c67cdf9553`; 279 generated files match independent repeats and selected compilation/wire/lint checks pass. #732 low stale-readiness finding corrected in the root draft; final integration re-review still required.

- 2026-10-12 - #755 independent review completed with two medium findings: valid mixed recursive field/Extend graphs fail in either declaration order (false cycle or incomplete copied graph), and conflicting peer JSON identities evade admission when declared through different metadata keys. `/root/effective_integration` is repairing both shared owners with reviewer repros, model/test updates and focused revalidation; `/root/review_effective_design` will re-review. No commit authorized. #743 design now favors shared pre-frame authentication admission plus preserved normal endpoint rechecks; an exactly-once callback assumption was rejected because the public authorization contract permits repeats.

- 2026-10-12 - `/root/grpc_admission_design` approved the seven-file custom-package follow-up against `651372ce`, manifest `71a015da22ecda29218ca83a45c724aaa342c4495868c2e1f36077c67cdf9553`, with no findings. Independent direct/model/279-file byte checks passed; final main integration review still required. CI at `651372ce` clears lint but both ci/race jobs fail the same projected-package matrix; no other test failure was observed. Root approved #743 revised readiness and copied it into the main draft; assignment waits for #732. Completed #716 scratch and old lifecycle repro modules were removed after the #722 owner confirmed they were superseded and active-process checks passed.

- 2026-10-12 - #731 readiness/red-only preparation assigned to `/root/basic_credential_design` in `/Users/luca/.codex/worktrees/basic-credential-validation/loom` at `651372ce`. Coordinate with frozen #732 alternatives; no production changes until scoped readiness approval and a stable dependency parent.

- 2026-10-12 - During #755 repair root identified an additional metadata comparison concern: raw `attributeJSONTag` strings do not include the name-only tag default `omitempty` applied by the renderer. New compatibility controls currently invert optional full/name-only equivalence. Implementer must align the rule with the actual tag owner and inherited/effective requiredness before freezing; this is not authorization for a separate transport policy.


- 2026-10-12 - Root confirmed the #755 metadata repair uses effective rendered JSON identity/options, following the accepted readiness rule. Optional full `one` versus name-only `one` conflicts; optional full `one,omitempty` versus name-only `one` is compatible; effective Required makes full `one` and name-only `one` compatible. Normalization belongs in `internal/encodingmeta`, with unchanged codegen delegation and existing requiredness/presence facts. The authored-partial-contract alternative was rejected. Both peer orders and local/inherited comparisons require rendered controls before re-review.

- 2026-10-12 - #731 readiness checkpoint is frozen with ten failing HTTP wire cases and two failing request-plan cases; valid/absent HTTP controls and twelve gRPC metadata controls pass. `/root/review_basic_readiness` is independently reviewing the scoped proposal. Production remains dependent on #732 integration. #722 bounded comparison passes: five parent/candidate designs compile and vet, independent candidate generations are identical, and differences are confined to expected WebSocket adapters; adversarial validation continues.

- 2026-10-12 - #731 readiness independently approved by `/root/review_basic_readiness` with no findings; reviewer reproduced HTTP failures and passing gRPC controls. Implementation awaits #732 integration. #755 resolved-tag tests also exposed inherited Required disappearing after intermediate composition consumed Bases; the existing AllRequired owner now traverses retained parent constraints, with multilevel red/green controls. Final repair gates are running.


- 2026-10-12 - Root began #681 inventory/red-only preparation in managed worktree `/Users/luca/.codex/worktrees/exact-numeric-bounds/loom` at `651372ce`. Twenty direct bound-preservation cases fail above float64's exact integer range; four small-number controls pass. The draft identifies DSL/effective representation as the shared owner and keeps #685 value-range admission and #692 target array literals separate. Exact representation, model evidence and readiness review remain outstanding; no production change is authorized.


- 2026-10-12 - #722 froze 145 implementation paths, then root added three shared guidance files in its isolated worktree. Combined 148-file manifest `/tmp/loom-ws-722-with-guidance-manifest.json`, SHA-256 `112cab2db6ea0a6971e3ab08c489e6e892d6b563a557dcab10844b79a335212a`, awaits `/root/review_effective_repair`'s separate verdict against `46b627ef`; final main integration remains pending. Independent runtime race, shutdown model and all fifteen retained generation manifests pass review so far. The full HTTP package timeout remains explicitly recorded.

- 2026-10-12 - #732 root guidance and the previously requested stale-readiness correction are now frozen in the auth worktree. New 26-file manifest `/tmp/loom-auth-with-guidance-manifest.json`, SHA-256 `de893d9595419757c6d180b3f7c70262b45dcb88924dc01d147e27a0aaeddcae`; original implementation is unchanged, but these documentation deltas require final integration review. #681 readiness/model investigation delegated to `/root/exact_numeric_design` in the recorded worktree, without production authorization.


- 2026-10-12 - `/root/review_effective_repair` approved the frozen #722 148-file worktree with no findings, independently passing runtime races, generated lifecycle wire checks and the 252-state shutdown model. The same reviewer separately approved the #732 documentation correction/new guides against its unchanged implementation and the carrier follow-up's one-paragraph guide addition; the latter is now eight files, manifest `/tmp/loom-projection-packages-with-guidance-manifest.json`, SHA-256 `7ae367742687c24fd2c6a20c3a2ce919a749a0a01b9b095f30e12d44ee257c45`. Each still needs final integrated-diff authorization before its own commit. #755 canonical named-owner binding now passes direct self/mutual recursion and snapshot mutation checks; final generated/schema comparison is running.


- 2026-10-12 - **Delivered #755 / #757 witness:** `4f7892a2b6b351786616a69fe7bbc3bee9202375` pushed to `origin/main`; GitHub confirms both closed (twelve delivered reports total). `/root/review_effective_repair` authorized the exact 59-file staged diff `b59d7c2abf66832bdb298623a450b8682e492a182550ecde9bde9aee5e5a8249`, final manifest `f138d283f484d04eb352da73b296c3afbff209c6a8a82a9f22ad5e5e0b4407c5`, with no unresolved findings. Original recursion and JSON identity findings, requiredness and structural binding follow-ups, and both low documentation/style corrections were confirmed. Durable evidence: `roadmap/effective-design-readiness.md` and `expr/tla/effective_design/README.md`. Completed task scratch removed after reviewer completion and active-process check; source/tests/models are committed.


- 2026-10-12 - **Delivered projected-package follow-up:** `c735c6360047dd964a3fe309de31ed3b3d065973` pushed to `origin/main`. `/root/review_effective_repair` authorized the exact eight-file diff `0c2eb151864e44e6b62220c1f227c5852feb63437342cd00f6b47fa3da3bba2b` against `4f7892a2`, with no findings. All 279 integrated generated files match both the previously compiled candidate and a new independent repeat; direct regression and three existing package-name compile/build/vet cases pass. This repairs the known CI failure family; remote CI outcome is not yet claimed. #732 has now integrated cleanly and is running three-transport wire and bounded output checks.


- 2026-10-12 - **Delivered #732:** `433d2e741b43b35783d293d0de60cabeca0be6a2` pushed to `origin/main`; GitHub confirms closure (thirteen reports total). `/root/review_effective_repair` authorized the exact 26-file diff `ec42f77572914b68257e97438d6f39cad2158d74ec510dc2d83f90e2c584fcff` against `c735c636`, no findings. All 108 artifacts match independent repeats; three-transport wire/race tests, affected units and scoped lint pass. Evidence: `roadmap/auth-credential-readiness.md`. #731 production released to its existing worktree after safe fast-forward to this parent. #722 all 148 frozen files integrated cleanly; its agent owns final bounded validation, with a verified pushed module pin replacing the old temporary v0.0.0 input. #743 managed worktree creation is pending operation `43a7a4ce-31e8-467e-a680-8b330be348d8` at this parent.

- 2026-10-12 - #681 readiness froze eleven non-production files in its worktree, manifest `/tmp/loom-exact-numeric-readiness-manifest.json`, SHA-256 `065c1ac3379e2a1e35d063be4b2a8aa0bbd919d0fec12a7faf32bfe8f202d769`. Thirty-two DSL/import precision cases reproduce; source-admission/raw-node controls and nine bounded Lean claims pass. Root requested the genuinely unresolved public numeric-string syntax decision: exact decimal strings with hexadecimal strings rejected (recommended), or preserving hexadecimal strings with further cost/limit investigation. This question remains pending; no dependent parser policy or production is authorized.


- 2026-10-12 — **Delivered #722 / #711, #712 and #714 witnesses:** `56bc7eae` pushed to `origin/main`; all four reports are confirmed closed (seventeen total). `/root/review_effective_repair` authorized the exact 148-file integrated diff `1fe7645447eb6b09672e7f08f8ccbfe3b551aac8d4af43a5b30f822ccf19e8b8` against `433d2e74`, with no unresolved findings. Five designs produce 114 files identical to the reviewed candidate and independent repeat; focused runtime race and generated real-socket checks pass. Evidence: `roadmap/websocket-lifetime-readiness.md` and `http/tla/websocket_lifetime/README.md`. Regenerate adapters for the new shared configuration and context attachment contract. #739 and other protocol-specific findings remain open.

- 2026-10-12 — Completed #732 and #722 temporary comparison/checker outputs were removed after final review and active-process checks; durable evidence is committed. Managed archival of both worktrees was refused because they are protected by a pinned task or workspace. Their source remains attached; no manual worktree deletion was attempted.

- 2026-10-12 — #743 managed worktree creation completed at `/Users/luca/.codex/worktrees/grpc-stream-admission/loom`, parent `433d2e74`; approved production implementation released to `/root/grpc_admission_design`. Metadata snapshots and the shared authentication runner must preserve cancellation, callback context, declaration ordering and normal endpoint rechecks. #731 implementation has expanded its red HTTP/JSON-RPC cases; an existing defaulted-Basic CLI compile defect is retained separately, outside this decoder-validation repair.

- 2026-10-12 — #681 policy response: the user emphasized cross-language interoperability and did not accept decimal-only strings. Root withdrew the Go-literal justification and resumed readiness investigation with existing hexadecimal-string support retained as the candidate. No parser narrowing, new numeric complexity limit or production implementation is authorized. The earlier pending choice is superseded by this investigation.

- 2026-10-12 — Recovery checkpoint independently reviewed by `/root/exact_numeric_design`, committed and pushed as `b46345a0`; the exact two-file diff was `5a209b03d34b3c75f325867077cb10b537402b8e2c53836ab56f027c41b4ecde`. Root began #792 inventory/red/model preparation in `/Users/luca/.codex/worktrees/recursive-import-references/loom` at `56bc7eae`. Three recursive component shapes fail Go initialization; matching narrow named-lookup controls evaluate successfully. The bounded reference model reproduces the failure and checks the candidate over 19,683 classified graphs. Production remains gated on full reference-context readiness and independent review. #706/#719/#793 serialization was inspected only and remains separate.

- 2026-10-12 — **Delivered #731:** `601a4338b` pushed to `origin/main`; GitHub confirms closure (eighteen reports total). `/root/review_effective_repair` approved the isolated implementation after a low formatting correction; `/root/websocket_lifecycle` authorized the exact 204-file integrated diff `8c8d304eaa62f2240535c42f8296715e4c3035ddf99ca1ce1eab79c00a26b2e4` against `b46345a0`, no findings. Seven designs / 144 artifacts match the reviewed candidate and independent repeat; integrated HTTP/JSON-RPC wire/race and direct gates pass. Existing bounded compilation/fixture evidence is reused. Evidence: `roadmap/basic-credential-validation-readiness.md`. The separate, reproduced CLI default representation defect is filed as #794 and remains open.

- 2026-10-12 — #681 readiness v2 is frozen as seventeen non-production files, manifest `/tmp/loom-exact-numeric-readiness-v2-manifest.json`, SHA-256 `a5d3db15e3fef936dffe8cce2aa4c329435b41a86a250f2d32546599e4acd8ee`. Forty-four precision regressions reproduce across DSL, decimal import and YAML radix import; sixteen scoped Lean claims and bounded interval/protocol probes pass. Root released independent readiness review to `/root/review_effective_repair`; no production is authorized. Existing finite hexadecimal strings are preserved in the candidate, with source-specific JSON/YAML admission. Runtime JSON-shape numeric consumers and their existing capacity-failure contract require explicit reconciliation.

- 2026-10-12 — #792 readiness investigation delegated to `/root/websocket_lifecycle` in the existing recursive-import worktree. Root's red/model checkpoint remains the starting point; constructor-specific symbolic reference support, eager cycles and error/union paths need a concrete shared plan before production.

- 2026-10-12 — Completed #731 cleanup: removed 44 task-owned scratch paths after review and durable evidence preservation. Managed worktree archival was rejected because it is protected by a pinned task/workspace; checkout retained unchanged.

- 2026-10-12 — CI `38296412526` exposed three distinct regression causes. Root fixed expression test context isolation (exact reviewed diff `2c4e527b35b576392d962091fd1b36a22a02677600c824ea8dcd995a730852d6`, pushed `b7e1b1f8`; same shuffled full expr race suite passes 5.966s), then corrected the explicit-body seam assertion to distinguish declaration and physical value carrier (exact reviewed diff `4644ac78e68266ba6b70f1f533c30ccdfce924adad7e9d37bfbc4c1fb002540e`, pushed `27da0247`; focused nullable/body gates pass 0.833s). `/root/review_effective_repair` approved each atomic commit separately. No production was changed by either maintenance repair.

- 2026-10-12 — Filed #795 for confirmed generated WebSocket close cancellation before queued data/null consumption. Three union wire runs fail on `601a4338`; nested collection CI also fails. `/root/websocket_lifecycle` paused #792 readiness, preserved its model/reference-context checkpoint, and reproduced the missing lifecycle sequence in TLA+. Candidate bounded peer wait is under contract/seam investigation; no test weakening is authorized.

- 2026-10-12 — #681 v3 readiness independently approved by `/root/review_effective_repair`: nineteen-file manifest SHA-256 `e369c61af21823c15d07f7570cfe55fa18246f61fc906dad44ee2e8ad9e58f3b`. Runtime `pkg` shape matching and prepared schema membership now belong to the proposed common exact owner; evaluator errors must propagate fatally, and unrenderable imported overflow requires a precise diagnostic. Production released after the worktree safely advanced to `27da0247`, all nineteen readiness hashes preserved. This is implementation authorization, not delivery evidence.

- 2026-10-12 — #743 isolated 41-file implementation independently approved after both low findings were repaired. Final pinned comparison repeated all eighteen modules and six builds/vets; all 71 artifacts and intended differences match. Root integrated onto `27da0247`, added three owning guides/index entries, reran owners and generated races (21.183s), and repeated integrated output/compilation. Exact staged 44-file diff `d329325650f2067fe05471eb9a485d521423871f71c3239beb9c0be4c7e20ce3` is under final integration review. Trackers are excluded from that commit.

- 2026-10-12 — **Delivered #743:** exact final 44-file diff `d329325650f2067fe05471eb9a485d521423871f71c3239beb9c0be4c7e20ce3` approved by `/root/review_effective_repair`, committed and pushed as `d8ccf0ee`; GitHub confirms closure (nineteen reports total). Gofmt/filesize checks pass. Removed 39 completed task scratch paths after durable evidence preservation; managed archival remains blocked by protected-worktree status.

- 2026-10-12 — Root approved concrete #795 readiness after inspecting the new counterexample and bounded peer-wait model. Production released to `/root/websocket_lifecycle` in `/Users/luca/.codex/worktrees/websocket-close-order/loom`. One terminal deadline covers writer acquisition, final JSON and physical-peer wait; raw Close and half-close remain distinct. Model correspondence and synchronized socket regressions are mandatory before final review.

- 2026-10-12 — Recovery checkpoint reviewed after two low presentation fixes, committed and pushed `214bab44`; exact two-file diff `4a5f08ce25f4816eee99bcf4f0dcf974c0a7461911d53a729ac922a5ac74cf3a`. Inventory contains 117 reports including #794/#795.

- 2026-10-12 — #792 readiness independently approved, twelve-file manifest `be7d7efb2eb79856fceb4645d708e07194c74ec18c0391f9cc52e0b19275a2f6`. Root assigned implementation to `/root/review_effective_repair` (a different final reviewer will be required). Worktree advanced to `214bab44` with all readiness hashes preserved. Direct planner reds establish per-occurrence ownership and mandatory-cycle diagnostics before production; coordinated numeric/importer file boundaries with #681.

- 2026-10-12 — **Delivered #795:** fresh reviewer `/root/review_ws_close_order` approved exact 13-file integrated diff `ec13e6f6ac898172c5f66793f17e0ee87a9760adad511033c7b48def128b0740` against `214bab44`, no findings. Committed/pushed `2e4c8022`; GitHub confirms closure (twenty reports total). Integrated HTTP/JSON-RPC runtime race checks pass 7.606s/2.091s and original generated regressions pass 13.445s. Independent positive/negative model reruns agree. Durable evidence is `roadmap/websocket-close-order-readiness.md`; completed task scratch removed and protected managed checkout retained when archive was rejected.

- 2026-10-12 — Assigned #794 readiness to `/root/basic_cli_defaults`: shared Basic CLI argument metadata must agree with physical presence/default representation; no emitted-code workaround or unrelated M7 policy rewrite. Root retains production release and final delivery ownership.

- 2026-10-12 — **Delivered #794:** `8c7d8d711b9e2fe54fbff1dabb9b49fd361ed170` pushed to `origin/main`; GitHub confirms closure (21 reports). `/root/review_basic_cli_defaults` approved the exact seven-file integrated diff `abcdab8399a328bcfd0ab1773be527a1a6b188bdc59c1008109b90990bb39bb3` against `2e4c8022`, no findings. Four shared metadata additions restore each Basic credential’s effective CLI default. HTTP/JSON-RPC full generated CLI build/vet/race and default/override wire tests pass; 67 compared artifacts show exactly six intended CLI changes and zero repeat differences. Root integrated focused tests passed (6.998s/6.996s). Evidence: `roadmap/basic-cli-defaults-readiness.md`. M7 presence policy remains separate. All 18 completed `/tmp/loom-basic-cli-794*` scratch paths were deleted after review; managed worktree archival was refused as protected/pinned, so its checkout remains attached. The implementer now owns readiness-only #756/#729 in `/Users/luca/.codex/worktrees/effective-wire-bindings/loom`; seven current-parent reds and six controls precede the shared binding proposal.

- 2026-10-12 — **CI assertion maintenance:** `589e96b1` pushed after `/root/review_effective_repair` approved the exact two-file test-only diff `09e8afc9f656d5c1d9acba82ca9b87f05bf0553e10e4c51e7288fa994a6bc5e6` against `8c7d8d71`, no findings. Three assertions expected the shared decoder error variable inside the body declaration block; they now check both declarations separately and preserve EOF, validation and presence assertions. Current-parent reds reproduced and the focused optional object/value body tests passed (0.857s). CI run `38300194426` also failed on the already-repaired #795 and a Go proxy download error; this is not an overall CI pass.

- 2026-10-12 — **Delivered #792:** `58916bc96d8057874242c6fc330f44bc1c684274` pushed to `origin/main`; GitHub confirms closure (22 reports). `/root/review_recursive_import` approved isolated 22-file source and exact 25-file integrated diff `828d47c2ee300eb706c569403950e1b7ef424e002a0c5b89b3f24f901d742180` against `8b5d265c`, no findings. Independent focused acceptance passed 104.206s; root current-main direct/evaluated/generated acceptance passed 110.129s. Eighteen evaluated cases, nine durable generated cases, 192 exact-repeat artifacts and 24 Vacuum specs cover constructor-sensitive recursive references; 32 parent acyclic and 64 parent mechanism-control artifacts remain identical. Evidence: `roadmap/recursive-import-readiness.md` and `internal/openapiimport/tla/schema_references/README.md`. All 29 completed #792 scratch paths were removed after review; managed archival was refused as protected/pinned. The implementer now continues readiness-only #784 in the recorded view-union worktree; main/shared guide changes remain root-owned.

- 2026-10-12 — **Delivered #784:** `6194743c9485dd242762d230fabc05519ecc1765` pushed and closed. `/root/review_view_union_final` independently authorized the exact 25-file diff `721467ca5aad4483bc000fde33e7b0d7f12c845996d3ec4add3af115d30e256d` against `58916bc9`, no findings. Prepared body roots own package declarations and validators; one allocator retains union identity. Eleven modules / 267 artifacts compile/vet and repeat exactly, with five intended server-only removals. Independent focused tests passed 12.228s/10.276s; integrated tests 18.873s/13.880s. The corrected untagged controls exposed a separate required-pointer validation bug, filed as #796 with a durable parent reproduction.
- 2026-10-12 — **Delivered #692:** `febb41bce2d2b80b36f73c67316f6c7d2cc9d226` pushed and closed. `/root/review_typed_defaults_final` authorized exact nine-file diff `c0b83dc6e6407371dfdf1f64ff5fc6e012051f477aace483377b1c5bd83d2a7b` against `6194743c`, after confirming a mechanical low-severity import grouping fix. Shared recursive target-typed collection literals replace competing assignment/alias formatters. Forty representation cases, 32 application gates, native/Optional/Nullable race/vet checks and HTTP/JSON-RPC compilation pass. Seven specimens / 109 artifacts repeat exactly; six intended default-assignment files change. Integrated focused checks passed 5.023s/2.763s/2.301s. #685, #693 and M7 policy remain separate.
- 2026-10-12 — After final review and delivery, removed all 90 completed #784/#692 scratch paths, including generated trees, manifests, checker output and reviewer overlays. Durable source/probes and summarized evidence are committed in their readiness/model records. Managed archival refused both worktrees as protected/pinned; their attached checkouts remain intact. Active numeric, wire and #796 evidence is retained.


- 2026-10-12 — **Delivered #756 / #729 witness:** `f40d1677c60fa3d3e2c17d6103c96a3fec76e860` pushed and both reports closed (26 total). `/root/review_wire_bindings_final` approved isolated 27-file source and exact 31-file integrated diff `f9fa882e894e3ced125d8c93a28c1fa8d6b0946282713dab2a7a6cc53f19b9b3` against `dc88a33c2`, no findings. Eleven designs / 167 artifacts compile/vet and repeat exactly; reviewer verified all 501 parent/candidate/repeat hashes and extra binding probes. Integrated expr/HTTP/JSON-RPC/OpenAPI gates pass. Evidence: `roadmap/effective-wire-binding-readiness.md`. Selected built-in error body failure is separately filed as #797.

- 2026-10-12 — **Delivered #796:** `d0fefa59ed98f20ee4cfe52f4f2c16871135eb1d` pushed and closed (27 total). `/root/review_projected_validation_final` approved exact 15-file integrated diff `03b460defa865a6626f8d627ed1fda37974016d9bc1ce8b4cfe1267d87a86ce6` against `f40d1677`, no findings. Nine generated runtime/race fixtures and ten module builds/vets pass; 90 verified hashes show eight intended views-only changes and exact repeats. Integrated gates pass 24.293s/6.118s. Evidence: `roadmap/projected-union-validation-readiness.md`. #709 remains a distinct recursion repair.

- 2026-10-12 — **Delivered #762:** `07cf7ae41ede92ff9cbc9f9e580204489da29b64` pushed and closed (28 total). `/root/review_multipart_final` approved exact 18-file integrated diff `5348d273ed8baa2a65e7aae456c98750d65fc1c1ca60d4fb01934d8ed3ae6d09` against `d0fefa59`, no findings. Nine runtime/race fixtures, optional/array/map reviewer probes, ordinary goldens and lint pass. Twelve designs / 192 artifacts compile/vet, only eight custom server decoders change, all repeats identical; reviewer verified 576 physical hashes. Integrated tests pass 27.151s. Evidence: `roadmap/multipart-decode-ownership-readiness.md`.

- 2026-10-12 — Completed scratch cleanup removed 98 wire-binding, 33 projected-validation and 31 multipart task paths after review and delivery. Durable probes, models, hashes and concise results remain committed. Managed archival of the two wire worktrees and the projected/multipart worktrees was refused as protected/pinned; those checkouts remain attached. Numeric and deepObject evidence remains active and must not be deleted.

- 2026-10-12 — CI test repair delivered as `d57f96c6`: `stream_admission_test.go` now allows gRPC's documented Send EOF path and still validates the final Recv status, public problem, authentication count and service/receive barriers. A Header-synchronized rejection case fails the old assertion deterministically. Root's race/shuffle gate passes 16.050s; `/root/review_multipart_final` independently passes all three fixtures 16.571s and authorizes exact one-file diff `f619dbd835fdf026369e506338afc938317463e13c37a43d80eeb4e5c9ada246`, no findings. This fixes test ordering, not production behavior.

- 2026-10-12 — **Delivered #797:** `14ab5a758` pushed and confirmed closed. Independent reviewer approved the forty-file final and integrated diffs without findings, including public representation before projection, generic client reconstruction and whole-error formatter handling. Sixteen generated modules and malformed-response checks pass; 48 artifacts repeat exactly with nine intended Go differences; OpenAPI bytes are unchanged and validated. Both maintained SSE fixtures were regenerated/tested. Integrated manifest `e5ffdeebb6f5440b8eee4747b53e5f5c3c4f3f21b4a15758f1e671409a6c4f08`, diff `d24c996d610dddbcba2386b4d0d603142c4768c54294bac2388cb4e80d98b096`. Cleanup removed 46 task-owned temporary paths; receipt `/tmp/loom-797-cleanup-receipt.json` SHA `27d43b51917e8e35668c151ceaf1a81e2e42007c39baf4da2792da01a7c32b85`. Managed archive refused protected worktrees, left intact.
- 2026-10-12 — **JSON final correction review:** first final review found two medium defects: nested non-null map values lost null before validation, and the embedded external import harness imported a framework internal package. Both corrected through existing JSON presence/carrier/conversion ownership and a valid external harness; revised 105-file manifest `/tmp/loom-uniform-json-revised-final-manifest.json` SHA `b38b484912fd2c982368e8d6640f27e7d2460ffe5a53d8d2c261e76eec46e136`. `/root/review_uniform_json_corrections` owns re-review. No delivery is claimed yet; preserve #709 during eventual integration of the two HTTP registry files.
- 2026-10-12 — **#802 readiness approved and implementation released:** a medium interleaving gap was corrected by consuming the first event before fanout mutation. Revised manifest `54cea2e692309478b28c42f016a699a41363994244d04f8a11069cd3e0ae5990`; seven expected parent failures, seventeen passing controls under both race tiers, four model outcomes independently confirmed. `/root/review_value_admission_readiness` is implementing ordered registration identities in `/Users/luca/.codex/worktrees/reader-registration-identity/loom` on `af05fa8aa`. #773 remains separate. Sequence remains #802 → #800 → #687.
- 2026-10-12 — **#693 readiness started:** root reproduced four accepted object defaults that generate map-to-struct compile errors; scalar and typed collection controls pass. `/root/object_default_readiness` owns `/Users/luca/.codex/worktrees/object-default-representation/loom` on `30cabf633`, with its own durable readiness plan and admission/generation probes. Reconcile already-repaired admission with the remaining shared target-literal representation before implementation; no production is authorized yet.
- 2026-10-12 — **#709 cleanup:** 133 task-owned temporary paths removed after validation/review. Exact manifests, delivery and compact evidence preserved in `/tmp/loom-recursive-709-cleanup-receipt.json`, SHA `5284747237a8dcccb7d2619aa4fea15ffd5308824b0de2007794d5c208bb8fdb`. Managed protected worktree remains intact.

- 2026-10-12 — **Delivered #802:** `81192e7a9` pushed and closed. Ordered registration objects replace parallel slices; read admission carries identity through reply/fanout and cursor completion only updates the same installed registration. Independent final and integrated reviews approved without findings. Race suites, real Redis, vet and model checks pass; no Remove-drain guarantee is added. Integrated manifest `28d84847dccf850e42aacceda536e3382f8e47babde85b4e453546d204958c5d`, diff `64d3c3e2c401a4f095432c2ed987fe76abc385a8a42c34327582bd9855614f61`.
- 2026-10-12 — **Delivered #706 / #719 / #793 witnesses:** `1d94c755e` pushed and all three closed. The accepted uniform JSON contract derives runtime, generated carriers, OpenAPI and import admission from `internal/httpencoding`. Both medium review findings were corrected and independently confirmed; exact integrated 108-file diff approved without findings. Main overlap checks cover recursive/named-nullable HTTP/JSON-RPC/gRPC and problem projection; fresh parent/candidate/repeat comparison checks twelve artifacts, six intended changed files and generated build/vet/wire. Integrated manifest `71dfbf73b02569fdd07985f99a0241bc8d9a0d228ffa6ea8038742f25875fd33`, diff `3523d68aacde7ea279ad633a1eddb350cfb89030ae8bda8d6acdbf28c370c2cc`. Regenerate clients and servers together; update handwritten bracket requests.
- 2026-10-12 — **Reconciled and closed #798:** the original saved object-enum design now generates/builds/vets against pushed `81192e7a9`. Six generated-validator runtime controls pass with race checking (1.375s): members 1/2 and absent optional object accepted; 3/-1 and present incomplete object rejected. The delivered #709 named-validator owner supplies `&actual` to the complete pointer-taking validator. No separate workaround or production change was needed. Receipt inputs/logs are `/tmp/loom-798-current-parent*`; the durable design remains `expr/lean/value_projection/numeric_object_enum_generator_probe.go.txt`.
- 2026-10-12 — **Filed #803:** root independently reproduced an anonymous optional object containing named children generating `actual != nil` on a value struct, without defaults/enums, at `30cabf633` and the integrated JSON candidate. Durable probe in the #693 worktree is `codegen/tla/default_representation/anonymous_nested_probe.go.txt`; it is isolated from #693. This is not the now-fixed named object-enum case.
- 2026-10-12 — **#693 implementation released:** independent readiness review approved eleven-file manifest `d28135cf2390fcc1bd3d4bdb465261bc89625af00172358a1fa9d9b8361c0b38`, no findings. Eight generated object shapes and Native/Optional/Nullable carriers reproduce the source-literal defect; ordinary gates pass. Typed model passes fifteen states, legacy/shallow fail ten. `/root/object_default_readiness` implements one resolved-value/target-context renderer and lossless default representability checks; ordinary open-object decoding and default-application gates remain separate.
- 2026-10-12 — **#801 readiness approved:** `/root/review_import_credentials_readiness` approved twelve-file manifest `35e7bcfc3bc768800fc1ae8dae82e5c1a05984d831cbb7db90d2488426beadae`. Proposed importer physical-source grouping preserves all role tags and AND/OR requirements; source scope lists that the DSL cannot represent must fail explicitly. Eleven reds, twelve controls, generated mounted authorization and a 728-state model were independently confirmed, with no findings. Implementation is authorized by the readiness gate; its release follows this checkpoint.
- 2026-10-12 — **#800 readiness continues:** `/root/review_value_admission_readiness` is testing bounded native Redis operations and close ownership after #802. A driver probe disproved the proposed `WithTimeout` facade's hook-preservation premise: it drops caller ProcessHooks in the pinned driver. No production change or upstream publication is authorized from that rejected design. Preserve native hook/metric semantics; investigate supported alternatives and a self-contained dependency reproduction before selecting the owner.

## Recovery and delivery

Read this entire file, repository `AGENTS.md`, `.agents/skills/loom-framework/SKILL.md`, and the applicable model README before resuming. Inspect `git status`, `git log`, attached managed worktrees, and agent status before assigning work. Never assume a running agent or uncommitted diff is abandoned. Do not use git clean, stash, reset, or checkout.

Milestones are ordered by priority and integration dependency. Milestones 2 and 3 may be implemented concurrently in isolated managed worktrees; integration remains sequential. Milestone 4 can start when a slot becomes available. Later milestones require a bounded readiness review of their concrete inventory and acceptance matrix before implementation. Do not turn the whole issue list into an undifferentiated rewrite.

One structural task has one owning ticket and one atomic commit. Before implementing a family, root selects one existing owning structural ticket and records related reports as regression witnesses in a readiness record; do not combine unrelated tickets in one commit. Record the ticket, parent SHA, worktree, changed paths, failing baseline, passing commands, intended generated-output differences, review verdict, commit SHA and push status in this file's Status section. Findings are not complete merely because code is written. A checked delivery item means reviewed, committed, pushed, and integrated.

Before any implementation edit, the task owner writes a readiness record under `roadmap/` naming the owning ticket, intended parent, included and excluded reports, exact file/test/model loci and checker commands, accepted/rejected fixture cases, baseline/current byte-comparison method, compile commands and intended output differences. An independent reviewer must approve that concrete slice. The broad milestone checklist alone never releases implementation. Record deeper public-policy blockers separately and continue independent ready tasks.

Before every commit, an agent that did not implement the change reviews the exact final diff against its intended parent. State the acceptance rule and run its checks first. Resolve every finding according to AGENTS.md, obtain re-review, and make no changes between authorization and commit. Integration that changes a reviewed diff requires re-review. Root alone updates this tracker and `roadmap/ROADMAP.md`; implementation agents hand back documentation deltas affecting shared skills for coordinated integration.

Use direct regressions and affected packages, then bounded generated comparisons against the baseline: compare exact bytes, list intended differences, compile generated Go, and repeat generation in independent processes for determinism. Use existing harnesses and representative fixtures; the full exported-design corpus is not a task gate. Set `LOOM_DIR` to the implementing worktree for local temp-module tests. Preserve model sources and summarized evidence; remove task-owned temporary output only after review and after checking active users.

## Accepted contracts and decision boundaries

- Nearest scope declaring `Security`, `SessionSecurity`, or `NoSecurity` owns the complete authentication declaration. Same-scope alternatives retain documented OR/AND composition; contradictory anonymous-only and authenticated declarations fail validation.
- Projection recursively preserves only the selected view through objects, collections, unions, and supported recursive references. Excluded-field changes cannot alter serialized output. Transport encoders consume this projection.
- Unknown service errors disclose a generic public message while retaining their cause for server-side diagnosis. Designed public errors and native protocol termination keep their contracts.
- Defaults fill absence and never replace explicit zero. Presence is independent of value. Explicit empty scalars survive encoding. Explicitly present empty arrays/maps must survive as empty values, distinct from absence. This supersedes the old nil/empty policy when Milestone 7 is implemented; model presence explicitly rather than infer it from length.
- Runtime owners control resource lifetime; generators provide typed adapters. Worker publication must not expose an unready worker. Lost replies require reconciliation rather than assuming rollback.
- WebSocket buffer overload is terminal: close the connection and cancel its context when its configured buffer budget is exceeded. Do not silently discard a message and continue. Bound raw-message storage explicitly and document memory outside that bound; the independently configured per-message limit remains a separate check.
- Local scheduler cancellation is provisionally intended to stop local participation, not delete the global schedule. Consumer evidence and a recorded contract decision are required before changing that behavior.
- `Extend` same-type redeclarations refine inherited constraints without weakening guarantees. For example, an inherited minimum 0 plus derived maximum 100 admits 0–100; incompatible types, contradictory constraints, or weakening inherited requiredness fail design validation. Preserve authored intent and document the exact composition of defaults/metadata in slice readiness.
- `gen/` is wholly framework-owned and regenerated. Reject create-only (`SkipExist`) declarations inside it before live mutation. Outside it, create-only files preserve any existing destination and create an absent destination atomically.

## Milestones

### Milestone 1: Durable inventory and execution readiness

Goal: Establish recoverable ownership and a reviewed first implementation wave.

Acceptance Criteria

- An independent reviewer approves this plan's first-wave milestones against current code; later work remains explicitly gated on its scoped readiness review.
- Every open code-review finding #679–#793, excluding closed duplicate #775, has a recorded family or residual disposition in `roadmap/framework-correctness-inventory.md`. Include referenced older #670. Dispositions do not assert unverified fixes.
- Worktree paths and owners are recorded in Status before implementation.

Checklist

- [x] Read this plan and the framework skill; recite milestone order, exit criteria, named commands, test-first procedure, independent-review gate, commit/push handoff, and repository constraints before editing implementation.
- [x] Fetch reports with `gh issue list --repo CaliLuke/loom --state open --label code-review --limit 200 --json number,title,body,labels,url`; reconcile their owners in `roadmap/framework-correctness-inventory.md`.
- [x] Review first-wave loci with `rg -n 'EffectiveRequirements|inheritedRequirements|inheritedSessionAuths|buildConstructorCode|ErrorSafeMessage|newErrorResponse' expr codegen security http grpc jsonrpc pkg` and map every relevant consumer to acceptance coverage.
- [x] Obtain independent readiness and exact-diff review of the planning artifact; run `git diff --check`; commit and push the initial recovery plan.

### Milestone 2: One effective authentication contract

Goal: Resolve authentication once and preserve it across extraction, validation and execution.

Acceptance Criteria

- `expr/method_test.go` and security tests prove API/service/method overrides in both session/ordinary directions, same-scope alternatives, anonymous alternatives, and rejection of contradictory NoSecurity declarations.
- Generated HTTP, gRPC and JSON-RPC security tests prove usable Basic/Bearer alternatives, credential constraints and matching effective OpenAPI requirements. Metadata-only authentication runs before blocking gRPC reads; message-carried credentials retain explicit ordering constraints.
- Models reproduce the old scope/ordering failures and check the corrected invariants; exact baseline/output comparisons cover selected security fixtures and compile emitted adapters.

Checklist

- [x] Deliver #731 Basic credential validation at `601a4338b`; independently reviewed, pushed and closed. Evidence: `roadmap/basic-credential-validation-readiness.md`. Separate CLI default compile issue #794 remains open.
- [x] Deliver #732 credential-alternative extraction through shared security analysis; reviewed, pushed and closed at `433d2e74`. Evidence: `roadmap/auth-credential-readiness.md`.
- [x] Deliver #724 nearest-scope ownership with #725 contradiction witness; independent review, commit `544c57f3`, push and integration complete. Evidence: `expr/tla/authentication_scope/README.md`.
- [ ] Inventory `rg -n 'EffectiveRequirements|SessionAuths|Requirements|DecodeBearer|BasicAuth|writeGRPCStreamInputDecode' expr security codegen/service http/codegen grpc/codegen jsonrpc/codegen`; reconcile #724, #725, #729, #731, #732, #743, #752 and #728 by exact cause, splitting distinct structural tasks before implementation.

- Read-only follow-up reconciliation: #729's blanket suggested rejection conflicts with the delivered, supported Basic-or-Bearer alternative contract (#732). #756 establishes explicit duplicate wire-name loss; #729 adds implicit security mappings after validation. Investigate one effective wire-binding owner before validation, but distinguish a Basic composite credential and disjoint authorization framing alternatives from two independently authored values overwriting one slot. Preserve HTTP header case-insensitivity, query/cookie case sensitivity, explicit/implicit/inherited mappings, and valid same-name mappings in different locations. Do not close both issues from a generic duplicate-name check without these controls.
- [ ] Add failing direct regressions at `expr/method_test.go` and relevant transport security tests; record the failing commands and scope-selection model before implementation.
- [x] Replace independent security inheritance in `expr/method.go` with one effective owner; derive consumer behavior from it and delete superseded selection paths.
- [ ] Run `go test ./expr ./security ./codegen/service -count=1`; run bounded security tests in `./http/codegen`, `./grpc/codegen`, `./jsonrpc/codegen`, recording exact filters and generated fixtures.
- [ ] Complete credential-alternative and stream-admission tasks only after their own test-first, model, generated comparison and independent-review obligations are ready; keep payload-dependent authorization at its valid boundary.
- [ ] Update `docs/dsl-reference.md`, relevant authentication guides and consumer skill; obtain exact-diff review, commit each owning structural ticket atomically, push and integrate.

### Milestone 3: Recursive view projection

Goal: Make selected views authoritative for every nested output shape and transport.

Acceptance Criteria

- Direct service conversion tests prove arrays, maps, unions, adjacent result fields and recursive supported objects use the same selected-view semantics in both conversion directions.
- Generated HTTP, JSON-RPC and gRPC tests prove excluded secrets never appear, including streaming results; declaration order cannot change projection or produce inconsistent body declarations.
- Selected normal and adversarial fixtures generate deterministically and compile; excluded-field perturbation leaves wire output unchanged.

Checklist

- [x] Deliver projected-package ownership follow-up `c735c636`; reviewed and pushed, with 279 deterministic generated artifacts and existing package-name compilation controls. Evidence: `roadmap/projection-carrier-packages-readiness.md`.
- [x] Deliver #788 recursive selection and conversion ownership with #730/#779 witnesses; independent review, commit `ce3f6a73`, push and integration complete. Evidence: `codegen/service/tla/recursive_projection/README.md`.
- [ ] Inventory `rg -n 'buildConstructorCode|ViewScope|Project|viewedResultBodyInit|collectHTTPUnionTypes' codegen/service http/codegen grpc/codegen jsonrpc/codegen`; reconcile #730, #779, #788, #784 and #727, recording distinct causes before grouping.
- [ ] Add failing direct tests beside `codegen/service/views_test.go` and generated wire regressions using existing view fixtures; record baseline failures and a projection ownership model before changing conversions.
- [ ] Replace direct-field special handling and generic semantic fallback in `codegen/service/service_data_views_init.go` with one recursive view-aware conversion owner; align view/body declaration collection with the selected projection.
- [ ] Run `go test ./codegen/service -count=1`; run bounded viewed-result tests in `./http/codegen`, `./grpc/codegen` and `./jsonrpc/codegen`, recording exact filters and fixtures.
- [ ] Compare selected baseline/current generated bytes, compile them, repeat generation in separate processes and record every intended difference in durable evidence beside the owner.
- [ ] Update `docs/code-generation.md` and affected consumer guidance; obtain exact-diff review, commit the structural ticket atomically, push and integrate.

### Milestone 4: Safe public error disclosure

Goal: Apply one public-disclosure policy while preserving each protocol's error classification.

Acceptance Criteria

- A plain internal error containing a sentinel secret never exposes that text in HTTP bodies, gRPC status/details or JSON-RPC envelopes; the server retains the cause.
- Designed errors, safe-message overrides, wrapped errors, joins, context cancellation and deadlines retain their explicitly tested classification and public contracts. Unary and stream paths are covered separately where their renderers differ. `grpc/error_contract.go` retains classification ownership; tree-wide errors.As must not select one joined branch as aggregate disclosure policy.

Checklist

- [x] Inventory `rg -n 'ErrorSafeMessage|newErrorResponse|EncodeServerError|InternalError|err.Error' pkg/error_remedy.go pkg/error* http/error* http/codegen/error* http/codegen/service_data_response_errors.go grpc/error.go grpc/error_contract.go grpc/error* jsonrpc/codegen/handler_sections.go jsonrpc/codegen/stream_sections_websocket.go`; reconcile #713 without replacing protocol classification with a generic status policy.
- [x] Add failing disclosure regressions to runtime error tests and generated transport tests; inspect existing error-ownership models before implementing the common public-message decision.
- [x] Implement the shared disclosure owner in `pkg` and make the three transport error boundaries consume it.
- [x] Run `go test ./pkg ./http ./grpc ./jsonrpc -count=1` and bounded generated error tests in `./http/codegen`, `./grpc/codegen`, `./jsonrpc/codegen`; record exact filters, baseline comparison and compile evidence.
- [x] Update `docs/error-handling.md`; obtain exact-diff review, commit #713 atomically, push and integrate.

### Milestone 5: Complete stream lifetime ownership

Goal: Make runtime owners authoritative for connection closure, cancellation and serialized operations.

Acceptance Criteria

- WebSocket close always releases the socket even when protocol writes fail; peer failure and handler exit terminate owned resources under the documented policy.
- Bounded message handling and disconnect detection have explicit backpressure and cancellation contracts; no second reader is introduced accidentally.
- gRPC send serialization and operation-context behavior are proven in their own protocol owner and compiled adapters.

Checklist

- [x] Deliver #716 message admission at `46b627ef` and #722 lifetime ownership at `56bc7eae`, with #711/#712/#714 witnesses; independently reviewed, pushed and closed. Evidence: `http/tla/websocket_message_limit/README.md` and `http/tla/websocket_lifetime/README.md`.
- [ ] Inventory `rg -n 'WriteClose|ReadMessage|SetReadLimit|WithContext|WebSocketStream|conn.Close' http jsonrpc grpc/codegen`; reconcile #711, #712, #714, #716, #722, #726, #758 and #739 into protocol-specific structural tickets.
- [ ] Record exact lifetime states, resource owners and bounded acceptance cases beside `http/websocket.go`; obtain scoped readiness review before implementation.
- [ ] Extend the owning TLA models with current-failure controls; run their README commands; add failing socket/race and generated-handler regressions.
- [ ] Move remaining generated lifecycle decisions into `http.WebSocketStream` and the appropriate JSON-RPC/gRPC runtime owners; delete duplicated protocol execution.
- [ ] Run `go test -race ./http ./jsonrpc ./grpc -count=1` plus bounded generated stream tests and compile comparisons; record selected cases and exact commands.
- [ ] Update stream guides and consumer skill; review, commit, push and integrate each structural ticket separately.

### Milestone 6: Validate the effective design

Goal: Ensure generators receive a fully resolved, validated semantic contract.

Acceptance Criteria

- Inherited field-number and mapped-name collisions fail as source-attributed design errors, before generator panics or silent overwrites.
- Valid effective shapes retain deterministic output; validation does not silently discard authored declarations or mutate shared bases.

Checklist

- [x] Deliver #755 effective Extend composition with #757 witness at `4f7892a2`; independently reviewed, pushed and closed. Evidence: `roadmap/effective-design-readiness.md`. Remaining validation reports require separate scope reconciliation.
- [ ] Inventory `rg -n 'PrepareValidateFinalize|Finalize|Merge|validateRPCObjectTags|ElemName' eval expr`; reconcile #755, #757, #756, #701, #702, #703, #699, #710, #760, #780 and #781 by actual validation owner.
- [ ] Record phase dependencies and the accepted constraint-refinement contract for Extend redeclarations in the owning design documentation; obtain scoped readiness review before implementation.
- [ ] Add failing effective-shape tests and model existing phase-order counterexamples; move composition before the checks that require it without globally swapping unsafe evaluator phases.
- [ ] Run `go test ./eval ./expr ./dsl -count=1`, affected generator packages and bounded generated accepted/rejected fixtures; record byte differences and compilation results.
- [ ] Update DSL documentation; review, commit, push and integrate each owning structural ticket separately.

### Milestone 7: Exact values and explicit presence

Goal: Preserve authored numeric meaning and explicit values through shared conversion contracts.

Acceptance Criteria

- Bounds above 2^53 and integer extrema preserve their exact meaning; invalid defaults and enum members fail at design time; accepted collection literals compile using their declared types.
- Absent/default/zero/empty/null cases follow the accepted scalar contract across supported transports. Explicitly present empty collections survive as empty values, distinct from absence, under the accepted public decision.

Checklist

- [ ] Inventory `rg -n 'Minimum|Maximum|typedDefaultValueLiteral|UseDefault|omitempty|omitzero' expr codegen http/codegen grpc/codegen jsonrpc/codegen`; reconcile #681, #685, #692, #693, #695, #696, #690 and existing `roadmap/value-contract-inventory.md` obligations.
- [ ] Record the exact numeric domain and presence matrix beside the existing value-contract design; obtain scoped readiness review before implementation.
- [ ] Add failing exact-bound, target-literal and scalar presence regressions; update relevant Lean/TLA source and correspondence before replacing lossy representations.
- [ ] Run `go test ./expr ./codegen ./codegen/service -count=1` and selected transport presence/default tests; run the affected proof README commands and bounded generated comparisons.
- [ ] Document regeneration and scalar contract changes; review, commit, push and integrate each structural ticket separately.

### Milestone 8: Generated identities share real scopes

Goal: Allocate generated declarations and references together without hiding authored identity.

Acceptance Criteria

- Adversarial authored names cannot collide with framework declarations, helper names or imports; references use the allocated declaration identity.
- Declaration ordering and existing numbered suffixes cannot change correctness; supported names compile across the affected generators.

Checklist

- [ ] Inventory `rg -n 'NameScope|Unique|Goify|TypeName|ImportName' internal/naming codegen/service http/codegen grpc/codegen internal/openapiimport`; reconcile #691, #694, #700, #733, #734, #735, #736, #770, #786 and #688 to real output scopes.
- [ ] Record each scope's declaration/reference owner and collision fixture; obtain readiness review and add failing tests plus bounded allocation-model controls.
- [ ] Route each selected scope through its shared allocator; remove recomputed names and suffix-based references.
- [ ] Run `go test ./internal/naming ./codegen ./codegen/service -count=1` and scoped generator/importer tests; compare and compile adversarial generated fixtures across independent generations.
- [ ] Review, commit, push and integrate each structural ticket separately.

### Milestone 9: Shared parameter wire contracts

Goal: Make runtime codecs and OpenAPI admission/export agree on accepted parameter serialization.

Acceptance Criteria

- Accepted location/style/explode/shape combinations round-trip through import, generation and real request decoding without changed meaning.
- Unsupported combinations fail with source-specific diagnostics; recursive component imports compile and preserve graph references under a separately scoped importer task.

Checklist

- [x] Reconcile #706, #719 and #793 against import/export/runtime witnesses; record the user-adopted serialization table and pass independent readiness review.
- [x] Implement the shared named-field contract in `internal/httpencoding`, with existing JSON presence carriers and generated conversion ownership; independently review and deliver `1d94c755e`.
- [x] Validate affected owner packages, mounted/imported generated build/vet/wire cases, independent-process bytes and eight libopenapi/Vacuum specimens; verify integration with recursive validation across transports.
- [x] Update OpenAPI/HTTP guides and consumer/framework skills; push and close the three proven serialization witnesses. Recursive import #792 was separately delivered in `58916bc9`.
- [ ] Reconcile remaining #791 separately; imported credential-source #801 has its own readiness and acceptance gate.

### Milestone 10: Transactional resource construction and publication

Goal: Prevent failed construction from leaking resources or publishing unusable distributed participants.

Acceptance Criteria

- Every injected local acquisition failure releases resources once; retries do not accumulate subscriptions or goroutines.
- Only ready workers receive jobs; lost publication replies, retries and cleanup converge under explicit ownership rules on real Redis.
- Scheduler cancellation and job bookkeeping match the recorded local/global lifetime decision.

Checklist

- [ ] Inventory `rg -n 'AddNode|newWorker|SetAndWait|initialWorkerHeartbeat|Schedule|ErrJobExists' pulse/pool`; reconcile #687, #704, #738 and #737, keeping observability constructor rollback #763 as a separate owner.
- [ ] Record acquisition/publication states and scheduler consumer evidence beside owning code; obtain readiness review before changing lifecycle contracts.
- [ ] Model failure and lost-reply counterexamples; add table-driven fault injection and real-Redis regressions before implementing local cleanup and distributed publication changes.
- [ ] Run `go test -race ./pulse/pool ./pulse/rmap ./pulse/streaming -count=1`, affected model README commands and `make test-pulse-redis`; distinguish miniredis evidence from real Redis behavior.
- [ ] Update Pulse guidance; review, commit, push and integrate each structural ticket separately.

### Milestone 11: Residual findings and final reconciliation

Goal: Leave every report with an evidence-backed disposition and no hidden unfinished architectural work.

Acceptance Criteria

- The inventory distinguishes delivered fixes, reproduced residual defects, duplicates, unsupported cases and public decisions; no issue is closed solely by similarity to a repaired family.
- File overwrite safety, proxy-address trust, logging/debug wrappers and uncovered protocol paths have explicit owners and bounded next actions.
- Integrated validation passes and temporary worktrees/output are cleaned only after all active users finish.

Checklist

- [x] Deliver #708 proxy-address trust at `9880865a`; independently reviewed, pushed and closed. Evidence: `http/tla/forwarded_client/README.md`.
- [x] Deliver promoted #670 artifact write-intent ownership; independent review, commit `41b4a8a3`, push and integration complete. Evidence: `cmd/loom/tla/artifact_write_intent/README.md`.
- [x] Deliver #741 example interceptor declarations through that shared writer; independent review, commit `23f2af5d`, push and integration complete. Evidence: `roadmap/example-interceptor-ownership-readiness.md`.
- [ ] Reconcile inventory residuals, including #670/#741 file ownership and #708 proxy trust, against final integrated source; promote each supported defect to a separately reviewed bounded task.
- [ ] Run targeted adversarial review of repaired owners and their consumers; record new counterexamples and update task acceptance before implementation.
- [ ] Run `make lint`, `make test` and integration checks selected by actual changed behavior; record exact source and outstanding blockers. Do not start a full exported-design audit without a named purpose and cost budget.
- [ ] Confirm every delivery commit was independently reviewed and pushed; update `roadmap/ROADMAP.md` and this tracker to reflect remaining active work.
- [ ] Preserve durable evidence and archive completed managed worktrees; remove task-owned model/compiler output after checking active processes.

- 2026-10-12 — **Delivered #681:** exact numeric ownership committed/pushed as `66a9dc9090a06cd4136be1e5ebdaf26883ff90fa`; ticket closed. Final independent reviewer `/root/review_numeric_final` approved 173 staged files against `ef0205d71e308acd26cae2fdc1467dc1270ae485`, manifest SHA `f6eaa75b61d82db93c711ea44d3a3c43bb41490a2a366ca7a3d0555bc30e7d1d`, diff SHA `e7ba259e9961e933fd628a962dfebde19defe2cbbd2d8daee54510966da0eec1`. Shared numeric owner, source normalization, native thresholds and required annotation preparation replace lossy Float64 paths. Isolated proof/conformance and three-transport tests pass; integration adds root compile-only, recursive import/wire/multipart/projected-validation tests, and 32 exactly repeated generated artifacts matching the reviewed candidate. Only six intended ordinary numeric artifacts change; exact-bound input changes from parent rejection to compiled/vetted output. Independent recursive generation passed 29.992s. #685, #798 and #799 remain open and are not claimed fixed.
- 2026-10-12 — #709 readiness accepted after one medium test-scope amendment; final ten-file manifest `b3c56f80c00b43ff381d3ac1bef73d0bcb5204071554bc7d52b47719f00ceb70`. Production released on numeric owner `66a9dc909`. #706/#719/#793 readiness accepted after one medium 3.1 witness correction; manifest `14610d692bc9535c66d4131b2aad19fc37fb12758c32707fb9b2f121958e7f15`. Serialization policy remains pending, with a concrete cross-language JSON recommendation presented to the user. Pool and typed-value admission investigations continue independently.

- 2026-10-12 — **Accepted serialization policy:** user selected uniform JSON for named query/form objects and maps (`filter={"color":"red"}`), including shallow maps. Preserve scalar and repeated scalar-array formats. Document coordinated client/server regeneration and handwritten-request migration; no depth-dependent format or compatibility codec. The shared URL serialization owner and importer/exporter contracts must be revised to this policy before production. Pool #687 readiness review found a real prerequisite: supported zero-block sink reads prevent Close from terminating until an event arrives; cleanup must solve this owner before promising constructor rollback. That production slice remains gated, with the independent probe retained.

- 2026-10-12 — Filed [#800](https://github.com/CaliLuke/loom/issues/800) after independent and root reproduction at `d57f96c6`: active Sink XREADGROUP BLOCK 0 survives local cancellation, so Close waits for another event. The issue retains the complete terminating reproduction. This is a separate streaming lifecycle prerequisite for #687; no production fix or unconditional rollback guarantee is claimed. Pool investigation manifest is `c3c64b7f5de53bd841a673353e880ec3b7d05a8240c3c0417d5bdb667d86269f`. #685 readiness manifest `d940f6ce541322f6c62dda81df15cbe52fc0e2b78da0b1baeec37f642f74dcbe` is under fresh review. Root #797 readiness records eight compilation failures, two normal controls and a 24-state projection model; it is not yet independently approved.

- 2026-10-12 — #685 isolated final review passed without findings on manifest `9679bc649d67e6ce9fccff0af47c8ba2c243ba09dd0b24b5764d58c99da0fe8f`. Root integrated the exact 22 files and shared guidance for final review; delivery remains pending. All 135 retained parent/candidate/repeat artifacts agree, all three transport entry points reject invalid values before gen/, and proof/conformance/runtime gates pass. Separate stale combined-auth fixture correction `8aec9136a` is reviewed and pushed; its existing endpoint golden is unchanged.
- 2026-10-12 — #709 is frozen for final review: 81-file manifest `78009a3e2f1494edd8a37e2ebdc59af468742182f5db1ce9084796405f25c312`, 201 artifacts, 35 intended changes and zero independent-repeat differences; 13 generated modules build/vet. Revised uniform JSON readiness manifest `e02f889e8f2b7d186744303ae4cfb631cfd7996fabd364cbc8f2cdd16df5958d` is independently approved and implementation is active. #797 readiness manifest `ed599d7d7f82cb3b46638a5112ee603af31ed2b6b0cb664b1130caabb1d20d0f` passed independent assessment with no findings; implementation released to root.
- 2026-10-12 — #800 now reproduces in eight actual Redis8.10.2 Reader/Sink, default/context-enabled and zero/long-block cases, synchronized by server CLIENT LIST. All join after an event and preserve caller clients. A bounded model rejects context-only and server-block-only cancellation; real TCP paused-server probes demonstrate a finite pipeline socket budget, but no production mechanism is approved. Durable sources remain beside pool readiness in `pulse/streaming/tla/read_shutdown/`. Delivered #681 temporary outputs were cleaned after active-owner confirmation.

- 2026-10-12 — **Delivered #685:** `1168ec6c1af34a9f6436aa3c84056e29ca8ee7fa` reviewed, pushed and closed; total thirty closed. Integrated 27-file manifest `b63f65967bb2b81e527d27841efd5eb288f5764701f86a3909d6a9f217276b79` approved with no findings. Root expr/DSL/value-contract/service suites pass. #709 final review found one medium named-nullable object carrier mismatch; the shared callable owner repair is implemented and final bounded replay is pending. No #709 delivery is claimed.
- 2026-10-12 — #797 shared public ProblemResponse projection now compiles all thirteen selected/pruned/full/empty/custom cases; mounted wire and generated-client checks are in progress. #706/#719/#793 JSON serialization passes generated client/server and imported-client controls; non-null occurrence rejection is being aligned with existing JSON decoding. #801 is filed after independent current-main reproduction of valid Bearer/OAuth alternatives producing conflicting credential fields. Its existing source fixture remains in `internal/openapiimport/symmetry_test.go`; do not weaken that contract to make the test pass.
- 2026-10-12 — #761 must precede #800: a retained `$` cursor loses events between completed reads, including when another stream blocks fanout. Red regressions pass their failure expectations against both miniredis and actual Redis8.10.2; snapshot/publication models are prepared for review. Native positive BLOCK has a finite driver socket budget (BLOCK plus ten seconds), preserving command hooks and stream metrics; the prior pipeline candidate is not selected. No new subsecond Close guarantee is adopted.

- 2026-10-12 — #761 readiness independently approved by `/root/review_reader_cursor_readiness`, exact eleven-file manifest `7cfc9b349393046deb17f2d884d83dfe62faced8c88c21e3be801f3fa61578f1`, no findings. Independent miniredis/real-Redis race witnesses and all five model outcomes agree. Implementation released to `/root/review_value_admission_readiness`. New #802 independently reproduces an old admitted read rewinding a removed/re-added explicit cursor; it remains a separate atomic registration-identity repair after #761, within the same owner, before #800 and #687. No claim that #761 resolves it.

- 2026-10-12 — **Delivered #709:** `25179e87b8b5d4e9cf5bb233d1818600469d060d` pushed and confirmed closed. Independent reviewer approved exact integrated 87-file diff `c8d300fa3d2704765e225e9fe6d437648d0a9080277b1baa5ca6e9e1dd4a9e7b`; main shared suites and selected three-transport checks pass. **Delivered #761:** `af05fa8aa1b664e42d962ba7df617966a99bcbaf` pushed and confirmed closed. Exact integrated 21-file diff `0b7eeeb3f81b5065807da1f6e9df87baf9bc55172afb2cb874a010ec6e9807e5` approved independently; streaming/options, selected pool callers, actual Redis and bounded models pass. Total thirty-two closed. #802 readiness is next, then #800 and #687.
- 2026-10-12 — Uniform JSON final review found two medium blockers: named query/form JSON subtrees bypass non-null collection carriers, and a final test edit inserted an illegal internal import into the external imported-client harness. `/root/basic_cli_defaults` paused #801 readiness and is repairing both through the existing representation owner and rerunning invalidated evidence. No JSON repair delivery is claimed. #797 continues in `/Users/luca/.codex/worktrees/problem-projection-integration/loom` on `25179e87b`: sixteen generated wire cases pass, 48 artifacts per parent/candidate/repeat are deterministic, and candidate modules compile/vet. OpenAPI JSON/YAML validation and lint pass; final review remains pending.
- 2026-10-12 — #685 cleanup completed after review/delivery and active-process checks: 27 owned paths including ignored Lean `.lake`, five generated/archive directories, logs and temporary harnesses removed. Final manifests and concise comparison evidence remain in `/tmp/loom-685-cleanup-receipt.json`, SHA `c4a1b62f5679af8d7a3f7110f71e2a4fa88c5a4792f089bd55b751a91865f384`. Managed archive was refused because the worktree is pinned; its source remains intact.

- 2026-10-12 — **Delivered #693:** `39f9ce17c`, reviewed 34-file integrated
  diff `c2815f4dbe25c8cb2c040f4e9296d2fa6e477fd4f1646f58f97f1b8d2cc46fa0`,
  pushed/closed. Shared resolved-value/target-context literals cover three
  transports. Four generated modules build/vet; 65 outputs repeat exactly.
- 2026-10-12 — **Delivered #801:** `7b29289ae`, reviewed 28-file integrated
  diff `e1e798a4e2d1dbb45e6a0adb7770fe26d9d8866b489bf2da8926200d0f181a4c`,
  pushed/closed. Physical credential grouping preserves roles/requirements;
  partial imports retain only effective root security. Three generated modules
  build/vet and all 66 outputs match approved source and isolated repeat.
- 2026-10-12 — **Delivered #683:** `ebc06ec94`, reviewed 12-file integrated
  diff `3027cfc123bae1446057510f77288132cbaa2d80ddfbd06fd2b1a47aab6b644c`,
  pushed/closed. Lifecycle-locked subscriber admission returns closed channels
  after shutdown without starting a Reader. Full streaming race, real-Redis
  controls, concurrent repetitions and 34-state model passed.
- 2026-10-12 — Upstream PR #4095 correction: all twelve failed CI jobs shared
  the introduced minimal-client clone panic. Guarded hook copying preserves
  original zero-state behavior; six relevant tests pass under race detection
  (5.763s; independent 5.385s). Complete stable-parent two-file diff
  `e4f5dcf042935f2fa407210a62ac874a6c4f192bd77757ef3399224d85ab8d57`
  independently authorized the amended commit `b77660a`. CI is pending, not
  claimed green; public communication contains no framework internals.


## Delivery checkpoint: 70 reports

- #690: `7b28984a7`, reviewed root against final 15-file manifest
  `334089f524aeccdedf2de8cdef8d1706bb6a77b445a5932628f02145720f206f`.
  Five designs, 55 exact-repeat artifacts, six intended decoder changes; mounted
  race/default and pointer-default tests pass. Root independent tests 9.242s;
  integrated tests 8.166s. Integrated diff SHA
  `9bcb1e23adcab4270fbf3bdadb25896877889389e4abb234824e8fe6f805f012`.
- #819: `6b21d37fd`, fresh reviewer `/root/basic_cli_defaults`, no findings.
  Final 13-file manifest `7aa7713335fda1b10d2db1bf5d07e0ac58e405c73ae0c680a849541f7f3b0e0d`;
  77 artifacts repeat exactly, six validator/CLI changes; model passes 104/64.
  Integrated direct/generator tests pass 0.571s/10.898s. Integrated diff SHA
  `9847228657be25bb9a817928ba101d470227901d9230dab4c934b040e8d6e26d`.
- #680: `942e58c6f`, root independently approved 20-file manifest
  `009f4f62c44ba1631c9901f41aef05891852076daff9ba0f2f439bdc1c3b92d5`.
  Original read/error/Close and policy-order regressions, bounded retention,
  concurrent snapshots and unfinished-upload SSE pass. Root race 1.245s/1.233s;
  integrated race 1.615s/1.504s. Integrated diff SHA
  `ef254718d64c7b6a92e2dedb65b0dc07d9651b018628ed20332082404c74a82f`.
- All three commits are pushed and issues confirmed closed. No overall CI-green
  claim: run `38338504807` at older `73451e4da` failed ci/race and release
  eligibility; its five exported-design compile partitions, generated quality,
  OpenAPI, value-contract and real Redis jobs passed. Logs are being classified
  against subsequent deliveries. Run `38335487607` additionally exposed #828.
- Inventory adds #827 client Debug error ownership, #828 span completion
  observation, and #829 projected nullable declaration naming. #826 is an
  external contributor's PR for already-delivered #823, not an issue/report;
  it is not included in the report count.
- Cleanup of delivered temporary artifacts remains required after active
  consumers finish. Keep #823 comparison inputs while #824 reuses them; #690's
  owner is compacting delivered evidence without deleting broader M7 probes.

## Delivery checkpoint: 73 reports

- #824: `e90a9e45f`, independent final and integration reviewer
  `/root/review_uniform_json_corrections`; one LOW dependency-pin evidence
  finding corrected and confirmed. Integrated direct/generated controls pass.
- #830: `66c617456`, root reviewed the agent's exact four-file diff with no
  findings. Integrated decoder/WebSocket families pass (0.787s); representative
  output, compile/vet and independent repeat evidence is unchanged.
- #828: `0d5736d64`, independent reviewer `/root/union_carrier_recovery`;
  LOW declaration order/GoDoc finding corrected and confirmed. Integrated
  race count10 passes (1.564s). No production tracing change.
- All three are pushed and confirmed closed. See current execution queue for
  remaining prerequisites, the accepted bounds policy and new assignments.


## Delivery checkpoint: 76 reports

- #695: `e64a14d1d`, root independently approved the 21-file candidate and
  exact integrated changes. Valid representative outputs retain all 63 artifacts;
  five invalid designs reject before generation. Full expr/DSL/shared codegen
  checks and bounded admission models pass. User explicitly accepted rejecting
  contradictory inherited bounds. Cleanup receipt:
  `/tmp/loom-695-cleanup-receipt.json` (SHA
  `e87004d396dbf7fbcabf01f25a6e9d9fc26935367e8fcb7a663a4b993bdee10e`).
- #827: `b0ef97856`, root independently approved the 23-path candidate and
  exact integrated diff `aca63290a3f74d00fa780d621fdd64d99a751c26555355643a932af3770ace6a`.
  Shared passive body capture replaces eager client response reads; per-exchange
  completion retains identity. Independent full race packages pass HTTP9.960s,
  middleware1.835s and transport1.338s. Runtime matrix covers I/O errors, streaming,
  concurrent Close, redaction, caps and once-only completion; bounded model passes.
- #831: `ab3b18d78`, `/root/review_uniform_json_corrections` independently
  approved final and integrated 19-file diffs without findings. Integrated SHA
  `a4a55b6a5ed28de9b9aa9937edaa81d2602bcf87ec7b7c8b9a38e86fbf187085`.
  All 121 artifacts verified, 32 intended conflicting-wrapper changes, repeated
  output identical and nominal controls unchanged. Ten mounted race modules,
  build/vet and frozen #684 overlay pass; independent owner/mounted checks pass.
- All three are pushed and closed. Live inventory snapshot
  `/tmp/loom-checkpoint-76-issues.json` confirms 153 unique reports, 76 closed.
  #832 is the new nullable-object transformation prerequisite for #829.
- Completed #828/#830 cleanup removed 28 inactive temporary targets after
  process checks; source worktrees remain protected. Receipts:
  `/tmp/loom-828-cleanup-receipt.json` (SHA
  `dd241a12c3f11a226e28de23bfb489b69532a36c9ee840f288c72420e7fdfb34`),
  `/tmp/loom-830-cleanup-receipt.json` (SHA
  `8fc185335c464a22703cb25fe3f7b5f560c4d8685819fa23c204fe224a8ba9bc`).

## Delivery checkpoint: 80 reports

- #832 `bf7ae2246`: shared nullable branch dispatch precedes concrete object
  helper admission. Independent final review and exact integration passed.
- #684 `27255d025`: wrapper kind owns contents requiredness. Four nested shapes
  retain empty/populated values; scalar/union contents remain required. Root
  verified the complete 21-file diff and 132 artifact hashes; mounted binary-wire
  race checks pass. Cleanup receipt `/tmp/loom-684-cleanup-receipt.json` retained.
- #833 `5e4b4457e`: OpenAPI and Lean conformance distinguish invalid source design
  admission from arithmetic intersection. All raw arithmetic cases remain;
  invalid source/target designs reject and valid schema controls pass. Independent
  review, Ajv and real Lean checks pass. Cleanup removed 157 MB of inactive output;
  receipt `/tmp/loom-833-cleanup-receipt.json` retained.
- #829 `a8783ef65`: shared concrete declaration and helper identity remains
  independent of nullable occurrence storage. Root independently reviewed all
  20 files and 45 evidence artifacts, verified 298 physical output hashes, and
  reran direct and complete nullable generated matrices (10.956s). All 83 current
  artifacts repeat exactly; 75 previously supported artifacts remain unchanged.
  #834 separately tracks nullable ResultType guard/argument handling.
- Inventory is 156 reports, 80 delivered/closed. #835 records shared default
  admission. Main CI `38346073415` at `5e4b4457e` remains in progress. CodeQL,
  generated quality, real Redis and four compile domains passed. OpenAPI passed
  the repaired #833 check, then failed another stale contradictory-byte oracle;
  Lean conformance also found stale named-key numeric and alias-length oracles.
  Their follow-up repair is assigned after #744 review. #833 corrected its
  recorded groups only; the overall admission-test migration remains incomplete.
  No overall green CI claim is made.
- #831 cleanup receipt `/tmp/loom-831-cleanup-receipt.json` retained; protected
  source worktrees remain intact. #829/#832 cleanup removed 636 inactive files (5.7 MB), with source inputs,
  hashes and compact receipt at `/tmp/loom-829-832-completion-receipt/receipt.json`.
  The standalone #834 witness remains active. Live snapshot
  `/tmp/loom-checkpoint-80-issues.json` confirms 156 distinct reports, 80 closed.

## Delivery checkpoint: 81 reports

- #744 `09aa506b4`: independent final and integrated reviews approved all 14
  files without findings. Context detaches configuration and slices under its
  logger lock; reusable WithOutputs detaches caller storage on every application.
  Buffer identity and fields remain unchanged. Full affected race tests pass
  (1.584s/1.393s), scoped lint zero issues; independent focused race checks pass
  (1.210s/1.243s). Current/shallow model counterexamples and detached 13/9 states
  remain aligned. Final integrated diff:
  `80e95992693338508173a39cf69e069478318c0695562569cbdb99162320c54e`.
- Pushed to main and GitHub confirmed #744 closed: 81 of 156 inventory reports.
  #746 request-scope readiness is next; #745 buffer ownership is still open.

## Delivery checkpoint: 83 reports

- #836 `9669d2815`: seven test/documentation files align invalid-design admission
  with delivered constraints while retaining raw mathematical empty-domain cases.
  Root independent IR/Ajv and two real Lean groups pass. Exact patch
  `43349dc4f0ce982fbf26f3b7460dd0080def263b585a944367fda09ed39e39d5`.
  Cleanup removed 424 temporary files (156940248 bytes); receipt
  `/tmp/loom-836-cleanup-receipt.json`, SHA
  `9be7c04569f34ba70ef371a2d2163834db1d106b6e847e6c2677058fedce9d6b`.
- #746 `7b59759c5`: HTTP, unary and streaming server requests create a fresh
  logging scope independently of request IDs/access logging. Root fields/options
  snapshot into a new buffer; policy receives the incoming request context.
  Independent final/integration review approved; full log/debug race and lint
  pass, request model fresh150/96 passes with two failing alternatives.
  Integrated exact patch
  `3843bbbd22a0d68d289af75dba695f80c46b742fd9b5d8c861c392f99a54a9dd`.
- #837 semantic view nullability, #838 CLI decoded-body presence and #839 stale
  protobuf collection goldens are separately reproduced and filed; 160 reports,
  83 delivered/closed. Their scope and dependency order are in the current queue.
- Latest main CI `38348749814` at `7b59759c5` is in progress. OpenAPI contract,
  generated-code-quality, real Redis and grpc/expr/jsonrpc compile jobs passed;
  CodeQL passed. No overall green claim while remaining jobs run.
- User approved libopenapi publication and original per-entry logger settings.
  PR658 is published and attached; #745 accepted policy is recorded in its plan.

## Delivery checkpoint: 87 reports

- #837 `2a8337700`: shared object projection copies intrinsic Nullable; occurrence
  attributes and selected-view identity remain separate. Root final/integrated
  review approved; direct semantic matrix and39 unchanged generated controls pass.
  Exact integrated diff `6f38f1aa3194e60acbd36c78b810a03af7a603be2daa3418f3162a8bd4542646`.
  The intrinsic generated compiler limitation remains #834, explicitly documented.
- #745 `ee4552dc9`: one private buffer owns queue, flush state and serialized
  output across With descendants. Immutable entry output snapshots implement the
  user decision. Full log/debug race and overlap detection pass. Independent exact
  integrated review approved diff
  `11db42febe0161cb21eb26e58c41e08fbe4d41f3a5d3905228faf8bd54f213b7`.
- #839 `8d2a446bf`: four golden expectations lose only obsolete protobuf contents
  requiredness. Current CI failed those exact four cases; corrected comparisons
  and containing-message/scalar/union/length controls pass. Exact five-file diff
  `189e01b57c91056efa20c9a2741464071a2da7f1213183f8cae17f18259ad95e`
  independently approved; no production changes.
- #750 `82368f12d`: stdlib JSON serialization and shared text/terminal escaping
  retain ordered fields, duplicate names and existing ordinary goldens. Root's
  MEDIUM depth-limit panic finding was fixed by per-value error containment,
  independently re-reviewed and tested with actual deeply nested values/siblings.
  Full integrated log/debug race suites pass2.049s/1.915s. Exact integrated diff
  `c5fe3a7816775cef6b2701b4f2769c91d3d308f66951b0f4fc593d01536407ff`.
  Bounded benchmark costs and intended formatting differences are in its plan.
- Cleanup after review/delivery: #746 removed19 temporary paths; receipt
  `/tmp/loom-746-cleanup-receipt.json`, SHA
  `d75f4d42aeea4adc21408c826ecd1ce35cedf06a3d3cc860ef3537d7a11779ac`.
  #745 removed20 paths; receipt `/tmp/loom-745-cleanup-receipt.json`, SHA
  `58247580fd5b8e4121d39bbb7f867d4187d925e97affa1eeae80d9a8e2c91cbe`.
  Durable sources, manifests and patches remain; #837 inputs needed by834 stay.
- CI38348749814 at7b59759c5 additionally passed value-contract and service compile;
  its CI failure was exactly the four now-repaired839goldens. Race and HTTP corpus
  were still running when inspected. New commits still require their own CI;
  no overall green result claimed.

## Delivery checkpoint: 88 reports

- #748 `2b27cd8e0`: shared SDK ParentBased composition preserves propagated
  local/remote parent decisions and existing root-ratio normalization. Twelve-file
  final/integrated review approved; actual HTTP propagation and runtime/HTTP/gRPC
  race suites pass. Exact integrated diff
  `3463b796680b2dc8eea1060ba285a018cbba29677d27c188d5c41e92157f2c91`.
- #840 was independently reproduced and filed for the JSON-RPC nullable-object
  body declaration; inventory is161 reports. #834 stays frozen until delivery.
- #763 readiness manifest `/tmp/loom-763-readiness.json`, SHA
  `7989070ff19b658127e426dbf4dd183378e2ccc18dcafa372c949eac9cc83bbe`,
  preserves actual metric/log failures, successful bootstrap controls and three
  model counterexamples. Checked staged construction passes169 states.
- Main CI38353191998 at2b27cd8e0 is queued; earlier current jobs are still running.
  Upstream libopenapi658 remains open with Windows/Linux and coverage checks green.
  No overall main CI success or dependency adoption is claimed.

## Delivery checkpoint: 90 reports

- #840 `0bc139d5e`: JSON-RPC success/error decoding consumes shared body ValueRef.
  Fifty-six generated runtime cases pass; the ordinary specimen is unchanged and
  the nullable specimen changes one declaration line. Independent final/integrated
  review approved diff
  `89e7dab1f322223198d7fd77d04fcc8f8ce6960baf3b085ebe68298e9dcca4c6`.
- #838 `0d9671589`: HTTP/JSON-RPC CLI builders consume a detached JSON decoding
  plan before native conversion. Required validation precedes defaults; bare JSON
  collection flags retain syntax and enforce element presence. Five designs yield
  169 artifacts/run,17 intended CLI/helper differences and exact isolated repeats;
  all parent/candidate modules compile/vet and mounted adversarial cases pass.
  Corrected41-file integration review approved diff
  `816c60a63d09e43ec3520ac293b794b6607e5bab621433a769f9914d6b019c45`.
  Native-to-wire default admission remains #835; it is not claimed delivered.
- #718 readiness is independently approved at manifest
  `f37a1c969b4ad834c84db46ef3e14be30227cebaf17daf2da3162ac06ea0a01c`.
  Root repeated37 runtime leaks; immutable diagnostic policy will consume actual
  credential bindings and prepared body names. Implementation moves to current
  main first; #721 default query visibility stays a separate public decision.
- #763 implementation/final manifest
  `0d1b5ad1105c108516561e813f10af497d465ad014477953e244a91424ea9f11`
  is under independent final review; full observability runtime/transport race
  checks and scoped lint pass. No delivery claimed before approval and push.
- #748 cleanup removed14 inactive temporary paths after process/lsof checks.
  Receipt `/tmp/loom-748-cleanup-receipt.json`, SHA
  `72b22a8466549a6d96b9cc39e911528e4e27f423067c337cf012a69a645a30de`.
  Durable models/tests and exact manifests/patches remain.

## Delivery checkpoint: 92 reports

- #763 `f7c2684b8`: staged telemetry construction preserves globals after failed
  initialization and shares reverse cleanup with normal shutdown. Fresh bounded
  rollback context retains values and joins cause/cleanup errors. Final/integrated
  review approved13-file diff
  `be7a360887afd088cf13881a6f325ba7e4ae5ce108f06b946d457119afd600ff`.
  Runtime integration with #748 sampling passes under race1.606s.
- #841 `477e86cb1`: regenerate mixedtick after CI38352030804 failed all five
  parity-dependent scenarios. Four of16 generated artifacts change;12 stay exact,
  and all16 match independent repeat. Build/vet and five race tests pass9.623s.
  Independent final review approved seven-file fixture/module/plan diff
  `707c001251ab33409732a761b7dd086fde0d09e8846a4f76ef76de576c9b3eb0`.
- Live all-issue reconciliation confirms162 tracked reports and92 closed.
  New source still requires its own CI; prior run passed OpenAPI, generated-code
  quality, value-contract, Redis and four compile partitions while race and HTTP
  corpus remained active. Its integration failure is the now-repaired fixture.
- #763 cleanup removed24 inactive temporary paths after process/lsof checks.
  Receipt `/tmp/loom-763-cleanup-receipt.json`, SHA
  `b4c46dc9d87d0c545be7da8e5989d69980b295f5737a7ce9cd7574d83006e7cf`.
  Durable model/test/README and exact patches/manifests remain. Managed archive
  refused #748/#763 because the application protects those worktrees; retained.
- #717 readiness plan is `roadmap/declared-error-properties.md` in its owning
  worktree. Shared problem document erases Temporary/Timeout/Fault before
  projection, then clients infer them from status. No implementation or changed
  public body-selection semantics are authorized by that investigation yet.

Checkpoint 93: #835 delivered as `772abd1a4`; its exact 34-file integration was
independently reviewed with no findings, preserving three newer guidance files.
The completed `0d9671589` CI race failure is the known libopenapi YAML-node race
(#820); upstream PR658 remains open with Windows/Linux and coverage checks green.
Mixedtick parity is repaired by #841. Newer main CI remains in progress/queued;
no overall green result is claimed.

## Delivery checkpoint: 95 reports

- #834 `e2ad69a64`: selected-view validation consumes one physical carrier owner;
  nullable projected payload and ordinary view metadata envelope retain distinct
  identities. Four generated fixtures pass, including root occurrence reuse;
  independent integrated diff approval covers 33 files, SHA
  `b7834ab4f80695f92036656d4bce29e74518121350e3e7bac7c3e08b0128766d`.
- #843 `5d3935d6b`: seven stale HTTP assertions now select helpers by responsibility
  and name and preserve both decoding and encoding coverage. All 36 generated
  artifacts per parent/candidate/repeat are identical; six modules compile/vet.
  Exact five-file review approved patch
  `5f5497ec033ea7353ed83be052da1f021e68865f9805f515ab069e319bae3b6c`;
  independent 0.671s and main integration 0.702s direct checks pass.
- Live all-issue reconciliation: 164 tracked reports, 95 closed. Newly filed #842
  is a distinct Basic CLI duplicate-binding defect, still open. Snapshot
  `/tmp/loom-checkpoint-95-all-issues.json`. No overall CI success is claimed.
- #717 shared problem-document implementation carries explicit boolean properties
  and removes status inference. Three-transport malformed/full checks pass 2.753s;
  expanded selected-body controls pass 29.812s before added malformed controls.
  Remaining schema/fixture/determinism/docs acceptance and final review are active.
- Completed #834/#837/#840 temporary outputs are removed after process checks;
  compact receipt `/tmp/loom-completed-834-837-840-receipts/cleanup.json`, SHA
  `c808153c50be3cbf861629acfed91db4677034c822e486b8d8b4f965095011ad`.
  #835 cleanup receipt `/tmp/loom-835-cleanup-receipt.json`, SHA
  `8f6b3cb9bb4356ac8cad9e8d91fc1537d814d9a7c4277cff6c4c52b4b3897543`.
  #841 cleanup receipt `/tmp/loom-841-cleanup-receipt.json`, SHA
  `d086c99b197a339f5917f5b0d8c083c1705460abed28687fbe807694bf630a15`.
  Durable source/model inputs and review manifests/patches remain. Active #718
  and omission inputs are preserved.

## Delivery checkpoint: 97 reports

- #718 `03e38fd16`: shared immutable diagnostics mask conventional and authored
  physical credential names. Generated metadata and example CLI wiring consume
  one owner. Response capture records trustworthy commit metadata; ambiguous
  previews are omitted without changing wire bytes. Independent correction review
  approved all 45 source/54 evidence hashes, including both disclosure witnesses
  and three model counterexamples. Root integration preserved newer guidance;
  exact patch SHA `3e9bd191819ddbdd1f5012ea76d916b636d1963d18cf39c7868c67085af4693c`.
  Four runtime race suites and generated credential transport suites pass.
- #717 `75e0ab159`: full problem responses carry explicit timeout/temporary/fault
  properties; shared reconstruction removes status inference. Pruned objects keep
  their omissions; scalar/empty bodies retain automatic header mappings. Fresh
  final and integration review approved 43 files. Integration patch SHA
  `d738d76f80ef51f164e705c7f50d6776f2bdcd3b1fe0bdb3655f7ae8a4c5e347`;
  all 21 selected-body cases and three-transport/property/malformed controls pass.
  Seven designs produce 118 deterministic artifacts; 17 changes are intended;
  all 14 modules build/vet. OpenAPI parsing/Vacuum and regenerated SSE parity pass.
- Inventory is 166 reports, 97 closed. #844 is independently approved authored-tag
  normalization readiness/report, manifest
  `d1f387788b5b203443e2bf316c56bd55eaee449117f54c92689c2f2c22c8c52e`.
  #845 is the separately reproduced JSON-RPC assertion failure from CI38359940773;
  its report is reviewed and test-only implementation is active. The same CI run
  passed OpenAPI, generated quality, Redis and four compile partitions; main job
  failed on #845 and a pool test's cross-map visibility assumption. Remaining jobs
  were still active when read; no overall green result is claimed.
- Upstream PRs go-redis4095 and libopenapi658 remain open with successful checks
  and no new human review. Dependency pins remain unchanged; #800/#820 are open.
- Prior agents repeatedly hit model capacity. Active replacements are
  `/root/diagnostic_redaction_finish`, `/root/omission_readiness_finish`,
  `/root/review_checkpoint95` and `/root/review_json_normalization` as slots permit.
  Root owns integration, the trackers and #721 readiness. Two new worktrees hit
  the application's 100-attachment identity limit; creation succeeded, attachment
  retry failed. Use the returned `jsonrpc-default-assertions` and
  `access-log-credentials` paths; do not create duplicate checkouts.


## Delivery checkpoint: 100 reports

- #845 `b963d8685`: named generated server admission and client construction
  assertions replace obsolete whole-output defaults. Mounted JSON-RPC tests cover
  absence, explicit values/empty, null and type/constraint rejection. Ten artifacts
  are byte-identical across parent/candidate/repeat; build/vet and focused race
  wrapper pass. Independent final and integration review approved the identical
  patch `49dff40e0a76a0eb44aa3be0573674b84773ace783e022102b6d98787b2856bb`.
- #844 `0f1fc3dc3`: one JSON-tag normalization owner retains authored full tags and
  exact exclusion. HTTP’s duplicate merger is removed. Independent review verified
  all 240 artifacts and six rendered specs: nine intended differences, 71 unchanged,
  eight modules build/vet, OpenAPI libopenapi/Vacuum and diagnostic-name checks pass.
  Root integration preserves the newer error-property skill guidance; 16 source
  files are exact, two skill additions are reviewed. Integrated patch
  `bd379bcc21f2750c22d6b51e75498e3e909f3b3b254cc4440025d29eebe7db01`.
  Automatic empty-versus-absent omission remains outstanding, with revised readiness
  assigned to `/root/omission_readiness_finish` on this delivered prerequisite.
- #846 `951475f2d`: a test-only repair observes the payload replica independently
  and asserts authoritative Redis bytes before/after repeated release. The original
  delayed-subscriber witness fails while Redis retains the payload; the corrected
  test passes that schedule, 20 race repetitions and Redis 8.10.2. Root independent
  race repetitions and main integration pass. Exact patch
  `9362817e6aeb3b00d51b65d3064c2e59446dfa341cfb12d2eb86a75ea30b2cc1`.
- `/tmp/loom-checkpoint-100-all-issues.json` confirms 167 distinct inventory reports,
  100 closed. CI38359940773 passed all five compile partitions, OpenAPI, generated
  quality, Redis and value-contract jobs. Its main test failures are #845/#846,
  now fixed; its race failure is the known libopenapi merge-node race #820, whose
  reviewed upstream PR658 remains open. New main CI runs are pending; no overall
  green result is claimed.
- #721 readiness manifest `0182f9eb2fe7543d612cec846a20d2a338d59b871d8b53661d9ef1f225905ec2`
  received independent approval with no findings. Twelve URL-disclosure probes and
  a compiled/vetted mixed server show the missing shared policy. Implementation
  is assigned to `/root/access_log_implementation`, using diagnostics.Policy.URL
  and per-host generated Fields; no separate credential list or visibility policy.
- #718 inactive comparison cleanup completed after process/open-file checks:
  `/tmp/loom-718-comparison-cleanup.json`, SHA
  `82357ad19182d96110b6655c6e8f9de3ee7da3e605b2457ad2ec5dc702b01a69`.
  Five generated/source-copy trees and 60 raw logs were removed; compact hashes,
  reports and durable reproduction inputs remain. Archiving the completed #717
  parent worktree was refused because the app marks it protected; it remains intact.

- Completed #717 temporary generated/checker output and raw logs are cleaned:
  `/tmp/loom-717-cleanup-receipt.json`, SHA
  `718ddc579e2e4f8234dbdda9e02e728fd1c6dbe487798aa57d2843ad747f8741`.
  Four inactive trees/32 logs were removed after ps/lsof checks, retaining 391 file
  hashes, result summaries, final patches and durable model/reproduction sources.
- Completed #844 cleanup receipt `/tmp/loom-844-cleanup-receipt.json`, SHA
  `7e232dca6631e6bdc20b699be979c34bc96911d1159c81c6971352dc6d4b2d60`,
  records 25 inactive output/log paths removed. M7 continues in the successfully
  created `json-presence-carriers` checkout; attachment/retry hit the same app
  identity limit, so use that existing path rather than creating another.

## Delivery checkpoint: 102 reports

- #721 `22d000aa8`: access-log URLs use the shared immutable diagnostic policy;
  each generated server example receives only its own HTTP/JSON-RPC service
  metadata. Independent review verified 414 artifact hashes, generated build/vet
  and runtime race checks. Integrated patch
  `c77e895287dd3f30e048968ba41611e548225d3aabf1d2fad30e8271f5761b42`
  was reviewed with no findings; focused main checks passed. Pushed and closed.
- #842 `99e2c33d6`: shared CLI source selection preserves first complete direct
  inputs, whole-body identity/order and detached slice storage. All 17 generated
  modules build/vet; 68 parser/payload/wire cases pass with race detection. Root
  independent review verified 53 source/evidence hashes and all 666 artifact
  hashes: 22 intended CLI-only differences, zero repeat differences. Six selected
  generated runtime specimens pass independently (10.099s/10.005s), as do focused
  integration tests (3.813s/3.676s). Exact integrated patch
  `544f08cf292a220c88f173e91d4e6ed003fe499ada63d8c69defd019bc797ac2`
  approved with no findings, committed, pushed and closed.
- #847 is separately reproduced and filed: selected request bodies use logical
  names instead of custom Go field names, even without authentication. Preserve
  `/tmp/loom-scalar-renamed-standalone/design/design.go` and its compilation log.
  No implementation claimed; do not fold it into CLI input coalescing.
- #848 automatic JSON omission is filed from the reviewed current-parent report.
  Readiness manifest `0d4d45f9d0c9fc712d7008c87fdc34226ccce4aa88cda0e851ca87b07706c127`
  passed independent review. `/root/omission_readiness_finish` implements in
  `/Users/luca/.codex/worktrees/json-presence-carriers/loom`, parent `0f1fc3dc3`.
  The type-neutral field plan belongs in internal/encodingmeta; canonical service
  layout is shared with expr composition. Full authored tags and custom codecs
  retain their contracts. No new user decision is pending.
- #694 readiness revision belongs to `/root/access_log_implementation` in
  `/Users/luca/.codex/worktrees/service-symbol-ownership/loom`, parent `951475f2d`.
  Review found that early reservations in the shared NameScope rename valid
  methods. Separate package/member ownership must also account for method-derived
  package helpers so splitting scopes does not expose new collisions. The agent
  is adding ordinary method and Foo/FooServerStream coupling witnesses before
  focused re-review; no production implementation is authorized yet.
- CI38364357703 at `0f1fc3dc3` passed all five compile partitions, OpenAPI,
  generated-code quality, value contracts and Redis. Main/race jobs expose stale
  inline-body tag assertions and omitted diagnostic files in manual test-module
  assembly; root reproduced these on `22d000aa8`. The race job also reproduces
  known libopenapi merge-node mutation #820. Logs are
  `/tmp/loom-ci-0f1-main.log`, `/tmp/loom-ci-0f1-race.log` and
  `/tmp/loom-ci-22d-regressions.log`. Newer main CI remains pending.
- Completed #721 temporary cleanup removed 11 inactive directories and 25 raw
  logs after process/open-file checks. Receipt `/tmp/loom-721-cleanup-receipt.json`,
  SHA `84ece62b7f6bcbecc2582c14c0f5abf128ac6e9911d3706aa619d3f354de31f3`,
  retains 862 file hashes and summarized evidence. Its optional full package run
  has no usable completion result; no pass is claimed or rerun required by its
  bounded acceptance. Unregistered source worktrees remain intact.
- Completed #845 cleanup removed three generated trees and 12 logs after checks;
  `/tmp/loom-845-cleanup-receipt.json`, SHA
  `bf29271f0125ea8a4359daeb219130fbedd18d5324cd7849bdf53277ee93b848`.


## Delivery checkpoint: 105 reports

- #849 `212700f49` replaces obsolete literal-tag assertions with complete
  anonymous struct identity checks. Root verified the final manifest and all
  36 generated artifact hashes; independent race controls passed (3.038s).
  Pushed and closed. Cleanup receipt `/tmp/loom-849-cleanup-receipt.json`, SHA
  `a8f142ad6d70ddcbf30a7bfcc3103396f33e445c1f17823df80510762d93d831`,
  preserves compact evidence after removing inactive generated outputs/logs.
- #851 `fe63e6b02` regenerates the stale JSON-RPC ticktock CLI file. Its bytes
  equal the parent regeneration and candidate output (SHA
  `91aaf3b6bd91eec50c7b2e577807027e3aa7f37f074ec3c9a11bcc02270ffcca`).
  Root independently reviewed the exact one-file diff; complete fixture tests
  and vet passed. Pushed and closed; no generator behavior changed.
- #850 `a25383ce1` adds the established diagnostic artifact owner to all three
  shared test-module builders and removes the per-caller workaround. Root
  approved the exact six-file diff with no findings, verified 15 source/evidence
  hashes and 362 artifact hashes, and independently ran the new race-enabled
  assembly controls (HTTP 4.889s, JSON-RPC 7.488s). Existing affected name sweeps
  passed. All 118 prior artifacts are identical; only four metadata files were
  added and all 122 candidate artifacts match a fresh-process repeat. Pushed
  and closed. Final manifest `/tmp/loom-850-final-manifest.json`, SHA
  `884963e0808b3e31311ad846b47283789b6d87ed11070d7eabc8b1dc3c4ddce4`.
- #852 is independently reviewed and filed. The existing import-reservation
  assertion combines unrelated generated files because their generators share
  a source file. `/root/diagnostic_redaction_finish` owns the rendered-file
  assertion repair in `ci-test-owner-readiness`; preserve actual collision
  controls and the controlled-fixture provenance assumption. Do not add a
  production reservation just to silence this false positive.
- #694 revised readiness passed independent review. Separate method/member and
  package/helper naming owners now compile the 14 selected modules; final
  comparison and review remain pending in `service-symbol-ownership`. Corrected
  shared method identity also corrects the previously suffix-renamed gRPC RPC;
  document regeneration requirements without adding compatibility aliases.
- #847 readiness passed independent review. Root implements one selected-field
  record retaining logical name, mapped key, actual service Go name and effective
  transform attribute in `selected-body-field-identity`, parent `99e2c33d6`.
  All 28 bounded generated modules compile; mounted request/response and CLI
  validation plus final artifact comparison remain in progress. No delivery
  or final acceptance claimed yet.
- #848 bounded implementation acceptance is complete; final exact-diff review
  remains pending in `json-presence-carriers`. Exclude the already delivered
  #849 assertion files and #851 CLI regeneration from that ticket's packet.
  The broad optional HTTP run timed out and is incomplete evidence; it is not
  a substitute for or a failure of the recorded focused acceptance.
- GitHub snapshot `/tmp/loom-checkpoint-105-all-issues.json` confirms the three
  new closures. Inventory now has 173 distinct reports, 105 closed. CI still
  has the separately tracked assertion failure and upstream libopenapi race;
  no full-green claim is made. Continue the backlog without waiting for upstream
  where independent work is available.


## Delivery checkpoint: 108 reports

- #852 `1074b3cdf`: root independently approved the exact five-file diff and
  verified 14 source/evidence hashes plus 3,778 captured artifact hashes. All
  original per-service output remained identical; expanded JSON-RPC alias cases
  compiled, and independent collector/assertion race checks passed (1.226s and
  2.879s). Pushed and closed. Inactive cleanup receipts:
  `/tmp/loom-850-cleanup-receipt.json` SHA
  `621141ac1026000e4fc142ecc9fce269a32214c229c1071c934eca46ced9932e`
  and `/tmp/loom-852-cleanup-receipt.json` SHA
  `78b1c8c6522849f5504247fa34004e045a8736f8799e10d1ac4ef0d40313afcb`.
- #848 `8ce69db06`: fresh reviewer approved 122 files, 25 evidence records,
  372 generated artifact hashes and rendered OpenAPI evidence. Shared physical
  storage determines automatic omission and finalized composition identity;
  full authored tags and opaque codecs retain their contracts. Independent
  generated HTTP/JSON-RPC checks passed (4.927s), with focused semantic and
  OpenAPI controls. Integrated patch
  `1bdd852b170ee0665add70a197388af06abb62301d504f0d42c37bce6b91b5f4`
  passed a separate narrow re-review; four guidance files preserve intervening
  #721/#842 text. Focused main checks passed (HTTP 7.602s, JSON-RPC 4.661s).
  Pushed and closed. The earlier optional broad timeout remains incomplete
  evidence, not a full-suite pass.
- #694 `fce66acb1`: fresh reviewer approved all 40 source files and 18 evidence
  records, 729 artifact hashes and 225 compiled-code hashes. All 14 selected
  modules build/vet; 243 artifacts repeat exactly, with 94 intended name/reference
  changes and 18 unchanged OpenAPI documents. Independent generated controls and
  gRPC descriptor/client-call checks passed (8.256s); the Owned model passes
  21 generated / 14 distinct states. Integrated patch
  `bde863b135473db5df04c492f859efed82c08d30bb3f35754be8e30a886067c9`
  passed narrow re-review preserving #848 tags and intervening documentation.
  Main service (32.301s), shared CLI (0.215s) and generated checks (4.844s) pass.
  Pushed and closed. Regeneration is required for corrected artificial gRPC
  method suffixes; no compatibility alias was added.
- Latest upstream snapshots are `/tmp/loom-upstream-658-status-latest.json` and
  `/tmp/loom-upstream-4095-status-latest.json`. Both remain open with unchanged
  reviewed heads; no new external message was sent.

- Follow-up reports #853 (mounted JSON-RPC selected response leaks excluded
  sibling fields) and #854 (required selected view uses an incorrect constructor
  pointer carrier) were independently reproduced on pushed source, reviewed and
  filed. They are separate from #847 physical field names. The live inventory
  contains 175 reports, still 108 delivered; #854 readiness takes priority as a
  prerequisite for complete required-viewed acceptance. Outgoing report hashes:
  `6b469c00a769db6a1ac99332c71ae994a64255f4eb0f496d1b0b9d662a071479`
  and `ba1b02757cc925d4a28a4a0861272063a5aa4510d91f5aaa3a39b870b5c51420`.


## Delivery checkpoint: 110 reports

- #855 `14e092327`: one fixture mapping extraction repairs required testdata lint.
  Fresh exact-diff review approved; all 237 artifacts across 17 variants match
  parent and independent repeat. Seven generated build/vet/race scenarios and
  scoped lint pass. Pushed and confirmed closed. Cleanup preserved hashes and
  summarized evidence in `/tmp/loom-855-cleanup-receipt.json`, SHA
  `4e2b4383da8a396733584875f5e3604ccfb01cae40839612ec4a8e26927a4657`.
- #857 `f609b2817`: synchronize the nested HTTP integration module with the
  shared test helper's existing dependency graph. Readonly parent compilation
  reproduces CI's missing checksum; candidate full integration tests, nested lint
  and clean tidy diff pass. Fresh reviewer independently confirmed the exact
  two-file patch `e492771bd3b4f08406649e0318dbb29490694de192eef99628efd1c0783a0a5e`
  and readonly compilation. Pushed and confirmed closed; no production or test
  assertion changes. GitHub snapshot `/tmp/loom-checkpoint-110-all-issues.json`
  confirms 110 closures among 178 distinct tracked reports.
- #691 readiness manifest
  `133516e23c2ff3df9477c147f53e60b3760e3534f7969d62f4f36ee404667595`
  passed independent review of sources, evidence, 345 artifact hashes and five
  models. #754 revised readiness manifest
  `0fb740fb9b68307257a730186bde2e3c91d1e49ec309efe940f38666e7952cb2`
  passed fresh review resolving the medium helper-ABI finding. Both implementations
  are active, not delivered. #854 readiness manifest
  `f6628f59dad27e1aff1396bc3828bf3be4676ea661b44b00a8892e87233e1486`
  passed root's independent table/model checks; named-view source conversion
  discovered during implementation remains under ownership review.
- #856 preserves the separately reproduced forbidden `init` import alias across
  transports/examples. It remains open; #691 does not claim that repair.
- Delivered #694 cleanup removed inactive artifacts and raw logs after process
  checks. Receipt `/tmp/loom-694-cleanup-receipt.json`, SHA
  `5e8dd7141b4565ab4adffd57110ded3a8d6895ef3ec1a654b7c611f9dd23fe48`.
  Active source packets, current reproduction inputs and shared tools remain.

## Delivery checkpoint: 114 reports

- #691 delivered `65f652a43`: exact independent final/integration review, 21
  generated modules, 479 artifacts with 169 intended changes and exact repeats;
  wire/schema controls unchanged. Cleanup receipt `/tmp/loom-691-cleanup-receipt.json`.
- #754 delivered `56c46446f`: independent final/integration review, six specimens,
  66 artifacts with eight intended private-helper changes, mixed-order and
  generated adapters/race controls. Cleanup receipt `/tmp/loom-754-cleanup-receipt.json`.
- #858 delivered `8faa2e6a6`: shared helper now owns required views; seven caller
  compensations removed. Parent58/candidate60 artifacts differ only by two missing
  view files; previous complete assembly and fresh repeat are byte-identical.
  Independent race/build/vet regression passes5.502s. Final manifest
  `0b0fd794e773af67d8751516dec41dd2918ba225752be87a7a02fe9273851224`.
- #854 delivered `4710891ff`: root independently approved all18 files and verified
  144 artifact hashes; separate integration review preserved #691/#754 guidance.
  Integrated direct and complete mixed build/vet/race checks pass0.619s/4.129s.
  Original required-view and optional named nil/empty/value controls are retained.
  Final manifest `b02855cae7d52a92fa8de0e3fa6d9372770f41a6d9146b100add9402e91a68eb`.
  #735 shared-alias witness and #853 envelope projection remain open.
- #859 mixed-result streaming accessor and #860 stale generated-name harness
  reports were independently reproduced before filing. GitHub snapshot confirms
  114 closed of181 tracked reports. No complete green CI claim is made.

- 2026-10-13 — **Checkpoint120:** #860/#847/#862/#861/#720/#740 pushed and
  confirmed closed. #847 final manifest2d2313e3 and integration wire matrix pass;
  #861 retains165 identical artifact triples plus18 pinned Vacuum specs;
  #862 negative identity controls prove assertions remain meaningful;
  #720 final manifestcfffe910/integrationb0577088,16 modules231 artifacts;
  #740 final manifest6f750495/integrationae7a1e41,6 specimens78 artifacts.
  Independent final and integration receipts are under `/tmp/loom-N-*verdict.json`.
  Latest main9b82560da;183 tracked reports,120 confirmed closed. Continue #853
  shared projection, #735 constructor identity readiness and remaining CI triage.

- 2026-10-13 — **Checkpoint124:** #863/#864/#865/#853 independently reviewed,
  pushed and confirmed closed; 124 of 186 tracked reports delivered. Main
  `fae477d0e` contains shared response projection before protocol framing.
  Original final patch `10c6bc8537`, integrated patch `83f2a81014`, fresh
  integration verdict `5e79a99333`; current 39-file fixture output repeats exactly,
  both apps build/vet/race, integrated mounted matrix and full `make lint` pass.
  Continue #735 constructor ownership and #859 streaming interceptor readiness;
  upstream prerequisites and GitHub aggregate CI remain unresolved.

- 2026-10-13 — **Checkpoint125:** #866 pushed as `b2b6f1480` and confirmed
  closed after independent exact-final approval. Inventory now has 188 reports,
  125 closed; #867 separately records reproduced mixed unary client dispatch.
  #859 readiness approved and implementation authorized; #735 continues bounded
  constructor validation, and #726 is modeling gRPC stream cancellation ownership.

- 2026-10-13 — **Checkpoint128:** #735 delivered `56a78637a`, #859 delivered
  `c8515931c`, #869 delivered `641c103c2`; all independently reviewed,
  pushed and confirmed closed. #735 retained 264 exact artifact triples and
  composed constructor/client controls. #859 retained 445 artifact hashes and
  all four generated integration harnesses passed. #869 repairs 21 stale
  expectations; affected HTTP/gRPC groups pass. New #868 records the separately
  reproduced bodyless JSON-RPC compilation regression. 128 of 190 tracked
  reports are closed; full CI remains unresolved.

- 2026-10-13 — **Checkpoint129:** #868 delivered `ab9da36ea` after root final
  and separate integration review. Main generated compiler regressions and
  mounted response matrix pass (JSON-RPC 5.479s, generator 2.825s). Only four
  endpoint bindings change in the bounded comparison. 129 of 190 tracked
  reports are confirmed closed. Continue #790 ID ownership readiness, #867 mixed
  dispatch and #726 gRPC cancellation; upstream #820 remains a race-job blocker.

- 2026-10-13 — **Checkpoint130:** #870 delivered `2e5fb203b` after exact
  independent review, pushed and confirmed closed. Its required-lint correction
  preserves 218 generated artifacts across parent/candidate/repeat; all 16
  modules compile and mounted source-selection controls pass. Inventory has
  191 reports, 130 closed and 61 open. #790 readiness re-review approved after
  correcting WS route witnesses; implementation authorized. #867 implementation
  includes shared CLI consumers. #734 readiness independently approved. Root
  review exposed two #726 lifetime edges, both reproduced and repaired before
  final acceptance. Latest aggregate CI remains unresolved.

- 2026-10-13 — **Checkpoint134:** #726 delivered `9508a1cb1` and #867
  delivered `cdc581066`, with independent final and integration reviews.
  #758 independently reconciled and closed against #726 using actual TCP race
  controls for explicit rejection and serialized 16,000-message completion.
  134/191 tracked reports are closed; #734 delivered `555d13306` after
  independent final review. #790 implementation and
  #782 readiness continue; #705 binary wire-key policy is pending. #820 remains
  the known upstream CI race blocker; current main CI is still active.

- 2026-10-13 — **Checkpoint138:** #782 delivered `4813696b7`, #873 delivered
  `f6ca917ab`, #874 delivered `d212ddc51` and #871 delivered `4f569fad1`;
  all independently reviewed, pushed and confirmed closed. Inventory adds four
  demonstrated CI findings #871–#874: 138/195 reports closed. #782 changes only
  five accessor artifacts without broadening invocation; #871 centralizes native
  terminal-framing classification while preserving real errors. #790 main
  integration follows final re-review; #872 follows approved readiness.
  Required CI remains unresolved; #820 upstream race and #705 policy remain open.

- 2026-10-13 — **Checkpoint143:** #790 delivered `e3c3ad28b`, #872 delivered
  `5c6535b6d`, #875 delivered `57316cb41`, #876 delivered `f90c6d8f0`, and
  #699 delivered `71a0758f6`; independently reviewed, pushed and confirmed closed.
  Inventory adds two reproduced test-maintenance reports: 143/197 closed.
  #699 preserves all 57 selected generated artifacts exactly; invalid local
  duplicates now fail before generation while narrower overrides remain valid.
  User accepted derived binary metadata keys for #705. #682 final review and
  #697 implementation continue; #723/#700/#705 readiness remains active.
  Latest required CI is unresolved; no aggregate green claim.

- 2026-10-13 — **Checkpoint147:** #682 delivered `f5435f493`, #878 delivered
  `0277e0f36`, #697 delivered `af21e5098`, and #723 delivered `a7b09d5a8`;
  independently reviewed, pushed and confirmed closed. #697 shares semantic
  dependency order without changing parser signatures. #723 uses one optional
  diagnostic callback owner across runtime and generated handlers; all 80 mounted
  and direct cases pass. #877 separately tracks named metadata element storage;
  its readiness and #705 readiness passed independent review. #700 readiness and
  design admission investigation continue. Required CI remains unresolved.

- 2026-10-13 — **Checkpoint153:** #700 delivered `f0dbd2e20`, #698 delivered
  `dc558fcbf`, #877 delivered `6d2c03510`, and #701 delivered `e631f0a89`;
  #881 delivered `bbe2008ac` and #882 delivered `5e7ddce00`;
  independently reviewed, pushed and confirmed closed. Server receiver identities
  are centralized; CLI help text stays literal; metadata storage preserves named
  service types; invalid union identities fail during evaluation. Inventory adds
  demonstrated CI fixture/coverage findings #881/#882: 153/203 closed. All 12 affected SSE tests pass;
  35 fixture artifacts repeat exactly. #705/#703/#879 implementation and root
  #880 readiness continue. No aggregate CI green claim.

- 2026-10-13 — **Checkpoint155:** #703 delivered `2da0fd46e` and #705 delivered
  `22dddd67f`, independently reviewed, pushed and confirmed closed. The field
  number parent comparison was rebuilt from an exact archive after invalid
  overlay evidence was discovered; all 30 artifacts match. Binary metadata has
  124 artifacts per source group with 13 intended changes and exact repeats;
  integrated native checks and full expr/DSL race suites pass. #883 adds one
  separate service-client receiver report: 155/204 closed. #879 final review,
  #880 implementation and #702 readiness continue.
