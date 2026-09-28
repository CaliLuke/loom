------------------------ MODULE BodyTypeNames ------------------------
EXTENDS Naturals, FiniteSets
CONSTANT Mode
Roots == {"request", "response", "stream"}
Names == 1..3
VARIABLES preferred, nested, pending, allocated, used
vars == <<preferred, nested, pending, allocated, used>>
Init == /\ preferred \in [Roots -> Names]
        /\ nested \in SUBSET Names
        /\ pending = Roots
        /\ allocated = [r \in Roots |-> 0]
        /\ used = nested \cup {preferred[r] : r \in Roots}
Assigned == {allocated[r] : r \in Roots \ pending}
Fresh == CHOOSE n \in 1..6: n \notin used /\ \A m \in 1..6: m < n => m \in used
Allocate(r) == LET name == IF Mode = "legacy" THEN preferred[r]
                          ELSE IF preferred[r] \in nested \cup Assigned THEN Fresh
                          ELSE preferred[r]
              IN /\ r \in pending
                 /\ allocated' = [allocated EXCEPT ![r] = name]
                 /\ pending' = pending \ {r}
                 /\ used' = used \cup {name}
                 /\ UNCHANGED <<preferred, nested>>
Next == (\E r \in pending: Allocate(r)) \/ (pending = {} /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
NestedTypesPreserved == Assigned \cap nested = {}
DistinctBodies == \A a,b \in Roots \ pending: a # b => allocated[a] # allocated[b]
Terminates == <>(pending = {})
======================================================================
