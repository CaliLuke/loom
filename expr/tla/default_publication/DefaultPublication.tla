-------------------- MODULE DefaultPublication --------------------
EXTENDS Naturals, FiniteSets
CONSTANT Policy
VARIABLES captured, local, publication, live, prepared, done, mutated
vars == <<captured, local, publication, live, prepared, done, mutated>>
Positions == 1..5
Absent == [present |-> FALSE, owner |-> 0, value |-> 0]
Entry(owner, value) == [present |-> TRUE, owner |-> owner, value |-> value]
\* 1 declares; 2 publishes an inherited default; 3 authors an equal override;
\* 4 inherits semantically but does not publish a reference-local annotation.
\* 5 projects a distinct target default; its semantic source has owner 1.
Published == [p \in Positions |-> CASE p \in {1, 2} -> Entry(1, 7)
                                  [] p = 3 -> Entry(3, 7)
                                  [] p = 5 -> Entry(5, 8)
                                  [] OTHER -> Absent]
Authored == [p \in Positions |-> p \in {1, 3, 5}]
SourceOwner == [p \in Positions |-> IF p = 3 THEN 3 ELSE 1]
Init == /\ captured = FALSE
        /\ local = [p \in Positions |-> FALSE]
        /\ publication = [p \in Positions |-> Absent]
        /\ live = Published
        /\ prepared = [p \in Positions |-> Absent]
        /\ done = {}
        /\ mutated = FALSE
Capture == /\ ~captured
           /\ captured' = TRUE
           /\ publication' = live
           /\ local' = IF Policy = "promote"
                        THEN [p \in Positions |-> live[p].present]
                        ELSE Authored
           /\ UNCHANGED <<live, prepared, done, mutated>>
Mutate == /\ captured /\ ~mutated
          /\ live' = [live EXCEPT ![1] = Entry(5, 9)]
          /\ mutated' = TRUE
          /\ UNCHANGED <<captured, local, publication, prepared, done>>
Prepare == \E p \in Positions \ done:
           /\ captured
           /\ prepared' = [prepared EXCEPT ![p] =
               CASE Policy = "local" -> IF local[p] THEN publication[p] ELSE Absent
                 [] Policy = "live" -> live[p]
                 [] Policy = "source-owner" -> IF publication[p].present
                     THEN [publication[p] EXCEPT !.owner = SourceOwner[p]] ELSE Absent
                 [] OTHER -> publication[p]]
           /\ done' = done \cup {p}
           /\ UNCHANGED <<captured, local, publication, live, mutated>>
Spec == Init /\ [][Capture \/ Mutate \/ Prepare]_vars
LocalAuthorship == captured => local = Authored
PublishedDefaults == \A p \in done: prepared[p] = Published[p]
NoInventedDefault == 4 \in done => ~prepared[4].present
EqualValueOwners == {1, 3} \subseteq done => prepared[1].owner # prepared[3].owner
===================================================================
