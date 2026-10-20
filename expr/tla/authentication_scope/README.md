# Authentication scope ownership

Issue #724 owns this repair; #725 is a contradiction-admission witness at the
same declaration boundary. `MethodExpr.selectedSecurity` selects ordinary and
session declarations together, before session lowering, payload injection or
transport mapping. A local ordinary declaration must not inherit outer sessions,
and a local session declaration must not inherit outer ordinary requirements.
`NoSecurity` overrides inherited declarations but rejects authenticated local
peers. `Security()` remains an anonymous OR alternative.

## Model and correspondence

`AuthenticationScope.tla` explores 1,024 authored inputs: three scopes, every
subset of ordinary/session/anonymous declarations, and method NoSecurity present
or absent. The candidate uses the implementation's method/service/API fallback
on complete declaration pairs. `NearestScope` independently chooses the greatest
index containing a declaration. Session lowering becomes two alternatives.
`ConflictRejected` requires rejecting exactly local authenticated contradictions.

- `legacy.cfg`: independent inheritance adds the service Basic alternative to
  a method session declaration. It violates `NearestScope` at depth 2.
- `legacy-contradiction.cfg`: NoSecurity silently accepts a method Basic
  declaration and clears its requirements. It violates `ConflictRejected` at
  depth 2.
- `checked.cfg`: both invariants pass over 2,048 distinct states, depth 2.

The model abstracts one ordinary scheme and one session contract; it does not
prove arbitrary scheme identity, Go expression copying, declaration order,
credential parsing, callback behavior or generated programs. It treats each
ordinary declaration as an opaque alternative. Tests cover the additional AND
scheme/scopes and ordered alternatives, payload and transport boundaries.

Production correspondence:

| Owner | Regression |
| --- | --- |
| `expr/method.go`: `selectedSecurity`, effective queries and Finalize | `TestAuthenticationScope`, `TestAuthenticationScopeQueries` |
| `expr/method_validate_security.go`: contradictory local declarations | `TestNoSecurityConflict`, both declaration orders and both declaration kinds |
| `expr/method_session.go`: selected-session payload injection | `TestAuthenticationScope`, credential field presence |
| HTTP mapping finalization, service endpoint analysis, OpenAPI operation requirements | `TestHTTPSecurityScope`, parsed OpenAPI, compiled adapters and wire rejection |
| gRPC and JSON-RPC transport analysis | `TestGRPCSecurityScope`, `TestJSONRPCSecurityScope`, compiled adapters and selected endpoint callbacks |

`expr/testdata.SecurityScopeMatrixDSL` additionally supplies one bounded corpus
for generated HTTP, JSON-RPC and gRPC adapter runtime tests: API inheritance,
service inheritance, method replacement, same-scope alternatives and
`NoSecurity`. Each adapter verifies accepted and rejected credentials, callback
and service invocation counts, and its own protocol failure representation.
HTTP session carriers and gRPC session metadata have separate runtime controls.
Compiled endpoint tests retain both replacement directions, bearer-to-cookie
fallback and failed credentials.

These runtime checks extend implementation correspondence beyond the model's
opaque authentication alternatives. They do not add a proof of credential
parsing, network ordering or arbitrary generated programs. No model transition
or invariant changes for this coverage improvement.

## Run

Use the existing TLA+ 1.7.4 tools jar, SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
From this directory, set `TLA_JAR` to that jar and `AUTH_MODEL_OUTPUT` to a fresh
task-owned temporary directory:

```sh
java -cp "$TLA_JAR" tlc2.TLC -config legacy.cfg -metadir "$AUTH_MODEL_OUTPUT/legacy" AuthenticationScope.tla
java -cp "$TLA_JAR" tlc2.TLC -config legacy-contradiction.cfg -metadir "$AUTH_MODEL_OUTPUT/contradiction" AuthenticationScope.tla
java -cp "$TLA_JAR" tlc2.TLC -config checked.cfg -metadir "$AUTH_MODEL_OUTPUT/checked" AuthenticationScope.tla
```

Legacy configurations must fail the named invariants (exit 12). A parser/tool
failure is not reproduction. Checked must exit 0. Preserve source and this
summary; remove state directories and traces after independent review.

From the repository root:

```sh
go test ./expr -run 'TestAuthenticationScope|TestNoSecurityConflict' -count=1
go test ./expr ./security ./codegen/service -count=1
LOOM_DIR="$PWD" go test ./http/codegen ./grpc/codegen ./jsonrpc/codegen -run 'Test.*SecurityScope|Test.*SessionSecurity|TestOpenAPIOperationSecurity' -count=1
```

## Validation evidence

Validated 2026-10-12 with Go 1.27.2 darwin/arm64 and Corretto Java 25.0.4.1.
Baseline source: `4535a403048510f055fc59ec3dfed41e3769ad58`; candidate is the #724
scope-selection change. The direct tests were added and failed before production
edits: three ordinary overrides retained jwt/key; three reverse overrides
incorrectly demanded Basic tags; an anonymous override retained sessions; four
NoSecurity contradictions were accepted. All commands above now pass. The final
model sources produced the expected two counterexamples and checked success.

Bounded generation reused existing rendering helpers, identical fixture inputs
and the module path `example.com/scope`, separately in baseline and candidate
processes. The baseline was a `git archive` extraction initialized with `git init`
so `loomsource.RepositoryRoot` resolved; each process bound `LOOM_DIR` to its own
source. No baseline source was replaced by candidate production code. New
parameterized fixture `expr/testdata.SecurityScopeDSL` was copied unchanged into
baseline. A session override without Basic fields is rejected by baseline DSL
validation, while candidate compilation succeeds. This admission difference is
intentional; there is no valid baseline output to compare for that case.

Cases and exact generated-byte comparison:

| Input | Baseline vs candidate | Candidate independent repeat |
| --- | --- | --- |
| `SecurityScopeDSL("http", false)` | 8 of 13 generated files differ | all 13 identical |
| `SecurityScopeDSL("grpc", false)` | 5 of 11 differ | all 11 identical |
| `SecurityScopeDSL("jsonrpc", false)` | 5 of 10 differ | all 10 identical |
| `SecurityScopeDSL` with true, each transport | baseline DSL rejection; candidate compiles | all 13/11/10 identical |
| HTTP `apiLevelSessionCookieSecurityDSL` | all 13 identical | all identical |
| gRPC `grpcAPISessionSecurityDSL` | all 11 identical | all identical |
| JSON-RPC `jsonrpcAPISessionSecurityDSL` | all 10 identical | all identical |
| service `EndpointWithAPISessionSecurityDSL` | both files identical | both identical |
| service `EndpointWithServiceSessionSecurityNoSecurityDSL` | both files identical | both identical |
| checked-in HTTP `quality/design` | all 17 files identical, including manifest and OpenAPI JSON/YAML | covered by bounded session repeat above |

All baseline and candidate accepted modules compiled using `go mod tidy` and
`go test ./... -run '^$'`. The two service modules used `go test ./...`. The
quality fixture was copied, its Loom replace redirected to each source, then
regenerated using `go run -mod=mod github.com/CaliLuke/loom/cmd/loom gen
example.com/http-quality/design` and tested using `go test ./...`; both pass.
No checked-in generated output needs changing.

Every intended generated difference:

- Each transport's `scope/service.go` loses inherited Auth/Cookie fields and
  JWT/APIKey authorizer interface methods; `scope/endpoints.go` loses those
  fallback authorizers and constructor arguments.
- HTTP and JSON-RPC `scope/client/encode_decode.go`,
  `scope/server/encode_decode.go`, and `scope/server/types.go` lose inherited
  header/cookie encoding, decoding and constructor inputs. Basic remains.
- gRPC files at the same client/server relative paths lose jwt/key metadata
  encoding, decoding and payload constructor inputs. Basic remains.
- HTTP `http/cli/scope/cli.go` and `http/scope/client/cli.go` lose the inherited
  auth/cookie arguments. Rendered OpenAPI has only the Basic operation
  requirement and corresponding used security component.
- Reverse overrides now admit session-only payloads, emit only session
  authorizers/bindings, and do not demand Basic fields. Contradictory local
  NoSecurity declarations now fail before finalization.

## Separate residual

During matrix construction, an existing no-payload session injection panic was
observed at baseline, unrelated to scope selection. Minimal DSL body for
`expr.RunDSL`:

```go
jwt := JWTSecurity("jwt")
session := SessionAuth("session", func() {
    BearerTransport(jwt, "auth")
})
Service("scope", func() {
    Method("read", func() {
        SessionSecurity(session)
        HTTP(func() {
            GET("/")
        })
    })
})
```

`expr/method_session.go` initializes `candidate` to Object for Empty, then its
named-payload renaming branch asserts that candidate is UserType because Empty
implements UserType. Evaluation reports `panic: interface conversion:
*expr.Object is not expr.UserType: missing method Attribute`. This task's scoped
fixture uses an ordinary object payload; the separate residual remains open.

## Repeating the bounded comparison

The comparison probes are temporary test files, not production test APIs. To
repeat, extract the baseline into a fresh directory, initialize its Git metadata
without committing, and copy only `expr/testdata/security_scope.go` from candidate:

```sh
mkdir "$AUTH_BASELINE"
git archive 4535a403048510f055fc59ec3dfed41e3769ad58 | tar -x -C "$AUTH_BASELINE"
git -C "$AUTH_BASELINE" init
cp expr/testdata/security_scope.go "$AUTH_BASELINE/expr/testdata/security_scope.go"
```

In each transport's codegen package, create a temporary `package codegen` test
with the following loop, importing `os`, `path/filepath`, `testing`, and
`fixture "github.com/CaliLuke/loom/expr/testdata"`. Substitute the row's runner,
renderer and existing control DSL below. Copy this same probe into baseline.

```go
func TestAuthScopeSnapshot(t *testing.T) {
    output := os.Getenv("AUTH_SCOPE_OUTPUT")
    if output == "" {
        t.Skip("comparison probe only")
    }
    names := []string{"override", "control"}
    if os.Getenv("AUTH_SCOPE_REVERSE") != "" {
        names = append(names, "reverse")
    }
    for _, name := range names {
        design := fixture.SecurityScopeDSL(transport, name == "reverse")
        if name == "control" {
            design = controlDSL
        }
        root := runDSL(t, design)
        dir := filepath.Join(output, transport, name)
        if err := os.MkdirAll(dir, 0700); err != nil {
            t.Fatal(err)
        }
        renderModule(t, dir, "example.com/scope", root)
    }
}
```

| transport | runDSL | renderModule | controlDSL |
| --- | --- | --- | --- |
| `http` | `RunHTTPDSL` | `renderHTTPModule` | `apiLevelSessionCookieSecurityDSL` |
| `grpc` | `RunGRPCDSL` | `renderGRPCResponseContractModule` | `grpcAPISessionSecurityDSL` |
| `jsonrpc` | `RunJSONRPCDSL` | `renderJSONRPCModule` | `jsonrpcAPISessionSecurityDSL` |

For HTTP, also render `renderOpenAPIJSON(t, openapiv3.Files, root)` and write its
bytes to `dir/gen/openapi.json`; import the existing OpenAPI v3 package. Parse
coverage is in the durable `TestHTTPSecurityScope` test.

Run from each source root with its own `LOOM_DIR`, output directory and a fresh
Go test process. Set `AUTH_SCOPE_REVERSE=1` only for candidate snapshots; baseline
reverse rejection is proven by the failing direct test, not by comparing output.

```sh
LOOM_DIR="$PWD" AUTH_SCOPE_OUTPUT="$AUTH_OUTPUT" go test ./http/codegen ./grpc/codegen ./jsonrpc/codegen -run '^TestAuthScopeSnapshot$' -count=1
```

Repeat candidate into a second output directory. Compare each entire `gen` tree
by relative path and bytes (including OpenAPI), then compile accepted modules.
For the two service-only controls, the analogous service-package probe calls
`codegen.RunDSL`, `NewServicesData`, `Files`, `EndpointFile`, and each file's
`Render` with the same module path. Add a Go 1.27.2 `go.mod` with a local Loom
replace before tidying/testing. The quality fixture procedure is given above.
Remove all temporary probe files from candidate before review; preserve only
these inputs, commands and summarized results after review.
