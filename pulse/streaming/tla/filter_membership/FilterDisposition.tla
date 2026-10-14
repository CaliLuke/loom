--------------------- MODULE FilterDisposition ---------------------
EXTENDS Naturals
CONSTANT Mode
VARIABLES kind, stage, pending, otherPending, stored, automatic, failures
vars == <<kind, stage, pending, otherPending, stored, automatic, failures>>
Init == /\ kind \in {"accepted", "filtered", "malformed"}
        /\ pending \in BOOLEAN /\ otherPending = TRUE /\ stored = TRUE
        /\ stage = "received" /\ automatic = FALSE /\ failures = 0
Classify ==
 /\ stage = "received"
 /\ stage' = IF kind = "accepted" THEN "application"
             ELSE IF Mode = "current" /\ kind = "filtered" THEN "done" ELSE "ack"
 /\ UNCHANGED <<kind, pending, otherPending, stored, automatic, failures>>
AckSuccess == /\ stage = "ack" /\ stage' = "done" /\ pending' = FALSE
              /\ automatic' = TRUE
              /\ UNCHANGED <<kind, otherPending, stored, failures>>
AckFailure == /\ stage = "ack" /\ failures = 0
              /\ stage' = "reclaim" /\ failures' = 1
              /\ UNCHANGED <<kind, pending, otherPending, stored, automatic>>
LostReply == /\ stage = "ack" /\ failures = 0
             /\ stage' = "reclaim" /\ pending' = FALSE /\ automatic' = TRUE
             /\ failures' = 1 /\ UNCHANGED <<kind, otherPending, stored>>
Reclaim == /\ stage = "reclaim" /\ pending /\ stage' = "ack"
           /\ UNCHANGED <<kind, pending, otherPending, stored, automatic, failures>>
Next == Classify \/ AckSuccess \/ AckFailure \/ LostReply \/ Reclaim
Spec == Init /\ [][Next]_vars /\ WF_vars(Classify) /\ WF_vars(AckSuccess) /\ WF_vars(Reclaim)
NoTerminalOrphan == stage = "done" => ~pending
NoApplicationAck == kind = "accepted" => ~automatic
OtherGroupPreserved == otherPending /\ stored
RejectionLeavesPEL == kind # "accepted" ~> ~pending
=============================================================================
