------------------------ MODULE SelectedBodyField ------------------------
EXTENDS Naturals, TLC
CONSTANT Mode
VARIABLES renamed, phase, semantic, member
vars == <<renamed, phase, semantic, member>>
DeclaredMember == IF renamed THEN "ActualValue" ELSE "Value"
Init == /\ renamed \in BOOLEAN /\ phase = "select"
        /\ semantic = "" /\ member = ""
Select == /\ phase = "select"
          /\ semantic' = IF Mode = "physical-only" THEN DeclaredMember ELSE "value"
          /\ member' = IF Mode = "logical-only" THEN "Value" ELSE DeclaredMember
          /\ phase' = "emit" /\ UNCHANGED renamed
Emit == /\ phase = "emit" /\ phase' = "done"
        /\ UNCHANGED <<renamed, semantic, member>>
Next == Select \/ Emit
Spec == Init /\ [][Next]_vars
SemanticIdentity == phase # "select" => semantic = "value"
PhysicalIdentity == phase # "select" => member = DeclaredMember
=============================================================================
