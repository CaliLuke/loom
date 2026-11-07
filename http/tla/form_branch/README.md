# Form branch projection

Root form unions match schemas against exact logical JSON projected directly
from one immutable `RawPairs` carrier. They never re-encode typed candidates:
Go integer widths and assignment success must not change schema membership.
The media owner decides whether the body is present. A present empty body
projects `{}`; see `UntaggedPresence` for optional-body rules.

## Captured ownership

`FormBranch.tla` checks ownership when an exploded field has alternatives
claiming `x` and `y`. A selected `x` branch may ignore `y` internally, but the
field's captured union namespace still owns both. Treating only selected keys
as consumed makes `y` an unrelated root member and changes root closure.

- `selected.cfg` reproduces the defect at raw `{x, y}`, selected `x`: invariant
  failure, exit 12, 19 generated/distinct states, depth 2.
- `captured.cfg` preserves all alternative claims: passes, exit 0, 48 generated
  and 32 distinct states, depth 2.

## Separate logical and physical domains

Review found that placing unknown physical names into logical JSON also fails:
`x=ok&filter=extra` can contain a declared `filter` field exploding to `x` and
an unrelated unknown raw `filter` key. Open branches must ignore that unknown
value without replacing the known logical property or rejecting its name.

`DomainProjection.tla` models this collision, object closure, and explicit-open
annotations. The old collision rejection (`collision.cfg`) fails
`SchemaAgreement`: 55 generated/distinct states, depth 2. The separated domains
(`domains.cfg`) pass `SchemaAgreement`, `PhysicalAnnotations`,
`ExplicitOpenEvaluates`, and `ImplicitOpenLeavesUnknown`: 144 generated,
96 distinct states, depth 2.

The finite universe includes unrelated and empty keys. These models establish
only the stated ownership/domain invariants. Exact-one accounting is owned by
`unionmatch`; JSON syntax, Unicode, number domains, repetitions, nesting and
value projection are implementation-test obligations.

## Implementation correspondence

- `FormKeySet` captures runtime ownership. `FormObjectJSON` validates all static
  field descriptors and namespace overlaps before invoking any raw callback.
- `FormProjection.Value` contains only declared logical JSON values. `Members`,
  `Unclaimed`, and `FieldClaims` retain physical decoded keys separately, with
  complete union-wide field claims. Residual ownership excludes other fields'
  explicit literals and deepObject prefixes.
- `FormProjection.Schema` combines logical schema membership with the captured
  branch closed-object policy. Open objects ignore unknown values.
  `Annotations` maps evaluated logical properties back to physical claims;
  only explicit `additionalProperties: true` evaluates unknown physical keys.
  Outer union closure follows original branch counting in `MatchUntaggedText`.
- `FormJSONValue` retains JSON and explicit null. `FormTextValue` uses exact
  `ParamValueJSON`, sharing static shape validation with `TextBranch.Schema`
  and `TextBranch.Annotations` before any input-dependent result.
- `FormTextUnionValue` uses `SelectSchema`, then selected annotations for closure.
  Recursive views retain nested exact-one structure. Omission differs from a
  present mismatch. This path never invokes typed assignment.
- `FormMapJSON` visits every decoded key, including the empty key, and delegates
  each value to the captured JSON/text/collection/union codec. Its logical and
  physical names coincide, and annotations retain this identity.
- `http/form_branch_test.go` covers the model counterexamples, unknown-name
  collision across open/closed/explicit-open branches, original ambiguity counts,
  static configuration failures after early mismatches, exact numbers, names,
  null, omission, empty roots, repetitions, delimiters, malformed input, nested
  unions, residual claims, map values, and evaluator error propagation.

## Explicit opt-in checks

Run with Java 25 and TLA+ tools 1.7.4; never add these to ordinary gates:

```sh
for policy in selected captured; do
  java -XX:+UseParallelGC -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" \
    tlc2.TLC -workers 1 -metadir "/tmp/loom-form-branch-$policy" \
    -config "http/tla/form_branch/$policy.cfg" \
    http/tla/form_branch/FormBranch.tla
done
for policy in collision domains; do
  java -XX:+UseParallelGC -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" \
    tlc2.TLC -workers 1 -metadir "/tmp/loom-form-branch-$policy" \
    -config "http/tla/form_branch/$policy.cfg" \
    http/tla/form_branch/DomainProjection.tla
done
```

The selected and collision controls deliberately fail. Delete temporary checker
states after independent review; retain sources, configurations and findings.

## Projection mismatch does not erase physical ownership

Schema projection and typed decoding are independent branch domains. The former
implementation returned an empty projection when a field's exact JSON view
mismatched. A later successful typed decode could then see no unknown keys and
incorrectly satisfy a closed object. `MismatchProjection.tla` models independent
schema/typed value results and a closed object's unknown physical members.
`mismatch-discard.cfg` reproduces that incorrect admission;
`mismatch-retain.cfg` retains the complete key set before projecting values.
The model does not prove JSON conversion or generated callback wiring.

`FormObjectJSON` and `FormMapJSON` now return nil `Value` but complete `Members`,
`Unclaimed`, and `FieldClaims` after an ordinary value mismatch. Map keys are
captured before any value callback. Invalid UTF-8 physical names still reject
before either value domain, and configuration/evaluator errors remain fatal.
`TestFormProjectionKeepsPhysicalClaimsOnMismatch` covers both mismatch returns
and invalid JSON, repeated keys, complete map claims, and independent open/closed
typed admission. Schema callers must continue to honor the returned match flag;
retained ownership is not a successful logical projection.

Run the two controls explicitly with the toolchain above:

```sh
for policy in discard retain; do
  java -XX:+UseParallelGC -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" \
    tlc2.TLC -workers 1 -metadir "/tmp/loom-form-mismatch-$policy" \
    -config "http/tla/form_branch/mismatch-$policy.cfg" \
    http/tla/form_branch/MismatchProjection.tla
done
```
