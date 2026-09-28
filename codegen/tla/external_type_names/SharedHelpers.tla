------------------------ MODULE SharedHelpers ------------------------
EXTENDS Naturals, FiniteSets
CONSTANT Mode
Services == {"first", "second"}
Names == 1..5
VARIABLES authored, local, remaining, helper
vars == <<authored, local, remaining, helper>>
Fresh(base, reserved) == CHOOSE n \in Names: n >= base /\ n \notin reserved
                            /\ \A earlier \in Names: (earlier >= base /\ earlier < n) => earlier \in reserved
Reserve(reserved) == LET first == IF 1 \in authored THEN reserved \cup {Fresh(1,reserved)} ELSE reserved IN
                    IF 2 \in authored THEN first \cup {Fresh(2,first)} ELSE first
Init == /\ authored \in SUBSET {1,2}
        /\ local \in [Services -> SUBSET {1,2}]
        /\ remaining = Services
        /\ helper = [s \in Services |-> 0]
Declare(s) == /\ s \in remaining
              /\ helper' = [helper EXCEPT ![s] = Fresh(1, IF Mode = "checked" THEN Reserve(local[s]) ELSE local[s])]
              /\ remaining' = remaining \ {s}
              /\ UNCHANGED <<authored, local>>
Next == (\E s \in remaining: Declare(s)) \/ (remaining = {} /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
AuthoredNamesPreserved == \A s \in Services \ remaining: helper[s] \notin authored
Terminates == <>(remaining = {})
=============================================================================
