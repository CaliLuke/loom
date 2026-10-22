----------------------- MODULE DocumentationSchema -----------------------
EXTENDS Naturals, FiniteSets
CONSTANT Mode
Contexts == {"schema", "runtime"}
VARIABLES seen, result
vars == <<seen, result>>
Init == /\ seen = {}
        /\ result = [c \in Contexts |-> "unchecked"]
Validate(c) ==
  /\ result[c] = "unchecked"
  /\ LET cached == IF Mode = "shared" THEN seen # {} ELSE c \in seen
         decision == IF cached THEN "accept"
                     ELSE IF c = "schema" /\ Mode # "old" THEN "accept"
                     ELSE "reject"
     IN /\ result' = [result EXCEPT ![c] = decision]
        /\ seen' = seen \cup {c}
Next == \E c \in Contexts : Validate(c)
Spec == Init /\ [][Next]_vars
SchemaAccepted == result["schema"] \in {"unchecked", "accept"}
RuntimeRejected == result["runtime"] \in {"unchecked", "reject"}
=============================================================================
