# WebSocket message policy (#716): prerequisite review

Status: production shape approved by the root; candidate implemented; acceptance evidence recorded below, awaiting independent review.
This is a separate atomic ticket from #722. It makes inbound message admission
finite; it does not introduce a reader pump or change dispatch concurrency.

## Contract and shape

Use the issue's concrete default: `DefaultMaxRequestBodyBytes` (32 MiB) for one
inbound WebSocket message; reject a larger message with WebSocket close 1009.
Use Gorilla's existing `SetReadLimit` and its frame/fragment handling. Inspection
of pinned `conn.go` shows `advanceFrame` checks compressed frame payload bytes
before `NextReader` wraps the message in a decompressor. Therefore `SetReadLimit`
alone cannot bound expanded messages. The shared runtime must also bound reads
from Gorilla's `NextReader` to `M+1` expanded bytes and reject expansion over `M`
with close 1009. Retain Gorilla's protocol/framing parser; do not reimplement it.
The policy must cover both generated HTTP and JSON-RPC connection owners; client
symmetry is part of the approved #716 scope.

## Approved concrete API

Use one explicit immutable connection configuration. No context-carried policy,
JSON-RPC-specific read-limit field, conflicting-context check, or compatibility
constructor is proposed.

```go
type WebSocketConfig struct {
    maxMessageBytes int64
    configure ConnConfigureFunc
}
func NewWebSocketConfig(maxMessageBytes int64, configure ConnConfigureFunc) (WebSocketConfig, error)
func (c WebSocketConfig) MaxMessageBytes() int64
func NewWebSocketStream(config WebSocketConfig, policies ...StreamWritePolicy) *WebSocketStream
func (s *WebSocketStream) SetConn(conn *websocket.Conn, cancel context.CancelFunc) error
```

The config constructor rejects nonpositive sizes and MaxInt64 (the M+1 probe must not overflow); its private fields prevent
later invalid mutation. The zero config consistently means 32 MiB and no hook,
so generated defaults need no fallible configuration call. `MaxMessageBytes`
normalizes zero to the default and is the only read-limit default owner. Existing
`StreamWritePolicy` stays a separate optional input because it also governs SSE
writes; there is no reason to duplicate it in WebSocketConfig.

`NewWebSocketStream` now allocates an unattached owner with explicit settings.
`SetConn` is the **only** attachment/configuration operation; the old constructor
that directly wrapped a raw connection is removed. Direct runtime users allocate
with a config and call `SetConn`, checking its error. No raw adoption mode tries
to infer hidden Gorilla configuration. Nil connections are rejected by SetConn;
allocation without a connection is already represented by the constructor.

SetConn applies the selected Gorilla framing limit to the incoming connection
**before** invoking the hook, then adopts the returned connection. All reads use
the captured expanded-byte cap. The hook may install a smaller framing limit;
the owner never overwrites it afterward. A larger configured limit explicitly
replaces a preexisting smaller framing limit before the hook runs. Increasing
only the hook's framing limit does not enlarge the owner's expanded-byte cap;
choose a larger WebSocketConfig for that. No #722 reader may start before
configuration and attachment finish.

Replacement ownership is explicit, without socket identity heuristics:

- The hook owns closing/releasing the input if it returns a distinct connection.
  It must do so without invalidating the returned connection. The runtime cannot
  infer whether arbitrary underlying-connection wrappers alias each other.
- The stream owns the connection returned by the hook and applies the selected
  expanded-byte cap to it; a hook returning a replacement owns its framing
  configuration. It must not transfer an input it still expects the stream to
  retain alongside the replacement.
- A nil hook result is a configuration failure: SetConn closes the original input
  records that input as the terminal owned connection, and returns an error with close failure information. Retaining attachment state prevents HTTP error encoding after a successful upgrade. This centralizes the
  behavior currently present only in the JSON-RPC client generator.
- Closing the stream before or during configuration still closes the final
  attached socket immediately. Repeated attachment and concurrent Close use the
  same existing lifecycle owner; no second connection holder is introduced.

Generated HTTP `ConnConfigurer` remains the per-endpoint holder, but its fields
become `<Method>Config loomhttp.WebSocketConfig` instead of `<Method>Fn` callbacks.
`NewConnConfigurer(config loomhttp.WebSocketConfig)` assigns that config to every
endpoint. HTTP generated client/server constructor positions stay unchanged
(`*ConnConfigurer`, with nil meaning every endpoint uses zero/default config).
Each HTTP handler receives its WebSocketConfig instead of a raw callback.

Generated JSON-RPC client/server constructors replace their existing raw
`ConnConfigureFunc` argument with `loomhttp.WebSocketConfig`. The generated client
stores it once and applies it on every new physical connection; all logical
streams share that explicit configuration. JSON-RPC `StreamConfig` and its
request timeout/result-channel options remain unchanged. The runtime
`WebSocketHandlerSpec.Configure` field becomes `Config WebSocketConfig`.

The runtime reads `NextReader` through a maximum `M+1` expanded-byte reader.
Native Gorilla `ErrReadLimit` and expanded-limit violations retain a matchable
`websocket.ErrReadLimit` cause. Expanded overflow attempts close 1009 with a
bounded control-write deadline, then **always** closes the socket; a failed
control write cannot skip physical release. Native overflow already attempts
1009 in Gorilla, so do not send a duplicate control message. Cleanup errors can
be joined while retaining the limit cause. Repeated reads must retain the first
terminal read error after automatic close: adjust the runtime error precedence
rather than leave a bound-specific parallel state machine. This is the limited
terminal-cleanup dependency that #716 needs; broader #722 watcher/service ownership
remains separate.

Exact anticipated production files:

| File | Change |
| --- | --- |
| `http/websocket_config.go` (new) | Immutable WebSocketConfig, validation and single default lookup. |
| `http/websocket.go` | New constructor/SetConn signatures; shared configure/attach ownership; bounded NextReader consumption and terminal overflow cleanup. |
| `http/codegen/stream_sections_websocket_structs.go` | Per-endpoint config fields/constructor; server stream no longer stores separate hook. |
| `http/codegen/server_sections.go` | Pass per-endpoint WebSocketConfig into generated handler constructors. |
| `http/codegen/template_sources.go` | Handler config parameter and configured unattached stream allocation. |
| `http/codegen/stream_sections_websocket_common.go` | Replace callback/SetConn sequence with sole SetConn operation and error handling. |
| `http/codegen/client_jennifer.go` | Allocate with per-endpoint config, SetConn with existing cancel callback; no independent hook invocation. |
| `jsonrpc/stream_server.go` | WebSocketHandlerSpec.Config; allocate/configure through the shared owner. |
| `jsonrpc/codegen/handler_sections.go` | Pass server config into WebSocketHandlerSpec. |
| `jsonrpc/codegen/top_level_sections.go`, `top_level_server_init_sections.go` | Replace existing raw-hook argument/field with WebSocketConfig on client/server. |
| `jsonrpc/codegen/client_sections.go` | Allocate with stored config and call SetConn; remove duplicate callback/nil-result handling. |
| `docs/http-guide.md` | Default, expanded-byte semantics, hook ownership/precedence and concrete API migration. |

Update direct-runtime and generated-module consumers of these signatures,
including `jsonrpc/websocket_client_test.go`, `websocket_close_errors_test.go`,
`stream_server_websocket_test.go` and generated integration harnesses. Regenerate
changed HTTP/JSON-RPC WebSocket golden/section expectations. If generated example
wiring supplies nil at the changed JSON-RPC config position, update its owning
emitter and verify the selected compile fixtures; do not hand-edit examples.
No service interface, shared MustClose, gRPC, SSE protocol, response projection
or JSON-RPC request timer changes are intended. Shared skill/index edits remain
orchestrator-owned.

## Validation and review acceptance

First add a failing direct test proving the current wrapper admits a message
over the intended bound. Then cover exact limit and one byte over, fragmented
text and binary messages, invalid policy values, custom smaller and larger
limits, hook order and replacement behavior, before-attach terminal close,
normal message/decode-error recovery, close 1009 and physical socket release.
Include negotiated compression: inspect Gorilla's implementation and test
expanded payloads rather than treating compressed wire size as decoded size.

Generate bounded HTTP bidirectional and JSON-RPC WebSocket server/client fixtures
through existing module helpers, explicitly replace the runtime module path for
both sources, retain outputs and compare exact bytes against parent plus two
isolated candidate runs. Compile/vet the distinct output once and run oversized
and configured-limit cases over real sockets. Expected changes are attachment
policy/configuration and constructor plumbing only; payload shape, OpenAPI,
dispatch, framing, write deadline and observation are unchanged.

Use the same representative existing designs listed in the #722 readiness note,
but #716 only needs one HTTP bidirectional and one JSON-RPC response-ID fixture
plus a server-streaming client inbound-limit witness. Regenerate affected
WebSocket golden files via their update paths. No exhaustive corpus run.

## Memory accounting for the dependent pump

Recommendation for #722 after policy acceptance: a configurable 64 MiB default
framework-retained **raw message byte** budget, composed of a 32 MiB maximum
in-flight message plus a shared 32 MiB budget for queued frames and a raw frame
currently being decoded. Retain a frame's reservation until the runtime finishes
decoding it; moving a frame from queue to decoder must not free that reservation
prematurely. Enforce queue admission by actual bytes, not number of frames.

For a configured message cap `M` and total raw budget `T`, require `T >= 2*M`
with overflow-safe arithmetic; waiting/decode budget is `T-M`. The reader can
hold at most one additional complete/in-progress message of size `M`. Under
explicit overload semantics a frame that cannot enter the waiting/decode budget
terminates the connection visibly. It is not dropped followed by continued
successful reads. Aggregate overload can use a documented protocol close code
chosen during design; message-too-large remains 1009.

This is not a hard process-heap bound. Typed JSON objects can occupy more memory
than their encoded bytes; Go allocation/capacity overhead, compression buffers,
the current decoded object and application-retained results are additional.
`ReadMessage` transfers byte ownership to its caller, whose retention is outside
the runtime queue budget. A claim bounding arbitrary decoded objects would need
a different streaming decoder/ownership contract and is outside this ticket.
Tests/models must account separately for queue, in-flight and current decode
raw bytes and explicitly record these representation limits.

The user accepted overload termination on 2026-10-12;
it does not authorize an arbitrary per-message default. The 32 MiB default is
the accepted #716 policy, recorded separately before
production work begins.


## Candidate evidence (2026-10-12)

Owning ticket: #716 only. The candidate changes inbound admission and the
configuration/attachment API across generated HTTP and JSON-RPC peers. The
separate #722 red tests, lifetime model and roadmap are excluded from its file
manifest. No reader pump, aggregate buffer policy, dispatch concurrency, gRPC,
SSE protocol or response projection change is included.

Acceptance is bounded expanded-message admission before decode, terminal 1009
and physical release on overflow, one configuration/attachment owner preserving
hooks, truthful post-upgrade error handling, and mechanically propagated public
configuration across generated clients, servers and example CLI wiring.

Evidence:

- `http/websocket_message_policy_test.go`: default 32 MiB overflow, exact and
  exceeded limits, text/binary, fragmentation, compressed exact/expansion,
  smaller hook limit, larger explicit limit over a prior raw limit, and stable
  terminal error. Tests observe physical closure before test cleanup.
- `http/websocket_config_test.go`: invalid/overflow-prone config values,
  same/different repeated attachment, hook nil/replacement ownership, and
  Close racing configuration. Existing lazy-attach and write-policy tests stay
  in the affected runtime package suite.
- `http/codegen/websocket_message_limit_integration_test.go`: generated HTTP
  client/server compressed exact/overflow controls, plus nil-hook cleanup with
  no HTTP encoder fallback after successful upgrade; existing decode controls
  run in the same module.
- `jsonrpc/codegen/websocket_message_limit_test.go`: generated JSON-RPC
  client/server compressed normal/overflow controls, wrapped close 1009,
  ErrReadLimit, physical closure and zero service calls for rejected requests.
- `http/tla/websocket_message_limit`: old/unrestricted and framing-only controls
  fail ExpandedBound in 12 states each; complete owner passes all three
  invariants in 18 states. The README records assumptions and exact invocation.
- A standalone parent-source runtime witness retained under
  `/tmp/loom-ws-716-parent-witness` confirms the default and compressed oversized
  messages were accepted and the framing rejection left the local socket open.
  The valid exact-limit control passes. Its only source adaptation selects the
  parent's raw-connection constructor and applies the old raw framing limit.
- Runtime suites passed with `-race`; selected generator regressions and
  generated transport modules passed; `go vet ./http ./jsonrpc ./http/codegen
  ./jsonrpc/codegen` and `make lint-namescope` passed. Broad affected codegen
  package results are recorded separately in the final handoff.

Retained generation evidence is under `/tmp/loom-ws-716-comparison`, generated
with Go 1.27.2, the parent's exact source archive
`7d73607827eb2896e185b1438095b868a9d88df8`, and the candidate worktree. Both source
CLIs use the same basename `loom`, module path `example.com/specimen`, explicit
module replacement and LOOM_DIR. The initial comparison had differing CLI
basenames in generated command headers; rerunning with identical names removes
that harness artifact. All comparisons below use corrected inputs.

Each selected parent and candidate module ran CLI `gen`, `example`, `go mod
tidy`, `go build ./...` and `go vet ./...`. Two isolated candidate processes
produce identical bytes. SHA-256 manifests cover every generated/example file.
The HTTP server-streaming case was refreshed after the context-child cleanup
and config-field GoDoc changes; both HTTP cases were refreshed after final
field GoDoc changes.

| Existing design | Obligation | Exact intended differing files |
| --- | --- | --- |
| HTTP `StreamingResultDSL` | Lazy server attachment and server-streaming client cancellation hook/watcher, including no-hook default | `gen/http/streaming_result_service/{client,server}/{client or server,websocket}.go` (four files) |
| HTTP `BidirectionalStreamingDSL` | Bidirectional client/server configuration and attachment | `gen/http/bidirectional_streaming_service/{client,server}/{client or server,websocket}.go` (four files) |
| JSON-RPC `ViewedStreamResponseIDDSL` | Shared physical client config, server config, example server and CLI propagation; views/IDs unchanged | `cmd/idviews-cli/jsonrpc.go`, `cmd/idviews/jsonrpc.go`, `gen/jsonrpc/cli/idviews/cli.go`, `gen/jsonrpc/files/client/client.go`, `gen/jsonrpc/files/server/server.go` |

HTTP differences replace raw callbacks with immutable configs, move attachment
and hook execution into SetConn, route attach errors, and cancel the new child
context when its existing watcher exits. JSON-RPC differences replace its raw
callback argument/field, remove duplicate hook handling, and use explicit zero
config values in generated examples. All other emitted files are byte-identical,
including service/view types, codecs and any emitted non-WebSocket artifacts.
No checked-in fixture app contains the changed WebSocket constructor/hook API;
changed WebSocket goldens and section literals were refreshed from generator
output. No generated `gen/` file was edited manually.

The first fragmented-message test revision accidentally imposed a 32-byte
write buffer on the 32 MiB default test, causing a timeout and test cleanup race.
The small buffer now applies only to fragmentation controls, and the handler is
joined before test completion. The subsequent full runtime race suites passed;
no acceptance result depends on the failed test-harness revision.

Shared skill/index deltas remain root-owned: document WebSocketConfig as the
single connection-level read/default/hook owner, SetConn as the sole attachment
path, the expanded-byte bound and hook replacement contract, and link the
message-admission model separately from the pending lifetime model.


Broad codegen validation completed against the in-progress snapshot. Its only
failures were the intentionally stale HTTP server-init/WS goldens, an earlier
version of the new wrapped-CloseError assertion, and a real new JSON-RPC local
name collision (`wsstream`). The latter was fixed by using the existing reserved
local `stream` in the connection-bootstrap scope, without another reservation
list or naming exception. `TestServicePackageNamedLikeLocalGeneratedModuleBuilds/websocket`,
`TestJSONRPCWebSocketUsesSharedRuntimeStream`, the generated message-limit
regression, the server-init streaming golden, WS goldens and section assertions
all passed after their fixes. The affected JSON-RPC CLI comparison was refreshed
for the final emitted local name. Unaffected passing broad-run cases were reused;
the whole broad run was not repeated.

Final checks also include `make lint-gofmt` and `git diff --check`. The candidate
is intentionally uncommitted for root transfer and fresh independent review on
the integrated parent. Temporary logs/modules remain until that review finishes;
then preserve summarized evidence here and delete task-owned temporary outputs.

## Integrated-parent verification

# Integrated #716 validation

Candidate: `/Users/luca/code/loom-mono/loom`, based on
`9880865a1c2873cf455fbae979dc7de3a533071f`, with the integrated #716 patch.
This is implementer validation, not independent review. No main files were edited
by this validation agent.

Go: `go1.27.2 darwin/arm64`. Parent source was extracted from the exact commit;
current generation and test commands ran from main with explicit local source.
The JSON-RPC test module helper resolves the running checkout with git, hence
selected main. HTTP logs explicitly identify `/Users/luca/code/loom-mono/loom`.

## Generation comparison

Run script: `/tmp/loom_ws_integrated_compare.py`.
Retained outputs: `/tmp/loom-ws-716-integrated-comparison`.
Both CLIs have basename `loom`; all modules use `example.com/specimen` and an
explicit module replacement to the selected parent/main source. Each case runs
`go mod tidy`, `loom gen example.com/specimen/design`,
`loom example example.com/specimen/design`, and `go mod tidy`. Each distinct
parent/current output then runs `go build ./...` and `go vet ./...`.
Two fresh independent candidate processes produce identical bytes, so the
identical repeat does not need another compile/vet.

- HTTP `StreamingResultDSL`: 23 files; only client/client.go,
  client/websocket.go, server/server.go and server/websocket.go under the service
  transport path differ from the parent.
- HTTP `BidirectionalStreamingDSL`: 23 files; the same four transport file roles
  differ from the parent.
- JSON-RPC `ViewedStreamResponseIDDSL`: 22 files; only example CLI/server wiring,
  generated aggregate CLI, and generated client/server top-level files differ.

Every difference is configuration, attachment or related API propagation already
listed in #716 readiness. Service types, views and codecs remain byte-identical.
All current candidate files also match the earlier frozen worktree's selected
candidate output exactly, but all generation, compile and vet checks above were
performed fresh against the new parent/main inputs. Per-case unified diffs and
all file SHA-256 values are retained in the comparison directory.

## Runtime and generated checks

From main:

```sh
LOOM_DIR="$PWD" go test ./http ./jsonrpc -race -count=1
LOOM_DIR="$PWD" go test ./http/codegen ./jsonrpc/codegen \
  -run '^Test(WebSocketMessageLimitGeneratedIntegration|JSONRPCWebSocketMessageLimitGeneratedModule|GeneratedErrorDisclosure|RecursiveProjectionHTTP|RecursiveProjectionJSONRPC)$' \
  -count=1 -v
LOOM_DIR="$PWD" go test ./jsonrpc/codegen \
  -run '^TestGeneratedErrorDisclosure$' -count=1 -v
git diff --check
```

Runtime suites pass. HTTP/JSON-RPC generated limit tests and both recursive
projection orders pass. The initial JSON-RPC error-disclosure run exposed the
new parent fixture still passing a raw nil config. Root changed only that
constructor argument to `loomhttp.WebSocketConfig{}`; its targeted rerun passes.
The other passing checks remain valid. No full codegen suite was repeated.

The sole Go/template/module input delta during validation is that root-owned
fixture migration. Its final SHA-256 is
`8d1edffae24806225a055fc6d1f7992f3f12afa8abef881722ab9dde0e9911ee`.
Initial/final source-input manifests record 2,573 files. Root's independent
skill/readiness/model-README edits are outside production compiler inputs.

## Evidence identities

Machine-readable report: `/tmp/loom-ws-716-integrated-validation.json`, SHA-256
`12d90fe3bbcce0f63c3b53fd488c2845e02199b2db1cc605509dfa9776545178`.
It records hashes for both CLIs, source snapshots, logs and the generated-output
manifest. Generated-output manifest SHA-256:
`016c7cc1eba9bcb16ebcc1670472205fd36504b7dcfd1b747835bf8db9598f64`.
Final source-input manifest SHA-256:
`a37b3db8f7784812b4e7fa648160cd73005a519afdec9db3a72b3b7d4b20ddb3`.

All temporary outputs remain available for independent review. No #722 source
implementation was performed.

## Independent-review correction

The initial integrated review found one medium terminal-cause regression.
After context cancellation closed an active read, a later read returned the
low-level socket error instead of `ErrWebSocketStreamClosed`. Explicit Close
after an earlier nonclosing read failure had the same incorrect precedence.

The shared private `close(readCause)` transition now selects the cause at the
first physical close. Explicit/context closure selects the closed sentinel;
automatic read-failure cleanup retains its cause. Later errors caused by
closing cannot replace that cause. No generated/API output changed. Three
synchronized real-socket cases failed before repair, then passed twenty race
repetitions. Already-canceled preflight and stable-overflow controls pass.
The terminal-cause model's old controls fail in 11 and 7 states; the revised
model passes 16 states. Commands and correspondence are in the model README.

Implementer HTTP/JSON-RPC runtime race and vet checks pass. Root reruns those
packages on the integrated repair. Generated bytes, compilation and independent
repeat evidence above remain applicable because only runtime terminal ownership,
its tests and model documentation changed. The full pinned root golangci run
identified only five older projection/artifact findings, tracked in a separate
CI-maintenance task; no initial #716 lint finding was reported.
