-------------------------- MODULE PackageSymbols ---------------------------
EXTENDS Naturals, Sequences, FiniteSets
CONSTANT Mode
Fixed == {"Server", "DecodeReadRequest"}
Types == {"Server", "Server2", "Cat", "ValidateCat", "Choice", "NewChoiceCat"}
Candidates(n) == <<n, n \o "2", n \o "3", n \o "4", n \o "5", n \o "6">>
Family(id, n) == {n} \cup
    IF id = "Choice" THEN {n \o "Kind", n \o "KindCat", "New" \o n \o "Cat"}
    ELSE {"Validate" \o n}
VARIABLES pending, occupied, names, published
vars == <<pending, occupied, names, published>>
Init == /\ pending = Types
        /\ occupied = IF Mode \in {"missing", "late"} THEN {} ELSE Fixed
        /\ names = [id \in Types |-> ""]
        /\ published = Fixed
Allocate(id) ==
    LET choices == Candidates(id)
        admissible == {i \in 1..Len(choices) :
             IF Mode = "owned" THEN Family(id, choices[i]) \cap occupied = {}
             ELSE choices[i] \notin occupied}
        first == CHOOSE i \in admissible : \A j \in admissible : i <= j
        n == choices[first]
    IN /\ id \in pending
       /\ pending' = pending \ {id}
       /\ names' = [names EXCEPT ![id] = n]
       /\ occupied' = occupied \cup (IF Mode = "owned" THEN Family(id, n) ELSE {n})
       /\ published' = published \cup Family(id, n)
Next == (\E id \in pending : Allocate(id)) \/ (pending = {} /\ UNCHANGED vars)
Unique == (Mode = "late" /\ pending # {}) \/
          /\ \A id \in Types \ pending : Family(id, names[id]) \cap Fixed = {}
          /\ \A a,b \in Types \ pending : a # b => Family(a, names[a]) \cap Family(b, names[b]) = {}
ReferencesAgree == \A id \in Types \ pending : names[id] \in published
=============================================================================
