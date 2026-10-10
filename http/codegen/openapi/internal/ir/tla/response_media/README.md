# Responses sharing an HTTP status

OpenAPI has one response per status, while a Loom endpoint can return several
errors or tagged successes with that status. The old assignment replaced each
previous response, losing reachable media types and schemas (#642).

`ResponseMedia.tla` explores all six orders of three alternatives: two JSON
schemas and one HTML schema, with and without a conflicting shared metadata
entry. `legacy.cfg` reproduces loss of alternatives. `checked.cfg` accumulates
schema sets and rejects conflicting metadata; it passes 60 generated / 48
distinct states. Equality with the expected schema sets also checks order
independence within this bound.

The chosen projection uses `anyOf`, because alternatives can overlap. `oneOf`
would incorrectly reject values accepted by multiple branches. Media types
remain separate. Non-schema metadata conflicts produce a diagnostic instead
of silently dropping authored intent. Header schemas use the same union;
requiredness is the intersection of all alternatives, including absent headers.

## Correspondence and limits

- `document.go` collects success and error alternatives before status projection.
- `response_media.go` merges schemas, descriptions, examples and headers without
  mutating the transport responses, and rejects incompatible metadata.
- `TestResponseMediaDeclarationOrder` checks whole-document order independence.
- `TestMergeResponseAlternatives` checks schema unions, header case folding,
  requiredness, example retention, non-mutation and metadata diagnostics.
- `TestRenderedErrorMedia` parses both formats with libopenapi and compares bytes
  across independent processes. The fixture also passes Redocly.
- `TestErrorMediaGenerated` checks server/client round trips and each generated
  response contract for both declaration orders.

The model abstracts schema equivalence and metadata compatibility. It does not
prove schema extraction, `anyOf` serialization, example naming, header semantics,
or runtime dispatch. Those are Go test obligations. OpenAPI cannot express
correlations between a particular body alternative and a particular header;
individual generated response contract cases retain that association.

## Run

Use TLA+ 1.7.4's `tla2tools.jar`, SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`, and Java 25.
From this directory, point `TLA_JAR` to the installed jar:

```sh
java -cp "$TLA_JAR" tlc2.TLC -config legacy.cfg -metadir /tmp/loom-response-media-legacy ResponseMedia.tla
java -cp "$TLA_JAR" tlc2.TLC -config checked.cfg -metadir /tmp/loom-response-media-checked ResponseMedia.tla
```

The legacy run must report an invariant violation; a parser or tool error does
not reproduce the defect. The checked run must pass. Remove temporary checker
state and traces after validation and review.
