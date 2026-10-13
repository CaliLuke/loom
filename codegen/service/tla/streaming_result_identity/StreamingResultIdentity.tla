---------------------- MODULE StreamingResultIdentity ----------------------
EXTENDS Naturals, TLC
CONSTANTS Repair, Cases, Shapes
VARIABLES phase, kind, shape, validated, attribute, nominal, unary
vars == <<phase, kind, shape, validated, attribute, nominal, unary>>
Eligible == kind \in {"mixed", "ordinary", "final"}
Explicit == IF kind = "mixed" THEN "chunk" ELSE
            IF kind = "ordinary" THEN "summary" ELSE "absent"
Expected == IF ~Eligible THEN "absent" ELSE
            IF kind = "mixed" THEN "chunk" ELSE "summary"
Fields(id) == IF id = "absent" THEN {} ELSE
             IF shape = "shared" THEN {"value"} ELSE {id}
Init == /\ phase = "choose" /\ kind = "unset" /\ shape = "unset"
        /\ validated = "absent" /\ attribute = "absent"
        /\ nominal = "absent" /\ unary = "summary"
Choose == /\ phase = "choose"
          /\ kind' \in Cases /\ shape' \in Shapes
          /\ phase' = "analyze"
          /\ UNCHANGED <<validated, attribute, nominal, unary>>
Analyze == /\ phase = "analyze"
           /\ validated' = IF ~Eligible THEN "absent" ELSE
                           IF Repair = "current" THEN "summary" ELSE
                           IF Repair = "explicit-only" THEN Explicit ELSE Expected
           /\ attribute' = IF ~Eligible THEN "absent" ELSE
                           IF Repair \in {"current", "validation-only"} THEN "summary" ELSE
                           IF Repair = "explicit-only" THEN Explicit ELSE Expected
           /\ nominal' = IF ~Eligible THEN "absent" ELSE
                         IF Repair \in {"current", "validation-only", "attributes-only"} THEN "summary" ELSE
                         IF Repair = "explicit-only" THEN Explicit ELSE Expected
           /\ phase' = "done"
           /\ UNCHANGED <<kind, shape, unary>>
Next == Choose \/ Analyze
AuthoredFieldAccepted == phase # "done" \/ Fields(Expected) \subseteq Fields(validated)
AttributeIdentity == phase # "done" \/ attribute = Expected
NominalIdentity == phase # "done" \/ nominal = Expected
OrdinaryBoundary == unary = "summary"
EligibilityUnchanged == phase # "done" \/ (Eligible \equiv nominal # "absent")
=============================================================================
