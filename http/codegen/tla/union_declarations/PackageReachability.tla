----------------------- MODULE PackageReachability -----------------------
EXTENDS Integers, FiniteSets
CONSTANT Mode
VARIABLES clientSelection, serverSelection, requestUnion, declarations, phase
vars == <<clientSelection, serverSelection, requestUnion, declarations, phase>>
Packages == {"client", "server"}
Union == "Union"
Branch == "Branch"
Types == {"Body", Union, Branch}
Shape(selected) == IF selected THEN Types ELSE {"Body"}
BodyTypes(package) == Shape(IF package = "client" THEN clientSelection ELSE serverSelection)
                     \cup (IF requestUnion THEN {Union, Branch} ELSE {})
ReachableUnions(package) == BodyTypes(package) \cap {Union}
Registry(package) == IF Mode = "legacy" THEN ReachableUnions("client")
                     ELSE ReachableUnions(package)
Init == /\ clientSelection \in BOOLEAN
        /\ serverSelection \in IF clientSelection THEN BOOLEAN ELSE {FALSE}
        /\ requestUnion \in BOOLEAN
        /\ declarations = [p \in Packages |-> {}]
        /\ phase = "prepare"
Emit == /\ phase = "prepare"
        /\ declarations' = [p \in Packages |-> (BodyTypes(p) \ {Union}) \cup Registry(p)]
        /\ phase' = "done"
        /\ UNCHANGED <<clientSelection, serverSelection, requestUnion>>
Next == Emit
Spec == Init /\ [][Next]_vars
DependenciesDeclared == phase = "done" => \A p \in Packages:
                        Union \in declarations[p] => Branch \in declarations[p]
ReachableDeclared == phase = "done" => \A p \in Packages:
                     BodyTypes(p) \subseteq declarations[p]
NoUnreachableUnion == phase = "done" => \A p \in Packages:
                      declarations[p] \cap {Union} = ReachableUnions(p)
=============================================================================
