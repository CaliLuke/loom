# Retained source type and normalized value domains

`SourceType.tla` models a source assertion over a normalized non-null value
domain and an independent exact-one null count. The domain abstracts the result
of normalization and effective constraints, including finite enum refinement.
Source absence is not an assertion. Concrete source facts must contain the
normalized domain before the exporter can restore a redundant type assertion.

Run explicitly with Java and TLA+ 1.7.4; these are design tools, not CI gates:

```sh
java -cp /path/to/tla2tools.jar tlc2.TLC -config unchecked.cfg SourceType.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config unchecked-domain.cfg SourceType.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config checked.cfg SourceType.tla
```

The unchecked candidate admits incompatible retained types and loses null when
restoring a non-null type. The unchecked configurations independently reproduce the domain and null invariant failures. The checked
candidate rejects incompatible facts and retains null only for exactly one
accepting branch: 72 generated states, 36 distinct, no invariant failures.

`validateHTTPMediaSourceTypes` calls `http_source_type.go` before either form or
multipart capture. Concrete normalized kinds and every untagged branch establish
compatibility directly. Where shape alone cannot establish it, the existing
value owner supplies effective finite enum candidates and their JSON projection;
exact numeric bounds also admit an integer point or empty domain. Absence and
null are separate. Unprovable opaque or unconstrained kinds cannot justify a
retained concrete source assertion. This is internal provenance validation, not
a new type-implication API or alternative schema matcher.

`TestHTTPSourceTypeCompatibility` checks scalar/array unions, incompatible
assertions, integer/number refinement, finite enums and null/absence controls.
`TestEncodingSourceTypePreservesNullMultiplicity` checks exporter correspondence:
original union branches remain intact, and the restored assertion does not add
null when two branches already accept it. The model abstracts exact constraint
solving, JSON projection, alias graphs and byte representations; normal tests
exercise those owners independently. See the OpenAPI field-encoding model for
application and cross-branch Encoding Object reconciliation.
