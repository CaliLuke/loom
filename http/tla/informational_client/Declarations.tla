--------------------------- MODULE Declarations ---------------------------
EXTENDS PerDo
\* This static frontend boundary makes PerDo's declared-100 assumption explicit.
\* Parsing, Go constraints and native Expect handling remain implementation tests.
\* Zero denotes the 1XX class; 999 is the unmatched sentinel.
Keys == {100, 103, 0}
Owner(keys, code) == IF code \in keys THEN code
                    ELSE IF 0 \in keys THEN 0 ELSE 999
Observed(keys, key) == {code \in 100..199: code # 101 /\ Owner(keys, code) = key}
Emitted(keys, key) == Observed(keys, key) \ {100}
Admitted(keys, key, constrained, allowed) ==
  /\ key \in keys
  /\ ~(Mode = "reject-declared100" /\ key = 100)
  /\ (~constrained \/
      (IF Mode = "emission-constraints" THEN Emitted(keys,key)
       ELSE Observed(keys,key)) \cap allowed # {})
DeclaredHundredAvailable ==
  /\ Admitted(Keys, 100, FALSE, {})
  /\ Admitted(Keys, 100, TRUE, {100})
  /\ Admitted({0}, 0, TRUE, {100})
ExactShadowing == ~Admitted(Keys, 0, TRUE, {100})
NoHundredEmission == \A key \in Keys: 100 \notin Emitted(Keys,key)
NoUpgradeObservation == \A key \in Keys: 101 \notin Observed(Keys,key)
=============================================================================
