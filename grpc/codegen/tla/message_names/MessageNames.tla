--------------------------- MODULE MessageNames ---------------------------
EXTENDS Naturals, FiniteSets, Sequences
CONSTANT Shared
Positions == {"endpoint", "anonymous", "nested"}
Names == 1..9
DesignNames == {1}
Candidates(p) == CASE p = "endpoint" -> <<1, 2, 3, 4>>
                  [] p = "anonymous" -> <<2, 5, 6, 7>>
                  [] OTHER -> <<5, 8, 9>>
VARIABLES reservations, returned, counts, stable
vars == <<reservations, returned, counts, stable>>
Init == /\ reservations = [n \in Names |-> "free"]
        /\ returned = [p \in Positions |-> 0]
        /\ counts = [p \in Positions |-> 0]
        /\ stable = TRUE
Available(p, n) == n \notin DesignNames /\ reservations[n] \in {"free", p}
First(p) == CHOOSE i \in 1..Len(Candidates(p)):
              Available(p, Candidates(p)[i]) /\
              \A j \in 1..(i-1): ~Available(p, Candidates(p)[j])
Allocate(p) == LET n == Candidates(p)[First(p)] IN
  /\ counts[p] < 2
  /\ stable' = (stable /\ (returned[p] = 0 \/ returned[p] = n))
  /\ returned' = [returned EXCEPT ![p] = n]
  /\ counts' = [counts EXCEPT ![p] = @ + 1]
  /\ reservations' = IF Shared \/ p # "endpoint"
                        THEN [reservations EXCEPT ![n] = p]
                        ELSE reservations
Done == \A p \in Positions: counts[p] = 2
Next == (\E p \in Positions: Allocate(p)) \/ (Done /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
Unique == \A p, q \in Positions: p # q /\ returned[p] # 0 /\ returned[q] # 0
                                    => returned[p] # returned[q]
Stable == stable
Terminates == <>Done
=============================================================================
