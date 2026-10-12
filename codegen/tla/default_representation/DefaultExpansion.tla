----------------------- MODULE DefaultExpansion -----------------------
EXTENDS Naturals, Sequences, TLC
CONSTANT Checked
VARIABLE recipe, pending, rejected, phase
vars == <<recipe, pending, rejected, phase>>
Children(node) == IF node = "A" THEN <<"B", "B">>
                  ELSE IF recipe = "cycle" THEN <<"A">> ELSE <<>>
Frame(node, path) == [node |-> node, path |-> path]
Extend(frame) == [i \in 1..Len(Children(frame.node)) |->
    Frame(Children(frame.node)[i], Append(frame.path, frame.node))]
HasCycle(path) == \E i,j \in 1..Len(path) : i # j /\ path[i] = path[j]
Init == /\ recipe \in {"shared", "cycle"}
        /\ pending = <<Frame("A", <<>>)>> /\ rejected = FALSE /\ phase = "expand"
Step == /\ phase = "expand" /\ Len(pending) > 0
        /\ LET frame == Head(pending)
               repeats == frame.node \in {frame.path[i] : i \in 1..Len(frame.path)}
           IN IF Checked /\ repeats
              THEN /\ rejected' = TRUE /\ pending' = <<>>
              ELSE /\ pending' = Extend(frame) \o Tail(pending)
                   /\ UNCHANGED rejected
        /\ UNCHANGED <<recipe, phase>>
Finish == /\ phase = "expand" /\ pending = <<>> /\ phase' = "done"
          /\ UNCHANGED <<recipe, pending, rejected>>
Spec == Init /\ [][Step \/ Finish]_vars
FinitePath == \A i \in 1..Len(pending) : ~HasCycle(pending[i].path)
CorrectOutcome == phase = "done" => (rejected = (recipe = "cycle"))
=============================================================================
