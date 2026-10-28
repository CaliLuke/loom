--------------------------- MODULE Requirements ---------------------------
EXTENDS TLC
CONSTANT Flatten
VARIABLES requirements, required, granted
Schemes == {"a", "b"}
Names == {"read", "admin"}
Init == /\ requirements \in SUBSET (SUBSET Schemes)
        /\ required \in [Schemes -> SUBSET Names]
        /\ granted \in [Schemes -> SUBSET Names]
Expected == \E branch \in requirements: \A scheme \in branch:
              required[scheme] \subseteq granted[scheme]
Projected(branch, scheme) == IF Flatten
    THEN UNION {required[s]: s \in branch}
    ELSE required[scheme]
Actual == \E branch \in requirements: \A scheme \in branch:
            Projected(branch, scheme) \subseteq granted[scheme]
Agreement == Actual = Expected
Next == UNCHANGED <<requirements, required, granted>>
Spec == Init /\ [][Next]_<<requirements, required, granted>>
=============================================================================
