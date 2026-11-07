--------------------------- MODULE DefaultStorage ---------------------------
EXTENDS TLC
CONSTANT Mode
VARIABLES source, target, input, materialize, named, phase, output, storage, collection, elementStorage
vars == <<source, target, input, materialize, named, phase, output, storage, collection, elementStorage>>
Init == /\ source \in {"native", "optional", "nullable"}
        /\ target \in {"native", "optional", "nullable"}
        /\ (source = "nullable") = (target = "nullable")
        /\ input \in {"absent", "zero", "empty", "value", "null"}
        /\ input = "null" => source = "nullable"
        /\ materialize \in BOOLEAN /\ named \in BOOLEAN
        /\ phase = "source" /\ output = input /\ storage = target
        /\ collection \in BOOLEAN /\ elementStorage = "declared"
\* DefaultForField has already selected a default and admitted only optional
\* source absence. Pointer/deferred/required controls set materialize to FALSE.
Apply == input = "absent" /\ materialize
Convert == /\ phase = "source" /\ phase' = "converted"
           /\ output' = IF Apply /\ ~(Mode \in {"old", "wrap-only"} /\ source = "native" /\ target = "optional") THEN "default" ELSE input
           /\ storage' = CASE Apply /\ Mode = "old" /\ source = "optional" /\ target = "optional" -> "native"
                              [] Apply /\ Mode = "inferred" /\ target # "native" /\ named -> "unnamed"
                              [] OTHER -> target
           /\ elementStorage' = IF Apply /\ collection /\ Mode = "raw-elements" THEN "native" ELSE "declared"
           /\ UNCHANGED <<source, target, input, materialize, named, collection>>
Next == Convert
Spec == Init /\ [][Next]_vars
DeclaredStorage == phase = "converted" => storage = target /\ elementStorage = "declared"
FillAbsence == phase = "converted" /\ Apply => output = "default"
PreserveSupplied == phase = "converted" /\ ~Apply => output = input
=============================================================================
