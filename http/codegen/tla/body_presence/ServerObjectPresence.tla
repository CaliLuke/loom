-------------------- MODULE ServerObjectPresence --------------------
EXTENDS TLC
CONSTANT Mode
VARIABLES source, requiredBody, requiredMember, selected, bindings,
          phase, carrier, error, envelope, member, converted,
          bound, invoked, unsafe
vars == <<source, requiredBody, requiredMember, selected, bindings,
          phase, carrier, error, envelope, member, converted,
          bound, invoked, unsafe>>
Init == /\ source \in {"absent", "empty", "fields", "malformed", "null"}
        /\ requiredBody \in BOOLEAN /\ requiredMember \in BOOLEAN
        /\ selected \in BOOLEAN
        /\ bindings \in {"none", "optional-absent", "valid", "invalid"}
        /\ phase = "observe" /\ carrier = FALSE /\ error = "none"
        /\ envelope = FALSE /\ member = FALSE /\ converted = FALSE
        /\ bound = FALSE /\ invoked = FALSE /\ unsafe = FALSE
ExpectedError == IF source \in {"malformed", "null"} THEN "decode"
                 ELSE IF source = "absent" /\ requiredBody THEN "missing_body"
                 ELSE IF source = "empty" /\ requiredMember THEN "missing_member"
                 ELSE IF bindings = "invalid" THEN "binding"
                 ELSE "none"
ExpectedPresent == source \in {"empty", "fields"}
Observe == /\ phase = "observe"
           /\ carrier' = (ExpectedPresent \/
                  (source = "absent" /\ ~selected /\ Mode \in {"value", "validate-only"}))
           /\ error' = IF source \in {"malformed", "null"} THEN "decode"
                        ELSE IF source = "absent" /\ requiredBody THEN "missing_body"
                        ELSE "none"
           /\ phase' = IF error' # "none" THEN "done"
                       ELSE IF Mode = "early-return" /\ source = "absent" /\ ~selected
                            THEN "done" ELSE "validate"
           /\ invoked' = (Mode = "early-return" /\ source = "absent" /\ ~requiredBody /\ ~selected)
           /\ UNCHANGED <<source, requiredBody, requiredMember, selected, bindings,
                          envelope, member, converted, bound, unsafe>>
Validate == /\ phase = "validate"
            /\ error' = IF carrier /\ requiredMember /\ source # "fields" /\
                           ~(Mode = "validate-only" /\ source = "absent")
                         THEN "missing_member" ELSE "none"
            /\ phase' = IF error' # "none" THEN "done" ELSE "elements"
            /\ UNCHANGED <<source, requiredBody, requiredMember, selected, bindings,
                           carrier, envelope, member, converted, bound, invoked, unsafe>>
Elements == /\ phase = "elements"
            /\ bound' = (bindings = "valid")
            /\ error' = IF bindings = "invalid" THEN "binding" ELSE "none"
            /\ phase' = IF error' # "none" THEN "done" ELSE "construct"
            /\ UNCHANGED <<source, requiredBody, requiredMember, selected, bindings,
                           carrier, envelope, member, converted, invoked, unsafe>>
Construct == /\ phase = "construct"
             /\ unsafe' = (Mode = "pointer-only" /\ ~carrier /\ ~selected)
             /\ member' = carrier
             /\ converted' = carrier
             /\ envelope' = (selected \/ bindings # "none" \/ carrier)
             /\ invoked' = ~unsafe'
             /\ phase' = "done"
             /\ UNCHANGED <<source, requiredBody, requiredMember, selected, bindings,
                            carrier, error, bound>>
Next == Observe \/ Validate \/ Elements \/ Construct
Spec == Init /\ [][Next]_vars
SafeConstruction == ~unsafe
Outcome == phase = "done" => error = ExpectedError /\ invoked = (ExpectedError = "none")
BodyPresence == phase = "done" /\ invoked => member = ExpectedPresent
EnvelopePresence == phase = "done" /\ invoked =>
                    envelope = (selected \/ bindings # "none" \/ ExpectedPresent)
BindingPreservation == phase = "done" /\ invoked => bound = (bindings = "valid")
BodyConversion == phase = "done" /\ invoked => converted = ExpectedPresent
=============================================================================
