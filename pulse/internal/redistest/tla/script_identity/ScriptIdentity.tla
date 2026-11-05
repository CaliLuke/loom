-------------------------- MODULE ScriptIdentity --------------------------
EXTENDS Naturals, FiniteSets
CONSTANT TranslateLookup
VARIABLES published, cached, expected, agrees
vars == <<published, cached, expected, agrees>>
Digests == {1, 2}
Rewritten(d) == d + 2

Init == /\ published = {}
        /\ cached = {}
        /\ expected = {}
        /\ agrees = TRUE

\* Publish the immutable digest translation before sending rewritten source.
Publish(d) == /\ published' = published \cup {d}
              /\ UNCHANGED <<cached, expected, agrees>>

\* A successful SCRIPT LOAD or EVAL adds the actual rewritten digest.
Commit(d) == /\ d \in published
             /\ cached' = cached \cup {Rewritten(d)}
             /\ expected' = expected \cup {d}
             /\ UNCHANGED <<published, agrees>>

\* Failed loads do not commit. Flush keeps immutable translations but removes
\* every executable script; concurrent publication cannot resurrect a script.
Flush == /\ cached' = {}
         /\ expected' = {}
         /\ UNCHANGED <<published, agrees>>

Lookup(d) == LET actual == IF TranslateLookup /\ d \in published
                          THEN Rewritten(d) ELSE d
             IN /\ agrees' = ((actual \in cached) = (d \in expected))
                /\ UNCHANGED <<published, cached, expected>>

Next == (\E d \in Digests: Publish(d) \/ Commit(d) \/ Lookup(d)) \/ Flush
Spec == Init /\ [][Next]_vars
LookupAgrees == agrees
CacheIdentity == cached = {Rewritten(d): d \in expected}
=============================================================================
