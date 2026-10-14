---------------- MODULE Admission ----------------
EXTENDS TLC
CONSTANT Strategy
VARIABLES fixture, observed, readStarted, phase, accepted
vars == <<fixture, observed, readStarted, phase, accepted>>
Init == /\ fixture \in {"pending", "published"}
        /\ observed = (Strategy = "Controlled" /\ fixture = "published")
        /\ readStarted = FALSE /\ phase = "committed" /\ accepted = FALSE
Publish == /\ Strategy = "Current" /\ phase = "committed" /\ ~observed
           /\ observed' = TRUE
           /\ UNCHANGED <<fixture, readStarted, phase, accepted>>
Observe == /\ phase = "committed"
           /\ readStarted' = ~observed /\ phase' = "observed"
           /\ UNCHANGED <<fixture, observed, accepted>>
Check == /\ phase = "observed" /\ phase' = "checked"
         /\ accepted' = IF Strategy = "Current" THEN readStarted
                         ELSE readStarted = (fixture = "pending")
         /\ UNCHANGED <<fixture, observed, readStarted>>
Next == Publish \/ Observe \/ Check
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
ValidFixture == phase = "checked" => accepted
Completion == <> (phase = "checked")
====================================================
