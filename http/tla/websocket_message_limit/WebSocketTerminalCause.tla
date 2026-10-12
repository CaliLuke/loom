--------------------- MODULE WebSocketTerminalCause ---------------------
EXTENDS Naturals
CONSTANTS Design, CloseOrder
VARIABLES closed, readErr, owner, phase, kind, localUsed
vars == <<closed, readErr, owner, phase, kind, localUsed>>

Init == /\ closed = FALSE /\ readErr = "none" /\ owner = "none"
        /\ phase = "reading" /\ kind \in {"network", "limit"}
        /\ localUsed = FALSE

ReadFailure == /\ phase = "reading" /\ phase' = "failed"
               /\ UNCHANGED <<closed, readErr, owner, kind, localUsed>>

OverflowCleanup ==
    /\ phase = "failed" /\ kind = "limit" /\ phase' = "cleaned"
    /\ closed' = TRUE
    /\ owner' = IF closed THEN owner ELSE "limit"
    /\ readErr' = IF ~closed /\ Design = "revised" THEN "limit" ELSE readErr
    /\ UNCHANGED <<kind, localUsed>>

RecordFailure ==
    /\ (phase = "failed" /\ kind = "network") \/ phase = "cleaned"
    /\ phase' = "done"
    /\ readErr' = IF readErr = "none" /\ (Design = "old" \/ ~closed)
                   THEN kind ELSE readErr
    /\ UNCHANGED <<closed, owner, kind, localUsed>>

LocalClose ==
    /\ ~localUsed
    /\ CloseOrder = "any"
        \/ (CloseOrder = "before" /\ phase = "reading")
        \/ (CloseOrder = "after" /\ phase = "done" /\ kind = "network")
    /\ localUsed' = TRUE /\ closed' = TRUE
    /\ owner' = IF closed THEN owner ELSE "local"
    /\ readErr' = IF ~closed /\ Design = "revised" THEN "none" ELSE readErr
    /\ UNCHANGED <<phase, kind>>

Next == ReadFailure \/ OverflowCleanup \/ RecordFailure \/ LocalClose
Spec == Init /\ [][Next]_vars
LocalCloseOwnsCause == owner = "local" => readErr = "none"
OverflowOwnsCause == owner = "limit" /\ phase = "done" => readErr = "limit"
OwnerIsTerminal == (owner # "none") = closed
=============================================================================
