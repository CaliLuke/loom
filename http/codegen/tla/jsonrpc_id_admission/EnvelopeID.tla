------------------------- MODULE EnvelopeID -------------------------
EXTENDS Naturals
CONSTANT Mode
VARIABLES phase, envelope, params, required, nullable, hasDefault, normalized,
          validated, admitted, delivered
vars == <<phase, envelope, params, required, nullable, hasDefault, normalized,
          validated, admitted, delivered>>
Values == {"absent", "null", "valid", "invalid"}
Default(v) == IF v = "absent" /\ hasDefault /\ ~required THEN "valid" ELSE v
Valid(v) == v = "valid" \/ (v = "absent" /\ ~required) \/ (v = "null" /\ nullable)
Expected == Default(envelope)
Init == /\ phase = "choose"
        /\ envelope = "absent" /\ params = "absent"
        /\ required = FALSE /\ nullable = FALSE /\ hasDefault = FALSE
        /\ normalized = "absent" /\ validated = "absent"
        /\ admitted = FALSE /\ delivered = "absent"
Choose == /\ phase = "choose"
          /\ \E e, p \in Values, r, n, d \in BOOLEAN:
               /\ envelope' = e /\ params' = p
               /\ required' = r /\ nullable' = n /\ hasDefault' = d
          /\ phase' = "normalize"
          /\ UNCHANGED <<normalized, validated, admitted, delivered>>
Normalize == /\ phase = "normalize"
             /\ normalized' = Default(IF Mode = "owned" THEN envelope ELSE params)
             /\ phase' = "validate"
             /\ UNCHANGED <<envelope, params, required, nullable, hasDefault,
                            validated, admitted, delivered>>
Validate == /\ phase = "validate"
            /\ admitted' = Valid(normalized)
            /\ validated' = normalized
            /\ phase' = "deliver"
            /\ UNCHANGED <<envelope, params, required, nullable, hasDefault,
                           normalized, delivered>>
Deliver == /\ phase = "deliver"
           /\ delivered' = IF Mode = "owned" \/ (Mode = "current" /\ envelope \in {"absent", "null"})
                            THEN normalized ELSE IF Mode = "late" THEN Expected ELSE envelope
           /\ admitted' = admitted /\ (Mode # "late" \/ Valid(Expected))
           /\ validated' = IF Mode = "late" THEN Expected ELSE validated
           /\ phase' = "done"
           /\ UNCHANGED <<envelope, params, required, nullable, hasDefault,
                          normalized>>
Next == Choose \/ Normalize \/ Validate \/ Deliver
ExactAdmission == phase # "done" \/ (admitted = Valid(Expected))
CheckedValue == phase # "done" \/ ~admitted \/ delivered = validated
CanonicalSource == phase # "done" \/ ~admitted \/ delivered = Expected
=====================================================================
