# gRPC binary metadata — #705

Readiness investigation against `555d13306`; no production changes yet.

1. [x] Trace metadata extraction, encoder/decoder consumers and response required-key contracts.
2. [x] Reproduce arbitrary-byte failures separately for request metadata, headers and trailers; retain explicit `-bin` and printable-byte controls.
3. [x] User accepted automatic standard `-bin` wire-key derivation for `Bytes`; preserve already suffixed keys.
4. [ ] Establish one evaluated binding owner before validation/codegen, including alias/array shape, mapped names, collision admission and required-key consistency. Inspect actual consumers and reject per-renderer spelling rules.
5. [ ] Independently review readiness and bounded acceptance, then implement the accepted owner.
6. [ ] Compare bounded parent/candidate/repeat generation, compile and exercise mounted binary/ordinary/absence/default/required and independently implemented peer cases.
7. [ ] Independently review the exact final diff, commit, push and close.

The protocol defines binary metadata through `-bin` names and base64 wire values;
standard runtimes encode/decode those values. Source:
https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-HTTP2.md . Plain keys accept
printable ASCII only. This is a cross-language protocol rule, not a Go-only codec.

`extractMetadata` currently passes the mapped element name unchanged into
`MetadataData.Name`. Both encoding and decoding use it. Response contract analysis
separately reads the evaluated mapped attributes, so changing the codegen name
alone would leave required-header/trailer checks inconsistent. Derived-name
collisions must be validated before generation rather than silently combining
unrelated fields. Existing absence/default and required admission rules remain.

Accepted contract: derive `x-raw-bin` from `Bytes` mapped to `x-raw`, and preserve
explicitly suffixed keys. Generated clients and servers carry arbitrary bytes
through the standard cross-language metadata protocol. Handwritten peers must use
the resulting wire key. Rejecting every unsuffixed binary mapping was considered
and not selected; it would require extra authoring without improving the wire
contract. Do not add a private encoding under ordinary keys. Reject collisions
between distinct fields after derivation before generating code.


## Original witness and controls

`TestBinaryMetadataRoundTrip` generates six bounded modules: request, header and
trailer placements, each using a plain key or an explicitly suffixed key. On the
published owner, all three plain-key cases fail for `00ff01` and a newline-bearing
value, while printable `hello` succeeds. All three explicit `x-raw-bin` cases pass
under the race detector. Generation and module compilation succeed for every case;
failures occur at the actual gRPC transport boundary. Combined generation/runtime
run: 12.927s. `/tmp/loom-705-parent.log` retains exact diagnostics and
`/tmp/loom-705-parent` retains generated input/output for readiness review.

The maintained #690 metadata-default model explicitly excludes binary-key framing.
Its defaults and required-before-default semantics are useful controls, not proof
of this repair. Bounded consumer inspection found gRPC registry designs using
Bytes in message fields rather than metadata, and the quality fixture uses text
metadata. No broad consumer-compatibility claim follows from that sample.

## Binding design model

`expr/tla/grpc_metadata_bindings` models current raw names, an encoder-only patch,
unchecked suffix derivation and the two candidate public policies. Current fails
binary framing in 58 distinct states; encoder-only fails consumer agreement in
58; unchecked derivation merges two fields in 71. Both a shared derived binding
with collision admission and an explicit-suffix admission owner pass (168
states generated/112 distinct each). This establishes why binding selection must
precede all consumers; it does not select the public policy.
