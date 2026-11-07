---------------------------- MODULE ValidationImports ----------------------------
EXTENDS FiniteSets
CONSTANT Mode
VARIABLES owner, needs
Chunks == {"matcher", "method"}
Files == {"union.go", "branch.go"}
Init == /\ owner \in [Chunks -> Files]
        /\ needs \in SUBSET Chunks
Next == UNCHANGED <<owner, needs>>
Required == {owner[chunk] : chunk \in needs}
Published == CASE Mode = "old" -> {owner[chunk] : chunk \in needs \cap {"method"}}
               [] Mode = "blanket" -> Files
               [] OTHER -> {owner[chunk] : chunk \in needs}
Coverage == Required \subseteq Published
Absence == Published \subseteq Required
Spec == Init /\ [][Next]_<<owner, needs>>
=============================================================================
