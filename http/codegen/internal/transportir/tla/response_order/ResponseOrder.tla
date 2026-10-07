------------------------ MODULE ResponseOrder ------------------------
EXTENDS Naturals, Sequences, FiniteSets
CONSTANT Mode
VARIABLES authored, matches, planned, done
vars == <<authored, matches, planned, done>>
IDs == {0, 1, 2, 3}
Orders == {s \in [1..4 -> IDs]: {s[i]: i \in 1..4} = IDs}
Tagged(s) == SelectSeq(s, LAMBDA x: x # 0)
Stable(s) == Append(Tagged(s), 0)
Legacy(s) == LET p == CHOOSE i \in 1..4: s[i] = 0
             IN [s EXCEPT ![p] = s[4], ![4] = 0]
First(s, selected) == LET eligible == {i \in 1..Len(s): s[i] = 0 \/ s[i] \in selected}
                         first == CHOOSE i \in eligible: \A j \in eligible: i <= j
                     IN s[first]
Init == /\ authored \in Orders
        /\ matches \in SUBSET {1, 2, 3}
        /\ planned = <<>>
        /\ done = FALSE
Plan == /\ ~done
        /\ planned' = IF Mode = "legacy" THEN Legacy(authored) ELSE Stable(authored)
        /\ done' = TRUE
        /\ UNCHANGED <<authored, matches>>
Next == Plan \/ (done /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
PreservesPriority == done => Tagged(planned) = Tagged(authored)
SelectsFirstMatch == done => First(planned, matches) = First(Stable(authored), matches)
PreservesResponses == done => {planned[i]: i \in 1..Len(planned)} = IDs
DefaultLast == done => planned[Len(planned)] = 0
======================================================================
