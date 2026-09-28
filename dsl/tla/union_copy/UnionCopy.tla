----------------------------- MODULE UnionCopy -----------------------------
EXTENDS Naturals, FiniteSets
CONSTANT Mode
Nodes == {"First", "Second"}
Owners == {"Left", "Right"}
Graphs == [Nodes -> SUBSET Nodes]
Reach(graph, root) == {root} \cup graph[root] \cup UNION {graph[n]: n \in graph[root]}
VARIABLES graph, root, ready, copied, mutated, metadata, sourceMeta, failed
vars == <<graph, root, ready, copied, mutated, metadata, sourceMeta, failed>>
Init == /\ graph \in Graphs
        /\ root \in Nodes
        /\ ready = {} /\ copied = {} /\ mutated = {}
        /\ metadata = [o \in Owners |-> 0] /\ sourceMeta = 0
        /\ failed = FALSE
Resolve(n) == /\ n \notin ready /\ ready' = ready \cup {n}
              /\ UNCHANGED <<graph, root, copied, mutated, metadata, sourceMeta, failed>>
Copy(o) == /\ o \notin copied
           /\ copied' = copied \cup {o}
           /\ failed' = (failed \/ (Mode = "deep" /\ ~(Reach(graph, root) \subseteq ready)))
           /\ metadata' = [metadata EXCEPT ![o] = sourceMeta]
           /\ UNCHANGED <<graph, root, ready, mutated, sourceMeta>>
Mutate(o) == /\ o \in copied \ mutated
             /\ mutated' = mutated \cup {o}
             /\ metadata' = [metadata EXCEPT ![o] = 1]
             /\ sourceMeta' = IF Mode = "shared" THEN 1 ELSE sourceMeta
             /\ UNCHANGED <<graph, root, ready, copied, failed>>
Done == ready = Nodes /\ copied = Owners /\ mutated = Owners
Next == (\E n \in Nodes: Resolve(n)) \/ (\E o \in Owners: Copy(o) \/ Mutate(o))
        \/ (Done /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
NoIncompleteTraversal == ~failed
SourceUnchanged == sourceMeta = 0
CopiesIndependent == \A o \in copied \ mutated: metadata[o] = 0
Terminates == <>Done
=============================================================================
