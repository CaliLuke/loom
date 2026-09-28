-------------------------- MODULE ProcessLease --------------------------
EXTENDS Naturals, FiniteSets
CONSTANT Mode
VARIABLES parent, lease, guard, pending, started, live
vars == <<parent, lease, guard, pending, started, live>>
Init == /\ parent = TRUE /\ lease = TRUE /\ guard = "absent"
        /\ pending = FALSE /\ started = FALSE /\ live = {}
StartGuard == /\ parent /\ lease /\ guard = "absent" /\ Mode # "legacy"
              /\ guard' = "alive"
              /\ UNCHANGED <<parent,lease,pending,started,live>>
BeginChild == /\ parent /\ lease /\ ~pending /\ ~started
              /\ (Mode \in {"legacy","late-guard"} \/ guard = "alive")
              /\ pending' = TRUE
              /\ UNCHANGED <<parent,lease,guard,started,live>>
FinishChild == /\ pending /\ pending' = FALSE /\ started' = TRUE
               /\ live' = IF Mode \in {"legacy","late-guard"} \/ guard = "alive" THEN {"child"} ELSE {}
               /\ UNCHANGED <<parent,lease,guard>>
Fork == /\ "child" \in live /\ "descendant" \notin live
        /\ live' = live \cup {"descendant"}
        /\ UNCHANGED <<parent,lease,guard,pending,started>>
ParentDies == /\ parent /\ parent' = FALSE /\ lease' = FALSE
              /\ UNCHANGED <<guard,pending,started,live>>
Cancel == /\ parent /\ lease /\ lease' = FALSE
          /\ UNCHANGED <<parent,guard,pending,started,live>>
ChildExits == /\ "child" \in live /\ live' = live \ {"child"} /\ lease' = FALSE
              /\ UNCHANGED <<parent,guard,pending,started>>
Reap == /\ guard = "alive" /\ ~lease
        /\ (Mode # "checked" \/ ~pending)
        /\ (Mode # "inherited-lease" \/ live = {})
        /\ guard' = "dead" /\ live' = {}
        /\ UNCHANGED <<parent,lease,pending,started>>
Next == StartGuard \/ BeginChild \/ FinishChild \/ Fork \/ ParentDies \/ Cancel \/ ChildExits \/ Reap
Spec == Init /\ [][Next]_vars /\ WF_vars(FinishChild) /\ WF_vars(Reap)
Owned == live # {} => parent \/ guard = "alive"
Reaped == ~lease ~> (live = {} /\ ~pending)
=============================================================================
