---------------------------- MODULE FieldTags ----------------------------
EXTENDS FiniteSets
CONSTANT Recursive
VARIABLES pending, seen, checked
vars == <<pending, seen, checked>>
\* A selected root reaches shared/cyclic messages through a collection or
\* union carrier. External owns its protobuf implementation and is opaque.
Nodes == {"root", "carrier", "nested", "external", "hidden"}
Objects == {"root", "nested", "hidden"}
Edges == [n \in Nodes |-> CASE n = "root" -> {"carrier", "external"}
                         [] n = "carrier" -> {"nested"}
                         [] n = "nested" -> {"root"}
                         [] n = "external" -> {"hidden"}
                         [] OTHER -> {}]
Init == /\ pending = {"root"} /\ seen = {} /\ checked = {}
Visit == \E n \in pending:
    /\ seen' = seen \cup {n}
    /\ checked' = IF n \in Objects THEN checked \cup {n} ELSE checked
    /\ pending' = (pending \ {n}) \cup
         (IF Recursive /\ n # "external" THEN Edges[n] \ (seen \cup {n}) ELSE {})
Done == /\ pending = {} /\ UNCHANGED vars
Next == Visit \/ Done
Coverage == pending = {} => checked = {"root", "nested"}
Opaque == "hidden" \notin seen
Spec == Init /\ [][Next]_vars
=============================================================================
