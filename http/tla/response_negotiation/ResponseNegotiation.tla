---------------------- MODULE ResponseNegotiation ----------------------
EXTENDS TLC
CONSTANT AddVary
VARIABLES firstAccept, nextAccept, fixed, phase, cached, varies, received
vars == <<firstAccept, nextAccept, fixed, phase, cached, varies, received>>
Accepts == {"json", "html", "default"}
Representation(accept) == IF fixed \/ accept # "html" THEN "json" ELSE "html"
Init == /\ firstAccept \in Accepts
        /\ nextAccept \in Accepts
        /\ fixed \in BOOLEAN
        /\ phase = "prime"
        /\ cached = "none"
        /\ varies = FALSE
        /\ received = "none"
Prime == /\ phase = "prime"
         /\ cached' = Representation(firstAccept)
         /\ varies' = (AddVary /\ ~fixed)
         /\ phase' = "request"
         /\ UNCHANGED <<firstAccept, nextAccept, fixed, received>>
Request == /\ phase = "request"
           /\ received' = IF ~varies \/ nextAccept = firstAccept
                          THEN cached ELSE Representation(nextAccept)
           /\ phase' = "done"
           /\ UNCHANGED <<firstAccept, nextAccept, fixed, cached, varies>>
Next == Prime \/ Request
Spec == Init /\ [][Next]_vars
CorrectRepresentation == phase = "done" => received = Representation(nextAccept)
NoHTMLSubstitution == (phase = "done" /\ Representation(nextAccept) = "json")
                      => received # "html"
=============================================================================
