# Basic authentication presence

A Basic header has two string components and one presence bit. The service
payload can also represent each optional component as absent. Encoding sends
when either component is required or present and converts an absent optional
counterpart to the empty component. Decoding a valid header supplies both
components; a missing or invalid header leaves optional fields absent and fails
when either component is required. This is protocol projection, not fallback
authentication.

`BasicPresence.tla` explores all four requiredness combinations, absent/empty/
nonempty component values, and encoded/missing/invalid headers. Required fields
cannot be absent. The model abstracts base64 parsing to header validity; Go's
`Request.BasicAuth` and generated regression tests check the concrete parser.
Scalar fields with defaults are represented as present values; the model does
not prove default initialization, named Go types, code generation, or callbacks.

- `legacy-encode.cfg` reproduces #600: optional absent user plus present empty
  password suppresses the header and violates `PreservesPresence`.
- `legacy-decode.cfg` reproduces #600: an absent header fabricates two present
  empty fields and violates `AbsentStaysAbsent`.
- `checked.cfg` verifies presence, rejection and round-trip projection across
  300 generated / 225 distinct states.

`basic_auth.go` owns the analyzed plan consumed by the request encoder/decoder
in `template_sources_request.go` and `template_sources_request_decoder.go`.
`TestBasicPresencePlan` checks analyzed requiredness, and
`TestBasicPresenceGenerated` compiles generated clients/servers and checks all
four combinations, missing/empty/nonempty components, and malformed headers.
Its named-string fixture also checks conversion independently of presence.
Existing credential tests cover plain String and all-present controls.

Use TLA+ 1.7.4 `tla2tools.jar` (SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`).
From this directory:

```sh
java -cp "$TLA_JAR" tlc2.TLC -config legacy-encode.cfg -metadir /tmp/loom-basic-old-encode BasicPresence.tla
java -cp "$TLA_JAR" tlc2.TLC -config legacy-decode.cfg -metadir /tmp/loom-basic-old-decode BasicPresence.tla
java -cp "$TLA_JAR" tlc2.TLC -config checked.cfg -metadir /tmp/loom-basic-checked BasicPresence.tla
```

The legacy runs must fail their named invariants; parsing/tool failures do not
count as reproductions. The checked run must pass. Keep generated checker state
and traces outside this directory and remove them after review.
