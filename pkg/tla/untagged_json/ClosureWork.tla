------------------------------ MODULE ClosureWork ------------------------------
EXTENDS Naturals
CONSTANT Mode, MaxDepth
VARIABLES depth, visits
vars == <<depth, visits>>
Init == /\ depth = 0 /\ visits = 1
Next == /\ depth < MaxDepth
        /\ depth' = depth + 1
        /\ visits' = IF Mode = "rematch" THEN 2 * visits + 2 ELSE visits + 2
Spec == Init /\ [][Next]_vars
LinearVisits == visits <= 2 * depth + 1
===============================================================================
