# Additional-member dependency closure

Generated declarations and conversion helpers must cover every reachable type,
including types reachable only through an object's additional-member map.
`DependencyClosure.tla` models a record with a declared member, a map of union
values, two union alternatives, and a recursive edge back to the record.

The old traversal omits the record-to-additional-carrier edge. `Current.cfg`
reproduces the incomplete dependency set: `Complete` fails after three states,
with only the record and its declared member discovered. `Complete.cfg` includes
that edge and checks every visitation order: 28 generated / 17 distinct states,
depth 8, with `TypeOK` and `Complete` passing. Repeated and recursive references
are visited once, so the correction terminates without a recursion-depth limit.

The implementation owners are `appendNestedHelpers` in
`codegen/go_transform_collections.go`, and `collectHTTPUnionTypes` plus
`collectUnionBranchUserTypesSeen` in `http/codegen/service_data_typesupport.go`.
All three must treat additional members as structural children. The shared
conversion collector serves both HTTP and JSON-RPC; fixing emitted callers or
adding declarations in individual endpoints would leave dependency discovery
incomplete.

`TestGeneratedAdditionalComposite` in the HTTP and JSON-RPC generators checks
object values, union values, nested collections, and recursive additional
members without an incidental declared or synthesized field reaching the same
type. It compiles the generated packages, round-trips their values through the
transport, and checks rejected member values do not invoke the service.

This finite model establishes closure and termination for its explicit graph.
It does not prove type identity, naming, codec semantics, or arbitrary graph
construction. Those remain implementation and generated-output obligations.

Run explicitly from the repository root with TLA+ tools 1.7.4 (TLC 2.19,
revision 5a47802; JAR SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`):

```sh
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir /tmp/loom-additional-current \
  -config codegen/tla/additional_dependencies/Current.cfg \
  codegen/tla/additional_dependencies/DependencyClosure.tla
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir /tmp/loom-additional-complete \
  -config codegen/tla/additional_dependencies/Complete.cfg \
  codegen/tla/additional_dependencies/DependencyClosure.tla
```

The current configuration must fail `Complete`, not parsing; the corrected
configuration must pass. Model execution is opt-in and is not a build, test,
commit, CI, or release gate. Remove temporary checker outputs after review.

## Implementation validation

Compared the parent `ba1c91f8bdbdd780f1f2b95860d048849e7f8fb0` plus the new
regression inputs against the corrected generator using Go 1.27.3. Both new
HTTP and JSON-RPC specimens failed to compile before the fix. They now compile
and pass valid round trips and rejected-value checks. The full shared codegen
and HTTP representation packages pass; existing additional-member runtime,
transformation-helper, and JSON-RPC untagged-array controls pass.

The maintained 23-design output comparison plus these two new designs contains
846 parent raw/rendered sections. All 834 unchanged sections match byte for
byte. Intended changes are confined to the new specimens: six generated
encode/decode or CLI files gain the missing object/count/recursive conversion
helpers (12 raw/rendered sections), and four `types_unions.go` files gain the
previously absent union declarations and codecs (eight new sections). No
sections disappear. All 25 designs are byte-stable across five independent
generation processes. The checked-in streaming fixtures have no affected output;
no generated fixture was edited or regenerated for this correction.

Scratch evidence for this run is under
`/tmp/loom-fresh-audit/additional-fix`: `red.log`, `green.log`, `packages.log`,
`controls.log`, `determinism.log`, and `output-comparison.txt`. The complete
before/after output snapshots and `output.diff` remain available for independent
review. They are temporary evidence, not tracked build artifacts.
