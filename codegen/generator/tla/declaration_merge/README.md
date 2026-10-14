# Shared declaration merge model

The merger must distinguish renderer categories from Go declaration identity.
This bounded model explores both contribution orders for distinct declarations,
identical duplicates, conflicting definitions and repeated effect-only declarations
(init or blank-only). Category ownership loses distinct declarations/effects and
silently admits conflicts. Declaration ownership preserves occurrences and rejects
conflicts independent of order. Definition equality and identity extraction are
assumptions here; Go AST tests establish the actual type/method/group boundaries.
Const iota, comments, import aliases and multi-name specs are outside this model.

Run with pinned TLC 1.7.4 jar SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`:

```sh
model_tmp=$(mktemp -d)
java -cp "$TLA_JAR" tlc2.TLC -workers 1 -metadir "$model_tmp/states" -config owned.cfg DeclarationMerge.tla
```

Repeat legacy.cfg for the category-owner counterexample. Store state and logs
outside the repository and remove them after independent review. Correspondence:
`generator.mergeFilesByPath`, private rendered declaration merge, and
`TestSharedDeclarationMerge` plus `TestSharedDeclarationFilenameCollision`.

Recorded TLC results: owned passes 20 generated / 18 distinct states; legacy
fails DeclarationCoverage (exit 12). The first legacy run with all invariants
also witnessed ConflictAdmission failure. Both checks correspond to direct
rendered regressions; no checker output is committed.
