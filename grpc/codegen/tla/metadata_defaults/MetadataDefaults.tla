------------------------ MODULE MetadataDefaults ------------------------
EXTENDS TLC
CONSTANT Mode
VARIABLES input, required, hasDefault, decoded, done
vars == <<input, required, hasDefault, decoded, done>>
Inputs == {"absent", "zero", "empty", "ordinary", "malformed"}
Init == /\ input \in Inputs
        /\ required \in BOOLEAN
        /\ hasDefault \in BOOLEAN
        /\ decoded = "uninitialized"
        /\ done = FALSE
Decode == /\ ~done
          /\ decoded' = IF input = "absent"
                         THEN IF required THEN "missing"
                              ELSE IF hasDefault /\ Mode = "presence"
                                   THEN "default" ELSE "absent"
                         ELSE IF input = "malformed" THEN "invalid" ELSE input
          /\ done' = TRUE
          /\ UNCHANGED <<input, required, hasDefault>>
Next == Decode
Spec == Init /\ [][Next]_vars
DefaultOnAbsence == done /\ input = "absent" /\ ~required /\ hasDefault => decoded = "default"
SuppliedPreserved == done /\ input \in {"zero", "empty", "ordinary"} => decoded = input
RequiredPreserved == done /\ input = "absent" /\ required => decoded = "missing"
MalformedPreserved == done /\ input = "malformed" => decoded = "invalid"
=============================================================================
