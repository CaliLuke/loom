--------------------------- MODULE CapturedField ---------------------------
EXTENDS Naturals
CONSTANT Capture
VARIABLES position, application, nullable, input, loweredApplication, accepts, phase
vars == <<position, application, nullable, input, loweredApplication, accepts, phase>>
Init == /\ position \in {"root", "member"}
        /\ application \in {"value", "items"}
        /\ nullable \in BOOLEAN
        /\ input \in {"missing", "null", "value"}
        /\ loweredApplication = "unset" /\ accepts = FALSE /\ phase = "ready"
Next == /\ phase = "ready" /\ phase' = "done"
        /\ loweredApplication' = IF Capture THEN application ELSE "value"
        /\ accepts' = (input # "missing" /\
             (input # "null" \/ (nullable /\ (Capture \/ position = "root"))))
        /\ UNCHANGED <<position, application, nullable, input>>
ApplicationAgreement == phase = "done" => loweredApplication = application
NullAgreement == phase = "done" =>
   accepts = (input # "missing" /\ (input # "null" \/ nullable))
Spec == Init /\ [][Next]_vars
=============================================================================
