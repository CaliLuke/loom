-------------------------- MODULE SourceDialect --------------------------
EXTENDS FiniteSets
CONSTANT Policy
VARIABLES custom, target, selected, phase, blocked
vars == <<custom, target, selected, phase, blocked>>

\* The grammar walker classifies target subtrees, not arbitrary data keys.
Targets == {"schema", "body", "header", "exampleData", "security"}
SchemaBearing == {"schema", "body", "header"}
Init == /\ custom \in BOOLEAN
        /\ target \in Targets
        /\ selected \in BOOLEAN
        /\ phase = "source"
        /\ blocked = FALSE
Transplant == /\ phase = "source"
              /\ blocked' = CASE Policy = "discard" -> FALSE
                                 [] Policy = "document" -> custom
                                 [] OTHER -> custom /\ target \in SchemaBearing
              /\ phase' = "bundled"
              /\ UNCHANGED <<custom, target, selected>>
Next == Transplant \/ (phase = "bundled" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
\* An existing selection closure retains only blockers actually reached.
Diagnosed == selected /\ blocked
Expected == selected /\ custom /\ target \in SchemaBearing
ContractRetained == phase = "bundled" => (Diagnosed <=> Expected)
=============================================================================
