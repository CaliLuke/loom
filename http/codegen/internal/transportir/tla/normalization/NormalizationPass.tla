---------------------- MODULE NormalizationPass ----------------------
EXTENDS Naturals, FiniteSets, TLC
CONSTANTS N, Recapture
Nodes == 1..N
(* A chain of named aliases; every layer contributes a distinct predicate. *)
Meaning(n) == n..N
VARIABLES captured, output, next, work
vars == <<captured, output, next, work>>
Init == /\ captured = [n \in Nodes |-> {}]
        /\ output = [n \in Nodes |-> {n}]
        /\ next = 0
        /\ work = 0
Capture == /\ next = 0
           /\ captured' = [n \in Nodes |-> Meaning(n)]
           /\ work' = N
           /\ next' = 1
           /\ UNCHANGED output
Lower == /\ next \in Nodes
         /\ output' = [output EXCEPT ![next] = captured[next]]
         /\ work' = work + IF Recapture /\ next > 1 THEN N-next+1 ELSE 0
         /\ next' = next+1
         /\ UNCHANGED captured
Done == /\ next = N+1
        /\ UNCHANGED vars
Next == Capture \/ Lower \/ Done
TypeOK == /\ captured \in [Nodes -> SUBSET Nodes]
          /\ output \in [Nodes -> SUBSET Nodes]
          /\ next \in 0..(N+1)
          /\ work \in Nat
FrozenInput == next > 0 => \A n \in Nodes : captured[n] = Meaning(n)
LoweringPreservesMeaning == \A n \in Nodes : n < next => output[n] = Meaning(n)
OneInputAnalysis == work <= N
Complete == <>(next = N+1)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
======================================================================
