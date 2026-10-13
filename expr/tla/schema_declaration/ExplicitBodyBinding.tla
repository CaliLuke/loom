------------------------ MODULE ExplicitBodyBinding ------------------------
EXTENDS Naturals, FiniteSets
CONSTANT Mode
VARIABLES completed, bindings, authoredOwner
vars == <<completed, bindings, authoredOwner>>
Endpoints == {1, 2}
Levels == 0..2
NoSource == 0
Authored == 3
Init ==
    /\ completed = {}
    /\ bindings = [e \in Endpoints |-> [level \in Levels |-> NoSource]]
    /\ authoredOwner = Authored
Finalize(e) ==
    /\ e \notin completed
    /\ completed' = completed \cup {e}
    /\ bindings' = [bindings EXCEPT ![e] =
          [level \in Levels |-> IF Mode \in {"current", "root-only"} /\ level > 0 THEN NoSource ELSE e]]
    /\ authoredOwner' = IF Mode \in {"current", "shared-spine"} THEN e ELSE authoredOwner
Next == (\E e \in Endpoints : Finalize(e))
        \/ (completed = Endpoints /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
CompleteBinding == \A e \in completed : \A level \in Levels : bindings[e][level] = e
AuthoredOwnerPreserved == authoredOwner = Authored
IndependentOccurrences == \A e \in completed : bindings[e][0] = e
=============================================================================
