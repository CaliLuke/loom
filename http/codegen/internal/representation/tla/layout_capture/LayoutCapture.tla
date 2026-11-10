-------------------------- MODULE LayoutCapture --------------------------
EXTENDS Naturals, FiniteSets
CONSTANT Mode
VARIABLES registry, captured, snapshots, overrides
vars == <<registry, captured, snapshots, overrides>>
Keys == {"root", "nested", "sibling", "unrelated"}
Targets == {1, 2, 3}
Initial == [k \in Keys |-> FALSE]

Init == /\ registry = Initial
        /\ captured = {}
        /\ snapshots = [t \in Targets |-> Initial]
        /\ overrides = [t \in Targets |-> FALSE]

Capture(t) == /\ t \notin captured
              /\ captured' = captured \cup {t}
              /\ snapshots' = [snapshots EXCEPT ![t] = registry]
              /\ overrides' = [overrides EXCEPT ![t] = TRUE]
              /\ UNCHANGED registry

Mutate == /\ captured = Targets
          /\ registry = Initial
          /\ registry' = [k \in Keys |-> TRUE]
          /\ UNCHANGED <<captured, snapshots, overrides>>

Next == Mutate \/ (\E t \in Targets : Capture(t))
Spec == Init /\ [][Next]_vars

\* A captured root override must not change a sibling's registered layout.
Read(t, k) == IF k = "root" THEN overrides[t]
              ELSE IF Mode = "borrow" THEN registry[k]
              ELSE snapshots[t][k]
SnapshotStable == \A t \in captured : Read(t, "sibling") = snapshots[t]["sibling"]
LocalOverride == \A t \in captured : Read(t, "root") = TRUE
\* Reachable-only capture cannot answer a later sibling publication.
StoredKeys(t) == IF Mode = "reachable" THEN {"root", "nested"} ELSE Keys
PublicationComplete == \A t \in captured : "sibling" \in StoredKeys(t)
StoredFacts == Cardinality(Keys) +
               (IF Mode = "dense" THEN Cardinality(captured) * Cardinality(Keys)
                ELSE Cardinality(captured))
LinearStorage == StoredFacts <= Cardinality(Keys) + Cardinality(captured)
=============================================================================
