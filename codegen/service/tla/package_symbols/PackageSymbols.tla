-------------------------- MODULE PackageSymbols ---------------------------
EXTENDS Naturals, Sequences, FiniteSets
CONSTANT ReserveFixed
Fixed == {"Client", "Service"}
Types == {"Client", "Client2", "Service", "Ordinary"}
Candidates(name) == <<name, name \o "2", name \o "3", name \o "4", name \o "5">>
VARIABLES pending, occupied, allocation
vars == <<pending, occupied, allocation>>
Init == /\ pending = Types
        /\ occupied = IF ReserveFixed THEN Fixed ELSE {}
        /\ allocation = [name \in Types |-> ""]
Allocate(name) ==
    LET choices == Candidates(name)
        free == {i \in 1..Len(choices) : choices[i] \notin occupied}
        first == CHOOSE i \in free : \A j \in free : i <= j
        selected == choices[first]
    IN /\ name \in pending
       /\ pending' = pending \ {name}
       /\ occupied' = occupied \cup {selected}
       /\ allocation' = [allocation EXCEPT ![name] = selected]
Next == (\E name \in pending : Allocate(name))
        \/ (pending = {} /\ UNCHANGED vars)
NoCollision == \A name \in Types \ pending : allocation[name] \notin Fixed
DistinctTypes == \A a,b \in Types \ pending : a # b => allocation[a] # allocation[b]
=============================================================================
