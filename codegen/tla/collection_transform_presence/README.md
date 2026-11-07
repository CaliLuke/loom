# Collection conversion presence

`CollectionTransform.tla` explores the shared collection conversion owner in
`codegen/go_transform_collections.go`. Source and destination layouts independently
choose JSON element presence; aliases own their nested layout. Container absence,
empty containers, cardinality, and nested element storage must survive conversion.

Run explicitly with TLC 1.7.4 (no default build or CI dependency):

```sh
java -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -config owned.cfg CollectionTransform.tla
```

Repeat with `source-only.cfg`, `outer-only.cfg`, `alias-cast.cfg`, and
`drop-slot.cfg` for negative controls. The original source-only renderer fails
`TargetCarrier`; wrapping only the outer slot fails `NestedCarrier`; blindly
casting an alias fails `TargetCarrier`; skipping an absent map value fails
`ContainerPreserved`. The corrected policy derives both levels from the target
layout and retains map keys even for absent source wrappers. A native destination
cannot express element absence, so that invalid slot becomes its zero value while
membership remains intact. A JSON destination retains the absent wrapper.

The model abstracts concrete values, nullable representation, conversion errors,
and generated Go syntax. `TestCollectionTransformsPreserveTargetPresence` compiles
and executes both directions for arrays/maps of scalar, object, union, nested,
aliased, and nullable values, including nil/empty and absent-wrapper cases. Its
named-layout cases verify that a JSON alias inside a native outer context still
owns its element carriers. Required-array empty materialization uses the same whole-collection type owner;
`TestRequiredCollectionTargetLayout` compiles this path with a JSON alias inside
a native parent. Optional object fields are converted through the same
presence assignment owner in `go_transform_presence.go`.

The checked policy explored 768 states (512 distinct); the four negative controls
failed after 329, 361, 345, and 396 states respectively. These bounded checks inform
the design; implementation tests remain the ordinary regression gate.
