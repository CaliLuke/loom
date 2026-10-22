------------------------- MODULE FingerprintInput -------------------------
EXTENDS Naturals, Integers, FiniteSets
CONSTANT Mode
VARIABLES visited, last, captured, admitted, sourceOrigin, policyOrigin, observed, work
vars == <<visited, last, captured, admitted, sourceOrigin, policyOrigin, observed, work>>
N == 4
Derived == 3
Meaning(i) == IF i = Derived THEN i + 10 ELSE i
Origin(i) == IF i = Derived THEN i + 100 ELSE i
InputCost(i) == N - i + 1
Init == /\ visited = {}
        /\ last = 0
        /\ captured = [i \in 1..N |-> i]
        /\ admitted = (Mode # "SkipAdmission")
        /\ sourceOrigin = 0
        /\ policyOrigin = 0
        /\ observed = 0
        /\ work = N
Visit(i) == /\ i \in (1..N) \ visited
            /\ LET fresh == i = Derived /\ Mode # "StaleDerived"
                   value == IF fresh THEN Meaning(i) ELSE captured[i]
                   origin == IF fresh THEN Origin(i) ELSE i
               IN /\ observed' = value
                  /\ sourceOrigin' = origin
                  /\ policyOrigin' = IF Mode = "CopiedPolicy" /\ i = 2
                                      THEN i + 100 ELSE origin
                  /\ work' = work + IF Mode = "Recapture" \/ fresh
                                    THEN InputCost(i) ELSE 0
            /\ visited' = visited \cup {i}
            /\ last' = i
            /\ UNCHANGED <<captured, admitted>>
Step == \E i \in 1..N: Visit(i)
Done == /\ visited = 1..N /\ UNCHANGED vars
Next == Step \/ Done
SemanticInput == last = 0 \/ observed = Meaning(last)
PolicyOrigin == sourceOrigin = policyOrigin
RepresentationAdmission == admitted
AnalysisBudget == work <= N + InputCost(Derived)
FrozenInput == captured = [i \in 1..N |-> i]
Complete == <>(visited = 1..N)
Spec == Init /\ [][Next]_vars /\ WF_vars(Step)
=============================================================================
