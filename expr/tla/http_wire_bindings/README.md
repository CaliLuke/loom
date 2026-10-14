# Effective HTTP wire bindings

This model explores admission of two operations that may address one physical
HTTP slot. It informed the shared binding plan in `expr/http_wire_bindings.go`.

A binding retains its slot, ordered source tuple, codec, explicit/implicit origin
and membership in a common AND requirement. A Basic tuple has username and
password positions; a raw or Bearer tuple has one credential source. All Basic
schemes in one method read the same DSL username/password tags. A field shared
by independent operations is identified by source identity, not by equal string
values. Header case folding and location scoping are abstracted into slot IDs.

Accepted pairs either use different slots, are the same operation (same ordered
source tuple and codec), or are disjoint Basic/Bearer alternatives with disjoint
source fields. Distinct raw values and independent Bearer credentials cannot be
inferred from one header. An AND requirement cannot send two independent values
in one slot, even when their frame families differ. Exact reuse of one source
and codec may serve multiple scheme checks, including an AND requirement.

The candidate deliberately does not reject or reinterpret required/defaulted
credentials. The original readiness hypothesis did that; generated evidence
showed both useful required-credential controls and the already-known
always-send default interaction. Requiredness remains the #732 payload contract;
default/presence handling remains M7. This model also does not claim that two
simultaneously populated alternative credentials can both survive one slot.

## Reproduce

Use the existing pinned TLA+ tools JAR at release 1.7.4, SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
This artifact reports TLC 2.19 (`5a47802`). It does not support the newer
`-noGenerateSpecTE` argument. Run from this directory:

```sh
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config legacy.cfg -metadir /tmp/loom-wire-legacy WireBindings.tla
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config blanket.cfg -metadir /tmp/loom-wire-blanket WireBindings.tla
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config explicit-only.cfg -metadir /tmp/loom-wire-explicit WireBindings.tla
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config checked.cfg -metadir /tmp/loom-wire-checked WireBindings.tla
```

- `legacy` accepts independent colliding inputs and must fail
  `NoAmbiguousDecode` or `NoConjunctiveOverwrite`.
- `blanket` rejects every distinct operation sharing a slot and must fail
  `PreserveDisjointAlternatives` (the already-supported Basic-or-Bearer contract).
- `explicit-only` validates only authored mappings and must fail a safety
  invariant when an inferred binding also occupies the slot. This rejects the
  tempting incomplete repair that validates declarations before auth insertion.
- `checked` must pass all three invariants.

The readiness run at parent `8c7d8d711b9e2fe54fbff1dabb9b49fd361ed170`
with Java 25 produced 3,648 initial states. `checked` visited 7,296 states and
passed. `legacy` failed `NoAmbiguousDecode` at 3,657 states, `blanket` failed
`PreserveDisjointAlternatives` at 3,793, and `explicit-only` failed
`NoAmbiguousDecode` at 3,659. The state counts are finite exploration evidence,
not a proof over arbitrary designs.

## Correspondence and limits

`expr/http_wire_bindings_test.go` retains direct rejected-design
witnesses for explicit, inherited and inferred collisions, header case folding,
path/query/cookie slots, success/error headers and session cookies. Its positive
controls cover remapping precedence, case-sensitive query/cookie slots, separate
locations, shared source/framing and Basic-or-Bearer alternatives.
`http/codegen/wire_bindings_test.go` compiles valid generated transports and checks retained values, correct auth
selection and absence of credential contamination. Its parent readiness probe
first reproduced value loss and auth failures; invalid designs now stop before
generation. The JSON-RPC companion checks the same shared contract.
`expr/http_wire_bindings_internal_test.go` checks detached plan ownership,
requirement coalescing and error-header phase agreement. `ensureWireBindings`
owns forward mapping enumeration and selected body exclusions;
`disjointAlternative` implements the modeled admission predicate. Response/error
occurrences use the same occupancy validator after implicit header selection.
The minimal `shared_source_probe.go.txt` can be copied temporarily to
`http/codegen/wire_binding_shared_source_probe_test.go`, then run with
`LOOM_DIR="$PWD" go test ./http/codegen -run TestWireBindingSharedBasicSourceProbe`.
It demonstrates that reusing a source field as both Basic username and Bearer
credential reaches incompatible consumers, including duplicate CLI locals. The
repair must reject incompatible source/codec operations at their owner; renaming
those locals does not resolve the representation conflict.

The finite model has two slots and three source fields. Its pairwise predicate
is intended to be applied to every pair in a real occupancy group. It does not
prove extraction, HTTP parsing, wire-name validity, source provenance, inheritance,
body inference, validation phase ordering, immutable ownership, arbitrary graph
size, named Go types, generated compilation or OpenAPI fidelity. Ordinary fields
are represented as raw operations that coexist, not as auth alternatives. Basic
component aliasing inside one composite is outside the model. Concrete Go tests
must establish these boundaries, including before/after finalization agreement.
Do not claim the model establishes product intent or expands supported codecs.

Keep these sources, configs, instructions and the minimal reproduction. Remove
only task-owned checker state, traces and temporary probe copies after review,
when no active process needs them.


## Separate selected-error-body reproduction

`selected_error_body_probe.go.txt` retains an unrelated, parent-confirmed codegen
failure excluded from this repair's acceptance. Copy it temporarily to
`http/codegen/wire_binding_selected_error_probe_test.go` and run:

```sh
LOOM_DIR="$PWD" go test ./http/codegen -run '^TestWireBindingSelectedErrorBodyProbe$' -count=1
```

The pushed parent and candidate both fail generated compilation: client
`body.Code` / `body.Detail` selectors target a string and locals are redeclared;
the server references nonexistent `(*loom.ServiceError).Detail`. The larger
parent/candidate response-body probe also produced byte-identical artifacts.
Do not mark this defect fixed when closing the mapping reports. Remove temporary
probe copies after running; root owns its separate backlog reconciliation.
