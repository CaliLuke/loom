----------------------- MODULE ResponseBody -----------------------
EXTENDS TLC, Naturals
CONSTANT Checked
VARIABLES raw, success, restore, readError, closeError, phase, closes, errors
vars == <<raw, success, restore, readError, closeError, phase, closes, errors>>
Init == /\ raw \in BOOLEAN /\ success \in BOOLEAN /\ restore \in BOOLEAN
        /\ readError \in BOOLEAN /\ closeError \in BOOLEAN
        /\ phase = "decode" /\ closes = 0 /\ errors = {}
Decode == /\ phase = "decode" /\ phase' = "close"
          /\ errors' = IF readError THEN {"read"} ELSE IF ~success THEN {"decode"} ELSE {}
          /\ UNCHANGED <<raw, success, restore, readError, closeError, closes>>
Transferred == raw /\ success /\ ~readError
Close == /\ phase = "close" /\ phase' = "done"
         /\ closes' = IF Transferred THEN 0 ELSE IF Checked \/ ~restore THEN 1 ELSE 0
         /\ errors' = IF Checked /\ ~Transferred /\ closeError THEN errors \cup {"close"} ELSE errors
         /\ UNCHANGED <<raw, success, restore, readError, closeError>>
Next == Decode \/ Close \/ (phase = "done" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
Ownership == phase = "done" => closes = (IF Transferred THEN 0 ELSE 1)
Failures == phase = "done" => /\ (readError => "read" \in errors)
                              /\ (~Transferred /\ closeError => "close" \in errors)
=============================================================================
