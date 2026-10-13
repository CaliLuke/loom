---------------------- MODULE ReferenceComparison ----------------------
EXTENDS Naturals, FiniteSets
CONSTANT Mode
VARIABLES phase, registry, expectedOverlay, actualOverlay, observed
Consumers == {1, 2}
vars == <<phase, registry, expectedOverlay, actualOverlay, observed>>
Contract(overlay) == [values |-> IF overlay THEN {2} ELSE {1, 2},
                      annotation |-> IF overlay THEN "local" ELSE "base"]
Init == /\ phase = [c \in Consumers |-> "pending"]
        /\ registry = TRUE
        /\ expectedOverlay \in BOOLEAN
        /\ actualOverlay \in BOOLEAN
        /\ observed = [c \in Consumers |-> Contract(FALSE)]
Resolve(c) == /\ phase[c] = "pending"
              /\ phase' = [phase EXCEPT ![c] = IF registry THEN "resolved" ELSE "dangling"]
              /\ observed' = [observed EXCEPT ![c] =
                   Contract(IF Mode = "drop-overlay" THEN FALSE ELSE actualOverlay)]
              /\ registry' = IF Mode = "eager-prune" THEN FALSE ELSE registry
              /\ UNCHANGED <<expectedOverlay, actualOverlay>>
Prune == /\ \A c \in Consumers : phase[c] = "resolved"
         /\ registry
         /\ registry' = FALSE
         /\ UNCHANGED <<phase, expectedOverlay, actualOverlay, observed>>
Next == (\E c \in Consumers : Resolve(c)) \/ Prune \/ UNCHANGED vars
ReferenceTargetsRemainAvailable == \A c \in Consumers : phase[c] # "dangling"
ComparisonPreservesContract == \A c \in Consumers : phase[c] = "resolved" =>
  (observed[c] = Contract(expectedOverlay)) = (actualOverlay = expectedOverlay)
TypeOK == expectedOverlay \in BOOLEAN /\ actualOverlay \in BOOLEAN
          /\ registry \in BOOLEAN
          /\ phase \in [Consumers -> {"pending", "resolved", "dangling"}]
Spec == Init /\ [][Next]_vars
=============================================================================
