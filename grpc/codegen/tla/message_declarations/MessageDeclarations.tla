----------------------- MODULE MessageDeclarations -----------------------
EXTENDS Naturals, FiniteSets, Sequences
CONSTANT SafeAllocation, CheckDeclarations
Owners == {"design", "array", "map", "alias", "compatible", "goalias"}
Generated == {"array", "map"}
Names == 1..5
Shape(p) == CASE p \in {"design", "compatible", "goalias"} -> "D"
                [] p = "alias" -> "X"
                [] p = "array" -> "A"
                [] OTHER -> "M"
Proto(p, n) == IF p = "goalias" THEN 9 ELSE n
Candidates(p) == IF p = "array" THEN <<1, 2, 3, 4, 5>> ELSE <<2, 3, 4, 5>>
VARIABLES claims, assigned, published, protoNames, accepted, rejected
vars == <<claims, assigned, published, protoNames, accepted, rejected>>
Init == /\ claims = [n \in Names |-> IF n = 1 THEN "design" ELSE "free"]
        /\ assigned = [p \in Owners |-> IF p \in Generated THEN 0 ELSE 1]
        /\ published = [n \in Names |-> "free"]
        /\ protoNames = [n \in Names |-> 0]
        /\ accepted = {}
        /\ rejected = {}
Available(p, n) == claims[n] \in {"free", p}
First(p) == CHOOSE i \in 1..Len(Candidates(p)):
              Available(p, Candidates(p)[i]) /\
              \A j \in 1..(i-1): ~Available(p, Candidates(p)[j])
Allocate(p) == LET n == IF SafeAllocation THEN Candidates(p)[First(p)]
                        ELSE Candidates(p)[1] IN
  /\ p \in Generated /\ assigned[p] = 0
  /\ assigned' = [assigned EXCEPT ![p] = n]
  /\ claims' = IF SafeAllocation THEN [claims EXCEPT ![n] = p] ELSE claims
  /\ UNCHANGED <<published, protoNames, accepted, rejected>>
Collect(p) == LET n == assigned[p] IN
  /\ n # 0 /\ p \notin accepted \cup rejected
  /\ IF CheckDeclarations /\ (published[n] \notin {"free", Shape(p)} \/ protoNames[n] \notin {0, Proto(p, n)})
        THEN /\ rejected' = rejected \cup {p}
             /\ UNCHANGED <<published, protoNames, accepted>>
        ELSE /\ published' = IF published[n] = "free"
                                  THEN [published EXCEPT ![n] = Shape(p)]
                                  ELSE published
             /\ protoNames' = IF protoNames[n] = 0 THEN [protoNames EXCEPT ![n] = Proto(p, n)] ELSE protoNames
             /\ accepted' = accepted \cup {p}
             /\ UNCHANGED rejected
  /\ UNCHANGED <<claims, assigned>>
Done == accepted \cup rejected = Owners
Next == (\E p \in Owners: Allocate(p) \/ Collect(p)) \/ (Done /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
WireCorrect == \A p \in accepted: published[assigned[p]] = Shape(p)
NamesMatch == \A p \in accepted: protoNames[assigned[p]] = Proto(p, assigned[p])
GeneratedAccepted == rejected \cap Generated = {}
UniqueGenerated == \A p \in Generated: assigned[p] # 0 =>
  /\ assigned[p] # 1
  /\ \A q \in Generated: p # q /\ assigned[q] # 0 => assigned[p] # assigned[q]
Terminates == <>Done
=============================================================================
