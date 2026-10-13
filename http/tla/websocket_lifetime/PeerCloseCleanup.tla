-------------------- MODULE PeerCloseCleanup --------------------
EXTENDS Naturals
CONSTANT Policy
VARIABLES acknowledged, terminal, released, cleanup, physicalFailure,
          writeFailure, dataAttempted, dataError
vars == <<acknowledged, terminal, released, cleanup, physicalFailure,
          writeFailure, dataAttempted, dataError>>
Init == /\ acknowledged = FALSE
        /\ terminal = FALSE
        /\ released = FALSE
        /\ cleanup = {}
        /\ physicalFailure \in BOOLEAN
        /\ writeFailure \in BOOLEAN
        /\ dataAttempted = FALSE
        /\ dataError = ""
PeerAck == /\ ~acknowledged /\ ~terminal
           /\ acknowledged' = TRUE
           /\ UNCHANGED <<terminal,released,cleanup,physicalFailure,writeFailure,dataAttempted,dataError>>
WriteError == IF acknowledged THEN {"already-sent"}
              ELSE IF writeFailure THEN {"write-failed"} ELSE {}
PhysicalError == IF physicalFailure THEN {"physical-failed"} ELSE {}
LocalClose == /\ ~terminal
              /\ terminal' = TRUE
              /\ released' = TRUE
              /\ cleanup' = (CASE Policy = "current" -> WriteError
                                  [] Policy = "suppress-all-close-errors" -> {}
                                  [] OTHER -> WriteError \ {"already-sent"}) \cup PhysicalError
              /\ UNCHANGED <<acknowledged,physicalFailure,writeFailure,dataAttempted,dataError>>
ReaderPublishes == /\ acknowledged /\ ~terminal
                   /\ terminal' = TRUE
                   /\ released' = TRUE
                   /\ cleanup' = PhysicalError
                   /\ UNCHANGED <<acknowledged,physicalFailure,writeFailure,dataAttempted,dataError>>
DataWrite == /\ acknowledged /\ ~terminal /\ ~dataAttempted
             /\ dataAttempted' = TRUE
             /\ dataError' = IF Policy = "suppress-all-writes" THEN "" ELSE "already-sent"
             /\ UNCHANGED <<acknowledged,terminal,released,cleanup,physicalFailure,writeFailure>>
Next == PeerAck \/ LocalClose \/ ReaderPublishes \/ DataWrite \/ (terminal /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
NoSpuriousCloseError == terminal => "already-sent" \notin cleanup
PhysicalRelease == terminal => released
KeepPhysicalError == terminal /\ physicalFailure => "physical-failed" \in cleanup
KeepWriteError == terminal /\ ~acknowledged /\ writeFailure => "write-failed" \in cleanup
DataWritesFail == dataAttempted => dataError = "already-sent"
=============================================================================
