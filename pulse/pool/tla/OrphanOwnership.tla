------------------------- MODULE OrphanOwnership -------------------------
EXTENDS Naturals
CONSTANT CheckCurrent
VARIABLES owner, payload, observed, queued, safe
vars == <<owner, payload, observed, queued, safe>>
Init == /\ owner = FALSE /\ payload = 1 /\ observed = 0
        /\ queued = FALSE /\ safe = TRUE
Observe == /\ observed = 0 /\ observed' = payload
           /\ UNCHANGED <<owner, payload, queued, safe>>
Claim == /\ ~owner /\ payload # 0 /\ owner' = TRUE
         /\ UNCHANGED <<payload, observed, queued, safe>>
Stop == /\ payload # 0 /\ owner' = FALSE /\ payload' = 0
        /\ UNCHANGED <<observed, queued, safe>>
Replace == /\ payload # 2 /\ payload' = 2 /\ owner' = FALSE
           /\ UNCHANGED <<observed, queued, safe>>
Sweep == /\ observed # 0 /\ ~queued
         /\ CheckCurrent => (~owner /\ payload = observed)
         /\ queued' = TRUE
         /\ safe' = (~owner /\ payload # 0 /\ payload = observed)
         /\ UNCHANGED <<owner, payload, observed>>
Next == Observe \/ Claim \/ Stop \/ Replace \/ Sweep
Spec == Init /\ [][Next]_vars
CurrentRequeue == safe
=============================================================================
