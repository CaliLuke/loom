# Object openness under Extend (#1062)

`expr.AttributeExpr.ObjectOpenness` owns the authored policy for undeclared
members: unset (open), explicit open, or closed. This model decides how
`Extend` composes that policy. JSON Schema `allOf` applies each subschema's
`additionalProperties` to every key that subschema does not declare. Loom's
derived type semantically declares all inherited fields, so its own policy
covers only keys outside the composed set.

`ObjectOpenness.tla` enumerates one or two bases over the keys `id` and
`note`, a derived type that may declare `note`, open or closed policies on every
source, and three value domains with a compatible pair and a disjoint one. It
compares Loom's composed object with the `allOf` meaning over every instance of
the keys `id`, `note` and `x`.

- Flatten: the behavior before #1062. A field takes only its declarations and
  one composed policy covers the rest. Exits 12, Sound violation in the initial
  state: a closed base with no members, extended by a derived type that declares
  `note`, admits `note` although the base forbids it.
- PerSource: each source's policy also constrains the fields it does not
  declare. A field with an empty domain rejects the design. Exits 0 with Sound,
  RejectionJustified and ImplementationAgrees, 16896 generated/8448 distinct
  states.

Use TLA+ tools 1.7.4, TLC 2.19 revision 5a47802, JAR SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
Validated with OpenJDK 25.0.5. From this directory:

```sh
for cfg in Flatten PerSource; do
  java -XX:+UseParallelGC -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 \
    -metadir "/tmp/loom-1062-$cfg" -config "$cfg.cfg" ObjectOpenness.tla
done
```

Flatten must fail Sound, not parsing. Keep logs and checker state outside the
repository and remove them after independent review.

## Design decisions

- Composition is per source. With open and closed policies only, a closed
  base admits the composition exactly when it declares every composed key.
  `ImplementationAgrees` checks that `closedBasesDeclare` in
  `expr/attribute_composition.go` admits the same designs as the per-source
  composition. The derived type inherits the closure.
- Rejecting such a design is a representation and admission policy, not a
  consequence of the model. `RejectionJustified` establishes only that no
  document accepted by the sources carries the forbidden field. Because every
  field here is optional, documents without that field, such as `{}`, stay
  valid. Loom rejects the design rather than generate a field that no valid
  document can carry, and rather than silently drop the authored member.
- Reopening a closed base with an explicit `"true"` is rejected as authoring
  hygiene. Under `allOf` the base closure still applies, so an explicit open
  policy on the derived type would be misleading. The model does not
  distinguish unset from explicit open.
- A base is the named type passed to `Extend`, followed through aliases. Its
  effective policy is the conjunction of the chain: closed when any link is
  closed, otherwise the nearest authored policy. This holds even for an alias
  that validation never visits. Validation separately rejects an alias or
  occurrence that authors `"true"` over a closed type. The model treats each
  base as already resolved.

## Implementation correspondence and limits

`TestExtendClosedBasePerSource` covers a derived member, a member declared
before `Extend`, a peer base member, chained and aliased closed bases,
reopening through `Extend`, an alias or an occurrence, closure inheritance,
refinement of a declared member, a member outside the JSON wire and the
migration that moves closure to the derived type. Occurrence policies, which
the model does not cover, select a representation variant at analysis time:
`TestObjectOpennessOccurrencePolicy` and `TestObjectOpennessOccurrenceVariants`
cover OpenAPI, including recursive, Extend-composed and result types, and
`TestPlanAppliesOccurrenceOpenness` covers untagged union matching. Value admission, examples,
value plans and OpenAPI emission read the policy only through
`ObjectOpenness`; `scripts/lint_object_openness.sh` keeps other code from
reading the metadata key.

The model abstracts keys as JSON wire names and values as small domains. It
excludes typed additional-member schemas (#1013), requiredness, nullability,
recursion, views and transport codecs. Declaration compatibility is the
existing `Extend` check; the model only requires that compatible domains
intersect.

## Selected result views (#1065)

`ProjectionOpenness.tla` checks the ownership boundary after field selection.
Projection preserves the declaration's authored policy, including the distinction
between unset and explicit open under an API-wide closed-object default. Closure
applies to the selected object's members. It is not copied onto a reference
wrapper. `CopyObjectOpenness` owns the detached policy copy; projection does not
copy unrelated naming or transport metadata.

Run the same TLC command with `ProjectionOpenness.tla` and these configurations:

| Configuration | Result | Generated / distinct states |
| --- | --- | --- |
| projection-discard | AuthoredPolicy fails: explicit policy is lost | 17 / 17 |
| projection-collapse | AuthoredPolicy fails: unset becomes explicit open | 13 / 13 |
| projection-checked | AuthoredPolicy and RenderedPolicy pass | 24 / 24 |

The bounded model covers three authored policies, two API defaults and two
selected field sets. It assumes a valid completed result declaration and does
not model code generation, recursion or typed additional members. Result views
with AdditionalAttributes remain rejected by ordinary design validation.
`TestProjectRetainsObjectOpenness` checks detached policy copies for root, nested
and collection projections with default/tiny selections. The rendered specimen
checks selected fields, requiredness and effective policy under both API defaults;
its generated HTTP service/client/server compile checks and parser/Vacuum checks
cover consumers outside the model.
