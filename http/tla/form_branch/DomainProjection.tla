------------------------ MODULE DomainProjection ------------------------
EXTENDS FiniteSets
CONSTANT Policy
VARIABLES raw, closed, explicitOpen, matched, evaluated, phase
vars == <<raw, closed, explicitOpen, matched, evaluated, phase>>
Claims == {"x"}
Unknown == raw \ Claims
Init == /\ raw \in SUBSET {"x", "filter", "other", ""}
        /\ closed \in BOOLEAN
        /\ explicitOpen \in BOOLEAN
        /\ ~(closed /\ explicitOpen)
        /\ matched = FALSE /\ evaluated = {} /\ phase = "project"
Project == /\ phase = "project"
           /\ matched' = ((~closed \/ Unknown = {}) /\
                          (Policy # "collision" \/ "filter" \notin Unknown))
           /\ evaluated' = (raw \cap Claims) \cup (IF explicitOpen THEN Unknown ELSE {})
           /\ phase' = "done"
           /\ UNCHANGED <<raw, closed, explicitOpen>>
Next == Project \/ (phase = "done" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
SchemaAgreement == phase = "done" => matched = (~closed \/ Unknown = {})
PhysicalAnnotations == phase = "done" => evaluated \subseteq raw
ExplicitOpenEvaluates == phase = "done" /\ explicitOpen => evaluated = raw
ImplicitOpenLeavesUnknown == phase = "done" /\ ~explicitOpen => evaluated \cap Unknown = {}
=============================================================================
