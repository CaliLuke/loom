---------------------- MODULE IndexTraversal ----------------------
EXTENDS Naturals, Sequences, FiniteSets, TLC

CONSTANTS Mode, AnchorKind
VARIABLES pending, globallySeen, covered, cycleEntered
vars == <<pending, globallySeen, covered, cycleEntered>>

Task(occ, kind, ancestors) ==
  [occurrence |-> occ, kind |-> kind, ancestors |-> ancestors]
Init == /\ pending = <<Task("first", "mapping", {}), Task("second", "mapping", {})>>
        /\ globallySeen = {}
        /\ covered = {}
        /\ cycleEntered = FALSE

\* Two authored paths alias one source mapping. That mapping supplies Bound
\* and a cyclic alias edge. The source identity is anchor, independently of
\* the newly allocated effective view pointer at either occurrence.
Step == /\ Len(pending) > 0
        /\ LET task == Head(pending)
               rest == Tail(pending)
               repeated == "anchor" \in task.ancestors
               skip == Mode = "Raw" \/
                       (Mode = "GlobalSeen" /\ "anchor" \in globallySeen) \/
                       (Mode = "PathSeen" /\ repeated) \/
                       (Mode = "MapOnly" /\ AnchorKind = "mapping" /\ repeated)
           IN IF task.kind = "leaf"
              THEN /\ pending' = rest
                   /\ covered' = covered \cup {task.occurrence}
                   /\ UNCHANGED <<globallySeen, cycleEntered>>
              ELSE IF skip
                   THEN /\ pending' = rest
                        /\ UNCHANGED <<globallySeen, covered, cycleEntered>>
                   ELSE /\ pending' = rest \o
                           <<Task(task.occurrence, "leaf", task.ancestors \cup {"anchor"}),
                             Task(task.occurrence, "mapping", task.ancestors \cup {"anchor"})>>
                        /\ globallySeen' = globallySeen \cup {"anchor"}
                        /\ cycleEntered' = (cycleEntered \/ repeated)
                        /\ UNCHANGED covered

Spec == Init /\ [][Step]_vars /\ WF_vars(Step)
NoCycleReentry == ~cycleEntered
OccurrenceCoverage == Len(pending) = 0 => covered = {"first", "second"}
Complete == <> (Len(pending) = 0)
===================================================================
