------------------------ MODULE Annotation ------------------------
EXTENDS Naturals, FiniteSets, TLC
CONSTANT Mode
VARIABLES source, codec, retained, known, outcome
vars == <<source, codec, retained, known, outcome>>
Init == /\ source \in {"resolved", "opaque", "invalid"}
        /\ codec \in {"builtin", "external"}
        /\ retained \in BOOLEAN
        /\ known \in {"valid", "invalid"}
        /\ outcome = "pending"
Opaque == source = "opaque" \/ codec = "external"
Eligible == source # "invalid" /\ retained
Next == /\ outcome = "pending"
        /\ outcome' = IF ~Eligible THEN "rejected"
                      ELSE IF Opaque THEN
                        IF Mode = "old" /\ source # "opaque" THEN "rejected"
                        ELSE IF Mode # "blanket" /\ known = "invalid" THEN "rejected"
                        ELSE "unsupported"
                      ELSE IF known = "valid" THEN "emitted" ELSE "rejected"
        /\ UNCHANGED <<source, codec, retained, known>>
Spec == Init /\ [][Next]_vars
PreservePortable == (outcome # "pending" /\ Eligible /\ known = "valid" /\ Opaque)
                    => outcome = "unsupported"
RejectKnownFailure == (outcome # "pending" /\ known = "invalid") => outcome = "rejected"
NoInventedWire == (Opaque /\ outcome # "pending") => outcome # "emitted"
NoInvalidAdmission == (~Eligible /\ outcome # "pending") => outcome = "rejected"
=============================================================================
