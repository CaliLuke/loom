----------------------------- MODULE CarrierRole -----------------------------
EXTENDS Naturals
CONSTANT Captured
VARIABLES jsonLeaf, ambientJSON, visited, layout
vars == <<jsonLeaf, ambientJSON, visited, layout>>
Init == /\ jsonLeaf \in BOOLEAN /\ ambientJSON \in BOOLEAN
        /\ visited = {} /\ layout = FALSE
Record(owner) ==
    /\ owner \in {"ambient", "field"} \ visited
    /\ layout' = IF Captured THEN jsonLeaf
                  ELSE layout \/ (IF owner = "field" THEN jsonLeaf ELSE ambientJSON)
    /\ visited' = visited \cup {owner}
    /\ UNCHANGED <<jsonLeaf, ambientJSON>>
Next == \E owner \in {"ambient", "field"} : Record(owner)
Spec == Init /\ [][Next]_vars
CapturedLayout == visited = {"ambient", "field"} => layout = jsonLeaf
=============================================================================
