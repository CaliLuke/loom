--------------------------- MODULE TypeIdentity ---------------------------
EXTENDS Naturals, FiniteSets, Sequences
CONSTANTS Mode, Cycle
Types == 1..4
DesignID(t) == IF t \in {1, 2} THEN 0 ELSE t
Key(t) == IF Mode = "design" THEN DesignID(t) ELSE t
Needed == {1}
VARIABLES visited, typeKeys, collected, validatorKeys, current, path, outcome
vars == <<visited, typeKeys, collected, validatorKeys, current, path, outcome>>
Init == /\ visited = {}
        /\ typeKeys = {}
        /\ collected = {}
        /\ validatorKeys = {}
        /\ current = 1
        /\ path = <<>>
        /\ outcome = "running"
Visit(t) == /\ t \in Types \ visited
            /\ visited' = visited \cup {t}
            /\ collected' = IF Key(t) \in typeKeys THEN collected ELSE collected \cup {t}
            /\ typeKeys' = typeKeys \cup {Key(t)}
            /\ validatorKeys' = IF t \in Needed THEN validatorKeys \cup {Key(t)} ELSE validatorKeys
            /\ UNCHANGED <<current, path, outcome>>
Walk == /\ outcome = "running"
        /\ IF current = 0
              THEN /\ outcome' = "accepted"
                   /\ UNCHANGED <<current, path>>
              ELSE IF Key(current) \in {Key(path[i]): i \in 1..Len(path)}
                      THEN /\ outcome' = "rejected"
                           /\ UNCHANGED <<current, path>>
                      ELSE /\ path' = Append(path, current)
                           /\ current' = IF current = 1 THEN 2 ELSE IF Cycle THEN 1 ELSE 0
                           /\ UNCHANGED outcome
        /\ UNCHANGED <<visited, typeKeys, collected, validatorKeys>>
Done == visited = Types /\ outcome # "running"
Next == (\E t \in Types: Visit(t)) \/ Walk \/ (Done /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
VisitsEveryType == visited = Types => collected = Types
ValidatorsMatch == visited = Types => {t \in Types: Key(t) \in validatorKeys} = Needed
CycleDetection == outcome # "running" => (outcome = "rejected") = Cycle
Terminates == <>Done
=============================================================================
