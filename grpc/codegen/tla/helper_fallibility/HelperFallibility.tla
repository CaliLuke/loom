----------------------- MODULE HelperFallibility -----------------------
EXTENDS Naturals, FiniteSets

CONSTANT Mode
VARIABLES graph, ownErrors, firstSibling, secondSibling, phase

Helpers == {"leaf", "middle", "root"}
Graphs == {
  [leaf |-> {}, middle |-> {"leaf"}, root |-> {"middle"}],
  [leaf |-> {"leaf"}, middle |-> {"leaf"}, root |-> {"middle"}],
  [leaf |-> {"middle"}, middle |-> {"leaf"}, root |-> {"middle"}]
}
RECURSIVE Reach(_, _)
Reach(helper, depth) == IF depth = 0 THEN {helper}
                       ELSE {helper} \cup UNION {Reach(child, depth - 1): child \in graph[helper]}
NeedsError(helper) == \E reached \in Reach(helper, Cardinality(Helpers)): reached \in ownErrors
FirstRootError == firstSibling \/ NeedsError("root")
SecondRootError == secondSibling \/ NeedsError("root")

Declaration(helper) == CASE Mode \in {"root-inherit", "calls-only"} -> FirstRootError
                           [] Mode = "drop-errors" -> FALSE
                           [] OTHER -> NeedsError(helper)
RootCall(which) == CASE Mode \in {"root-inherit", "declarations-only"} ->
                            IF which = 1 THEN FirstRootError ELSE SecondRootError
                       [] Mode = "drop-errors" -> FALSE
                       [] OTHER -> NeedsError("root")
HelperCall(parent, child) == CASE Mode = "root-inherit" -> FirstRootError
                                 [] Mode = "declarations-only" -> NeedsError(parent)
                                 [] Mode = "drop-errors" -> FALSE
                                 [] OTHER -> NeedsError(child)

vars == <<graph, ownErrors, firstSibling, secondSibling, phase>>
Init == /\ graph \in Graphs
        /\ ownErrors \in SUBSET Helpers
        /\ firstSibling \in BOOLEAN
        /\ secondSibling \in BOOLEAN
        /\ phase = "planned"
Emit == /\ phase = "planned"
        /\ phase' = "emitted"
        /\ UNCHANGED <<graph, ownErrors, firstSibling, secondSibling>>
Spec == Init /\ [][Emit]_vars

RootABI == phase = "emitted" =>
           \A which \in {1, 2}: RootCall(which) = Declaration("root")
NestedABI == phase = "emitted" =>
             \A parent \in Helpers: \A child \in graph[parent]:
               HelperCall(parent, child) = Declaration(child)
ErrorPropagation == phase = "emitted" =>
                    \A helper \in Helpers: NeedsError(helper) => Declaration(helper)
LocalContract == phase = "emitted" =>
                 \A helper \in Helpers: Declaration(helper) = NeedsError(helper)
=======================================================================
