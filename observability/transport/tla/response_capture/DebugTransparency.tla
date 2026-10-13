----------------------- MODULE DebugTransparency -----------------------
EXTENDS Naturals, TLC
CONSTANT Mode
VARIABLES actual, captured, accepted, observed, flushes, forwarded, retained, omitted, steps
vars == <<actual, captured, accepted, observed, flushes, forwarded, retained, omitted, steps>>
Init == /\ actual = 0 /\ captured = 0 /\ accepted = 0 /\ observed = 0
        /\ flushes = 0 /\ forwarded = 0 /\ retained = 0 /\ omitted = FALSE /\ steps = 0
Final(code) == code = 101 \/ code >= 200
Commit(old, code) == IF old = 0 THEN code ELSE old
Header(code) == /\ steps < 3 /\ steps' = steps + 1
 /\ actual' = (IF Final(code) THEN Commit(actual, code) ELSE actual)
 /\ captured' = (IF Mode = "legacy-status" THEN code
                  ELSE IF Final(code) THEN Commit(captured, code) ELSE captured)
 /\ UNCHANGED <<accepted, observed, flushes, forwarded, retained, omitted>>
Write(n) == /\ steps < 3 /\ steps' = steps + 1
 /\ actual' = Commit(actual, 200)
 /\ captured' = (IF Mode = "legacy-status" THEN captured ELSE Commit(captured, 200))
 /\ accepted' = accepted + n
 /\ observed' = observed + (IF Mode = "legacy-bytes" THEN 2 ELSE n)
 /\ retained' = (IF Mode = "legacy-cap" THEN retained+n ELSE IF accepted+n > 2 THEN 0 ELSE accepted+n)
 /\ omitted' = (Mode # "legacy-cap" /\ accepted+n > 2)
 /\ UNCHANGED <<flushes, forwarded>>
Flush == /\ steps < 3 /\ steps' = steps + 1
 /\ flushes' = flushes + 1
 /\ forwarded' = forwarded + (IF Mode = "legacy-flush" THEN 0 ELSE 1)
 /\ actual' = Commit(actual, 200)
 /\ captured' = (IF Mode = "legacy-status" THEN captured ELSE Commit(captured, 200))
 /\ UNCHANGED <<accepted, observed, retained, omitted>>
Next == (\E code \in {101,103,201,500}: Header(code))
        \/ (\E n \in {0,1,2}: Write(n)) \/ Flush
        \/ (steps = 3 /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
Status == captured = actual
AcceptedBytes == observed = accepted
FlushTransparency == forwarded = flushes
BoundedCapture == retained <= 2 /\ omitted = (accepted > 2) /\
                  retained = (IF omitted THEN 0 ELSE accepted)
=============================================================================
