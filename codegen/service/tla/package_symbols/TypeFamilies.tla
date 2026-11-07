-------------------------- MODULE TypeFamilies ----------------------------
EXTENDS Naturals, Sequences, FiniteSets
CONSTANT AtomicFamilies
Types == {"Choice", "ChoiceKindCat", "NewChoiceCat", "Cat", "ValidateCat"}
Family(t, n) == IF t = "Choice" THEN {n, n \o "Kind", n \o "KindCat", "New" \o n \o "Cat"}
                ELSE IF t = "Cat" THEN {n, "Validate" \o n}
                ELSE {n}
Candidates(n) == <<n, n \o "2", n \o "3", n \o "4", n \o "5", n \o "6">>
VARIABLES pending, occupied, allocation
vars == <<pending, occupied, allocation>>
Init == /\ pending = Types
        /\ occupied = {}
        /\ allocation = [t \in Types |-> ""]
Allocate(t) ==
    LET candidates == Candidates(t)
        available == {i \in 1..Len(candidates) :
             (IF AtomicFamilies THEN Family(t, candidates[i]) ELSE {candidates[i]}) \cap occupied = {}}
        first == CHOOSE i \in available : \A j \in available : i <= j
        selected == candidates[first]
    IN /\ t \in pending
       /\ pending' = pending \ {t}
       /\ occupied' = occupied \cup (IF AtomicFamilies THEN Family(t, selected) ELSE {selected})
       /\ allocation' = [allocation EXCEPT ![t] = selected]
Next == (\E t \in pending : Allocate(t)) \/ (pending = {} /\ UNCHANGED vars)
DeclarationsDistinct == \A a,b \in Types \ pending :
    a # b => Family(a, allocation[a]) \cap Family(b, allocation[b]) = {}
=============================================================================
