# Selected body field identity

The bounded model distinguishes the logical selection key from the declared Go
member. `logical-only` models deriving every member selector from the logical
name and violates PhysicalIdentity when metadata renames a member. `physical-only`
models replacing the selection key itself with the Go name and violates
SemanticIdentity. `resolved` retains both identities and satisfies both invariants.

This is a small design comparison, not a proof of Go generation. It assumes valid
selection and correct GoifyAtt metadata resolution. It does not model inherited
metadata, mapped object keys, requiredness, defaults, nullability or projections.
Direct analysis and compiled generated fixtures must check those boundaries.
All body builders and constructor metadata must consume the same resolved member;
semantic lookups must retain the original key. The proposed owning helper is the
private selectedServiceField binding in http/codegen, not transport templates.

Run each configuration with Java Corretto25.0.4.1 plus official TLA+ tools1.7.4, JAR SHA256
936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88:

```sh
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -config resolved.cfg \
  -metadir /tmp/loom-selected-body-field-resolved SelectedBodyField.tla
```

Use separate fresh external metadirs for logical-only.cfg and physical-only.cfg.
Preserve sources/configurations and concise findings; delete checker output after
validation and independent review once no active consumer needs it.

Observed results: logical-only exits12 at four states, physical-only exits12 at
three states, and resolved exits0 after six states. The two negative results
name the expected invariants; they are not syntax/checker failures.
