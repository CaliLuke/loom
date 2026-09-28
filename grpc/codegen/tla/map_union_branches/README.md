# Named map union branches

A protocol buffer oneof cannot contain a map field directly. Loom already
represents each named map with a message containing a map `field`. This model
checks acceptance and preservation of the selected union branch when that
message is used inside a oneof, including nil and empty maps.

- `legacy.cfg` reproduces the validator rejecting named map branches.
- `drop-empty.cfg` models a candidate that omits the wrapper for nil or empty
  maps. It loses the selected branch after encoding.
- `checked.cfg` keeps the wrapper whenever the map branch is selected. It
  passes 45 generated states and 36 distinct states, preserving the branch
  and entries and reaching completion.

Run each configuration with a TLA+ tools jar:

```sh
java -cp /path/to/tla2tools.jar tlc2.TLC -config legacy.cfg -metadir /tmp/loom-map-branches-legacy MapBranches.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config drop-empty.cfg -metadir /tmp/loom-map-branches-empty MapBranches.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config checked.cfg -metadir /tmp/loom-map-branches-checked MapBranches.tla
```

The first two runs must fail `NamedMapSupported` and
`SelectedBranchPreserved`, respectively. The checked run must pass.

The finite model abstracts protobuf field numbering, key types, map entries,
Go rendering and malformed messages. It explicitly permits protobuf to
normalize nil maps to empty maps; branch selection must still survive.
Direct validation tests cover permitted and rejected map shapes. Generated
fixture tests check declarations, compile the proto and Go output, and test
conversions through actual protobuf serialization.
