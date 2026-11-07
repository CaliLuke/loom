-------------------------- MODULE ChildCarrier --------------------------
CONSTANT Policy
VARIABLES parentExtracted, childJSON, childPointer, valueMode, phase
vars == <<parentExtracted, childJSON, childPointer, valueMode, phase>>
Init == /\ parentExtracted \in BOOLEAN /\ childJSON \in BOOLEAN
        /\ childPointer \in BOOLEAN
        /\ valueMode = FALSE /\ phase = "child"
Child == /\ phase = "child"
         /\ valueMode' = CASE Policy = "inherited" -> parentExtracted
                            [] Policy = "reset" -> FALSE
                            [] Policy = "json-value" -> childJSON
                            [] OTHER -> ~childPointer
         /\ phase' = "done"
         /\ UNCHANGED <<parentExtracted, childJSON, childPointer>>
Next == Child
Spec == Init /\ [][Next]_vars
CallMatchesStorage == phase = "done" => valueMode = ~childPointer
=============================================================================
