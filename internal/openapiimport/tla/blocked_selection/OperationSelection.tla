------------------------ MODULE OperationSelection ------------------------
EXTENDS Naturals, Sequences
CONSTANTS SourcePolicy, RootPolicy
VARIABLES location, spelling, security, supplied, rootFailure, retained, projected, phase
vars == <<location, spelling, security, supplied, rootFailure, retained, projected, phase>>
Authored == IF location = "standard" THEN "/get"
            ELSE "/additionalOperations/" \o spelling
ProjectionAddress == IF SourcePolicy = "authored" THEN Authored
                     ELSE IF location = "standard" THEN "/get" ELSE "/" \o spelling
FilterAddress == IF SourcePolicy = "authored" THEN Authored
                 ELSE IF location = "standard" THEN "/get" ELSE "/purge"
Required == security # "public"
Available == IF security = "inherit" THEN ~rootFailure ELSE supplied
Init == /\ location \in {"standard", "additional"}
        /\ spelling \in {"PURGE", "purge"}
        /\ security \in {"inherit", "public", "local"}
        /\ supplied \in BOOLEAN /\ rootFailure \in BOOLEAN
        /\ retained = FALSE /\ projected = FALSE /\ phase = "project"
Project == /\ phase = "project"
           /\ projected' = (Required /\ Available /\
                 (security = "inherit" \/ ProjectionAddress = Authored))
           /\ phase' = "filter"
           /\ UNCHANGED <<location, spelling, security, supplied, rootFailure, retained>>
Filter == /\ phase = "filter"
          /\ retained' = ~((rootFailure /\ (RootPolicy = "raw" \/ security = "inherit")) \/
                 (security = "local" /\ (~supplied \/ ProjectionAddress # Authored) /\ FilterAddress = Authored))
          /\ phase' = "done"
          /\ UNCHANGED <<location, spelling, security, supplied, rootFailure, projected>>
Next == Project \/ Filter \/ (phase = "done" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
NoLostSecurity == phase = "done" /\ retained /\ Required => projected
NoFalseBlock == phase = "done" /\ (~Required \/ Available) => retained
=============================================================================
