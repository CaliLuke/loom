# Protobuf collection element admission (#740)

Typed object collection elements are non-null. Empty objects and empty
collections are values; an optional object field may be absent. Concrete gRPC
Nullable elements remain rejected by design validation. `Any` retains JSON null.
These rules come from `docs/dsl-reference.md` and `docs/grpc-guide.md`; the model
does not choose them.

The pre-fix conversion had three outcomes for the same nil object element:
inline populated object declarations panic, recursive helpers preserve nil, and
empty declarations manufacture an empty object. Preserving nil is insufficient:
protobuf serialization turns a native nil repeated-message or map-message value
into a present empty message. With no required child fields, validation accepts
that message. Admission must happen before this information loss.

## Candidates and recorded results

The model varies source category, array/map/optional-field position, inline/
recursive/empty declaration, conversion direction and unary/stream/error-detail
boundary. Four configurations compare the following designs:

| Configuration | Result | Meaning |
| --- | --- | --- |
| current | NoPanic fails, 433 states | Inline object conversion dereferences nil |
| guard-only | NoInvalidEmission fails, 865 states | Preserved nil reaches the boundary and can become an empty message |
| blanket-nil | ExactAdmission fails, 901 states | Valid optional absence, Any null and existing collection behavior are rejected |
| owner | All invariants pass, 1296 states | Reject only invalid object element occurrences before conversion and propagation |

Post-wire validation cannot distinguish the guard-only counterexample from an
authored valid empty message, so it cannot recover the lost admission decision.
The model treats native nil inner collections as the existing empty-wrapper
conversion control, not as a new promise to preserve collection presence. That
separate protobuf collection contract is outside #740.

Run sequentially from this directory with Corretto Java 25.0.4.1, TLC 1.7.4
(TLC2 2.19), JAR SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`:

```sh
for config in current guard-only blanket-nil owner; do
  mkdir -p "/tmp/loom-740-java-$config"
  java -Djava.io.tmpdir="/tmp/loom-740-java-$config" -XX:+UseParallelGC \
    -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" tlc2.TLC -workers 1 \
    -metadir "/tmp/loom-740-model-$config" -config "$config.cfg" ElementAdmission.tla
done
```

Negative configurations must exit 12 with their named invariant failure, not a
parse or tool error. Isolate Java temporary directories when running in parallel:
concurrent standard-module extraction produced one tool failure in investigation;
the isolated rerun produced the expected counterexample.

## Implementation correspondence

`elementNeedsAdmission` in `protobuf_element_admission.go` classifies the
physical source before `transformArrayElementAttrs` / `transformMapElementAttrs`
erase a protobuf wrapper. `conversionCanFail` walks source occurrences with their
source context; a target wrapper alone does not make a native scalar collection
fallible. The same predicate feeds `transformElement`, which rejects nil message
elements before field access, helper dispatch or empty-object allocation.
Optional object-field helper reuse retains its existing absence rule.

`InitData.ErrorAware`, helper declarations and calls query their own conversion
graph and context. The [helper fallibility model](../helper_fallibility/README.md)
covers that separate ABI obligation. Existing unary, streaming, CLI and
error-detail consumers propagate the errors; type files and CLI builders supply
the standard error import. Root constructors return a zero result on failure.

`TestNilCollectionConversionReadiness` and `TestNilElementWireReadiness` now
require rejection rather than the original ten panics. They also cover recursive
and empty declarations, anonymous/named wrappers, ordinary nil fields and valid
empty values. `TestElementAdmissionPhysicalCarrier` checks source classification;
`TestElementAdmissionSharedHelpers` checks shared Leaf and internally fallible
Node helpers in both method orders. The mounted transport test proves invalid
requests never invoke the service and invalid result/detail/stream values are not
published, while preceding valid stream items survive. The existing Any raw-error
status diagnostic is an adjacent concern, not the desired nil-error contract.

The model assumes correct classification and propagation; it does not prove
generated Go, arbitrary cyclic Go values, custom codecs, constraints, status
encoding or network delivery. Direct carrier tests cover service-value versus
protobuf-pointer union classification, but generated union collection designs
remain rejected by the existing public contract. Native nil inner collections
retain existing normalization rather than gaining a new presence guarantee.

An adjacent native nil map-value validator defect is recorded separately in the
readiness plan. It does not justify repairing shared validation inside #740 or
relying on post-serialization validation for this conversion boundary.
