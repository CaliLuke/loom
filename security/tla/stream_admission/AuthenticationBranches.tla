------------------------ MODULE AuthenticationBranches ------------------------
EXTENDS Naturals, Sequences
CONSTANT Mode
VARIABLES good, branch, scheme, ctx, phase, winner, calls
vars == <<good, branch, scheme, ctx, phase, winner, calls>>
Init == /\ good \in [1..4 -> BOOLEAN]
        /\ branch = 1 /\ scheme = 1 /\ ctx = <<>>
        /\ phase = "auth" /\ winner = 0 /\ calls = <<>>
Step == /\ phase = "auth"
        /\ LET id == 2 * (branch - 1) + scheme
               updated == Append(ctx, branch)
           IN /\ calls' = Append(calls, id)
              /\ IF good[id]
                 THEN /\ ctx' = updated
                      /\ IF scheme = 1
                         THEN /\ scheme' = 2 /\ branch' = branch /\ phase' = phase /\ winner' = winner
                         ELSE /\ scheme' = scheme /\ branch' = branch /\ phase' = "done" /\ winner' = branch
                 ELSE /\ IF branch = 1
                         THEN /\ branch' = 2 /\ scheme' = 1 /\ phase' = phase /\ winner' = winner
                              /\ ctx' = IF Mode = "leak" THEN updated ELSE <<>>
                         ELSE /\ branch' = branch /\ scheme' = scheme /\ phase' = "done" /\ winner' = 0 /\ ctx' = updated
        /\ UNCHANGED good
Next == Step
Spec == Init /\ [][Next]_vars
WinnerIsFirst == phase = "done" => winner =
  (IF good[1] /\ good[2] THEN 1 ELSE IF good[3] /\ good[4] THEN 2 ELSE 0)
WinningContext == winner # 0 => ctx = <<winner, winner>>
ANDOrder == \A i \in 1..Len(calls): calls[i] \in {1,3} \/ (i > 1 /\ calls[i-1] = calls[i]-1)
=============================================================================
