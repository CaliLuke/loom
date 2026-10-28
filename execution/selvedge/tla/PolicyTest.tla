--------------------------- MODULE PolicyTest ---------------------------
EXTENDS Naturals
CONSTANT AttemptDeadline
VARIABLES phase, attempts
vars == <<phase, attempts>>

Init == /\ phase = "ready"
        /\ attempts = 0

Claim == /\ phase = "ready"
         /\ attempts < 2
         /\ phase' = "running"
         /\ attempts' = attempts + 1

\* A delayed handler need not return before an authored attempt deadline.
Expire == /\ AttemptDeadline
          /\ phase = "running"
          /\ attempts = 1
          /\ phase' = "ready"
          /\ UNCHANGED attempts

\* A permanent result ends retrying only when it wins the attempt outcome.
ReturnPermanent == /\ phase = "running"
                   /\ phase' = "done"
                   /\ UNCHANGED attempts

Next == Claim \/ Expire \/ ReturnPermanent
Spec == Init /\ [][Next]_vars
SingleAttempt == phase = "done" => attempts = 1
=============================================================================
