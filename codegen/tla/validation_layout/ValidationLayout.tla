---------------------- MODULE ValidationLayout ----------------------
EXTENDS Naturals, FiniteSets, TLC
CONSTANTS Mode, PresenceCases
VARIABLES physical, recursive, pending, definitions, calls, structural
vars == <<physical, recursive, pending, definitions, calls, structural>>
Nodes == {"root", "peer", "leaf"}
Children(t) == CASE t = "root" -> {"peer"}
                   [] t = "peer" -> {"leaf"}
                   [] OTHER -> IF recursive THEN {"peer"} ELSE {}

(* A named Leaf has required primitive storage. Its actual declaration chooses
   pointer presence independently of the caller's union branch value mode. *)
Structural(t) == structural /\ t = "peer"
RECURSIVE Eligible(_, _, _)
Eligible(t, caller, seen) ==
 IF t \in seen THEN FALSE
 ELSE (Mode = "checked" /\ Structural(t))
      \/ (t = "leaf" /\ IF Mode = "old" THEN caller ELSE physical)
      \/ (\E child \in Children(t): Eligible(child, caller, seen \cup {t}))
CallMode(t) == t # "peer"
HasBody(t) == Structural(t) \/ (t = "leaf" /\ physical)
             \/ (\E child \in Children(t): Eligible(child, CallMode(t), {}))

Init == /\ physical \in BOOLEAN
        /\ structural \in PresenceCases
        /\ recursive \in BOOLEAN
        /\ pending = Nodes
        /\ definitions = {}
        /\ calls = {}
Compile == /\ pending # {}
           /\ \E t \in pending:
                /\ definitions' = IF HasBody(t) THEN definitions \cup {t} ELSE definitions
                /\ calls' = calls \cup {<<t, child>> : child \in {c \in Children(t): Eligible(c, CallMode(t), {})}}
                /\ pending' = pending \ {t}
           /\ UNCHANGED <<physical, recursive, structural>>
Spec == Init /\ [][Compile]_vars /\ WF_vars(Compile)
CallClosure == pending = {} => \A call \in calls: call[2] \in definitions
PredicateCoverage == pending = {} /\ physical =>
                     <<"root", "peer">> \in calls /\ <<"peer", "leaf">> \in calls
StructuralCoverage == pending = {} /\ structural =>
                      <<"root", "peer">> \in calls /\ "peer" \in definitions
NoFalseRequirement == pending = {} /\ ~physical /\ ~structural => calls = {} /\ definitions = {}
FiniteRegistry == definitions \subseteq Nodes /\ pending \subseteq Nodes
Terminates == <> (pending = {})
=============================================================================
