# Compatible component name hints

The model belongs to `componentNaming` and `compatibleSchemaName` in the OpenAPI
IR analyzer. Context-specific provisional acquisition may create a new component
for an inline body that retains a source declaration's noncanonical name hint.
The old desired-name rule required identical logical declarations. It lost that
hint even when the completed schemas were equal, so the final registry could not
compare them under one public name.

`BodyNameHint.tla` explores six pairs in fixed registration order: a declaration
followed by a full body with equal annotations, different annotations, different
publication dependencies, a partial body, a foreign name, or the same logical
declaration. Semantic shape, logical identity, compatible name and complete graph
identity are distinct inputs. The acquired annotation context is a separate input: a compatible public
name must not replace it or reseed a custom component projector. Complete identity includes annotations and recursive
publication, as provided by the existing complete-graph owner.

Run explicitly with the pinned TLA+ tools 1.7.4 JAR and Java 25:

```sh
java -XX:+UseParallelGC -Xmx256m -cp "$TLA2TOOLS_JAR" tlc2.TLC \
  -workers 1 -metadir /tmp/body-name-owned -config owned.cfg BodyNameHint.tla
```

Substitute `current.cfg`, `shapeOnly.cfg`, `ignoreHint.cfg` and `coupled.cfg` for negative
controls. The current rule fails `CompatibleDesired`; ignoring the final graph
fails `NoFalseMerge`; ignoring hint compatibility fails
`ForeignNamesRemainDistinct`. Coupling the public desired name to sampling
fails `AcquiredSamplePreserved`. The owned rule passes all seven invariants in 18
distinct states (24 generated). Executed with Amazon Corretto 25.0.5 and the
1.7.4 JAR. This is an opt-in design aid, not a CI or delivery requirement.

The model does not prove Go metadata extraction, canonical conflict handling,
schema sampling, callback purity, graph hashing or arbitrary registration-order
invariance. It assumes declaration registration precedes its body, as in
`BuildBodyTypes`. Direct `TestInlineBodyCompatibleNameHint`,
`TestInlineBodyHintForeignAndPartial`, `TestInlineBodyHintRetainsMethodSample` and
`TestInlineBodyHintCustomCallbacks` check concrete prepared/callback ownership.
`TestInlineBodyHintCustomGeneratedSampling` exercises the no-authored-example
custom projector path: its acquired context must survive desired-name alignment.
Both modeled components remain retained. Registration-only pruning is outside
this bound, as are visiting acquisition handles and recursive materialization.
The model does not establish termination of contributor publication cycles.
Existing canonical, discriminator and projection controls retain their separate
obligations. Neither type equality nor name compatibility authorizes sample
substitution, early context reuse or merging different completed schemas.
