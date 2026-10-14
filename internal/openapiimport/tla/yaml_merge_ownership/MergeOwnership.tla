----------------------- MODULE MergeOwnership -----------------------
EXTENDS Naturals, FiniteSets, TLC

CONSTANT Immutable
Workers == {"components", "lookup"}
VARIABLES phase, sourceVersion, reading, raced, views
vars == <<phase, sourceVersion, reading, raced, views>>

Init == /\ phase = [w \in Workers |-> "start"]
        /\ sourceVersion = 0
        /\ reading = {}
        /\ raced = FALSE
        /\ views = [w \in Workers |-> "none"]

\* Both workers have aliases to the same published mapping. Expansion reads
\* that mapping and its shared children before retaining its selected content.
Begin(w) == /\ phase[w] = "start"
            /\ phase' = [phase EXCEPT ![w] = "read"]
            /\ reading' = reading \cup {w}
            /\ UNCHANGED <<sourceVersion, raced, views>>

Expand(w) == /\ phase[w] = "read"
             /\ phase' = [phase EXCEPT ![w] = "publish"]
             /\ reading' = reading \ {w}
             /\ views' = [views EXCEPT ![w] = "effective"]
             /\ UNCHANGED <<sourceVersion, raced>>

\* The old helper writes node.Content on the shared node. The candidate only
\* assigns the caller's local node pointer to a shallow immutable view.
Publish(w) == /\ phase[w] = "publish"
              /\ phase' = [phase EXCEPT ![w] = "done"]
              /\ sourceVersion' = IF Immutable THEN sourceVersion ELSE 1
              /\ raced' = (raced \/ (~Immutable /\ reading # {}))
              /\ UNCHANGED <<reading, views>>

Next == \E w \in Workers: Begin(w) \/ Expand(w) \/ Publish(w)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
NoConcurrentWrite == ~raced
SourcePreserved == Immutable => sourceVersion = 0
EffectiveViews == \A w \in Workers: phase[w] = "done" => views[w] = "effective"
Complete == <> (\A w \in Workers: phase[w] = "done")
=====================================================================
