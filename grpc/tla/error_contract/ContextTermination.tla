----------------------- MODULE ContextTermination -----------------------
EXTENDS TLC
CONSTANT Preserve
VARIABLES kind, outer, stage, code, cause
vars == <<kind, outer, stage, code, cause>>
Kinds == {"canceled", "deadline", "mixed", "unanimous", "remote"}
Expected == IF outer THEN "owned" ELSE IF kind = "mixed" THEN "unknown"
            ELSE IF kind = "deadline" THEN "deadline" ELSE "canceled"
Init == /\ kind \in Kinds /\ outer \in BOOLEAN /\ stage = "server"
        /\ code = "pending" /\ cause = "original"
Server == /\ stage = "server"
          /\ code' = IF outer THEN "owned" ELSE IF Preserve THEN Expected
                     ELSE IF kind = "remote" THEN "canceled" ELSE "unknown"
          /\ stage' = "client" /\ UNCHANGED <<kind, outer, cause>>
Client == /\ stage = "client"
          /\ code' = IF ~Preserve /\ ~outer /\ kind = "remote" THEN "unknown" ELSE code
          /\ cause' = IF ~Preserve /\ ~outer /\ kind = "remote" THEN "fault" ELSE cause
          /\ stage' = "done" /\ UNCHANGED <<kind, outer>>
Next == Server \/ Client \/ (stage = "done" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
ServerContract == stage # "server" => code = Expected
ClientCause == stage = "done" => cause = "original"
=============================================================================
