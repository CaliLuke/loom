# Constraint domain admission (#695)

The local ValidationExpr validator accepts equal exclusive endpoints and
negative lengths. Ordinary named ancestry is not checked as one domain unless
a default or enum triggers effective admission. Extend separately implements
more-correct interval checks. The candidate shares one domain owner across
local expressions, effective constraints, attribute validation and composition.

`DomainAdmission.tla` explores numeric endpoints and two authored minimum
lengths plus a maximum over -1..1. Numeric endpoint exclusivity and whether the
constraints belong to separate inherited layers vary independently. A finite
witness oracle enumerates half-step numeric values and integer lengths; it
checks the candidate's endpoint-comparison formula without merely restating it.

- Local: reproduces the incomplete local-only comparison and negative-length
  admission. Exits 12, Admission violation, 437 generated/distinct states.
- MergedOnly: checks merged domain shape but forgets negative authored minimum
  lengths erased by a tighter nonnegative overlay. Exits 12, Admission violation,
  682 generated/distinct states.
- Checked: checks authored length eligibility before folding, then the selected
  exact numeric and length endpoints. Exits 0, 1297 generated/865 distinct states.

Use TLA+ tools 1.7.4, TLC 2.19 revision 5a47802, JAR SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
Validated with Corretto Java 25.0.4.1. From this directory:

```sh
for cfg in Local MergedOnly Checked; do
  java -XX:+UseParallelGC -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 \
    -metadir "/tmp/loom-695-$cfg" -config "$cfg.cfg" DomainAdmission.tla
done
```

Negatives must fail Admission, not parsing or action completeness. Keep logs and
checker state outside the repository and remove them after independent review.

## Implementation correspondence and limits

NumericValid corresponds to selected effective lower/upper endpoint comparison,
using exact NumericBound.Cmp and strictness from EffectiveNumericBound. LengthValid
corresponds to authored nonnegative validation followed by tighter-bound folding.
Local and MergedOnly show why fixing only the exclusive comparison or validating
only the lowered result does not establish the cross-layer contract.

Direct `TestValidationDomainAdmission` retains the original three failures and
mixed/inclusive/zero controls. `TestEffectiveConstraintDomainAdmission` covers
named ancestry, a masked negative minimum, precise decimal ordering beyond
float64 integer precision, and detached-source immutability. DSL tests cover
actual expression execution, strings/bytes/arrays/maps, ordinary named types,
and existing Extend rejection. They prevent accepting an unrelated DSL error
as evidence of domain admission.

This is bounded arithmetic exploration, not a general satisfiability proof. It
abstracts exact numbers by their ordering, assumes finalized acyclic ancestry,
and excludes enums, regex/format compatibility, arbitrary codecs, recursion,
nullable presence and discrete machine representability. A mathematically
nonempty interval with no representable native value remains outside this
repair. Tests cover exact-number parsing and real occurrence construction.
