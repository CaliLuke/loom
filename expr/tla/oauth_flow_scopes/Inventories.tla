------------------------- MODULE Inventories -------------------------
EXTENDS TLC
CONSTANT Flatten
VARIABLES shared, local, explicit
Flows == {"code", "client"}
Names == {"read", "write"}
Init == /\ shared \in SUBSET Names
        /\ local \in [Flows -> SUBSET Names]
        /\ explicit \in [Flows -> BOOLEAN]
Expected(f) == IF explicit[f] THEN local[f] ELSE shared
Inventory(f) == IF Flatten THEN UNION {Expected(g): g \in Flows} ELSE Expected(f)
Isolation == \A f \in Flows: Inventory(f) = Expected(f)
Next == UNCHANGED <<shared, local, explicit>>
Spec == Init /\ [][Next]_<<shared, local, explicit>>
=======================================================================
