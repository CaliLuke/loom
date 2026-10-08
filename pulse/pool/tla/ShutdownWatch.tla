---------------------------- MODULE ShutdownWatch ----------------------------
EXTENDS TLC
CONSTANT Reconcile
VARIABLES requested, subscribed, pending, checked, closed
vars == <<requested, subscribed, pending, checked, closed>>
Init == /\ requested = FALSE /\ subscribed = FALSE /\ pending = FALSE
        /\ checked = FALSE /\ closed = FALSE
Request == /\ ~requested /\ requested' = TRUE
           /\ pending' = subscribed
           /\ UNCHANGED <<subscribed, checked, closed>>
Subscribe == /\ ~subscribed /\ subscribed' = TRUE
             /\ UNCHANGED <<requested, pending, checked, closed>>
Check == /\ subscribed /\ ~checked /\ checked' = TRUE
         /\ closed' = IF Reconcile THEN requested ELSE closed
         /\ UNCHANGED <<requested, subscribed, pending>>
Notify == /\ checked /\ pending /\ pending' = FALSE
          /\ closed' = requested
          /\ UNCHANGED <<requested, subscribed, checked>>
Next == Request \/ Subscribe \/ Check \/ Notify
Spec == Init /\ [][Next]_vars /\ WF_vars(Request) /\ WF_vars(Subscribe)
        /\ WF_vars(Check) /\ WF_vars(Notify)
EventuallyClosed == <>closed
NoUnrequestedClose == closed => requested
=============================================================================
