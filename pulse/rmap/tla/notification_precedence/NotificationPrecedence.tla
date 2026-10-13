--------------------- MODULE NotificationPrecedence ---------------------
EXTENDS Naturals, TLC
CONSTANT Mode
VARIABLES buffer, owed, outgoing, phase, subscribed, produced, postCloseWrite
vars == <<buffer, owed, outgoing, phase, subscribed, produced, postCloseWrite>>
Strong(a, b) == IF a >= b THEN a ELSE b
Init == /\ buffer = 0 /\ owed = 0 /\ outgoing = 0 /\ phase = "idle"
        /\ subscribed = TRUE /\ produced = 0 /\ postCloseWrite = FALSE
Begin(k) == /\ subscribed /\ phase = "idle" /\ produced < 3
            /\ outgoing' = k /\ owed' = Strong(owed, k)
            /\ phase' = IF Mode = "current" THEN "send" ELSE "take"
            /\ produced' = produced + 1
            /\ UNCHANGED <<buffer, subscribed, postCloseWrite>>
Take == /\ phase = "take"
        /\ outgoing' = IF Mode = "owned" THEN Strong(buffer, outgoing) ELSE outgoing
        /\ buffer' = 0 /\ phase' = "send"
        /\ UNCHANGED <<owed, subscribed, produced, postCloseWrite>>
Publish == /\ phase = "send"
           /\ buffer' = IF buffer = 0 THEN outgoing ELSE buffer
           /\ postCloseWrite' = postCloseWrite \/ ~subscribed
           /\ phase' = "idle" /\ outgoing' = 0
           /\ UNCHANGED <<owed, subscribed, produced>>
Receive == /\ buffer > 0
           /\ owed' = IF buffer >= owed THEN 0 ELSE owed
           /\ buffer' = 0
           /\ UNCHANGED <<outgoing, phase, subscribed, produced, postCloseWrite>>
Close == /\ subscribed /\ phase = "idle"
         /\ subscribed' = FALSE
         /\ UNCHANGED <<buffer, owed, outgoing, phase, produced, postCloseWrite>>
Progress == (\E k \in 1..3: Begin(k)) \/ Take \/ Publish
Next == Progress \/ Receive \/ Close
Spec == Init /\ [][Next]_vars /\ WF_vars(Progress) /\ WF_vars(Receive)
TypeOK == /\ buffer \in 0..3 /\ owed \in 0..3 /\ outgoing \in 0..3
          /\ phase \in {"idle", "take", "send"} /\ produced \in 0..3
          /\ subscribed \in BOOLEAN /\ postCloseWrite \in BOOLEAN
PendingRetained == (subscribed /\ phase = "idle") => buffer >= owed
NoPostCloseWrite == ~postCloseWrite
EventuallyDelivered == (subscribed /\ owed > 0) ~> (~subscribed \/ owed = 0)
=============================================================================
