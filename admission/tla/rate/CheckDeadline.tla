------------------------ MODULE CheckDeadline ------------------------
CONSTANT Mode
VARIABLES phase, expired, signaled, admitted
vars == <<phase, expired, signaled, admitted>>
Init == /\ phase = "backend"
        /\ expired = FALSE
        /\ signaled = FALSE
        /\ admitted = FALSE
Expire == /\ phase # "checked"
          /\ ~expired
          /\ expired' = TRUE
          /\ UNCHANGED <<phase, signaled, admitted>>
Signal == /\ expired
          /\ ~signaled
          /\ signaled' = TRUE
          /\ UNCHANGED <<phase, expired, admitted>>
Answer == /\ phase = "backend"
          /\ phase' = "answered"
          /\ UNCHANGED <<expired, signaled, admitted>>
Check == /\ phase = "answered"
         /\ phase' = "checked"
         /\ admitted' = IF Mode = "err-only" THEN ~signaled
                        ELSE ~signaled /\ ~expired
         /\ UNCHANGED <<expired, signaled>>
Next == Expire \/ Signal \/ Answer \/ Check
Spec == Init /\ [][Next]_vars
DeadlineEnforced == phase = "checked" /\ expired => ~admitted
======================================================================
