# Credential alternative extraction (#732)

The bounded model separates strict bearer parsing from whether foreign
Authorization framing can leave an optional credential absent. The shared
`internal/securitygen.PlanBearer` owner permits that absence only if an OR
requirement omits the credential and the payload does not require it. Anonymous
requirements qualify; several requirements using the same credential do not.
HTTP/JSON-RPC header analysis and gRPC metadata analysis consume this plan.

The runtime `security.IsOtherAuthorizationScheme` recognizes an ASCII HTTP token
scheme followed by a space and a nonempty credential. It does not validate the
foreign credential. `security.DecodeBearer` remains unchanged and strict.
Generated decoders assign only successfully parsed bearer values; foreign
framing never fabricates a present empty token. Malformed matching framing and
authored validation remain global rejection. Existing callbacks retain the
ability to inspect absent optional values, including transport-owned cookies.

## Model and correspondence

Input classes are absent, foreign scheme, valid bearer and malformed bearer.
The `required` flag models unconditional requiredness of this bearer credential;
Basic component requiredness is covered by generated tests. The `shared`
topology represents alternatives that all need the same credential. The model
assumes the bearer callback rejects absent credentials and models callback
success/failure and resulting context selection. It abstracts actual parsing,
validation predicates, callback implementation and generated Go.

- `legacy.cfg` reproduces a global parser failure preventing a valid independent
  alternative: `OptionalAlternativeUsable` fails at depth 4 (exit 12).
- `checked.cfg` passes all four invariants: 387 generated / 381 distinct states,
  depth 4. It retains singleton, AND and required-credential rejection and
  models the existing OR/AND context contract.

Production correspondence:

| Owner | Regression |
| --- | --- |
| Shared alternative membership and wire classification | `TestBearerMapping`, `TestBearerAlternativePlan` |
| Foreign scheme framing and strict bearer parser | `TestOtherAuthorizationScheme`, unchanged `TestDecodeBearer` |
| HTTP and JSON-RPC POST projection | `TestCredentialAlternativesGenerated` in each codegen package |
| gRPC metadata projection | `TestCredentialAlternativesGenerated` in gRPC codegen, bufconn requests |
| Existing Basic requiredness and presence | `TestBasicPresenceGenerated`, new Required controls |
| Shared callback order/context | New transport wire tests: both OR orders, failed context discard, AND chaining |

Before production edits, the new generated HTTP `TestBasicAlternative` failed
with `invalid Authorization bearer credential` for `Authorization: Basic dXNlcjpwYXNz`. The model was checked
before production edits. The expanded generated tests retain named credential
strings and check pointer presence, decoder error names, wire error class,
callback order/context and service invocation. They are implementation evidence,
not consequences of the finite model.

## Run

Use TLA+ tools release 1.7.4 (TLC 2.19), jar SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`:

```sh
java -cp "$TLA_JAR" tlc2.TLC -config legacy.cfg -metadir "$AUTH_MODEL_OUTPUT/legacy" CredentialAlternatives.tla
java -cp "$TLA_JAR" tlc2.TLC -config checked.cfg -metadir "$AUTH_MODEL_OUTPUT/checked" CredentialAlternatives.tla
```

Legacy must violate `OptionalAlternativeUsable`; corrected must pass all four
invariants. Keep model sources and summarized evidence; remove checker state
and logs after independent review. Toolchain used: Go 1.27.2 darwin/arm64 and
Corretto Java 25.0.4.1.

```sh
go test ./internal/securitygen ./security ./codegen/service -count=1
LOOM_DIR="$PWD" go test ./http/codegen ./grpc/codegen ./jsonrpc/codegen -run 'TestCredentialAlternativesGenerated|TestBasicPresence|TestNamedCredential|TestConstrainedCredential|Test.*SessionSecurity|Test.*SecurityScope|TestDecode$|TestRequestDecoder$|TestRequestEncoder$' -count=1
```

The gRPC harness filters the module test run to its credential tests because the
renderer also creates response-contract scenario skeletons. All generated
packages compile; those unimplemented scenario skeletons are not new failures.


## Generated-output comparison

Baseline: `9880865a1c2873cf455fbae979dc7de3a533071f`. A `git archive` extraction,
initialized with local Git metadata, was used for parent generation. Each
process set `LOOM_DIR` to its own source; JSON-RPC's helper resolved that source's
Git root. The identical new fixture was copied into parent. Candidate generation
was repeated in another Go test process with a separate destination. A fixture
helper split for maintainability was separately regenerated at both revisions;
its outputs matched the already compiled snapshots exactly.

| Input | Artifacts | Parent/candidate differences |
| --- | ---: | --- |
| `CredentialAlternativesDSL("http")` | 12 | HTTP server decoder only |
| `CredentialAlternativesDSL("grpc")` | 11 | gRPC server decoder only |
| `CredentialAlternativesDSL("jsonrpc")` | 10 | JSON-RPC server decoder only |
| HTTP `PayloadJWTAuthorizationHeaderDSL` | 12 | None |
| HTTP `NamedCredentialFieldsDSL` | 12 | None |
| HTTP `BasicPresenceDSL` | 12 | None |
| HTTP `apiLevelSessionCookieSecurityDSL` | 12 | HTTP server decoder only |
| JSON-RPC `jsonrpcAPISessionSecurityDSL` | 10 | JSON-RPC server decoder only |
| `http/integration_tests/fixtures/quality` | 17 | HTTP accounts server decoder only |

All 108 candidate artifacts match their process-isolated repeat byte for byte.
The six changed files only add the foreign-scheme exception to the strict
bearer failure guard and assign the credential when parsing succeeded. Client,
service, interface, endpoint callback and unrelated decoder output is unchanged.
All parent/candidate modules compile and vet. Quality fixture OpenAPI JSON/YAML
and manifest are byte-identical; no OpenAPI semantics changed. Existing checked-in
security decoder goldens pass unchanged. The checked-in ticktock generated trees
have no credential extraction and need no regeneration.

To repeat, use `RunHTTPDSL`/`renderHTTPModule`,
`RunGRPCDSL`/`renderGRPCResponseContractModule`, and
`RunJSONRPCDSL`/`renderJSONRPCModule` in their existing codegen test packages with
module `example.com/alternatives`. Render each row in its own directory; compare
entire `gen` trees by relative name and exact bytes, including missing/extra
files. Compile accepted modules with `go mod tidy`, `go test ./... -run '^$'` and
`go vet ./...`. Repeat candidate generation without rebuilding unchanged bytes.
For quality, copy the fixture, rewrite its Loom replace to each source, run
`go run -mod=mod github.com/CaliLuke/loom/cmd/loom gen example.com/http-quality/design`,
then `go test ./...` and `go vet ./...`. Preserve inputs and summarized evidence;
remove temporary source archives, probes, binaries and logs after independent
review.
