-------------------------- MODULE RecursiveAliases --------------------------
EXTENDS FiniteSets
CONSTANT Rebind
Slots == {"root", "copy", "other"}
Nodes == Slots
IDs == {"A", "B"}
ID(n) == IF n = "other" THEN "B" ELSE "A"
VARIABLES canonical, bound, phase, normalized
vars == <<canonical, bound, phase, normalized>>
Init == /\ canonical = [id \in IDs |-> "none"]
        /\ bound = [s \in Slots |-> s]
        /\ phase = [s \in Slots |-> "unvisited"]
        /\ normalized = {}
Visit(s) == LET n == bound[s] IN
  /\ phase[s] = "unvisited"
  /\ IF canonical[ID(n)] = "none"
        THEN /\ canonical' = [canonical EXCEPT ![ID(n)] = n]
             /\ phase' = [phase EXCEPT ![s] = "active"]
             /\ UNCHANGED bound
        ELSE /\ bound' = IF Rebind THEN [bound EXCEPT ![s] = canonical[ID(n)]] ELSE bound
             /\ phase' = [phase EXCEPT ![s] = "done"]
             /\ UNCHANGED canonical
  /\ UNCHANGED normalized
Finish(s) == /\ phase[s] = "active"
             /\ normalized' = normalized \cup {bound[s]}
             /\ phase' = [phase EXCEPT ![s] = "done"]
             /\ UNCHANGED <<canonical, bound>>
Done == \A s \in Slots: phase[s] = "done"
Next == (\E s \in Slots: Visit(s) \/ Finish(s)) \/ (Done /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
NormalizedAliases == Done => \A s \in Slots: bound[s] \in normalized
CanonicalAliases == Done => \A s,t \in Slots: ID(bound[s]) = ID(bound[t]) => bound[s] = bound[t]
Terminates == <>Done
=============================================================================
