-------------------------- MODULE ContentCarrier --------------------------
EXTENDS Naturals
CONSTANT Recurse
VARIABLES mixed, jsonLeaf, layout, phase
vars == <<mixed, jsonLeaf, layout, phase>>
Init == /\ mixed \in BOOLEAN /\ jsonLeaf \in BOOLEAN
        /\ layout = FALSE /\ phase = "capture"
Capture == /\ phase = "capture"
           /\ layout' = (jsonLeaf /\ (~mixed \/ Recurse))
           /\ phase' = "done"
           /\ UNCHANGED <<mixed, jsonLeaf>>
Next == Capture
Spec == Init /\ [][Next]_vars
LeafOwnsLayout == phase = "done" => layout = jsonLeaf
=============================================================================
