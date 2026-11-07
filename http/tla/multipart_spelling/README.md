# Multipart literal field spelling

Multipart named Encoding Object styles use RFC6570 collection framing without
URI percent-encoding. A plus sign or percent escape is literal part data. The
bounded model contrasts a URL decoder reused unchanged with a literal adapter;
comma data in a delimited collection is unspellable and must fail publication.
Repeated parts and scalar parts retain comma data without that restriction.

Run explicitly with the repository's pinned TLC 1.7.4 jar:

```sh
java -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" tlc2.TLC -workers 1 \
  -config http/tla/multipart_spelling/literal.cfg \
  -metadir /tmp/loom-multipart-literal http/tla/multipart_spelling/MultipartSpelling.tla
```

Use `uri.cfg` for the negative control. The literal candidate must preserve all
modeled tokens; the URI candidate must fail `LiteralIdentity`. This finite token
model establishes the adapter choice, not a proof of MIME parsing, arbitrary
strings, schema/typed matching or Go code. The existing untagged-text and closure
models own independent matching and replay; multipart source-budget and body
presence models own wire limits and empty envelopes. Direct literal codec tests
and generated multipart fixtures establish implementation correspondence.

`MetadataPrecedence.tla` models absent, empty and nonempty authored metadata,
zero/one/two file fields, and absent/present file headers. Run it with
`metadata.cfg` (36 states pass) or `metadata_overwrite.cfg` (expected failure of
`Precedence`, 28 states). The explicit target always wins; only one uploaded
file supplies a missing target. `captureMultipartMetadata` in
`expr/http_multipart_contract.go` captures the eligible binding once.
`MultipartMetadataJSON` supplies the same logical value to schema matching and
typed decoding without adding physical part claims. Direct metadata tests and
`TestMultipartDerivedRootAgreement` in the generated fixture cover that mapping,
including explicit empty overrides and duplicate uploaded parts.

```sh
java -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" tlc2.TLC -workers 1 \
  -config http/tla/multipart_spelling/metadata.cfg \
  -metadir /tmp/loom-multipart-metadata http/tla/multipart_spelling/MetadataPrecedence.tla
```

The literal candidate visits 16 states; its URI-decoder negative control violates
`LiteralIdentity`. `DecodeMultipartParam` and `MultipartForm.AddParam` use the
shared style parser with literal spelling. The model does not cover arbitrary
MIME headers, file truncation, source error ownership, Go custom codecs, or
schema membership; runtime source-budget tests and the mounted generated fixture
cover those separate boundaries. These models remain explicit design tools and
are not CI requirements.
