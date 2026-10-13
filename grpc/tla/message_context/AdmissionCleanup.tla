--------------------------- MODULE AdmissionCleanup ---------------------------
EXTENDS TLC
CONSTANT Mode
VARIABLES phase, admitted, terminal, registering, registered
vars == <<phase, admitted, terminal, registering, registered>>
Init == /\ phase = "idle" /\ admitted = FALSE /\ terminal = FALSE
        /\ registering = TRUE /\ registered = FALSE
Register == /\ registering /\ registering' = FALSE
            /\ registered' = IF terminal /\ Mode = "owned" THEN FALSE ELSE TRUE
            /\ UNCHANGED <<phase, admitted, terminal>>
RejectCanceled == /\ phase = "idle" /\ phase' = "rejected"
                  /\ terminal' = (Mode = "blanket-receive")
                  /\ UNCHANGED <<admitted, registering, registered>>
Admit == /\ phase = "idle" /\ phase' = "active" /\ admitted' = TRUE
         /\ UNCHANGED <<terminal, registering, registered>>
CancelActive == /\ phase = "active" /\ phase' = "returned" /\ terminal' = TRUE
                /\ registered' = IF Mode = "owned" THEN FALSE ELSE registered
                /\ UNCHANGED <<admitted, registering>>
Next == Register \/ RejectCanceled \/ Admit \/ CancelActive
Spec == Init /\ [][Next]_vars
RejectedOperationKeepsRPC == ~admitted => ~terminal
TerminalDetachesCallbacks == terminal => ~registered
=============================================================================
