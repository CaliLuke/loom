---------------------------- MODULE ShutdownClose ----------------------------
EXTENDS TLC
CONSTANT Monotonic
VARIABLES closing, finished, observed, shutdown, closeKind
vars == <<closing, finished, observed, shutdown, closeKind>>
Init == /\ closing = FALSE /\ finished = FALSE /\ observed = FALSE
        /\ shutdown = FALSE /\ closeKind = FALSE
LocalClose == /\ ~closing /\ closing' = TRUE
              /\ UNCHANGED <<finished, observed, shutdown, closeKind>>
Observe == /\ ~observed /\ observed' = TRUE /\ closing' = TRUE
           /\ closeKind' = IF closing THEN closeKind ELSE TRUE
           /\ shutdown' = IF Monotonic THEN TRUE ELSE shutdown
           /\ UNCHANGED finished
\* close joins the watcher before publishing completion.
Finish == /\ closing /\ observed /\ ~finished /\ finished' = TRUE
          /\ shutdown' = IF Monotonic THEN shutdown ELSE closeKind
          /\ UNCHANGED <<closing, observed, closeKind>>
Next == LocalClose \/ Observe \/ Finish
Spec == Init /\ [][Next]_vars
Recorded == observed /\ finished => shutdown
=============================================================================
