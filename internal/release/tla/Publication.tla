----------------------- MODULE Publication -----------------------
EXTENDS Naturals, FiniteSets
CONSTANT Unsafe
VARIABLES selected, green, tag, published, main
vars == <<selected, green, tag, published, main>>
Init == /\ selected = 0 /\ green = {} /\ tag = 0 /\ published = FALSE /\ main = 1
Select == /\ selected = 0 /\ selected' = main /\ UNCHANGED <<green, tag, published, main>>
Advance == /\ main = 1 /\ main' = 2 /\ UNCHANGED <<selected, green, tag, published>>
Check == /\ selected # 0 /\ green' = green \cup {selected} /\ UNCHANGED <<selected, tag, published, main>>
Tag == /\ selected \in green /\ tag = 0 /\ tag' = IF Unsafe THEN main ELSE selected
       /\ UNCHANGED <<selected, green, published, main>>
Publish == /\ tag # 0 /\ published' = TRUE /\ UNCHANGED <<selected, green, tag, main>>
Next == Select \/ Advance \/ Check \/ Tag \/ Publish
Spec == Init /\ [][Next]_vars
TestedSource == tag = 0 \/ tag \in green
PinnedSource == tag = 0 \/ tag = selected
Complete == published => tag # 0
=================================================================
