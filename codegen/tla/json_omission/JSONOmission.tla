---------------------------- MODULE JSONOmission ----------------------------
EXTENDS TLC
CONSTANT Mode
VARIABLES carrier, input, required, phase, wire, projected
vars == <<carrier, input, required, phase, wire, projected>>
Carriers == {"native", "nilable", "explicit", "custom"}
States(c) == CASE c = "native" -> {"zero", "empty", "value"}
            [] c = "explicit" -> {"absent", "null", "zero", "empty", "value"}
            [] OTHER -> {"absent", "zero", "empty", "value"}
Init == /\ carrier \in Carriers
        /\ input \in States(carrier)
        /\ required \in BOOLEAN
        /\ required => input # "absent"
        /\ phase = "source"
        /\ wire = input
        /\ projected = input
CurrentOmit == ~required /\ (input = "absent" \/ (carrier # "explicit" /\ input = "empty"))
ZeroOmit == ~required /\ (input = "absent" \/ (carrier = "native" /\ input \in {"empty", "zero"}) \/ (carrier = "custom" /\ input = "value"))
CarrierOmit == IF carrier = "custom" THEN CurrentOmit ELSE ~required /\ input = "absent"
Emit == /\ phase = "source"
        /\ phase' = "encoded"
        /\ wire' = IF CASE Mode = "current" -> CurrentOmit
                        [] Mode = "all-zero" -> ZeroOmit
                        [] OTHER -> CarrierOmit
                   THEN "absent" ELSE input
        /\ projected' = IF carrier = "custom" THEN "unsupported" ELSE IF (IF Mode = "tags-only" THEN CurrentOmit ELSE
                            CASE Mode = "current" -> CurrentOmit
                              [] Mode = "all-zero" -> ZeroOmit
                              [] OTHER -> CarrierOmit)
                        THEN "absent" ELSE input
        /\ UNCHANGED <<carrier, input, required>>
Spec == Init /\ [][Emit]_vars
PreserveSupplied == phase = "encoded" /\ carrier # "custom" /\ input # "absent" => wire = input
ProjectionAgreement == phase = "encoded" /\ carrier # "custom" => projected = wire
CustomBoundary == phase = "encoded" /\ carrier = "custom" =>
                  /\ wire = IF CurrentOmit THEN "absent" ELSE input
                  /\ projected = "unsupported"
OmitAbsence == phase = "encoded" /\ input = "absent" => wire = "absent"
=============================================================================
