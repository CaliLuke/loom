----------------------- MODULE SpanCompletion -----------------------
EXTENDS Integers, TLC
CONSTANT Mode
VARIABLES returned, clientEnded, serverEnded, serverNotified, released, snapshot
vars == <<returned, clientEnded, serverEnded, serverNotified, released, snapshot>>
Init == /\ returned = FALSE /\ clientEnded = FALSE /\ serverEnded = FALSE
        /\ serverNotified = FALSE /\ released \in BOOLEAN /\ snapshot = -1
ReturnRPC == /\ ~returned /\ returned' = TRUE /\ clientEnded' = TRUE
             /\ UNCHANGED <<serverEnded, serverNotified, released, snapshot>>
Release == /\ returned /\ ~released /\ released' = TRUE
           /\ UNCHANGED <<returned, clientEnded, serverEnded, serverNotified, snapshot>>
ServerEnd == /\ released /\ ~serverEnded
             /\ serverEnded' = TRUE /\ serverNotified' = TRUE
             /\ UNCHANGED <<returned, clientEnded, released, snapshot>>
EarlyNotify == /\ Mode = "notify-first" /\ released /\ ~serverNotified
               /\ serverNotified' = TRUE
               /\ UNCHANGED <<returned, clientEnded, serverEnded, released, snapshot>>
Observe == /\ returned /\ snapshot = -1
           /\ (Mode = "old" \/ serverNotified)
           /\ snapshot' = (IF clientEnded THEN 1 ELSE 0) + (IF serverEnded THEN 1 ELSE 0)
           /\ UNCHANGED <<returned, clientEnded, serverEnded, serverNotified, released>>
Next == ReturnRPC \/ Release \/ ServerEnd \/ EarlyNotify \/ Observe
Spec == Init /\ [][Next]_vars
CompleteSnapshot == snapshot = -1 \/ snapshot = 2
=============================================================================
