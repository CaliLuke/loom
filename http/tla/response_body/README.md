# Response body ownership (#603)

The shared HTTP runtime owns consumption, optional restoration and cleanup.
Generated HTTP and JSON-RPC decoders supply typed decoding only. A successful
raw/file result transfers its original open body to the caller; every consumed
or failed response closes its original exactly once. Read/decode and cleanup
failures remain independently observable. Restoration must never lose the
original cleanup handle.

The model abstracts decode and close into separate transitions, enumerating
raw/ordinary, success/failure, restoration, read failure and cleanup failure.
The legacy control loses the original close when restoration replaces the body,
and drops cleanup errors. The checked design has one cleanup owner. It checks
96 distinct states. The model does not prove byte buffering, limits, callbacks,
Go defers, network I/O, panics or arbitrary decoder behavior; generated and
runtime tests establish those implementation seams. Raw decoder callbacks must
not consume the body on success. The model records no claim about callbacks
that violate that contract.

## Reproduce

Use official TLA+ tools v1.7.4 (TLC 2.19 revision `5a47802`), JAR SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
Runs used Corretto 25.0.4.1. From this directory:

```sh
for cfg in LegacyOwnership LegacyFailures Checked; do
  java -XX:+UseParallelGC -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 \
    -metadir "/tmp/loom-response-$cfg" -config "$cfg.cfg" ResponseBody.tla
done
```

`LegacyOwnership` fails `Ownership`, and `LegacyFailures` fails `Failures`,
both with exit 12. `Checked` passes both invariants with exit 0. Tool or parse
failures do not satisfy a negative control. Remove checker output after review.

## Correspondence

- `http/response_body.go`: `DecodeResponse` captures the original closer,
  restores bounded captured bytes, and composes errors with `errors.Join`.
  Successful raw/file metadata decoding transfers ownership; failed metadata
  decoding closes the original. Oversized bodies are rejected without replay.
- HTTP and JSON-RPC response renderers call that operation around their typed
  decode logic. HTTP raw endpoint code no longer closes a decoder-owned failure
  a second time. JSON-RPC unexpected-status reads propagate errors and use the
  shared diagnostic bound. Notification responses use the same operation.
- HTTP SSE handshakes use the same owner with `streamBody=true`: rejected
  status or content type closes once and retains cleanup errors; successful
  validation transfers the open body to the stream. Non-200 declared-error
  responses remain exclusively owned by their typed response decoder.
  `TestGeneratedSSEResponseOwnership` in the HTTP ticktock fixture reproduces
  the former ignored-close failures and checks rejection, typed errors,
  accepted empty content type, and successful transfer, with and without close
  failures (#632). `LegacyFailures` still fails and `Checked` passes 96 states;
  the model abstracts handshake validation as metadata decoding.
- `http.TestDecodeResponseOwnership` checks metadata failure, transfer,
  restoration and error composition. The existing response-limit test checks
  bounded reading. `jsonrpc.TestNotificationResponseCleanupErrors` checks
  notification partial replay and independent read/close causes.
- `codegen/generator.TestResponseBodyLifecycle` compiles/vets generated HTTP,
  JSON-RPC, raw and file clients. Its runtime matrix covers success, read,
  decode and close failures, combined errors, unexpected status, restoration
  enabled/disabled, exact close counts and original body identity on transfer.

The failing-first generated matrix reproduced ignored close failures and
leaked originals before implementation. The runtime operation removes both
local restoration implementations and the endpoint's redundant close.
