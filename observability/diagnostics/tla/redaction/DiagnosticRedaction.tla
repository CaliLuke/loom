---------------------- MODULE DiagnosticRedaction ----------------------
EXTENDS TLC
CONSTANT Mode
VARIABLES location, phase, supplied, captured, output
vars == <<location, phase, supplied, captured, output>>
Locations == {"camelCase", "customQuery", "customBody", "scalarBody", "path"}
Init == /\ location \in Locations
        /\ phase = "configure"
        /\ supplied = TRUE
        /\ captured = FALSE
        /\ output = "pending"
Configure == /\ phase = "configure"
             /\ captured' = CASE Mode = "NamesOnly" -> FALSE
                              [] Mode = "LogicalName" -> location = "customQuery"
                              [] OTHER -> supplied
             /\ phase' = "capture"
             /\ UNCHANGED <<location, supplied, output>>
Capture == /\ phase = "capture"
           /\ phase' = "complete"
           /\ UNCHANGED <<location, supplied, captured, output>>
MutateCaller == /\ phase = "complete"
                /\ supplied
                /\ supplied' = FALSE
                /\ captured' = IF Mode = "Borrowed" THEN FALSE ELSE captured
                /\ UNCHANGED <<location, phase, output>>
Complete == /\ phase = "complete"
            /\ output' = IF captured THEN "masked" ELSE "secret"
            /\ phase' = "done"
            /\ UNCHANGED <<location, supplied, captured>>
Next == Configure \/ Capture \/ MutateCaller \/ Complete
Spec == Init /\ [][Next]_vars
NoCredentialDisclosure == output # "secret"
=============================================================================
