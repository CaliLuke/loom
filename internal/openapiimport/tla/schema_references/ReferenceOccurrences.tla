------------------------ MODULE ReferenceOccurrences ------------------------
EXTENDS Naturals, FiniteSets
CONSTANTS Nodes, Mode
VARIABLES occurrences, emitted, phase
vars == <<occurrences, emitted, phase>>
Kinds == {"mandatory", "lookup", "symbolic"}
Edges(refs) == {<<ref[1], ref[2]>> : ref \in refs}
Mandatory(refs) == {ref \in refs : ref[3] = "mandatory"}
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
Eligible(refs) == refs \ Mandatory(refs)
ToLower(refs) == {ref \in Eligible(refs) : <<ref[1], ref[2]>> \in CycleEdges(Edges(refs))}
Init == /\ occurrences \in SUBSET (Nodes \X Nodes \X Kinds)
        /\ emitted = occurrences
        /\ phase = "plan"
Plan == /\ phase = "plan"
        /\ emitted' = IF Mode = "wrong_pair"
             THEN {ref \in occurrences : <<ref[1], ref[2]>> \notin Edges(ToLower(occurrences))}
             ELSE occurrences \ ToLower(occurrences)
        /\ phase' = "done"
        /\ UNCHANGED occurrences
Next == Plan
Spec == Init /\ [][Next]_vars
PreservesMandatoryOccurrences == Mandatory(occurrences) \subseteq emitted
Admitted == ~Cyclic(Edges(Mandatory(occurrences)))
AdmittedInitializersAcyclic == phase = "done" /\ Admitted => ~Cyclic(Edges(emitted))
OnlyEligibleOccurrencesLowered == emitted \subseteq occurrences /\ occurrences \ emitted \subseteq Eligible(occurrences)
=============================================================================
