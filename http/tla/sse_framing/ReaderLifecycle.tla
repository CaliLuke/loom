--------------------- MODULE ReaderLifecycle ---------------------
EXTENDS Naturals
CONSTANT Mode
VARIABLES reading, closed, stopped, stopWaiting, calls
vars == <<reading, closed, stopped, stopWaiting, calls>>
Init == /\ reading = FALSE /\ closed = FALSE /\ stopped = FALSE
        /\ stopWaiting = FALSE /\ calls = 0
Read == /\ ~closed /\ ~reading /\ ~stopped /\ calls < 2
        /\ reading' = TRUE /\ calls' = calls + 1
        /\ UNCHANGED <<closed,stopped,stopWaiting>>
Complete == /\ reading /\ reading' = FALSE
            /\ UNCHANGED <<closed,stopped,stopWaiting,calls>>
Close == /\ ~closed /\ closed' = TRUE /\ stopWaiting' = TRUE
         /\ stopped' = IF Mode = "unguarded" THEN TRUE ELSE stopped
         /\ UNCHANGED <<reading,calls>>
Stop == /\ stopWaiting /\ ~reading /\ stopped' = TRUE
        /\ stopWaiting' = FALSE /\ UNCHANGED <<reading,closed,calls>>
Next == Read \/ Complete \/ Close \/ Stop \/ UNCHANGED vars
Spec == Init /\ [][Next]_vars
ExclusiveIterator == ~(reading /\ stopped)
ClosedCannotStart == stopped => closed

=================================================================
