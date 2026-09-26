-------------------------- MODULE ClientClose --------------------------
EXTENDS Naturals
CONSTANT GuardClosed
VARIABLES closed, conn, lock, getting, closing, closeDone
vars == <<closed, conn, lock, getting, closing, closeDone>>
Init == /\ closed = FALSE /\ conn = FALSE /\ lock = "free"
        /\ getting = FALSE /\ closing = FALSE /\ closeDone = FALSE
GetStart == /\ lock = "free"
            /\ ~(GuardClosed /\ closed)
            /\ lock' = "get" /\ getting' = TRUE
            /\ UNCHANGED <<closed, conn, closing, closeDone>>
GetFinish == /\ getting /\ lock = "get"
             /\ conn' = TRUE /\ lock' = "free" /\ getting' = FALSE
             /\ UNCHANGED <<closed, closing, closeDone>>
CloseStart == /\ ~closed /\ closed' = TRUE /\ closing' = TRUE
              /\ UNCHANGED <<conn, lock, getting, closeDone>>
CloseFinish == /\ closing /\ lock = "free"
               /\ conn' = FALSE /\ closing' = FALSE /\ closeDone' = TRUE
               /\ UNCHANGED <<closed, lock, getting>>
Release == /\ conn /\ conn' = FALSE
           /\ UNCHANGED <<closed, lock, getting, closing, closeDone>>
Next == GetStart \/ GetFinish \/ CloseStart \/ CloseFinish \/ Release
Spec == Init /\ [][Next]_vars
NoConnectionAfterClose == closeDone => ~conn
=============================================================================
