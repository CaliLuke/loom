------------------------- MODULE UnionDeclarations -------------------------
EXTENDS Naturals, FiniteSets
CONSTANT Mode
Identities == {"Raw", "First", "Second", "Other"}
Occurrences == 1..8
Identity(i) == CASE i \in {1, 5} -> "Raw"
                [] i \in {2, 6} -> "First"
                [] i \in {3, 7} -> "Second"
                [] OTHER -> "Other"
Shape(id) == IF id = "Other" THEN "OtherShape" ELSE "SharedShape"
Key(id) == IF Mode = "legacy" THEN Shape(id) ELSE id
VARIABLES remaining, collected, visited, emitted
vars == <<remaining, collected, visited, emitted>>
Init == /\ remaining = Occurrences
        /\ collected = {}
        /\ visited = {}
        /\ emitted = [id \in Identities |-> 0]
Collect(i) == LET id == Identity(i) IN
              /\ i \in remaining
              /\ remaining' = remaining \ {i}
              /\ visited' = visited \cup {id}
              /\ collected' = collected \cup {Key(id)}
              /\ emitted' = IF Key(id) \in collected THEN emitted
                            ELSE [emitted EXCEPT ![id] = @ + 1]
Next == (\E i \in remaining: Collect(i))
        \/ (remaining = {} /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
EveryReferenceDeclared == \A id \in visited: emitted[id] = 1
NoDuplicateDeclarations == \A id \in Identities: emitted[id] <= 1
Terminates == <>(remaining = {})
=============================================================================
