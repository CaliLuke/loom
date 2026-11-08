---------------------- MODULE PhysicalLayouts ----------------------
EXTENDS Naturals
CONSTANTS IgnoreNative, CaptureTagged
Kinds == {"json", "form", "tagged-form", "nested-json"}
Declared(k) == k \notin {"form", "tagged-form"}
Recorded(k) == IF k = "tagged-form" /\ ~CaptureTagged THEN TRUE ELSE Declared(k)
VARIABLE kind, inherited, converted, phase
vars == <<kind, inherited, converted, phase>>
Init == kind = "json" /\ inherited = TRUE /\ converted = TRUE /\ phase = "idle"
Select == phase = "idle" /\ \E k \in Kinds:
  kind' = k /\ inherited' = TRUE /\ converted' = TRUE /\ phase' = "selected"
Convert == phase = "selected" /\ phase' = "converted"
  /\ converted' = (IF IgnoreNative THEN inherited \/ Recorded(kind) ELSE Recorded(kind))
  /\ UNCHANGED <<kind, inherited>>
Next == Select \/ Convert
Correspondence == phase = "converted" => converted = Declared(kind)
Spec == Init /\ [][Next]_vars
====================================================================
