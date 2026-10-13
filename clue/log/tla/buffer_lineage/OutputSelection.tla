------------------------ MODULE OutputSelection ------------------------
EXTENDS Naturals, TLC
CONSTANT Snapshot
VARIABLES setting, queued, emitted, changed
vars == <<setting, queued, emitted, changed>>
Init == /\ setting = "original"
        /\ queued = "original"
        /\ emitted = "none"
        /\ changed = FALSE
Reconfigure == /\ ~changed
               /\ setting' = "later"
               /\ changed' = TRUE
               /\ UNCHANGED <<queued, emitted>>
Flush == /\ emitted = "none"
         /\ emitted' = IF Snapshot THEN queued ELSE setting
         /\ UNCHANGED <<setting, queued, changed>>
Spec == Init /\ [][Reconfigure \/ Flush]_vars
OriginalOutput == emitted \in {"none", "original"}
=============================================================================
