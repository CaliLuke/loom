------------------------- MODULE BindingIdentity -------------------------
EXTENDS Naturals, FiniteSets
CONSTANT Policy
VARIABLES selected, admitted, phase
vars == <<selected, admitted, phase>>
Bindings == {"field", "map", "credential"}
Labels == [field |-> "MapParams", map |-> "MapParams", credential |-> "sid"]
\* The whole-query map overlaps every sibling; field and credential are disjoint.
Overlap(a, b) == a # b /\ (a = "map" \/ b = "map")
\* Name-keyed construction overwrites the authored field with the synthetic map.
Claims(B) == IF Policy = "labels" /\ Labels["field"] = Labels["map"] /\ {"field", "map"} \subseteq B
             THEN B \ {"field"} ELSE B
Init == /\ selected \in SUBSET Bindings
        /\ admitted = FALSE /\ phase = "pending"
Check == /\ phase = "pending"
         /\ admitted' = (\A a, b \in Claims(selected) : ~Overlap(a, b))
         /\ phase' = "done"
         /\ UNCHANGED selected
Next == Check \/ (phase = "done" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
Sound == phase = "done" /\ admitted => (\A a, b \in selected : ~Overlap(a, b))
Complete == phase = "done" /\ (\A a, b \in selected : ~Overlap(a, b)) => admitted
=============================================================================
