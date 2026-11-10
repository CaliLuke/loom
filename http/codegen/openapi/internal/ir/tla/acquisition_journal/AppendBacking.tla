------------------------- MODULE AppendBacking -------------------------
EXTENDS Naturals
CONSTANT Mode
VARIABLES prior, current, phase
vars == <<prior, current, phase>>
Init == /\ prior = [i \in 1..2 |-> IF i = 1 THEN 1 ELSE 0]
        /\ current = prior
        /\ phase = "ready"
Append == /\ phase = "ready"
          /\ current' = [current EXCEPT ![2] = 2]
          /\ prior' = IF Mode = "Shared" THEN current' ELSE prior
          /\ phase' = "appended"
Abort == /\ phase = "appended"
         /\ current' = prior
         /\ phase' = "aborted"
         /\ UNCHANGED prior
Commit == /\ phase = "appended"
          /\ phase' = "committed"
          /\ UNCHANGED <<prior, current>>
Next == Append \/ Abort \/ Commit
Spec == Init /\ [][Next]_vars
PrefixPreserved == prior[1] = 1 /\ current[1] = 1
NoRetainedBinding == phase = "aborted" => prior[2] = 0
=============================================================================
