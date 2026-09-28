------------------------- MODULE UnionDeclarations -------------------------
EXTENDS Naturals, FiniteSets, TLC
CONSTANT Mode
Types == 1..4
Shapes == [t \in Types |-> IF t = 3 THEN "other" ELSE "shared"]
VARIABLES allocated, names, visited, emitted, keys
vars == <<allocated, names, visited, emitted, keys>>
Init == /\ allocated = {}
        /\ names = [t \in Types |-> 0]
        /\ visited = {}
        /\ emitted = {}
        /\ keys = {}
Allocate(t) == /\ t \in Types \ allocated
               /\ allocated' = allocated \cup {t}
               /\ names' = [names EXCEPT ![t] = Cardinality(allocated) + 1]
               /\ UNCHANGED <<visited, emitted, keys>>
Collect(t) == /\ allocated = Types
              /\ t \in Types \ visited
              /\ LET key == IF Mode = "shape" THEN Shapes[t] ELSE names[t]
                 IN /\ visited' = visited \cup {t}
                    /\ emitted' = IF key \in keys THEN emitted ELSE emitted \cup {names[t]}
                    /\ keys' = keys \cup {key}
              /\ UNCHANGED <<allocated, names>>
Next == (\E t \in Types: Allocate(t)) \/ (\E t \in Types: Collect(t))
        \/ (visited = Types /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
NamesAreUnique == \A a,b \in allocated: names[a] = names[b] => a = b
EveryReferenceDeclared == visited = Types => emitted = {names[t]: t \in Types}
Terminates == <>(visited = Types)
=============================================================================
