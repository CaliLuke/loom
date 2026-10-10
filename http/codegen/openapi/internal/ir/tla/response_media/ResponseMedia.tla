------------------------ MODULE ResponseMedia ------------------------
EXTENDS Naturals, Sequences, FiniteSets, TLC
CONSTANT Mode
VARIABLES order, conflict, index, content, rejected
vars == <<order, conflict, index, content, rejected>>
IDs == {1, 2, 3}
Orders == {s \in [1..3 -> IDs]: {s[i]: i \in 1..3} = IDs}
Media(id) == IF id = 2 THEN "html" ELSE "json"
Schema(id) == IF id = 3 THEN "other" ELSE Media(id)
Init == /\ order \in Orders
        /\ conflict \in BOOLEAN
        /\ index = 1
        /\ content = [x \in {} |-> ""]
        /\ rejected = FALSE
Add == /\ index <= 3
       /\ LET id == order[index]
              media == Media(id)
              schema == Schema(id)
              incompatible == conflict /\ media \in DOMAIN content
          IN /\ content' = IF Mode = "legacy" THEN (media :> {schema})
                           ELSE IF rejected \/ incompatible THEN content
                           ELSE IF media \in DOMAIN content THEN
                               [content EXCEPT ![media] = @ \cup {schema}]
                           ELSE content @@ (media :> {schema})
             /\ rejected' = IF Mode = "legacy" THEN FALSE ELSE rejected \/ incompatible
       /\ index' = index + 1
       /\ UNCHANGED <<order, conflict>>
Next == Add \/ (index = 4 /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
PreservesAlternatives == index = 4 /\ ~conflict =>
    content = ("json" :> {"json", "other"}) @@ ("html" :> {"html"})
RejectsConflicts == index = 4 => rejected = conflict
======================================================================
