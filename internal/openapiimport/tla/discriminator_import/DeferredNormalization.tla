---------------------- MODULE DeferredNormalization ----------------------
EXTENDS Naturals, Sequences
CONSTANT Mode
VARIABLES phase, validHint, validSibling, hints, errors
vars == <<phase, validHint, validSibling, hints, errors>>

\* Two normalization copies stand for nullable and singleton wrappers. Only
\* the final copy is rendered; graph-wide admission happens after copying.
Init == /\ phase = "copy"
        /\ validHint \in BOOLEAN
        /\ validSibling \in BOOLEAN
        /\ hints = <<"pending", "pending", "pending">>
        /\ errors = <<FALSE, FALSE, FALSE>>

Copy == /\ phase = "copy"
        /\ phase' = "hint"
        /\ UNCHANGED <<validHint, validSibling, hints, errors>>

ResolveHint == /\ phase = "hint"
               /\ hints' = IF Mode = "value-copy"
                              THEN [hints EXCEPT ![1] = IF validHint THEN "valid" ELSE "invalid"]
                              ELSE [i \in 1..3 |-> IF validHint THEN "valid" ELSE "invalid"]
               /\ phase' = "sibling"
               /\ UNCHANGED <<validHint, validSibling, errors>>

ResolveSibling == /\ phase = "sibling"
                  /\ errors' = IF Mode = "shared-and-retargeted"
                                 THEN [errors EXCEPT ![3] = ~validSibling]
                                 ELSE [errors EXCEPT ![1] = ~validSibling]
                  /\ phase' = "done"
                  /\ UNCHANGED <<validHint, validSibling, hints>>

Next == Copy \/ ResolveHint \/ ResolveSibling
Spec == Init /\ [][Next]_vars
HintRetained == phase = "done" /\ validHint => hints[3] = "valid"
AdmissionPreserved == phase = "done" =>
                       ((hints[3] = "valid" /\ ~errors[3]) = (validHint /\ validSibling))
=============================================================================
