# Anonymous wrapper contracts (#831)

Three collection occurrences have the same protobuf wire shape: permissive,
strict, and a second copy of strict. Each needs the contract it authored.
The model explores every registration order. It separately checks complete
helper emission, faithful predicates and reuse of equal contracts.

The parent keys wrappers by wire shape and emits a presence-only helper even
for unconstrained contents. `current` therefore exposes predicate loss or
accidental tightening. `presence-only` removes that invented predicate, as
#684 proposes, and exposes the missing-helper dependency. Globally merging
constraints rejects valid permissive values. Per-position identity avoids
predicate loss but unnecessarily breaks equal-contract sharing. A contents
contract owner satisfies all three obligations.

Run from this directory with Corretto Java25.0.4.1 and TLC1.7.4 (TLC2 2.19):

```sh
export TLA_JAR="$HOME/.cache/loom-tools/tla2tools-1.7.4.jar"
java -cp "$TLA_JAR" tlc2.TLC -workers 1 -metadir /tmp/loom-831-model-current -config current.cfg WrapperIdentity.tla
java -cp "$TLA_JAR" tlc2.TLC -workers 1 -metadir /tmp/loom-831-model-contract -config contract.cfg WrapperIdentity.tla
```

JAR SHA256: `936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
Run the other configs identically, using distinct external metadirs.

| Mode | Expected result | Generated/distinct states |
| --- | --- | --- |
| current | PredicateFaithful counterexample | 5/5 |
| presence-only | HelperClosure counterexample | 5/5 |
| merged | PredicateFaithful counterexample | 5/5 |
| positional | EqualContractsShare counterexample | 8/7 |
| contract | all invariants pass | 14/10 |

Correspondence: `wrapNestedCollection` asks `collectionMessageName` to select
the generated declaration and UID from wire shape plus `collectionContract`; `anonymousMessageName` reserves its name; `collectValidationsR` and
`addUserTypeValidation` collect one complete helper per declaration identity.
Both call and definition must use that declaration. Existing message-declaration
models retain responsibility for explicit names, schema conflicts and recursive
normalization. This model's contract key is abstract: it does not justify a
MinLength-only string key or prove graph snapshot equality.

Implementation correspondence obligations: capture all effective predicate
facts through `expr.EffectiveConstraintsFor`, preserve ordered clause kind/value
without conflating source identity with predicate identity, and detach snapshots
before registration. Named edges retain their allocated identity and occurrence
facts; do not traverse mutable in-progress named declarations to construct an
anonymous contract. Equal snapshots may share; conservative non-sharing of
nonidentical fact sequences is permitted. This is not a decision procedure for
semantic equivalence of regexes or arbitrary custom values.

Direct tests distinguish length, enum, pattern and format contracts and retain
equal-contract sharing. Generated tests prove binary request validation for
array/array, array/map, map/array and map/map, in both declaration orders, plus
named collection boundaries. Maps use Elem inside MapOf to constrain their
values. An initial matrix accidentally constrained the temporary map itself;
that attempt is excluded. Accepted map evidence uses the corrected Elem DSL.
The implemented tests additionally cover response validators and mounted unary
request/result behavior under race, copied/mutated and recursive named facts,
all numeric endpoint kinds, conjunctive predicates and storage/nullability.
A four-specimen isolated comparison retains 121 artifacts per run and verifies
32 intended files change only for conflicting contracts. Declaration-collision
and recursive controls remain byte-identical. Parent and candidate modules
build/vet. The separate #684 typed-contents overlay passes without missing
helpers; it is not part of this atomic implementation.
A first model attempt mixed integer/string key sets and failed evaluation;
only the homogeneous-key runs above count as model evidence.
