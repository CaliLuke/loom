---------------------- MODULE PhysicalLayouts ----------------------
EXTENDS Naturals
CONSTANT IgnoreNative
Kinds == {"json", "form", "nested-json"}
Declared(k) == k # "form"
VARIABLE kind, inherited, converted, phase
vars == <<kind, inherited, converted, phase>>
Init == kind = "json" /\ inherited = TRUE /\ converted = TRUE /\ phase = "idle"
Select == phase = "idle" /\ \E k \in Kinds:
  kind' = k /\ inherited' = TRUE /\ converted' = TRUE /\ phase' = "selected"
Convert == phase = "selected" /\ phase' = "converted"
  /\ converted' = (IF IgnoreNative THEN inherited \/ Declared(kind) ELSE Declared(kind))
  /\ UNCHANGED <<kind, inherited>>
Next == Select \/ Convert
Correspondence == phase = "converted" => converted = Declared(kind)
Spec == Init /\ [][Next]_vars
====================================================================
