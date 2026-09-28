---------------------------- MODULE UnionNames -----------------------------
EXTENDS Naturals, Sequences, FiniteSets, TLC
CONSTANT Mode
Definitions == 1..3
Base(d) == IF d = 3 THEN "RS3" ELSE "RS"
Candidates(d) == IF d = 3 THEN <<"RS3", "RS32", "RS33", "RS34">>
                 ELSE <<"RS", "RS2", "RS3", "RS4", "RS5">>
Authored == {"RS2"}
Reserved == Authored \cup {Base(d): d \in Definitions}
VARIABLES order, ids, copies, memo, clones, names, used, next
vars == <<order, ids, copies, memo, clones, names, used, next>>
Init == /\ order = <<>>
        /\ ids = [d \in Definitions |-> 0]
        /\ copies = [d \in Definitions |-> 0]
        /\ memo = {<<<<"1", 0>>, 0>>}
        /\ clones = {}
        /\ names = [d \in Definitions |-> ""]
        /\ used = Authored
        /\ next = 1
Declare(d) == /\ ids[d] = 0
              /\ ids' = [ids EXCEPT ![d] = Len(order) + 1]
              /\ order' = Append(order, d)
              /\ UNCHANGED <<copies, memo, clones, names, used, next>>
Copy(d) == LET key == IF Mode = "legacy" THEN <<Base(d), 0>>
                     ELSE IF Mode = "untyped" THEN <<ToString(ids[d]), 0>>
                     ELSE <<Base(d), ids[d]>>
               matches == {entry \in memo: entry[1] = key}
               origin == IF matches = {} THEN d ELSE (CHOOSE entry \in matches: TRUE)[2]
           IN /\ ids[d] # 0 /\ copies[d] < 2
              /\ copies' = [copies EXCEPT ![d] = @ + 1]
              /\ memo' = memo \cup {<<key, origin>>}
              /\ clones' = clones \cup {<<d, origin>>}
              /\ UNCHANGED <<order, ids, names, used, next>>
Ready == \A d \in Definitions: copies[d] = 2
Available(d, k) == /\ Candidates(d)[k] \notin used
                    /\ (k = 1 \/ Mode = "suffix" \/ Candidates(d)[k] \notin Reserved)
Allocate == /\ Ready /\ next <= 3
            /\ LET d == IF Mode = "order" THEN order[next] ELSE next
                   k == CHOOSE k \in 1..Len(Candidates(d)):
                        Available(d, k) /\ \A j \in 1..(k-1): ~Available(d, j)
               IN /\ names' = [names EXCEPT ![d] = Candidates(d)[k]]
                  /\ used' = used \cup {Candidates(d)[k]}
            /\ next' = next + 1
            /\ UNCHANGED <<order, ids, copies, memo, clones>>
Done == next = 4
Next == (\E d \in Definitions: Declare(d) \/ Copy(d)) \/ Allocate
        \/ (Done /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
CopiesKeepDefinition == \A clone \in clones: clone[1] = clone[2]
DistinctNames == \A a, b \in Definitions: (a # b /\ names[a] # "" /\ names[b] # "") => names[a] # names[b]
AuthoredNamesPreserved == \A d \in Definitions: names[d] \notin Authored
StableNames == Done => names = [d \in Definitions |-> CASE d = 1 -> "RS" [] d = 2 -> "RS4" [] OTHER -> "RS3"]
Terminates == <>Done
=============================================================================
