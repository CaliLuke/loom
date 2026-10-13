--------------------- MODULE ResponseSourceBinding ---------------------
EXTENDS Naturals
CONSTANT Repair
VARIABLES phase, shape, forwarding, bound, consumed
vars == <<phase, shape, forwarding, bound, consumed>>
Shapes == {"no-result", "named-empty", "explicit-empty", "body", "metadata-only"}
HasReference == shape # "no-result"
IntendedBody == shape \in {"body", "explicit-empty"}
HasBody == IF Repair = "cardinality" THEN shape = "body" ELSE IntendedBody
Capture == shape = "metadata-only"
NeedsResult == HasBody \/ Capture \/ forwarding
Init == /\ phase = "choose" /\ shape = "unset" /\ forwarding = FALSE
        /\ bound = FALSE /\ consumed = FALSE
Choose == /\ phase = "choose"
          /\ \E s \in Shapes, f \in BOOLEAN:
               /\ shape' = s /\ forwarding' = f
          /\ phase' = "bind"
          /\ UNCHANGED <<bound, consumed>>
Bind == /\ phase = "bind"
        /\ bound' = IF Repair = "current" THEN HasReference \/ Capture \/ forwarding
                    ELSE NeedsResult
        /\ consumed' = NeedsResult
        /\ phase' = "done"
        /\ UNCHANGED <<shape, forwarding>>
Next == Choose \/ Bind
NoUnusedResult == phase # "done" \/ (bound => consumed)
SourceAvailable == phase # "done" \/ (consumed => bound)
PreservesAuthoredBody == phase # "done" \/ (HasBody = IntendedBody)
=============================================================================
