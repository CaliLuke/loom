------------------------------ MODULE DecodeWork ------------------------------
EXTENDS Naturals
CONSTANT Mode, MaxDepth
VARIABLES depth, visits
vars == <<depth, visits>>
Init == /\ depth = 0 /\ visits = 1
Next == /\ depth < MaxDepth
        /\ depth' = depth + 1
        /\ visits' = IF Mode = "discard" THEN 2 * visits + 1 ELSE visits + 1
Spec == Init /\ [][Next]_vars
SingleDecode == visits = depth + 1
===============================================================================
