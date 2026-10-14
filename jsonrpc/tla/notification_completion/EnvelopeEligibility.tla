---------------------- MODULE EnvelopeEligibility ----------------------
EXTENDS Naturals
CONSTANT Policy
VARIABLES phase, transport, valid, hasID, replies, dispatched
vars == <<phase, transport, valid, hasID, replies, dispatched>>
Notification == valid /\ ~hasID
Init == /\ phase = "received"
        /\ transport \in {"http", "sse", "websocket"}
        /\ valid \in BOOLEAN
        /\ hasID \in BOOLEAN
        /\ replies = 0
        /\ dispatched = FALSE
Handle == /\ phase = "received"
          /\ phase' = "done"
          /\ dispatched' = valid
          /\ replies' = IF Policy = "current" /\ transport # "http"
                         THEN IF hasID THEN 1 ELSE 0
                         ELSE IF Notification THEN 0 ELSE 1
          /\ UNCHANGED <<transport, valid, hasID>>
Next == Handle \/ (phase = "done" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
InvalidRequestsReply == phase = "done" /\ ~valid => replies = 1
NotificationsSuppress == Notification => replies = 0
InvalidNeverDispatch == ~valid => ~dispatched
RequestsReply == phase = "done" /\ hasID => replies = 1
=============================================================================
