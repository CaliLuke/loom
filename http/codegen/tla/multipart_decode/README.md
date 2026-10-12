# Multipart decoder ownership

`DecodeOwnership.tla` explores wrapper-local accumulator and return-signature
ownership alongside the existing body, bound-element and Basic validation order.
It does not model MIME parsing, resources, auth callbacks or Go expression types.

The ordinary decoder returns payload plus error. The custom multipart adapter
returns only error; its body callback runs before bound elements and the outer
Basic phase. A failing phase must prevent service invocation. The adapter owns
its own accumulator when elements can accumulate errors, even though its outer
request decoder deliberately excludes those elements from its accumulator need.

Run from the repository root with Java 25 and the pinned `tla2tools-1.7.4.jar`
(SHA256 `936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`,
reporting TLC 2.19):

```sh
java -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC \
  -config checked.cfg -metadir /tmp/loom-multipart-checked \
  http/codegen/tla/multipart_decode/DecodeOwnership.tla
```

Replace `checked.cfg` with `legacy.cfg` or `declaration-only.cfg` for expected
counterexamples. Legacy fails compilation agreement in 65 states: the custom
adapter uses a missing accumulator. Declaring the accumulator alone fails in 73
states: its stale cookie branch returns a payload from an error-only function.
The shared-partials candidate passes compilation agreement and service-barrier
properties across 152 states. These results are bounded design checks, not a Go
compiler proof. The static `Emit` facts explicitly summarize the observed source
branches; generated compilation tests establish their correspondence.

Implementation seams: `request_decode_plan.go` owns element and enclosing-decoder
facts; `server.go` supplies each wrapper's return signature; ordinary and custom
multipart wrappers consume the same element/conversion partials. Common partials
must not declare a second accumulator or choose an enclosing function signature.

Regression correspondence: `TestCustomMultipartBindingAccumulator` and
`TestCustomMultipartBindingsGenerated` reproduce missing declarations for path,
query, header and cookie bindings, including unconstrained string paths that
still emit a barrier. The cookie compile diagnostic additionally proves the
stale return signature. No-binding and existing generated multipart tests are
controls. Production acceptance adds callback errors, malformed binding input,
constraints, Basic/Bearer framing and service-invocation assertions to check the
phase abstraction. The model treats these individual failures as booleans and
cannot prove their concrete wire status, code or detail.

Keep model configurations and this correspondence record; remove checker states,
traces and task logs after final independent review, preserving concise evidence.
