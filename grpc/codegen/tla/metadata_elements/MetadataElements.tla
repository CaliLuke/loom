-------------------- MODULE MetadataElements --------------------
EXTENDS Naturals
CONSTANT Policy
VARIABLES named, constrained, hasDefault, kind, storage, codec, encodeCast,
          constructorCast, validationStorage, validationKept, defaultStorage, resolved
vars == <<named, constrained, hasDefault, kind, storage, codec, encodeCast,
          constructorCast, validationStorage, validationKept, defaultStorage, resolved>>
Kinds == {"bytes", "string", "boolean", "uint32", "float64"}
Init == /\ named \in BOOLEAN /\ constrained \in BOOLEAN /\ hasDefault \in BOOLEAN
        /\ kind \in Kinds
        /\ storage = "unset" /\ codec = "unset" /\ encodeCast = FALSE
        /\ constructorCast = FALSE /\ validationStorage = "unset"
        /\ validationKept = FALSE /\ defaultStorage = "unset" /\ resolved = FALSE
Resolve == /\ ~resolved
           /\ storage' = IF named /\ Policy \in {"current", "codec-only"}
                          THEN "unqualified-nominal" ELSE "primitive"
           /\ codec' = IF named /\ Policy = "current" THEN "unsupported" ELSE "primitive"
           /\ encodeCast' = (~named \/ Policy = "owned")
           /\ constructorCast' = (~named \/ Policy \in {"storage-only", "owned"})
           /\ validationStorage' = IF named /\ Policy # "owned" THEN "nominal" ELSE "primitive"
           /\ validationKept' = ~(named /\ Policy = "storage-only")
           /\ defaultStorage' = IF named /\ Policy # "owned" THEN "nominal" ELSE "primitive"
           /\ resolved' = TRUE
           /\ UNCHANGED <<named, constrained, hasDefault, kind>>
Next == Resolve \/ (resolved /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
ResolvableVariables == resolved => storage = "primitive"
CompleteCodec == resolved => codec = "primitive"
SourceConversion == resolved /\ named /\ kind \in {"string", "boolean", "float64"} => encodeCast
ServiceIdentity == resolved /\ named => constructorCast
ConstraintAgreement == resolved /\ constrained => validationKept /\ validationStorage = storage
DefaultAgreement == resolved /\ hasDefault => defaultStorage = storage
=============================================================================
