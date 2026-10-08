---------------------------- MODULE Bearer ----------------------------
EXTENDS TLC
CONSTANT Checked
VARIABLES bearer, raw, accepted, value, done
vars == <<bearer, raw, accepted, value, done>>
Init == /\ bearer \in BOOLEAN /\ raw \in {"valid", "Bearer valid", "Bearer x", "Basic valid"}
        /\ accepted = FALSE /\ value = "" /\ done = FALSE
Extract == IF bearer /\ raw = "Bearer valid" THEN "valid"
           ELSE IF bearer /\ raw = "Bearer x" THEN "x" ELSE raw
Framed == ~bearer \/ raw \in {"Bearer valid", "Bearer x"}
Valid(v) == v = "valid"
Decode == /\ ~done /\ done' = TRUE
          /\ accepted' = (IF Checked THEN Framed /\ Valid(Extract) ELSE Valid(raw))
          /\ value' = (IF Checked THEN Extract ELSE raw)
          /\ UNCHANGED <<bearer, raw>>
Next == Decode \/ (done /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
RoundTrip == done /\ bearer /\ raw = "Bearer valid" => accepted
Contract == done => accepted = (Framed /\ Valid(Extract))
=============================================================================
