---------------------- MODULE RecursiveProjection ----------------------
EXTENDS TLC
CONSTANT Mode
Shapes == {"direct", "adjacent", "array", "map", "union", "object", "recursive"}
Views == {"default", "full"}
VARIABLES shape, parentView, childView, secret, phase, visible, readExcluded
vars == <<shape, parentView, childView, secret, phase, visible, readExcluded>>
Init == /\ shape \in Shapes /\ parentView \in Views /\ childView \in Views
        /\ secret \in {"first", "second"} /\ phase = "input"
        /\ visible = {} /\ readExcluded = FALSE
Project == /\ phase = "input"
           /\ visible' = IF childView = "full" \/ (Mode = "legacy" /\ shape # "direct")
                          THEN {"id", secret} ELSE {"id"}
           /\ readExcluded' = (Mode = "legacy" /\ shape # "direct" /\ childView = "default")
           /\ phase' = "done"
           /\ UNCHANGED <<shape, parentView, childView, secret>>
Next == Project \/ (phase = "done" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
NoExcludedDisclosure == phase = "done" /\ childView = "default" => visible = {"id"}
NoExcludedRead == ~readExcluded
ExactSelection == phase = "done" => visible = (IF childView = "full" THEN {"id", secret} ELSE {"id"})
Terminates == <>(phase = "done")
=============================================================================
