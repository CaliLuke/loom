------------------- MODULE NotificationCompletion -------------------
EXTENDS Naturals
CONSTANT Policy
VARIABLES phase, idKind, streaming, origin, sendFails, terminal, replies,
          invoked, barrier, reason
vars == <<phase,idKind,streaming,origin,sendFails,terminal,replies,invoked,barrier,reason>>
HasID == idKind # "absent"
RequestFailure == origin \in {"params", "service"}
TransportFailure == origin \in {"read", "context", "write"} \/ (RequestFailure /\ HasID /\ sendFails)
Recoverable == RequestFailure /\ ~TransportFailure
Init == /\ phase = "first"
        /\ idKind \in {"absent", "string", "null"}
        /\ streaming \in BOOLEAN
        /\ origin \in {"params", "service", "read", "context", "write"}
        /\ sendFails \in BOOLEAN
        /\ terminal = FALSE /\ replies = 0 /\ invoked = FALSE
        /\ barrier = FALSE /\ reason = "none"
CompleteFirst == /\ phase = "first"
                 /\ phase' = "followup"
                 /\ terminal' = IF Policy = "blanket" /\ ~HasID THEN FALSE
                                ELSE TransportFailure \/ (Policy = "current" /\ streaming /\ origin = "params" /\ ~HasID)
                 /\ replies' = IF RequestFailure /\ HasID /\ ~sendFails THEN 1 ELSE 0
                 /\ invoked' = (origin = "service")
                 /\ reason' = IF origin = "params" THEN "invalid-params"
                              ELSE IF origin = "service" THEN "handler-error" ELSE "transport-error"
                 /\ UNCHANGED <<idKind,streaming,origin,sendFails,barrier>>
Followup == /\ phase = "followup"
            /\ phase' = "done"
            /\ barrier' = ~terminal
            /\ UNCHANGED <<idKind,streaming,origin,sendFails,terminal,replies,invoked,reason>>
Next == CompleteFirst \/ Followup \/ (phase = "done" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
NoNotificationResponse == ~HasID => replies = 0
InvalidParamsNeverInvoke == origin = "params" => ~invoked
KeepReason == phase # "first" /\ origin = "params" => reason = "invalid-params"
RecoverableConnection == phase = "done" /\ Recoverable => barrier
TransportFailureEscapes == phase # "first" /\ TransportFailure => terminal
=============================================================================
