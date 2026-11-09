-------------------------- MODULE NamedAdmission --------------------------
EXTENDS Naturals, FiniteSets
CONSTANT Policy
VARIABLES policies, links, extra, accepted, done
vars == <<policies, links, extra, accepted, done>>
Nodes == 1..3
RECURSIVE Reach(_, _)
Reach(n, seen) == IF n = 0 \/ n \in seen THEN {} ELSE {n} \cup Reach(links[n], seen \cup {n})
Closed == \E n \in Reach(1, {}): policies[n] = "closed"
Init == /\ policies \in [Nodes -> {"unset", "open", "closed"}]
        /\ links \in [Nodes -> 0..3]
        /\ extra \in BOOLEAN
        /\ accepted = FALSE
        /\ done = FALSE
Admit == /\ ~done
         /\ LET closed == CASE Policy = "occurrence" -> policies[1] = "closed"
                              [] Policy = "global" -> \E n \in Nodes: policies[n] = "closed"
                              [] OTHER -> Closed
            IN accepted' = ~(extra /\ closed)
         /\ done' = TRUE
         /\ UNCHANGED <<policies, links, extra>>
Spec == Init /\ [][Admit]_vars
Contract == done => accepted = ~(extra /\ Closed)
=============================================================================
