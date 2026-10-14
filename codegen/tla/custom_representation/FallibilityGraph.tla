---------------------- MODULE FallibilityGraph ----------------------
EXTENDS FiniteSets, TLC
CONSTANT Strategy
VARIABLES defaultEnabled, crossing, marked, visited, done
vars == <<defaultEnabled, crossing, marked, visited, done>>
Nodes == {"AFull", "BFull", "CFull", "ATiny", "BTiny", "Leaf", "NativeA", "NativeB"}
Direct == IF defaultEnabled \/ crossing THEN {"Leaf"} ELSE {}
Children(n) == CASE n = "AFull" -> {"BFull", "CFull"}
                 [] n = "BFull" -> {"AFull", "Leaf"}
                 [] n = "CFull" -> {"Leaf"}
                 [] n = "ATiny" -> {"BTiny"}
                 [] n = "BTiny" -> {"ATiny"}
                 [] n = "NativeA" -> {"NativeB"}
                 [] n = "NativeB" -> {"NativeA"}
                 [] OTHER -> {}
Expected == IF defaultEnabled \/ crossing THEN {"AFull", "BFull", "CFull", "Leaf"} ELSE {}
CanFail(n) == n \in Direct \/ Children(n) \cap marked # {}
Add(n) == IF Strategy = "SharedKey" /\ n = "AFull" THEN {n, "ATiny"}
          ELSE IF Strategy = "SharedKey" /\ n = "BFull" THEN {n, "BTiny"}
          ELSE {n}

Init == /\ defaultEnabled \in BOOLEAN
        /\ crossing \in BOOLEAN
        /\ marked = Direct
        /\ visited = {}
        /\ done = FALSE

Visit == /\ Strategy = "Visited"
         /\ ~done
         /\ \E n \in Nodes \ visited:
               /\ visited' = visited \cup {n}
               /\ marked' = IF CanFail(n) THEN marked \cup {n} ELSE marked
         /\ UNCHANGED <<defaultEnabled, crossing, done>>

Propagate == /\ Strategy # "Visited"
             /\ ~done
             /\ \E n \in Nodes \ marked:
                   /\ CanFail(n)
                   /\ marked' = marked \cup Add(n)
             /\ UNCHANGED <<defaultEnabled, crossing, visited, done>>

Finish == /\ ~done
          /\ IF Strategy = "Visited" THEN visited = Nodes
             ELSE \A n \in Nodes: CanFail(n) => n \in marked
          /\ done' = TRUE
          /\ UNCHANGED <<defaultEnabled, crossing, visited, marked>>

Next == Visit \/ Propagate \/ Finish
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
SelectedPairsOnly == marked \cap {"ATiny", "BTiny", "NativeA", "NativeB"} = {}
CompleteClassification == done => marked = Expected
Completion == <> done
=====================================================================
