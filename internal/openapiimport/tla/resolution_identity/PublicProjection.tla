------------------------- MODULE PublicProjection -------------------------
EXTENDS Naturals
CONSTANTS DisplayPolicy, LookupPolicy, OriginPolicy, TimingPolicy, ParsePolicy, RolePolicy
VARIABLES secret, moved, external, raw, lookedUp, display, origin, family, boundary, phase
vars == <<secret, moved, external, raw, lookedUp, display, origin, family, boundary, phase>>
Safe(value) == IF value = "credential-key" THEN "redacted-key" ELSE value
Init == /\ boundary \in {"node", "members", "components", "mapping"}
        /\ family \in {"analysis", "planner", "syntax"}
        /\ secret \in BOOLEAN /\ moved \in BOOLEAN /\ external \in BOOLEAN
        /\ raw = IF secret THEN "credential-key" ELSE "ordinary-key"
        /\ lookedUp = FALSE /\ display = "" /\ origin = FALSE /\ phase = "lookup"
Lookup == /\ phase = "lookup"
          /\ lookedUp' = ((IF LookupPolicy = "raw" THEN raw ELSE Safe(raw)) = raw)
          /\ phase' = "publish"
          /\ UNCHANGED <<secret, moved, external, raw, display, origin, family, boundary>>
Publish == /\ phase = "publish"
           /\ display' = IF ParsePolicy = "libraryFirst" /\ family = "syntax" THEN "redacted-key"
                        ELSE IF family = "syntax" /\
                           ((RolePolicy = "collectionsRaw" /\ boundary \in {"members", "components"}) \/
                            (RolePolicy = "mappingSplit" /\ boundary = "mapping")) THEN raw
                        ELSE IF family = "syntax" /\ boundary = "mapping" THEN "safe-reason"
                        ELSE IF DisplayPolicy = "safe" THEN Safe(raw) ELSE raw
           /\ origin' = ((external \/ (OriginPolicy = "occurrence" /\ moved)) /\ (TimingPolicy = "final" \/ family = "analysis"))
           /\ phase' = "done"
           /\ UNCHANGED <<secret, moved, external, raw, lookedUp, family, boundary>>
Next == Lookup \/ Publish \/ (phase = "done" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
IdentityPreserved == phase = "done" => lookedUp
NoCredentialDisclosure == phase = "done" => display # "credential-key"
MovedOriginRetained == phase = "done" => (origin = (external \/ moved))
NoOrdinaryLoss == phase = "done" /\ ~secret /\ boundary # "mapping" => display = raw
MappingReasonOnly == phase = "done" /\ family = "syntax" /\ boundary = "mapping" /\ RolePolicy = "complete" => display = "safe-reason"
=============================================================================
