-------------------------- MODULE ReferenceDepth --------------------------
EXTENDS Naturals, FiniteSets, Sequences, TLC

CONSTANTS N, Mode
ASSUME N >= 1 /\ Mode \in {"checked", "enumerate", "vertices", "collapseAll"}
Nodes == 1..N
Max(S) == CHOOSE m \in S : \A n \in S : m >= n
Min(S) == CHOOSE m \in S : \A n \in S : m <= n
Paths(V, E) == {s \in UNION {[1..n -> V] : n \in 1..Cardinality(V)} :
    /\ \A i, j \in DOMAIN s : i # j => s[i] # s[j]
    /\ \A i \in 1..(Len(s)-1) : <<s[i], s[i+1]>> \in E}
Reach(E, a, b) == a = b \/ \E p \in Paths(Nodes, E) : p[1] = a /\ p[Len(p)] = b
Component(E, a) == Min({b \in Nodes : Reach(E, a, b) /\ Reach(E, b, a)})
Groups(E) == {Component(E, a) : a \in Nodes}
Arcs(E) == {<<Component(E, e[1]), Component(E, e[2])>> : e \in {a \in E : Component(E, a[1]) # Component(E, a[2])}}
Oracle(E) == Max({Len(p) : p \in Paths(Groups(E), Arcs(E))})
Selected(E, a) == CASE Mode = "vertices" -> a
                   [] Mode = "collapseAll" -> 1
                   [] OTHER -> Component(E, a)
SelectedGroups(E) == {Selected(E, a) : a \in Nodes}
SelectedArcs(E) == {<<Selected(E, e[1]), Selected(E, e[2])>> : e \in {a \in E : Selected(E, a[1]) # Selected(E, a[2])}}

VARIABLES graph, heights, round
vars == <<graph, heights, round>>
Init == /\ graph \in IF Mode = "enumerate" THEN {Nodes \X Nodes} ELSE SUBSET (Nodes \X Nodes)
        /\ heights = [g \in SelectedGroups(graph) |-> 1]
        /\ round = 0
Step == /\ round < N
        /\ heights' = [g \in SelectedGroups(graph) |->
             Max({1} \cup {heights[e[1]] + 1 : e \in {a \in SelectedArcs(graph) : a[2] = g}})]
        /\ round' = round + 1
        /\ UNCHANGED graph
Next == Step
Spec == Init /\ [][Next]_vars
TypeOK == /\ graph \subseteq Nodes \X Nodes
          /\ heights \in [SelectedGroups(graph) -> 1..(N+1)]
          /\ round \in 0..N
ExactDepth == round = N => Max({heights[g] : g \in DOMAIN heights}) = Oracle(graph)
\* Cyclic DFS enumerates these simple paths instead of reusing component
\* results. This lower-bound witness is already superlinear at N=4.
BoundedWork == Mode # "enumerate" \/ Cardinality(Paths(Nodes, graph)) <= N + Cardinality(graph)
=============================================================================
