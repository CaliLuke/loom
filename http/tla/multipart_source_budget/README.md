# Multipart source budget

Design model for #759: one request's parser consumes a prefix, reports
successful decoding, and the service may dispatch. The parser's final boundary
or an early custom callback is not necessarily source EOF. Four specimens cover
an exact request, header/empty-part overhead, trailing epilogue, and data with a
terminal source error hidden by parser buffering. Size units abstract raw bytes;
no decoded value contributes to these specimens' old value-only budget.

`values` models current aggregate-value admission and must violate Admission.
`wire` installs the correct raw bound but omits successful completion; epilogue
and buffered source errors still violate Admission. `owned` requires completion
within that same bound and preservation of observed source errors before dispatch.
ExactAccepted also prevents an implementation that simply rejects everything.

Run with Java 25 and TLC 1.7.4 (jar SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`):

```sh
java -cp /Users/luca/.local/share/tlaplus/tla2tools-1.7.4.jar tlc2.TLC \
  -workers 1 -metadir /tmp/loom-759-model-owned \
  -config owned.cfg MultipartBudget.tla
```

Substitute `values` or `wire` for the negative variants. Expected exit12 with
Admission counterexample for both negatives; exit0 for owned. Deadlock checking
remains enabled. Results: values fails at 14 distinct states, wire fails at 15;
owned passes 20 generated/16 distinct states. This bounded model informs the runtime operation's completion
ownership. It is not a proof of multipart syntax or arbitrary user callbacks.

Correspondence: `DecodeMultipartRequestWithLimit` establishes the raw budget
before MultipartReader, observes `requestBodyLimitReadCloser.readErr`, and
completes the raw stream before returning success to both built-in and custom
generated decoders. `readSource` records the first error without changing Read
retry semantics, covered by the shared-wrapper regression. Direct multipart source tests cover
actual overhead, byte-exact limits, read-ahead/error behavior and Close ownership;
existing mounted request-limit/custom-binding harnesses cover 413 and dispatch.
Callback failures never dispatch and do not need extra draining, so they are
outside this success-path model and remain covered by existing tests. Request
cancellation/deadlines, MIME syntax, allocation multiplicative constants and
arbitrary user codec memory use are outside its assumptions.
