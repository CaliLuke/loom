------------------------------ MODULE TypeCopy ------------------------------
EXTENDS Naturals, FiniteSets

CONSTANT NominalCache
VARIABLES copies, pending
vars == <<copies, pending>>

Nodes == {"root", "text", "json"}
Edges(n) == CASE n = "root" -> {"text", "json"}
                [] OTHER -> {n}
Nominal(n) == IF n = "root" THEN "root" ELSE "Record"
Key(n) == IF NominalCache THEN Nominal(n) ELSE n
Shape(n) == CASE n = "root" -> "fields"
                [] n = "text" -> "x"
                [] OTHER -> "jsonX"

Init == /\ copies = [n \in Nodes |-> "absent"]
        /\ pending = {"root"}

Copy(n) ==
    LET cached == {p \in Nodes : copies[p] # "absent" /\ Key(p) = Key(n)}
        owner == IF cached = {} THEN n ELSE copies[CHOOSE p \in cached : TRUE]
    IN /\ n \in pending
       /\ copies' = [copies EXCEPT ![n] = owner]
       /\ pending' = (pending \ {n}) \cup
              IF cached = {} THEN {p \in Edges(n) : copies'[p] = "absent"} ELSE {}

Next == \E n \in pending : Copy(n)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)

TypeOK == /\ copies \in [Nodes -> Nodes \cup {"absent"}]
          /\ pending \subseteq Nodes
ShapesRetained == \A n \in Nodes : copies[n] # "absent" => Shape(copies[n]) = Shape(n)
DistinctNodes == \A a, b \in Nodes :
    copies[a] # "absent" /\ copies[b] # "absent" /\ a # b => copies[a] # copies[b]
EdgesRetained == pending = {} =>
    \A n \in Nodes : copies[n] # "absent" /\ Edges(copies[n]) = Edges(n)
Terminates == <>(pending = {})
=============================================================================
