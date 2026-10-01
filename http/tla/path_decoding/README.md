# Path matching and array decoding

This bounded model informed the #514 design. Escaping must survive routing
until the consumer knows which delimiters belong to the transport. The client
escapes each array element and joins them with literal commas. The server
must split those literal separators before decoding each element once.
Decoding the entire capture first loses the distinction between `%2C` and `,`.
Routing also needs registered literals and incoming paths in the same escaped
representation, independently of whether Go retains `URL.RawPath`.

`PathDecoding.tla` operates on token sequences, with explicit encode, join,
split and decode operations. It enumerates 42 arrays of one or two elements
drawn from six values (ordinary text, comma, slash, literal `%2C`, plus, and an
empty element), three route literals, and two raw-path conditions. Percent
escapes are atomic tokens: the model checks operation ordering, not the Go URL
parser, UTF-8 encoding, hex case, arbitrary path grammars or malformed escapes.
It excludes a zero-element array, whose empty representation cannot distinguish
it from one empty element. It assumes that the router preserves raw captures.

Run from this directory with the reusable TLA+ tools JAR:

```sh
java -XX:+UseParallelGC -Xmx512m -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -noGenerateSpecTE -metadir /tmp/loom514-tlc/LegacyArrays \
  -config LegacyArrays.cfg PathDecoding.tla
java -XX:+UseParallelGC -Xmx512m -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -noGenerateSpecTE -metadir /tmp/loom514-tlc/LegacyLiterals \
  -config LegacyLiterals.cfg PathDecoding.tla
java -XX:+UseParallelGC -Xmx512m -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -noGenerateSpecTE -metadir /tmp/loom514-tlc/Decoded \
  -config Decoded.cfg PathDecoding.tla
```

Recorded with Corretto 25.0.4.1 and TLC `2026.10.01.024053` (`0dab95e`). The
official `tlaplus/tlaplus` v1.8.0 release JAR has SHA-256
`1d99ab9ad6cf6fb9839dcc7d4a04fd262e136d452edf2a4928c5c18dd4b3468f`.

| Configuration | Expected and observed result |
| --- | --- |
| `LegacyArrays.cfg` | Exit 12: `ArrayRoundTrip` fails on a single `a,b` element. |
| `LegacyLiterals.cfg` | Exit 12: `LiteralMatches` fails when a literal needing escaping meets a raw-path request. |
| `Decoded.cfg` | Exit 0: both invariants pass over all 252 distinct states. |

These static input checks have no liveness claim. Actual route dispatch,
middleware pattern reporting, scalar/catch-all decoding, and generated client
round trips remain implementation test obligations.

## Contract decision and implementation correspondence

`Muxer.RawVars` is required alongside decoded `Vars`. An optional capability
would allow a mux to satisfy the interface and then fail on an otherwise valid
array endpoint. The required method makes the missing capability a compile-time
error for custom implementations. Existing custom muxes must expose escaped
captures directly; re-escaping decoded values cannot reconstruct delimiters.
The checked Auto-K and drum-meoh consumers use `NewMuxer`, which implements
both methods, so neither needs an adapter change.

| Obligation | Implementation and regression |
| --- | --- |
| Match literals in one representation | `http/mux.go` escapes registered literal fragments and selects an escaped routing path without changing the request URL. `TestVars` and `TestMuxMiddlewareMatchesEscapedPathLikeChi` cover literal spaces/non-ASCII text, slash values, catch-alls and authored metadata. |
| Preserve captures until their consumer can decode | `Muxer.RawVars` retains escaping; `Vars` decodes once. `TestRawVarsPreservesEscapedCapture` checks that the raw and decoded views remain distinct. |
| Preserve routing below a parent mux | `TestMuxMountedUnderChiRouter` checks that canonical routing respects the parent's remaining path. Parent routing context ownership is outside this static model. |
| Split before decoding array elements | `http.DecodePathArray` owns the algorithm. `TestDecodePathArray` and `TestDecodePathArrayRejectsMalformedEscape` cover values and error handling. |
| Use the raw contract in both generated branches | `TestPathArrayDecodeSplitsBeforeUnescapingElements` and `TestMultipartPathArrayDecodeUsesRawCapture` inspect normal and multipart output. |
| Preserve client-to-service values | `TestPathEscapingRoundTripIntegration` compiles and exercises the existing generated client/server fixture with literal, scalar, catch-all and array cases. |

No model claim covers malformed URL syntax or custom mux correctness. Runtime
and generated error-path tests cover failures beyond the model's valid-token
domain. Regeneration is needed to adopt the corrected array decoder.

Raw-capture preservation also requires the correct router context. During
implementation, a parent mount and a child route with identical pattern text
exposed an ownership ambiguity: matching the pattern string did not establish
that the child had routed the request. The runtime records routing depth on
entry and uses a separate local match until child dispatch adds its pattern.
The mounted-router regression covers that case. This ownership boundary is
tested in Go; it is not established by the static token model.

## Bounded generated comparison

The existing path-escaping fixture was compared against parent
`6e52c2dbe56dcf6e826375be90d2683f40ebe682` using Go 1.27.1 on darwin/arm64.
Eleven of twelve generated files are byte-identical. The only changed file is
`gen/http/pathesc/server/encode_decode.go`: its array decoder reads `RawVars`,
calls `DecodePathArray`, wraps malformed escapes in `DecodePayloadError`, and
no longer imports `strings`. Scalar decoding and all client output are unchanged.
Four focused array-decoder goldens separately record the same changes for
string and boolean array paths, including validation and array-only payloads.

The generated candidate passes tidy, vet and all existing fixture tests with
the added slash-in-literal-route and comma/percent/plus array cases. A second
isolated generation matches all twelve generated files and the test harness
byte for byte. Direct multipart rendering and named/transitive-array controls
cover those branches without a separate generated specimen matrix.
