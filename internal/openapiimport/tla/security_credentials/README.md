# Imported credential allocation

This model covers the source-adapter step before the existing effective HTTP
wire-binding owner. It does not replace `expr/tla/http_wire_bindings`: that owner
still decides whether different wire operations may coexist. Here, equivalent
source credential representations must receive one payload source, all semantic
scheme roles must survive, and grouping must not turn AND into OR.

`Inputs` contains seven representative credential occurrences. A physical key
contains location, canonical wire name, codec and component position. Basic
username/password are separate positions of one composite encoding; they cannot
be merged. Header normalization is abstracted into canonical names. Go tests
cover case-insensitive headers and case-sensitive query/cookie names.

The transition corresponds to the importer credential-plan builder:
`securityParameters` previously allocated by scheme identity; it now groups
before assigning payload field names in `security_credentials.go`. Payload role tags and explicit
HTTP mappings consume that same plan. Security requirements remain the original
list of conjunctions; they are never deduplicated by physical source identity.
`checks` abstracts arbitrary authentication success/failure for the two roles.

## Reproduce

Use Java 25 and the existing TLA+ 1.7.4 jar (reports TLC 2.19), SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
Run from this directory, substituting each configuration below:

```sh
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config checked.cfg -metadir /tmp/loom-security-credentials-checked Credentials.tla
```

| Config | Expected result | Readiness result |
| --- | --- | --- |
| scheme | Current allocation fails same-representation invariant | 369 distinct states, counterexample |
| slot | Merging solely by wire slot loses codec/component distinction | 377 distinct states, counterexample |
| codec | Merging solely by encoding loses physical location/name | 617 distinct states, counterexample |
| drop-role | Reusing a source but retaining just one auth role loses checks | 369 distinct states, counterexample |
| flatten | Grouping must not turn conjunction into alternatives | 374 distinct states, counterexample |
| checked | All three invariants | 728 distinct states, passed |

All runs have 364 initial states. Final states stutter explicitly; this is a
finite normalization model, not a liveness claim. The first checked run stopped
at a terminal-state deadlock; adding the intended terminal stutter leaves the
invariants unchanged and the final rerun above passes.

## Correspondence and limits

`security_credentials_readiness_test.go` first reproduced eight over-allocation cases and now verifies their repairs,
keeps distinct slots/codecs and operation overrides as controls, and checks a
hand-authored candidate representation through full generated compilation,
HTTP authentication and OpenAPI export. The candidate fixture contains one
attribute with both token/access-token tags and one API-key attribute tagged for
two schemes. It demonstrates feasibility; it is not importer implementation.
`security_credentials_import_test.go` additionally imports the actual JSON
fixture, generates/builds/vets the whole module and exercises its clients,
mounted auth hooks, requirement export and lossless scope diagnostics.

Partial operation projection is also outside the model: `documentForOperations`
retains root requirements only if a retained operation inherits them.
`security_credentials_partial_test.go` checks unsupported roots with public and
scoped secured overrides, blocked inheritance, strict rejection, the empty
subset and actual generated authentication. Valid inherited root requirements
remain covered by the existing partial-import tests.

Scopes are intentionally outside this physical grouping model. The existing
Loom requirement has one scope list applied to every scheme; OpenAPI carries one
list per scheme. A generated counterexample records that disagreement and direct
import tests demand explicit unsupported diagnostics for differing per-scheme
lists. Do not infer scope unions, scope intersections or a public policy from
this model. Header parsing, presence/default semantics, generated identifiers,
custom mappings, arbitrary graph size, repeated query values and malformed
credentials remain concrete-test obligations. Raw Authorization conflicts and
Basic/Bearer conjunctions remain under the existing expr admission model.

Keep source/configs and this summary. Checker state and raw logs stay outside
the repository and are removed after review and delivery.
