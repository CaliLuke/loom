-------------------------- MODULE UnionExpansion --------------------------
EXTENDS Naturals, FiniteSets, Sequences
CONSTANT Mode
Nodes == 1..3
Graphs == [Nodes -> SUBSET Nodes]
VARIABLES graph, root, pending, rejected
vars == <<graph, root, pending, rejected>>
Reach(n) == graph[n] \cup UNION {graph[m]: m \in graph[n]}
           \cup UNION {UNION {graph[k]: k \in graph[m]}: m \in graph[n]}
HasCycle == \E n \in {root} \cup Reach(root): n \in Reach(n)
Init == /\ graph \in Graphs /\ root \in Nodes
        /\ pending = {<<root>>} /\ rejected = FALSE
Expand(path) ==
  /\ path \in pending /\ ~rejected
  /\ LET n == path[Len(path)]
         cycle == n \in {path[i]: i \in 1..(Len(path)-1)}
     IN IF Mode = "checked" /\ cycle
        THEN /\ rejected' = TRUE /\ pending' = {}
        ELSE /\ rejected' = FALSE
             /\ pending' = (pending \ {path}) \cup {Append(path, m): m \in graph[n]}
  /\ UNCHANGED <<graph, root>>
Done == pending = {}
Next == (\E path \in pending: Expand(path)) \/ (Done /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
BoundedExpansion == \A path \in pending: Len(path) <= Cardinality(Nodes)+1
SafeAcceptance == Done /\ ~rejected => ~HasCycle
AccurateRejection == rejected => HasCycle
Terminates == <>Done
=============================================================================
