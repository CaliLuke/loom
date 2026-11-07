# Occurrence-owned union closure

The runtime closure design selects a private union declaration for an occurrence
that adds `unevaluatedProperties: false`. A shared open declaration must remain
open elsewhere. Copies must also remain distinct to the declaration collector
and NameScope; cloning the expression while deduplicating by the old identity
would silently select the wrong codec. Repeated occurrences with identical
policy and finalized contract should share the same declaration.

`UnionClosure.tla` explores every ordering of three occurrences: one open and
two closed uses of one declaration. `checked` retains source state and includes
policy in declaration/reference identity. `mutate` changes shared source state;
`collapse` copies policy but erases it from identity. The invariants require
unchanged source policy, correct policy at every visited reference, and no
duplicate declarations. Fair collection must terminate.

Independent review found that nominal IDs also merge localized contracts with
different discriminator/value keys. The `distinct` profile makes the third
occurrence use a different contract; `nominal` deliberately erases that contract
from the key and reproduces the incorrect merge. The implementation compares
complete finalized contract graphs within a nominal identity before reuse.

Run explicitly from this directory with Java 25.0.5 and tla2tools 1.7.4:

```sh
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 -config checked.cfg -metadir "$TMPDIR/union-closure-checked" UnionClosure.tla
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 -config mutate.cfg -metadir "$TMPDIR/union-closure-mutate" UnionClosure.tla
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 -config collapse.cfg -metadir "$TMPDIR/union-closure-collapse" UnionClosure.tla
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 -config distinct.cfg -metadir "$TMPDIR/union-closure-distinct" UnionClosure.tla
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 -config nominal.cfg -metadir "$TMPDIR/union-closure-nominal" UnionClosure.tla
```

This is a design comparison, not a reproduced shipped closure bug: Loom has not
previously admitted this metadata. TLC 1.7.4 on Corretto 25.0.5 checked all three
configurations on 2026-11-06. `checked` passed (14 generated states, 8 distinct,
depth 4), including fair termination. `mutate` violated `SourceUnchanged`
(3 generated/distinct states); `collapse` violated `ReferencePolicy`
(5 generated/distinct states). Both negative controls exited with status 12;
the checked model exited with status 0. After adding contract identity, all three
profiles retained those results. `distinct` also passed (14 generated, 8 distinct,
depth 4); `nominal` violated `ReferencePolicy` (8 generated, 7 distinct), exit 12.

The model assumes that finalized branch graphs are stable and that hash/name
allocation distinguishes policy keys. It does not prove string-key collision
freedom, alias copying, transport projection, recursive fields, JSON annotations,
or Go generation. The existing union declaration model covers reference/collector
agreement separately; pkg's UntaggedClosure covers selection before closure.
`expr/union_closure_finalize.go` applies the chosen policy after service and
transport finalization. It reserves every existing type name and ID before
allocating private named variants, retains the ID through copies and includes
closure in raw-union hashing. `union_closure_contract.go` compares complete
finalized contract graphs, excluding executed DSL callbacks, copy provenance
and derived caches. It retains keys, constraints, metadata and recursive edges;
comparison uses visited attribute pairs and does not call application codecs.
Finalized branch occurrences remain shared.
`dsl/union_closure_test.go` covers reversed occurrence ordering, repeated closed
uses, authored name collisions, declared closure, recursive open references and
copy preservation. `http/codegen/untagged_roots_test.go` compiles both declarations
and exercises local open/closed codecs on mounted HTTP endpoints, including
transactional union decoding and custom tagged-envelope keys. Prepared-plan
and runtime annotation tests cover the separate matching obligation.
Proof execution remains opt-in, outside CI and ordinary build/test commands.
