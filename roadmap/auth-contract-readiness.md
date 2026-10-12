# Authentication scope resolution readiness

Status: independently approved; root released implementation. Candidate validation is complete and awaits exact-diff review and integration. Evidence: `expr/tla/authentication_scope/README.md`.
Owner: auth agent. Worktree: `/Users/luca/.codex/worktrees/auth-contract/loom`.
Parent: `4535a403048510f055fc59ec3dfed41e3769ad58`.
Owning structural ticket: #724; #725 is a linked regression witness of the same declaration-resolution boundary, not a separate delivery task.

## Scope and acceptance

Select the nearest API/service/method scope with any authentication declaration,
then lower ordinary requirements and sessions from that scope together. Preserve
same-scope OR alternatives, multi-scheme AND requirements, scopes, explicit
anonymous `Security()`, and method `NoSecurity()` inheritance override. Reject
same-method `NoSecurity()` combined with authenticated `Security` or
`SessionSecurity` in either declaration order. Redundant anonymous declarations
are not authenticated contradictions. No transport independently reselects scopes.

Direct tests must cover API-to-service, API-to-method and service-to-method
replacement in both ordinary/session directions, combined same-scope declarations,
anonymous overrides, inherited and empty controls, NoSecurity contradictions,
pre-finalization queries versus finalized output, payload field injection and
header/cookie mapping absence after overrides. Repeated effective queries must
not mutate enclosing declarations. Preserve method-local copies of schemes.

The concrete risk is a restrictive Basic method accepting browser sessions.
The framework owns inheritance and derived credential injection. Applications
continue to own credential verification and authorization policy. The checked-in
HTTP quality fixture uses session credentials at method scope and should remain
byte-identical. A workspace search found session usages in framework fixtures;
no external consumer claim is made without locating its actual design.

## Owning code and consumers

- `expr/method.go`: split `inheritedRequirements` / `inheritedSessionAuths`,
  `Finalize`, `EffectiveRequirements`, `EffectiveSessionAuths`; replace split
  selection with one private owner, rather than transport compensation.
- `expr/method_validate_security.go`: reject contradictory local declarations.
- `expr/method_session.go`: credential injection runs during Method.Validate;
  it must consume only selected sessions before payload validation.
- `expr/http_endpoint_mapping.go`: transport-only cookie validation queries
  effective sessions during validation.
- `expr/http_endpoint_finalize.go`: finalized method requirements are copied to
  endpoint schemes; cookie inference and transport-owned lookup must see the same
  selected sessions, without resurrecting outer declarations.
- `expr/http_endpoint_security.go`: path credential validation consumes
  EffectiveRequirements before finalization.
- `codegen/service/service_data_methods.go`: endpoint requirements and cookie
  auth data consume the effective methods; HTTP/gRPC/JSON-RPC use that shared data.
- OpenAPI consumes evaluated HTTP requirements; API defaults remain document
  defaults, with operation requirements overriding them.

Reviewed implementation scope: `expr/method.go`, optionally a focused
`expr/method_security.go` extraction, `expr/method_validate_security.go`,
`dsl/security.go` (GoDoc), `docs/dsl-reference.md`. A consumer/framework skill
wording delta goes to root rather than being edited in this worktree.

## Test/model files and sequence

1. Add `expr/method_security_scope_test.go` with the matrix above. Run
   `go test ./expr -run 'TestAuthenticationScope|TestNoSecurityConflict' -count=1`
   before changing production code and record actual failures.
2. Add `expr/tla/authentication_scope/AuthenticationScope.tla`, `legacy.cfg`,
   `checked.cfg`, and `README.md`. Model three scopes with ordinary/session/
   anonymous alternatives and NoSecurity; legacy independent selection must
   violate nearest-scope ownership and legacy contradiction admission must fail
   a separate invariant. Candidate must satisfy selection and admission. Use
   TLA+ 1.7.4, existing jar `/Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar`
   (verify SHA-256 against repository pin). Run each configuration with
   `java -cp "$TLA_JAR" tlc2.TLC -config <config> -metadir <task-temp-dir> AuthenticationScope.tla`.
   Add a separate legacy-contradiction config if independent controls require it.
   The model is bounded scope selection, not a proof of Go DSL or wire behavior.
3. Implement the shared selection owner and contradiction validation. Re-run
   regressions, then `go test ./expr ./security ./codegen/service -count=1`.
4. Add focused assertions in existing `codegen/service/security_test.go`,
   `grpc/codegen/security_test.go`, `jsonrpc/codegen/security_test.go`, and new
   `http/codegen/security_scope_test.go`. Add compiled transport regression
   modules via existing renderHTTPModule/renderGRPCResponseContractModule/
   renderJSONRPCModule helpers, in corresponding `security_scope_test.go` files.
   HTTP Basic override must reject bearer/cookie-only calls and invoke only the
   Basic callback; selected session methods must retain bearer/cookie behavior.
   gRPC and JSON-RPC compiled adapters must expose only selected credentials and
   invoke the matching shared auth endpoint. Test ordinary-over-session and
   session-over-ordinary separately; do not exercise known broken shared-header
   Basic-or-Bearer decoding as acceptance of this scope task.
5. Extend existing OpenAPI operation-security test coverage to compare selected
   requirements and absent overridden session cookie bindings. Parse emitted
   document using existing libopenapi test helpers.
6. Run bounded transport filters:
   `go test ./http/codegen -run 'Test.*SecurityScope|Test.*SessionSecurity|TestOpenAPIOperationSecurity' -count=1`
   `go test ./grpc/codegen ./jsonrpc/codegen -run 'Test.*SecurityScope|Test.*SessionSecurity' -count=1`.
   Set `LOOM_DIR` to this worktree for local generated modules.

## Bounded generated comparison

Reuse existing session fixtures: HTTP `quality` checked-in fixture,
`codegen/service/testdata.EndpointWithAPISessionSecurityDSL`,
`EndpointWithServiceSessionSecurityNoSecurityDSL`, and existing grpc/jsonrpc
session harness designs. Add override cases to those test harness families,
without a new exhaustive generation harness. Copy the bounded test/fixture
inputs unchanged into a temporary baseline source extraction using `git archive`
of the parent. Initialize that extraction with `git -C "$AUTH_BASELINE" init`
(no commit, checkout, reset, stash or clean), verify
`git -C "$AUTH_BASELINE" rev-parse --show-toplevel`, and run baseline tests from
that root with `LOOM_DIR="$AUTH_BASELINE"`. This provides the Git root needed by
existing module helpers without remote-source selection. Run current commands
from this managed worktree with `LOOM_DIR` set to this worktree. Generate at
baseline/current with identical module path/toolchain,
compare every generated byte, and compile both accepted outputs. Baseline tests
expect the bug where applicable; rejected contradiction designs are generation
failures only after the correction. Repeat current generation in a second
independent process and compare exact output bytes. Keep generated fixture
identity metadata source-aware when interpreting differences, recording it
explicitly rather than silently ignoring files. Existing quality fixture must
have no semantic or code differences.

Expected differences are limited to removal of inherited session credential
fields, session transport bindings and fallback authorizer calls when ordinary
security wins; removal of inherited ordinary schemes/callbacks when sessions win;
operation OpenAPI requirements reflecting that same selected scope; and explicit
DSL rejection of contradictory NoSecurity declarations. Same-scope and inherited
controls remain identical. Capture actual changed files and any metadata-only
changes in the model README evidence section before review. Reuse source and
checks when later documentation changes do not invalidate them.

## Exclusions and delivery

#729 credential wire-binding collisions, #731 Basic constraints, #732 decoder OR
alternatives, #743 gRPC stream admission ordering, #752 per-scheme scopes and
#728 JSON-RPC WebSocket credential placement have distinct owners and remain
separate tasks. No broad cleanup, new dependencies, or generated-file hand edits.

After checks, run gofmt on touched Go files and `git diff --check`. Root assigns
an independent exact-diff reviewer against the stated parent. Resolve all findings
under AGENTS.md; do not commit or change an authorized diff until review allows
it. Root owns push/integration. Remove task-owned checker/build outputs only
after review and no active reader needs them, retaining durable evidence.

## Implemented fixture refinement

The three generated transport tests share parameterized design and endpoint test
fixtures in `expr/testdata/security_scope.go` and `security_scope_harness.go`;
these are not zero-argument exported design-corpus entries. Existing renderers
compile both replacement directions, and the HTTP test adds parsed OpenAPI and
wire behavior. The new direct regression file carries the planned method matrix.
No production transport change or security-test golden update was needed.
