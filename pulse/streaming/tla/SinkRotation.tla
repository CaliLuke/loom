--------------------------- MODULE SinkRotation ---------------------------
EXTENDS FiniteSets
CONSTANT Fixed
Streams == {"first", "second"}
VARIABLES active, members, current, owned, phase, panicked
vars == <<active, members, current, owned, phase, panicked>>
Init == /\ active = Streams /\ members = [s \in Streams |-> {"old"}]
        /\ current = "old" /\ owned = {"old"} /\ phase = "idle"
        /\ panicked = FALSE
Begin == /\ phase = "idle" /\ active # {}
         /\ owned' = IF Fixed THEN owned \cup {"new"} ELSE owned
         /\ phase' = "prepare"
         /\ UNCHANGED <<active, members, current, panicked>>
Commit == /\ phase = "prepare"
          /\ \E keepOld \in BOOLEAN:
              members' = [s \in Streams |->
                IF s \in active /\ (Fixed \/ s = (CHOOSE a \in active: TRUE))
                THEN (IF Fixed /\ ~keepOld THEN {} ELSE members[s]) \cup {"new"}
                ELSE members[s]]
          /\ current' = "new" /\ phase' = "done"
          /\ owned' = IF Fixed THEN owned ELSE {"new"}
          /\ UNCHANGED <<active, panicked>>
Fail == /\ phase = "prepare"
        /\ \E cleanupFails \in BOOLEAN:
            members' = [s \in Streams |->
              IF cleanupFails /\ s \in active THEN members[s] \cup {"new"} ELSE members[s]]
        /\ current' = IF Fixed THEN current ELSE ""
        /\ phase' = "done"
        /\ UNCHANGED <<active, owned, panicked>>
Remove(s) == /\ phase \in {"idle", "done"} /\ s \in active
             /\ members' = [members EXCEPT ![s] = @ \ (IF Fixed THEN owned ELSE {current})]
             /\ active' = active \ {s}
             /\ UNCHANGED <<current, owned, phase, panicked>>
Empty == /\ active = {} /\ panicked' = ~Fixed
         /\ UNCHANGED <<active, members, current, owned, phase>>
Next == Begin \/ Commit \/ Fail \/ Empty \/ (\E s \in Streams: Remove(s))
Spec == Init /\ [][Next]_vars
CurrentRegistered == phase # "prepare" => \A s \in active: current \in members[s]
NoGhost == \A s \in Streams \ active: members[s] = {}
NoPanic == ~panicked
=============================================================================
