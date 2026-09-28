# OpenAPI security binding allocation

`SecurityBindings` separates component-name allocation from operation emission.
Two bindings of one authored scheme compete for a generated name, and a third
scheme owns that name explicitly. Allocation uses a fixed sorted order; operation
emission explores every order.

Run from this directory with a TLA+ tools jar, one command at a time:

```sh
java -cp /path/to/tla2tools.jar tlc2.TLC -config legacy.cfg SecurityBindings.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config unreserved.cfg SecurityBindings.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config checked.cfg SecurityBindings.tla
```

The legacy model keys components only by authored scheme name. It violates
`BindingCorrect` after 8 distinct states when one location overwrites another.
Allocating variants without reserving authored names still fails that invariant
after 10 distinct states. Both failures and the checked candidate were run before
implementation. The checked model passes binding correctness, unique binding
names, canonical allocation, and termination: 17 generated states, 11 distinct
states.

## Conditional argument and limits

The document first collects distinct binding keys and reserves every authored
name. Sorted allocation keeps a single binding's authored name. Each variant
chooses the first name absent from both the reserved set and previously allocated
names. Induction on that order gives distinct names for distinct bindings.
Operation emission only reads this completed registry, so later endpoints cannot
overwrite the declaration an earlier endpoint references. Fixed sorting makes the
allocation independent of endpoint traversal order.

The finite model abstracts strings and locations to a bounded set. It assumes
correct canonical binding keys and a fixed total sorting order; it does not prove
HTTP header normalization, all possible strings, OAuth semantics, or OpenAPI
validity. Direct tests cover name sanitization collisions, reserved numeric
suffixes, header-case sharing, API defaults, endpoint order, anonymous and AND/OR
requirements, OAuth scope preservation, excluded endpoints, and external URIs.
Rendered specs are checked with libopenapi and Redocly.
