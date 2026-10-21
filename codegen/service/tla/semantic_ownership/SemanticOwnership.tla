------------------------- MODULE SemanticOwnership -------------------------
EXTENDS Naturals, FiniteSets
CONSTANTS Backends, SharedOwner, SharedRecord, SharedCache
VARIABLES captured, owners, selected, outcomes, cache
vars == <<captured, owners, selected, outcomes, cache>>

Init == /\ captured = {}
        /\ owners = [b \in Backends |-> "absent"]
        /\ selected = {}
        /\ outcomes = [b \in Backends |-> "unset"]
        /\ cache = [b \in Backends |-> "unset"]

Capture(b) == /\ b \notin captured
              /\ captured' = captured \cup {b}
              /\ owners' = [owners EXCEPT ![b] = IF SharedOwner THEN "semantic" ELSE b]
              /\ UNCHANGED <<selected, outcomes, cache>>

Expected(b) == IF b = "openapi" THEN "doc-example" ELSE IF b = "suppressed" THEN "suppressed" ELSE IF b = "transport" THEN "transport-example" ELSE "example"
Key(b) == IF SharedCache THEN "service" ELSE b
Selected(b) == IF b = "suppressed" \/ cache[Key(b)] = "unset" THEN Expected(b) ELSE cache[Key(b)]
Select(b) == /\ b \in captured \ selected
             /\ selected' = selected \cup {b}
             /\ outcomes' = IF SharedRecord
                            THEN [c \in Backends |-> Selected(b)]
                            ELSE [outcomes EXCEPT ![b] = Selected(b)]
             /\ cache' = IF b = "suppressed" THEN cache ELSE [cache EXCEPT ![Key(b)] = Selected(b)]
             /\ UNCHANGED <<captured, owners>>

Next == \E b \in Backends : Capture(b) \/ Select(b)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)

OneSemanticOwner == Cardinality({owners[b] : b \in captured}) <= 1
LocalSelection == \A b \in selected : outcomes[b] = Expected(b)
Completes == <> (selected = Backends)
=============================================================================
