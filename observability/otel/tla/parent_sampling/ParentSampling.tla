----------------------- MODULE ParentSampling -----------------------
EXTENDS TLC
CONSTANT HonorParent
VARIABLES validParent, remote, parentSampled, newRoot, rootDecision, sampled, phase
vars == <<validParent, remote, parentSampled, newRoot, rootDecision, sampled, phase>>
Init == /\ validParent \in BOOLEAN
        /\ remote \in BOOLEAN
        /\ parentSampled \in BOOLEAN
        /\ newRoot \in BOOLEAN
        /\ rootDecision \in BOOLEAN
        /\ sampled = FALSE
        /\ phase = "before"
HasParent == validParent /\ ~newRoot
Sample == /\ phase = "before"
          /\ phase' = "after"
          /\ sampled' = IF HonorParent /\ HasParent THEN parentSampled ELSE rootDecision
          /\ UNCHANGED <<validParent, remote, parentSampled, newRoot, rootDecision>>
Spec == Init /\ [][Sample]_vars
ParentDecision == phase = "after" /\ HasParent => sampled = parentSampled
RootDecision == phase = "after" /\ ~HasParent => sampled = rootDecision
=============================================================================
