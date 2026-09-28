# One wire representation for union examples

`ByteProjection.tla` checks branch selection for a typed byte value in an
untagged union with byte and text branches. The branches have enum constraints
over two abstract strings: the authored text (`raw`) and its distinct base64
encoding (`wire`). Each enum accepts either string or both. There are nine
enum combinations and three pipeline phases.

Run with Java and the TLA+ tools JAR:

```sh
java -cp /path/to/tla2tools.jar tlc2.TLC -config checked.cfg ByteProjection.tla
```

| Configuration | Result |
| --- | --- |
| `legacy.cfg` | Fails: typed bytes fail JSON-shape selection and the fallback emits raw text. |
| `per-branch.cfg` | Fails: interpreting bytes separately for each branch can choose the text branch after an ambiguous intermediate match. |
| `checked.cfg` | Passes: 27 generated and distinct states. |

`PreservesBytes` requires any emitted example to retain the byte encoding.
`PreservesUniqueExample` requires emission when that wire value matches exactly
one branch. The checked design computes one wire representation for every
branch and compares it with the enum representation emitted in the schema.
The typed input remains available for projection and final encoding.

The legacy behavior and the rejected per-branch repair were both reproduced
in Go tests before this model's checked design was implemented. The durable
regressions include byte/text branches with the same authored `Enum("hi")`:
their wire enums are distinct, so `[]byte("hi")` must stay `"aGk="`.

This bounded model assumes correct base64 encoding and enum projection. It
does not prove the Go implementation, arbitrary object graphs, other schema
constraints, empty bytes (whose two strings coincide), or serialization.
Direct tests, round-trip fuzzing, evaluated DSL tests, rendered JSON/YAML
checks, and generation/compile comparisons cover those implementation seams.
