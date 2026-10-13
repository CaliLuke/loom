---------------------- MODULE Construction ----------------------
EXTENDS Naturals, Sequences, TLC
CONSTANTS EagerPublication, Rollback, ReverseCleanup
VARIABLES enabled, failAt, stage, acquired, globals, attempted, cleanupErrors, phase
vars == <<enabled, failAt, stage, acquired, globals, attempted, cleanupErrors, phase>>
Reverse(s) == [j \in 1..Len(s) |-> s[Len(s)-j+1]]
Init == /\ enabled \in SUBSET (1..3)
        /\ failAt \in 0..3
        /\ stage = 1
        /\ acquired = <<>> /\ globals = <<>>
        /\ attempted = <<>> /\ cleanupErrors = {}
        /\ phase = "construct"
Skip == /\ phase = "construct" /\ stage <= 3 /\ stage \notin enabled
        /\ stage' = stage + 1
        /\ UNCHANGED <<enabled, failAt, acquired, globals, attempted, cleanupErrors, phase>>
Acquire == /\ phase = "construct" /\ stage <= 3 /\ stage \in enabled
           /\ stage # failAt
           /\ acquired' = Append(acquired, stage)
           /\ globals' = IF EagerPublication THEN Append(globals, stage) ELSE globals
           /\ stage' = stage + 1
           /\ UNCHANGED <<enabled, failAt, attempted, cleanupErrors, phase>>
Fail == /\ phase = "construct" /\ stage <= 3 /\ stage \in enabled
        /\ stage = failAt
        /\ phase' = IF Rollback THEN "rollback" ELSE "failed"
        /\ UNCHANGED <<enabled, failAt, stage, acquired, globals, attempted, cleanupErrors>>
Commit == /\ phase = "construct" /\ stage = 4
          /\ globals' = acquired /\ phase' = "success"
          /\ UNCHANGED <<enabled, failAt, stage, acquired, attempted, cleanupErrors>>
Cleanup == /\ phase = "rollback" /\ Len(attempted) < Len(acquired)
           /\ LET next == IF ReverseCleanup
                           THEN acquired[Len(acquired)-Len(attempted)]
                           ELSE acquired[Len(attempted)+1]
               IN /\ attempted' = Append(attempted, next)
                  /\ cleanupErrors' \in {cleanupErrors, cleanupErrors \cup {next}}
           /\ UNCHANGED <<enabled, failAt, stage, acquired, globals, phase>>
Finish == /\ phase = "rollback" /\ Len(attempted) = Len(acquired)
          /\ phase' = "failed"
          /\ UNCHANGED <<enabled, failAt, stage, acquired, globals, attempted, cleanupErrors>>
Next == Skip \/ Acquire \/ Fail \/ Commit \/ Cleanup \/ Finish
Spec == Init /\ [][Next]_vars
FailedGlobals == phase = "failed" => globals = <<>>
AllCleanupAttempted == phase = "failed" => Len(attempted) = Len(acquired)
CleanupOrder == attempted = SubSeq(Reverse(acquired), 1, Len(attempted))
SuccessGlobals == phase = "success" => globals = acquired
=================================================================
