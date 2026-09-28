# Nested union example selections

A synthesized nested union selects an outer and an inner branch. The old
expression generator returns only the innermost value. OpenAPI later tries to
infer branch tags from that value, but two outer branches with the same inner
shape cannot be distinguished. The regression in `nested_union_examples_test.go`
reproduces the missing outer envelope on scalar, collection and map examples.

`UnionSelections.tla` abstracts each example as its selected path of two tags.
It enumerates two occurrences, two outer tags and two inner tags, with both
processing orders. `legacy.cfg` models the loss of the outer selection and
fails `SelectionsPreserved`. `payload-cache.cfg` models a rejected strategy
that caches a complete envelope under the shared inner payload identity; it
fails when `s/Leaf` and `t/Leaf` reuse the same cached envelope.

`checked.cfg` retains the selection at each occurrence and checks preservation
and termination: 100 generated states, 68 distinct states. This is a finite
composition model, not a proof of Go implementation correctness. It does not
model recursive schemas, scalar coercions, validation, random number generation,
JSON/YAML encoding or authored examples. Those require direct and rendered tests.

Run with a local TLA+ tools jar:

```sh
java -cp /path/to/tla2tools.jar tlc2.TLC -config legacy.cfg UnionSelections.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config payload-cache.cfg UnionSelections.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config checked.cfg UnionSelections.tla
```

The first two runs are expected to fail their invariant. The checked run must
finish without an error. `example_generation.go` retains each selected branch
in a private copy, with a separate example memo. The direct regression checks
different outer tags sharing an inner value and protection of the caller's raw
memo; rendered tests cover JSON/YAML and independent generation processes.
