----------------------- MODULE DefaultApplication -----------------------
EXTENDS TLC
CONSTANT Mode
VARIABLES carrier, input, selected, materialize, required, phase, output
vars == <<carrier, input, selected, materialize, required, phase, output>>
Carriers == {"native", "pointer", "collection", "bytes", "optional", "nullable"}
States(c) == CASE c = "native" -> {"zero", "value"}
               [] c = "pointer" -> {"absent", "zero", "value"}
               [] c \in {"collection", "bytes"} -> {"absent", "empty", "value"}
               [] c = "optional" -> {"absent", "zero", "empty", "value"}
               [] OTHER -> {"absent", "null", "zero", "empty", "value"}
Init == /\ carrier \in Carriers
        /\ input \in States(carrier)
        /\ selected \in BOOLEAN
        /\ materialize \in BOOLEAN
        /\ required \in BOOLEAN
        \* Actual conversion follows required admission. Invalid missing-required
        \* input is not converted to value storage by the modeled pipeline.
        /\ required => input # "absent"
        /\ phase = "source"
        /\ output = input
Absent == CASE Mode = "zero" -> input = "absent" \/ (carrier = "native" /\ input = "zero")
               [] Mode = "nil-only" -> input = "absent" /\ carrier \in {"pointer", "collection", "bytes"}
               [] Mode = "bytes-kind" -> input = "absent" /\ carrier # "bytes"
               [] Mode = "target-only" -> TRUE
               [] OTHER -> input = "absent" /\ carrier # "native"
Convert == /\ phase = "source"
           /\ phase' = "converted"
           /\ output' = IF selected /\ materialize /\ ~required /\ Absent THEN "default" ELSE input
           /\ UNCHANGED <<carrier, input, selected, materialize, required>>
Next == Convert
Spec == Init /\ [][Next]_vars
PreserveSupplied == phase = "converted" /\ input # "absent" => output = input
FillAbsence == phase = "converted" /\ input = "absent" /\ selected /\ materialize => output = "default"
RetainDeferred == phase = "converted" /\ (~selected \/ ~materialize) => output = input
=============================================================================
