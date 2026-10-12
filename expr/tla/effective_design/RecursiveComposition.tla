---------------------- MODULE RecursiveComposition ----------------------
EXTENDS Naturals, FiniteSets, TLC
CONSTANT Behavior
VARIABLES cyclic, phase, done, shape, snapshot, rejected, ownerBound
vars == <<cyclic, phase, done, shape, snapshot, rejected, ownerBound>>
Nodes == {"Base", "Node"}
Parents(n) == IF n = "Node" THEN {"Base"}
              ELSE IF cyclic THEN {"Node"} ELSE {}
Members(n) == IF n = "Base" THEN {"next"} ELSE {"n"}
Complete == [n \in Nodes |-> IF n = "Node" THEN {"n", "next"} ELSE {"next"}]
Init == /\ cyclic \in BOOLEAN
        /\ phase = "check"
        /\ done = {}
        /\ shape = [n \in Nodes |-> Members(n)]
        /\ snapshot = {}
        /\ rejected = FALSE
        /\ ownerBound = FALSE
\* Base.next references Node in both inputs. Only one has an Extend cycle.
Check == /\ phase = "check"
         /\ rejected' = (cyclic \/ Behavior = "mixed-cycle")
         /\ phase' = IF rejected' THEN "finished" ELSE "compose"
         /\ UNCHANGED <<cyclic, done, shape, snapshot, ownerBound>>
Compose(n) ==
    /\ phase = "compose"
    /\ n \notin done
    /\ Parents(n) \subseteq done
    /\ shape' = [shape EXCEPT ![n] = @ \cup UNION {shape[p]: p \in Parents(n)}]
    /\ done' = done \cup {n}
    \* Copying inherited Base.next during Node composition sees its old shape.
    /\ snapshot' = IF n = "Node" /\ Behavior = "early-copy"
                    THEN shape["Node"] ELSE snapshot
    /\ UNCHANGED <<cyclic, phase, rejected, ownerBound>>
Finish == /\ phase = "compose"
          /\ done = Nodes
          /\ snapshot' = IF Behavior \in {"complete", "complete-unbound"} THEN shape["Node"] ELSE snapshot
          /\ ownerBound' = (Behavior = "complete")
          /\ phase' = "finished"
          /\ UNCHANGED <<cyclic, done, shape, rejected>>
Next == Check \/ (\E n \in Nodes: Compose(n)) \/ Finish
Spec == Init /\ [][Next]_vars
OnlyInheritanceCyclesRejected == phase = "finished" => rejected = cyclic
CopiesSeeCompleteShape == phase = "finished" /\ ~rejected => snapshot = Complete["Node"]
StructuralRecursionUsesOwner == phase = "finished" /\ ~rejected => ownerBound
=============================================================================
