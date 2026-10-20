----------------------- MODULE ValidatorOrdering -----------------------
EXTENDS FiniteSets
CONSTANT Mode
VARIABLES pending, declared, required, validators, phase
vars == <<pending, declared, required, validators, phase>>
Roots == {"ordinary", "union", "unrelated"}
Types(root) == CASE root = "ordinary" -> {"Item"}
                [] root = "union" -> {"Data", "Item"}
                [] OTHER -> {"Other"}
Needs(root) == IF root = "union" THEN Types(root) ELSE {}
Init == /\ pending = Roots
        /\ declared = {}
        /\ required = {}
        /\ validators = {}
        /\ phase = "collect"
Collect(root) ==
    /\ phase = "collect"
    /\ root \in pending
    /\ pending' = pending \ {root}
    /\ required' = required \cup Needs(root)
    /\ declared' = declared \cup Types(root)
    /\ validators' = IF Mode = "eager"
                     THEN validators \cup ((Types(root) \ declared) \cap required')
                     ELSE validators
    /\ UNCHANGED phase
Finalize ==
    /\ phase = "collect"
    /\ pending = {}
    /\ validators' = IF Mode = "deferred" THEN declared \cap required ELSE validators
    /\ phase' = "done"
    /\ UNCHANGED <<pending, declared, required>>
Next == (\E root \in Roots: Collect(root)) \/ Finalize
Spec == Init /\ [][Next]_vars
DependenciesDeclared == phase = "done" => required \subseteq validators
NoUnneededValidators == validators \subseteq required
=============================================================================
