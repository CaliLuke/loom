------------------------- MODULE SchemaReferences -------------------------
EXTENDS Naturals, FiniteSets
CONSTANTS Nodes, Mode
VARIABLES references, deferred, lowered, phase, admitted
vars == <<references, deferred, lowered, phase, admitted>>
RECURSIVE Closure(_, _)
Closure(edges, steps) ==
    IF steps = 0 THEN edges
    ELSE LET prior == Closure(edges, steps - 1)
         IN prior \cup {p \in Nodes \X Nodes :
                \E middle \in Nodes :
                    <<p[1], middle>> \in prior /\ <<middle, p[2]>> \in prior}
Cyclic(edges) == \E node \in Nodes : <<node, node>> \in Closure(edges, Cardinality(Nodes))
CycleEdges(edges) == {edge \in edges :
    <<edge[2], edge[1]>> \in Closure(edges, Cardinality(Nodes))}
Init == /\ references \in SUBSET (Nodes \X Nodes)
        /\ deferred \in SUBSET references
        /\ (Mode # "legacy" \/ deferred = references)
        /\ lowered = references
        /\ phase = "plan"
        /\ admitted = FALSE
Plan == /\ phase = "plan"
        /\ lowered' = IF Mode = "legacy" THEN references
                        ELSE references \ (deferred \cap CycleEdges(references))
        /\ admitted' = IF Mode = "legacy" THEN TRUE
                         ELSE ~Cyclic(references \ deferred)
        /\ phase' = "done"
        /\ UNCHANGED <<references, deferred>>
Next == Plan
Spec == Init /\ [][Next]_vars
GeneratedInitializersAcyclic == phase = "done" /\ admitted => ~Cyclic(lowered)
AllRepresentableGraphsAdmitted == phase = "done" => (admitted = ~Cyclic(references \ deferred))
OnlyDeferredReferencesLowered == lowered \subseteq references /\ references \ lowered \subseteq deferred
=============================================================================
