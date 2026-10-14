---------------- MODULE Completion ----------------
EXTENDS TLC
CONSTANT Strategy
VARIABLES queued, canceled, ended, phase, result
vars == <<queued, canceled, ended, phase, result>>
Init == /\ queued = FALSE /\ canceled = FALSE /\ ended = FALSE
        /\ phase = "waiting" /\ result = "none"
Emit == /\ phase = "waiting" /\ ~queued /\ ~ended
        /\ queued' = TRUE
        /\ UNCHANGED <<canceled, ended, phase, result>>
CallbackCancel == /\ Strategy = "Current" /\ queued /\ ~canceled
                  /\ canceled' = TRUE
                  /\ UNCHANGED <<queued, ended, phase, result>>
Deadline == /\ phase = "waiting" /\ ~canceled
            /\ canceled' = TRUE
            /\ UNCHANGED <<queued, ended, phase, result>>
End == /\ ~ended /\ (queued \/ canceled)
       /\ ended' = TRUE
       /\ UNCHANGED <<queued, canceled, phase, result>>
CurrentObserve == /\ Strategy = "Current" /\ phase = "waiting"
                  /\ (queued \/ canceled \/ ended)
                  /\ phase' = "done"
                  /\ result' \in (IF queued THEN {"success"} ELSE {}) \cup
                        (IF canceled \/ ended THEN {"failure"} ELSE {})
                  /\ UNCHANGED <<queued, canceled, ended>>
OwnedObserve == /\ Strategy = "Owned" /\ phase = "waiting"
                /\ (queued \/ canceled \/ ended)
                /\ phase' = "joining" /\ canceled' = TRUE
                /\ UNCHANGED <<queued, ended, result>>
Finish == /\ Strategy = "Owned" /\ phase = "joining" /\ ended
          /\ phase' = "done"
          /\ result' = IF queued THEN "success" ELSE "failure"
          /\ UNCHANGED <<queued, canceled, ended>>
Next == Emit \/ CallbackCancel \/ Deadline \/ End \/ CurrentObserve \/ OwnedObserve \/ Finish
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
ReceivedWins == phase = "done" /\ queued => result = "success"
Joined == Strategy = "Owned" /\ phase = "done" => ended
Completion == <> (phase = "done")
====================================================
