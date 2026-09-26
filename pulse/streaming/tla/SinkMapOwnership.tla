----------------------- MODULE SinkMapOwnership -----------------------
EXTENDS Naturals
CONSTANT Fixed
VARIABLES active, handle, openMaps, member, phase
vars == <<active, handle, openMaps, member, phase>>
Init == /\ active = TRUE /\ handle = TRUE /\ openMaps = 1
        /\ member = TRUE /\ phase = "idle"
StartRemove == /\ phase = "idle" /\ IF Fixed THEN handle ELSE active
               /\ active' = IF Fixed THEN active ELSE FALSE
               /\ phase' = "remove"
               /\ UNCHANGED <<handle, openMaps, member>>
RemoveOK == /\ phase = "remove" /\ member' = FALSE
            /\ active' = FALSE /\ phase' = "destroy"
            /\ UNCHANGED <<handle, openMaps>>
RemoveFail == /\ phase = "remove" /\ phase' = "idle"
              /\ member' \in {member, FALSE} \* reply may be lost
              /\ UNCHANGED <<active, handle, openMaps>>
DestroyFail == /\ phase = "destroy" /\ phase' = "idle"
               /\ UNCHANGED <<active, handle, openMaps, member>>
Finish == /\ phase = "destroy" /\ phase' = "idle"
          /\ handle' = IF Fixed THEN FALSE ELSE handle
          /\ openMaps' = IF Fixed THEN openMaps - 1 ELSE openMaps
          /\ UNCHANGED <<active, member>>
Add == /\ phase = "idle" /\ ~active /\ openMaps < 3
       /\ active' = TRUE /\ member' = TRUE /\ handle' = TRUE
       /\ openMaps' = IF Fixed /\ handle THEN openMaps ELSE openMaps + 1
       /\ UNCHANGED phase
Next == StartRemove \/ RemoveOK \/ RemoveFail \/ DestroyFail \/ Finish \/ Add
Spec == Init /\ [][Next]_vars
NoLostHandle == openMaps = IF handle THEN 1 ELSE 0
Retryable == (phase = "idle" /\ member /\ ~active) => (Fixed /\ handle)
=============================================================================
