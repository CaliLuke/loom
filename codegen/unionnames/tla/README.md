# Union member namespace

`../names.go` owns union storage and accessor names for service, view, HTTP,
JSON-RPC and gRPC consumers. The original direct Go identifier conversion can
produce a storage field `Kind` beside method `Kind()`, or duplicate accessors
for logical names such as `a-b` and `a_b`. Reserving only fixed methods does not
handle `AsText` beside branch `Text`, or a pre-existing suffixed sibling.

The allocator first sorts logical branch names and allocates unique accessor
suffixes while reserving all authored candidates. It then reserves the complete
fixed and per-branch method namespace before allocating storage fields. These
two namespaces are captured separately; renderers must not reconstruct either
name from the authored spelling. Wire tags remain authored values. Normal
members remain unchanged; conflicting public storage fields and duplicate
accessor suffixes gain numeric suffixes. Consumers must regenerate.

`MemberNames.tla` abstracts Go identifier conversion into an ordered sequence of
candidate names. Four bounded cases cover fixed methods, generated accessors,
sanitized duplicates and authored suffixes, plus an ordinary control. Its
invariants require unique method suffixes, unique fields, disjoint field/method
namespaces and unchanged ordinary names. Fresh suffix search is bounded at 20;
the implementation is unbounded. Sorting and Go identifier conversion are
covered by `TestResolve`, including reversed declaration order. The model does
not establish package-level type names, protobuf field names or arbitrary
Unicode conversion correctness.

Run explicitly from this directory with TLA+ 1.7.4:

```sh
java -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -config old.cfg MemberNames.tla
java -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -config old-sanitized.cfg MemberNames.tla
java -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -config allocated.cfg MemberNames.tla
```

Both old configurations fail their intended invariant. The allocated policy
passes 8 generated / 4 distinct states. No proof tool is required by normal
builds or CI. Service/view direct tests and generated HTTP/gRPC tests exercise
the implementation and its cross-consumer correspondence.
