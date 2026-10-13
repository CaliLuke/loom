-------------------------- MODULE NameOccupancy -----------------------------
EXTENDS Naturals, Sequences, FiniteSets, TLC
CONSTANT ClaimCandidates
Requests == {"first", "numbered", "repeat"}
Base == [r \in Requests |-> IF r = "numbered" THEN "Pet2" ELSE "Pet"]
Bases == {"Pet", "Pet2"}
Candidates(b) == <<b, b \o "2", b \o "3", b \o "4">>
VARIABLES pending, counts, occupied, names
vars == <<pending, counts, occupied, names>>
Init == /\ pending = Requests
        /\ counts = [b \in Bases |-> 0]
        /\ occupied = {}
        /\ names = [r \in Requests |-> ""]
Allocate(r) ==
    LET b == Base[r]
        candidates == Candidates(b)
        free == {i \in 1..Len(candidates) : candidates[i] \notin occupied}
        first == CHOOSE i \in free : \A j \in free : i <= j
        counted == IF counts[b] = 0 THEN b ELSE b \o ToString(counts[b] + 1)
        selected == IF ClaimCandidates THEN candidates[first] ELSE counted
    IN /\ r \in pending
       /\ pending' = pending \ {r}
       /\ counts' = [counts EXCEPT ![b] = @ + 1]
       /\ occupied' = occupied \cup {selected}
       /\ names' = [names EXCEPT ![r] = selected]
Next == (\E r \in pending : Allocate(r))
        \/ (pending = {} /\ UNCHANGED vars)
Distinct == \A a,b \in Requests \ pending : a # b => names[a] # names[b]
=============================================================================
