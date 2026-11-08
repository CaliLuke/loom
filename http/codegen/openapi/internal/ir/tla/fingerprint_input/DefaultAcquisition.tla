-------------------- MODULE DefaultAcquisition --------------------
EXTENDS Naturals, FiniteSets, Sequences
CONSTANT Policy
VARIABLES mode, prepared, preparations, acquisitionCallbacks, acquired
vars == <<mode, prepared, preparations, acquisitionCallbacks, acquired>>
Positions == {1, 2, 3}
Owner == [p \in Positions |-> IF p = 3 THEN 2 ELSE 1]
Requests == <<1, 2, 3, 1>>
Entries == {acquired[i] : i \in 1..Len(acquired)}
Init == /\ mode \in {"builtin", "opaque"}
        /\ prepared = {}
        /\ preparations = [p \in Positions |-> 0]
        /\ acquisitionCallbacks = 0
        /\ acquired = <<>>
Prepare == \E p \in Positions \ prepared:
           /\ prepared' = prepared \cup {p}
           /\ preparations' = [preparations EXCEPT ![p] = @ + 1]
           /\ UNCHANGED <<mode, acquisitionCallbacks, acquired>>
Key(p) == IF mode = "builtin" THEN <<"wire", Owner[p]>>
          ELSE IF Policy = "collapse" THEN <<"opaque", 0>>
          ELSE IF Policy = "position" THEN <<"opaque", p>> ELSE <<"opaque", Owner[p]>>
Acquire == /\ Len(acquired) < Len(Requests)
           /\ LET p == Requests[Len(acquired) + 1] IN
               /\ p \in prepared
               /\ acquired' = Append(acquired, [position |-> p, owner |-> Owner[p], key |-> Key(p)])
               /\ preparations' = IF Policy = "reprepare"
                                   THEN [preparations EXCEPT ![p] = @ + 1]
                                   ELSE preparations
           /\ acquisitionCallbacks' = acquisitionCallbacks + (IF Policy = "raw" THEN 1 ELSE 0)
           /\ UNCHANGED <<mode, prepared>>
Spec == Init /\ [][Prepare \/ Acquire]_vars
PreparedBeforeAcquire == \A a \in Entries: a.position \in prepared
CallbackFreeAcquisition == acquisitionCallbacks = 0
SinglePreparation == \A p \in Positions: preparations[p] <= 1
DistinctDefaults == \A a, b \in Entries: a.owner # b.owner => a.key # b.key
ReusedDefault == \A a, b \in Entries: a.owner = b.owner => a.key = b.key
RepeatedIdentity == \A a, b \in Entries: a.position = b.position => a.key = b.key
===================================================================
