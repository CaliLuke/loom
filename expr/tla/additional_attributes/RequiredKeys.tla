------------------------- MODULE RequiredKeys -------------------------
EXTENDS Naturals, FiniteSets
CONSTANT Storage
Keys == {"a", "a:b", "a,b", ""}
Values == {"zero", "value", "null"}
Tag(k) == CASE k = "a:b" -> "b" [] k = "a,b" -> "a" [] OTHER -> k
VARIABLES required, input, accepted, decoded, phase
vars == <<required, input, accepted, decoded, phase>>
Init == /\ required \in SUBSET Keys
        /\ input \in UNION {[D -> Values] : D \in SUBSET Keys}
        /\ accepted = FALSE /\ decoded = [k \in {} |-> "zero"] /\ phase = "decode"
Decode == /\ phase = "decode"
          /\ decoded' = IF Storage = "map" THEN input
                         ELSE [k \in {j \in DOMAIN input : j \notin required \/ Tag(j) = j} |-> input[k]]
          /\ accepted' = IF Storage = "map" THEN required \subseteq DOMAIN input
                          ELSE \A k \in required : Tag(k) \in DOMAIN input
          /\ phase' = "done"
          /\ UNCHANGED <<required, input>>
Next == Decode \/ (phase = "done" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
RequiredExact == phase = "done" => (accepted = (required \subseteq DOMAIN input))
Preserved == phase = "done" /\ accepted => decoded = input
=============================================================================
