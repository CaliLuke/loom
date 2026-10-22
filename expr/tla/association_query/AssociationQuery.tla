-------------------------- MODULE AssociationQuery --------------------------
EXTENDS Naturals, FiniteSets
CONSTANT IgnoreScope
VARIABLES edges, owners, root, duplicate, preferred, done
vars == <<edges, owners, root, duplicate, preferred, done>>
Nodes == 1..3
RECURSIVE Reach(_, _, _)
Reach(start, relation, steps) ==
    IF steps = 0 THEN {start}
    ELSE LET prior == Reach(start, relation, steps - 1)
         IN prior \cup {n \in Nodes : \E p \in prior : <<p, n>> \in relation}
Inverse == {<<p[2], p[1]>> : p \in edges}
Forward == owners \cap Reach(root, edges, Cardinality(Nodes))
Indexed == IF IgnoreScope THEN owners
           ELSE {n \in owners : root \in Reach(n, Inverse, Cardinality(Nodes))}
Outcome(candidates) ==
    IF candidates = {} THEN "missing"
    ELSE IF Cardinality(candidates) > 1 \/ ((candidates \cap duplicate) \ preferred) # {}
         THEN "ambiguous"
         ELSE CHOOSE n \in candidates : TRUE
Init == /\ edges \in SUBSET (Nodes \X Nodes)
        /\ owners \in SUBSET Nodes
        /\ root \in Nodes
        /\ duplicate \in SUBSET owners
        /\ preferred \in SUBSET duplicate
        /\ done = FALSE
Next == /\ ~done
        /\ done' = TRUE
        /\ UNCHANGED <<edges, owners, root, duplicate, preferred>>
Spec == Init /\ [][Next]_vars
SameOwners == Forward = Indexed
SameOutcome == Outcome(Forward) = Outcome(Indexed)
=============================================================================
