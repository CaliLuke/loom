--------------------- MODULE ResponseDescriptions ---------------------
EXTENDS Naturals, FiniteSets, TLC
CONSTANT Mode
VARIABLES order, index, components, refs
vars == <<order, index, components, refs>>
IDs == 1..4
Orders == {s \in [IDs -> IDs]: {s[i]: i \in IDs} = IDs}
Status(id) == IF id = 4 THEN 503 ELSE 400
Shape(id) == IF id = 3 THEN "other" ELSE "problem"
Description(id) == IF id = 1 THEN "first" ELSE "second"
Key(id) == IF Mode = "legacy" THEN <<Status(id), Shape(id), Description(id)>>
           ELSE IF Mode = "overmerge" THEN <<Status(id)>>
           ELSE <<Status(id), Shape(id)>>
Init == /\ order \in Orders
        /\ index = 1
        /\ components = [x \in {} |-> ""]
        /\ refs = [x \in {} |-> ""]
Add == /\ index <= 4
       /\ LET id == order[index]
              key == Key(id)
          IN /\ components' = IF key \in DOMAIN components THEN components
                              ELSE components @@ (key :> <<Status(id), Shape(id)>>)
             /\ refs' = refs @@ (id :> [key |-> key, description |-> Description(id)])
       /\ index' = index + 1
       /\ UNCHANGED order
Next == Add \/ (index = 5 /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
SameShapesShare == \A a, b \in DOMAIN refs:
    Status(a) = Status(b) /\ Shape(a) = Shape(b) => refs[a].key = refs[b].key
PreservesShape == \A id \in DOMAIN refs:
    components[refs[id].key] = <<Status(id), Shape(id)>>
PreservesDescription == \A id \in DOMAIN refs: refs[id].description = Description(id)
=======================================================================
