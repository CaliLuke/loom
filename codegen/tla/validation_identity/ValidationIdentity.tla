---------------------- MODULE ValidationIdentity ----------------------
EXTENDS Naturals, FiniteSets, Sequences
CONSTANT Mode
VARIABLES explicit, intrinsic, occurrence, pending, definitions, calls
vars == <<explicit, intrinsic, occurrence, pending, definitions, calls>>
Types == {"Node", "Peer"}
PhysicalName(t) == IF explicit[t] THEN "Wire" \o t ELSE t
Carrier(nullable) == IF nullable THEN "nullable-wrapper" ELSE "object-pointer"
Definition(t) == <<PhysicalName(t), Carrier(intrinsic[t])>>
Call(t) == <<IF Mode = "drop-metadata" THEN t ELSE PhysicalName(t),
             Carrier(IF Mode = "copy-presence"
                     THEN intrinsic[t] \/ occurrence[t] ELSE intrinsic[t])>>
Init == /\ explicit \in [Types -> BOOLEAN]
        /\ intrinsic \in [Types -> BOOLEAN]
        /\ occurrence \in [Types -> BOOLEAN]
        /\ pending = Types
        /\ definitions = {}
        /\ calls = {}
Emit == /\ pending # {}
        /\ \E t \in pending:
             /\ pending' = pending \ {t}
             /\ definitions' = definitions \cup {Definition(t)}
             /\ calls' = calls \cup {Call(t)}
        /\ UNCHANGED <<explicit, intrinsic, occurrence>>
Next == Emit
Spec == Init /\ [][Next]_vars
CallIdentity == pending = {} => calls = definitions
PhysicalNames == pending = {} => {d[1] : d \in definitions} = {PhysicalName(t) : t \in Types}
FiniteDeclarations == Cardinality(definitions) <= Cardinality(Types)
=============================================================================
