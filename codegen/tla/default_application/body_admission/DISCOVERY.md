# Discovery and controlled conversion copies

`TransformDiscovery.tla` explores the bounded identity boundary between conversion discovery and helper emission. It complements the unchanged Body-admission model; it does not model default evaluation, decoding, nullable storage, Go variable allocation or recursive value serialization.

Four canonical conversion nodes represent a root, two distinct selected occurrences A/B, and an unchanged sibling. A/B have the same physical helper spelling but different selected defaults (9/11). A has a recursive self-edge; the root reaches both and the unchanged sibling. B is locally fallible. Discovery computes source-effect and failure closure before helper allocation. Two authored traversal orders are checked separately: helper spelling is deterministic for each fixed order, not invariant under changing authored order.

Canonical occurrence tokens abstract source/target type identity, references and physical layout within one plan. They are not Go names or serialized values. Copies carry an explicit owning occurrence and physical key. This is one graph's correspondence, not another default-selection owner. Canonical nodes remain the fixed-point enumeration; concrete aliases do not add semantic nodes.

## Observed defects and modeled alternatives

The actual `TestAdmittedSourceDefaultHelperClosure` and `TestAdmittedSourceDefaultFailureClosure` failures showed mapped traversal's child copies missing the nodes discovered from original children. The retained `TestAdmittedSourceDefaultInheritedNullableHelper` failure independently showed a nullable concrete-root copy losing the admitted helper identity. `TestAdmittedSourceDefaultOccurrenceIdentity` also checks distinct selected defaults with the same physical type name and source immutability. The original immutability assertion compared a DupAtt snapshot with the original, although DupAtt adds origin metadata; that assertion was an oracle mismatch, not evidence of source mutation. The corrected before/after snapshot comparison passes. This model assumes read-only selected children and does not prove Go aliasing safety.

| Configuration | Meaning | Checked result |
| --- | --- | --- |
| `discovery-mapped-copy` | Emission uses fresh mapped-child keys absent from discovery. | `LookupComplete` counterexample, exit12, 25 states, depth7. |
| `discovery-nullable-copy` | Emission independently recreates the concrete nullable key instead of using the owned pair. | `LookupComplete` counterexample, exit12, 33 states, depth9. |
| `discovery-physical-only` | Rejected alternative: physical helper spelling alone identifies distinct selected semantics. | `DistinctSemanticHelpers` counterexample, exit12, 17 states, depth5. |
| `discovery-alias-overwrite` | Hypothetical robustness control: inserting a concrete key into the canonical key role overwrites another owner. | `CanonicalPreserved` counterexample, exit12, 14 states, depth4. |
| `discovery-owned` | Original mapped children plus a controlled concrete pair scoped to its owning graph node. | All nine invariants pass; exit0, 52 generated/distinct states, empty queue, depth13. |

The alias-collision row is deliberately **model-only**: this work did not establish a corresponding admitted Go witness. It rejects an unsafe identity representation if such a collision occurs; it does not establish a new public behavior requirement or claim that the exact abstract collision is reachable in Go. The physical-only row is likewise a rejected design, not attribution of a newly observed production failure. The first two rows correspond to the retained actual missing-key failures.

The owned run covers both a distinct concrete physical key and a key that would coincide with another canonical occurrence if its role were erased. It checks total lookup, original ownership/default agreement, failure closure, fixed-order helper allocation, distinct semantic helpers, recursive identity and controlled-copy correspondence. It assumes the caller carries the correct origin for its planned copy; direct Go tests must establish that correspondence. It cannot prove arbitrary graph shapes, all conversion paths or global pointer/hash uniqueness.

## Implementation correspondence

- `walkMatches` uses private `transformMappedAttribute` to detach parent/name facts while retaining actual selected children. Consumers must treat those original children as read-only; the source-immutability test is required.
- `transformFailurePlan` and `(*transformFailures).discoverObject` own discovery and closure. The ordered canonical nodes remain separate from lookup aliases. Allocate helpers only after effects/fallibility reach their fixed point.
- `concreteTransformPair` reuses each node's one detached source/target pair. In the implementation's tagged key representation, canonical `transformPair` keys leave `sourceCopy`/`targetCopy` nil; concrete aliases include the exact copied attribute pointers. `(*transformFailures).node` checks the owned-copy role before canonical lookup. Both roles live in the existing graph, and alias registration cannot overwrite a canonical key simply because physical type/reference/layout matches.
- Pointer identity is only a within-plan lookup mechanism, not an output seed. Caller-selected semantic identity and `NameScope` own generated helper names. The model's name tokens abstract that allocator for the two fixed authored traversals.

These correspondence statements require final source and regression review. The checker does not establish that the implementation threads every copy role correctly, nor does its success resolve generated storage/name failures elsewhere.

## Opt-in run

These companion runs used Amazon Corretto 25.0.5 and TLA+tools1.7.4/TLC2.19 revision5a47802. The original [Body-admission model](README.md) retains its separate historical toolchain attribution. `TLA_JAR` must have SHA256 `936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.

```sh
: "${TLA_JAR:?set TLA_JAR to the pinned JAR}"
java -version
shasum -a 256 "$TLA_JAR"
model_state_dir=$(mktemp -d "${TMPDIR:-/tmp}/loom-transform-discovery.XXXXXX")
java -XX:+UseParallelGC -Xmx256m -cp "$TLA_JAR" tlc2.TLC \
  -workers 1 -metadir "$model_state_dir" \
  -config discovery-owned.cfg TransformDiscovery.tla
```

Run each negative configuration separately with its own state directory and inspect the exact invariant/exit12. Parsing, evaluation and timeout failures do not count as negatives. The design runs used a 30-second process bound and each finished below one second. Preserve source/configs and concise receipts; clean owned checker state after validation and independent review. Never add execution or proof-tool installation to CI, hooks, ordinary builds/tests or release prerequisites.
