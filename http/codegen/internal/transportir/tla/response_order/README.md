# HTTP response selection order

`dsl.Tag` selects the first matching authored tag. The default response is
selected only when no tag matches, regardless of where it was declared.
Expression validation owns the exactly-one-default requirement (#604).

The model contains one default (ID 0) and three tagged responses. It explores
all 24 declaration orders and all eight matching-tag subsets. Tag evaluation
is abstracted to a set of matching IDs; it assumes valid expressions, distinct
response identities, and one untagged default. It does not model serialization,
headers, cookies, generated names, or how Go evaluates tag comparisons.

`legacy.cfg` models the old swap of the default and last response. TLC violates
`SelectsFirstMatch`: authored `<<0, 1, 2, 3>>`, matching tags `{1, 3}`, becomes
`<<3, 1, 2, 0>>`, selecting 3 instead of 1. This is the cause of #605.

`checked.cfg` models a stable partition: retain tagged response order and append
the default. TLC checks priority, first-match selection, response preservation,
and default-last across 576 generated / 384 distinct states. This supports the
partition design; it is a bounded model check, not a proof of arbitrary generated
Go or of extraction from an application design.

## Implementation correspondence

- `request_response.go` / `buildResponse` owns the selection order in the shared
  transport IR without mutating the authored expression.
- `http/codegen/service_data_response.go` consumes that order without swapping.
- `TestResponseSelectionOrder` checks IR order, attached body/header mappings and
  expression non-mutation for default-first, middle and last declarations.
- `TestResponseSelectionEncoders` covers rendered encoder order.
- `TestResponseSelectionGenerated` compiles the generated server and client and
  checks overlapping, individual and absent matches, statuses, headers, bodies
  and client decoding. These tests cover two tags; the model additionally checks
  three-tag cases where a middle default could reorder later matching tags.

## Reproduce

Use the TLA+ 1.7.4 release's `tla2tools.jar` (SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`).
From this directory, with `TLA_JAR` pointing to that installed jar:

```sh
java -cp "$TLA_JAR" tlc2.TLC -config legacy.cfg -metadir /tmp/loom-response-legacy ResponseOrder.tla
java -cp "$TLA_JAR" tlc2.TLC -config checked.cfg -metadir /tmp/loom-response-checked ResponseOrder.tla
```

The legacy run must fail `SelectsFirstMatch`; the checked run must pass. A tool
or parsing failure is not the expected counterexample. Keep checker state and
trace output outside the repository and remove it after review.
