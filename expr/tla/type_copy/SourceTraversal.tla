--------------------------- MODULE SourceTraversal ---------------------------
EXTENDS Naturals, FiniteSets
CONSTANT NominalCache
VARIABLES pending, seen, mutations
vars == <<pending, seen, mutations>>
Nodes == {"root", "direct", "collection"}
Edges(n) == IF n = "root" THEN {"direct", "collection"} ELSE {n}
Key(n) == IF NominalCache /\ n # "root" THEN "Leaf" ELSE n
Init == /\ pending = {"root"}
        /\ seen = {}
        /\ mutations = [n \in Nodes |-> 0]
Visit(n) == /\ n \in pending
            /\ IF Key(n) \in seen
                  THEN /\ pending' = pending \ {n}
                       /\ UNCHANGED <<seen, mutations>>
                  ELSE /\ seen' = seen \cup {Key(n)}
                       /\ mutations' = [mutations EXCEPT ![n] = @ + 1]
                       /\ pending' = (pending \ {n}) \cup Edges(n)
Next == \E n \in pending : Visit(n)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
Complete == pending = {} => \A n \in Nodes : mutations[n] = 1
ExactlyOnce == \A n \in Nodes : mutations[n] <= 1
Terminates == <>(pending = {})
=============================================================================
