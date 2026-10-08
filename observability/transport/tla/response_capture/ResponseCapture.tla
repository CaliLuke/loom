----------------------- MODULE ResponseCapture -----------------------
EXTENDS Naturals, TLC
CONSTANT Checked
VARIABLES actual, captured, steps
vars == <<actual, captured, steps>>
Init == /\ actual = 0 /\ captured = 0 /\ steps = 0
Final(code) == code = 101 \/ code >= 200
Header(code) == /\ steps < 3 /\ steps' = steps + 1
 /\ actual' = (IF actual = 0 /\ Final(code) THEN code ELSE actual)
 /\ captured' = (IF captured = 0 /\ (~Checked \/ Final(code)) THEN code ELSE captured)
Commit == /\ steps < 3 /\ steps' = steps + 1
 /\ actual' = (IF actual = 0 THEN 200 ELSE actual)
 /\ captured' = (IF captured = 0 THEN 200 ELSE captured)
Next == (\E code \in {101, 103, 200, 500}: Header(code)) \/ Commit
        \/ (steps = 3 /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
Status == captured = actual
=============================================================================
