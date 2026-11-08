# Union validation belongs to the physical declaration

HTTP union names and immutable JSON predicates can be shared between client and
server. Validation operands and JSON adapter callbacks depend on the declaration's
actual Go layout. A nullable object branch exposed a fixed native-value context:
client fields were pointers, but its enum check compared the pointer to a string.
The same authored native DSL fails with and without discriminator hints.

The initial `ValidationBinding` model rejected fixed defaults, first-package
capture and an enclosing-root layout in favor of the declaration registry. Its
later refinement checks that the JSON adapter receives the same validation and
that native scalar/inline branch storage is adapted independently of named field
layout. That refinement followed direct scalar/inline counterexamples against
the first candidate, before the storage correction. A malformed LET scope in
one model revision was corrected; its parser failure is not a counterexample.

Run explicitly with TLA+ tools 1.7.4, keeping checker state outside source:

```sh
java -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -workers 1 -metadir /tmp/union-validation-model -config default.cfg ValidationBinding
```

Repeat with `shared.cfg`, `root.cfg`, `stale-json.cfg` and `unadapted.cfg`:
each deliberately violates an invariant (exit 12). `declaration.cfg` checks all
24 states (exit 0). These are opt-in design tools, never ordinary test gates.
The two package orders and inherited/occurrence-nullability labels are bounded
inputs. Policy values abstract pointer versus value layout. This does not prove
Go syntax, JSON shape semantics, arbitrary type graphs, view projection, or the
nullable wrapper's own correctness; implementation tests establish those seams.

`collectHTTPUnionTypes` shares allocated names but publishes each physical
package through `bindHTTPUnionDeclaration`. `representation.ApplyLayout` applies
the recorded named declaration policy. Existing `AttributeContext.UnionBranchContext`
adapts root storage without replacing named member layout. The binder clones
field and JSON branch descriptors, derives validation there, and retains the
immutable schema/runtime predicates from `unionjson.Analyze`. Neither a later
package nor reversed allocation order can change an earlier publication.
The previous fixed-context validation path is removed from the allocation phase.

`TestHTTPUnionValidationBelongsToPhysicalDeclaration` covers intrinsic/occurrence
nullable branches, opposite registry-versus-root contexts and both package orders,
including independent JSON adapters. `TestHTTPUnionValidationPreservesBranchStorage`
checks native scalar, nullable scalar and inline-object operands. Existing form
binding and named-helper reuse tests remain controls.

`TestNullableUnionValidationGenerated` uses the existing generated-module batch
owner for four hinted/unhinted intrinsic/occurrence cases. It compiles service,
HTTP client/server, CLI and views, checks actual request status/problem detail and
service calls/state, and decodes both ordinary and viewed client responses.
Null and valid enum branches succeed; missing members and wrong enum values
reject without service invocation. Existing untagged-root generated controls
cover scalar/alias/collection/nested and nullable ordinary cases.

The persistent imported native specimens retain every authored enum and null
constraint. Output comparison uses identical CLI arguments and untouched bytes;
only nullable HTTP client validation is expected to change, while the ordinary
control and all OpenAPI output remain equal. Raw logs and compiled artifacts are
scratch review evidence, not committed model inputs.
