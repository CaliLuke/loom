----------------------- MODULE ProjectionIdentity -----------------------
EXTENDS TLC, Naturals, FiniteSets
CONSTANT Mode
Occurrences == {1, 2, 3}
Selected(i) == IF i = 2 THEN "full" ELSE "default"
Fields(view) == IF view = "full" THEN {"id", "secret"} ELSE {"id"}
Key(i) == IF Mode = "legacy" THEN "carrier" ELSE Selected(i)
Keys == {"carrier", "default", "full"}
VARIABLES pending, helpers, referenced
vars == <<pending, helpers, referenced>>
Init == /\ pending = Occurrences /\ helpers = [k \in Keys |-> {}]
        /\ referenced = [i \in Occurrences |-> {}]
Convert(i) == /\ i \in pending
              /\ helpers' = IF helpers[Key(i)] = {}
                            THEN [helpers EXCEPT ![Key(i)] = Fields(Selected(i))]
                            ELSE helpers
              /\ referenced' = [referenced EXCEPT ![i] = helpers'[Key(i)]]
              /\ pending' = pending \ {i}
Next == (\E i \in Occurrences: Convert(i)) \/ (pending = {} /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
ReferencesMatchSelection == \A i \in Occurrences \ pending: referenced[i] = Fields(Selected(i))
Terminates == <>(pending = {})
=============================================================================
