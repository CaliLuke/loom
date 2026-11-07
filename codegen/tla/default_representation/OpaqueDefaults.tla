-------------------------- MODULE OpaqueDefaults --------------------------
EXTENDS Naturals, FiniteSets, TLC
CONSTANT Mode
VARIABLES sample, emitted, phase
vars == <<sample, emitted, phase>>
Case(label, operation, proven, constructible) ==
    [label |-> label, operation |-> operation, proven |-> proven,
     constructible |-> constructible]
Cases == {
    Case("typed-map", "literal", TRUE, TRUE),
    Case("structural-imported-slice", "literal", TRUE, TRUE),
    Case("structural-imported-alias", "literal", TRUE, TRUE),
    Case("structural-empty-imported-map", "literal", TRUE, TRUE),
    Case("structural-wrong-import", "literal", FALSE, TRUE),
    Case("typed-empty-map", "literal", TRUE, TRUE),
    Case("import-alias", "literal", TRUE, TRUE),
    Case("builtin-byte-alias", "literal", TRUE, TRUE),
    Case("untyped-map", "literal", FALSE, TRUE),
    Case("unproven-assignable-map", "literal", FALSE, TRUE),
    Case("private-struct", "literal", TRUE, FALSE),
    Case("pointer-address", "literal", TRUE, FALSE),
    Case("text-capability", "text", TRUE, TRUE),
    Case("raw-json-capability", "json", TRUE, TRUE)
}
Identity(c) == IF Mode = "root-only" /\ c.label \in
    {"structural-imported-slice", "structural-imported-alias", "structural-empty-imported-map"}
    THEN FALSE ELSE c.proven
Supported(c) == c.operation # "literal" \/ (c.proven /\ c.constructible)
Init == /\ sample \in Cases /\ emitted = FALSE /\ phase = "ready"
Materialize == /\ phase = "ready"
               /\ emitted' = (Mode = "current" \/ (sample.operation # "literal" \/ (Identity(sample) /\ sample.constructible)))
               /\ phase' = "done" /\ UNCHANGED sample
Next == Materialize
Spec == Init /\ [][Next]_vars
NoUnsupportedLiteral == phase = "done" /\ emitted => Supported(sample)
PreserveCapabilities == phase = "done" /\ Supported(sample) => emitted
=============================================================================
