-------------------------- MODULE Completion --------------------------
EXTENDS Naturals
CONSTANT Policy
VARIABLES complete, copies, calls, phase
vars == <<complete, copies, calls, phase>>
Kinds == {"type", "result"}
Fields == {"id", "name"}
Init == /\ complete = [k \in Kinds |-> FALSE]
        /\ copies = [k \in Kinds |-> {}]
        /\ calls = [k \in Kinds |-> 0]
        /\ phase = [k \in Kinds |-> "pending"]
Localize(k) == /\ phase[k] = "pending"
               /\ LET registered == Policy = "checked" \/ k = "type"
                  IN /\ copies' = [copies EXCEPT ![k] = IF registered THEN Fields ELSE {}]
                     /\ complete' = [complete EXCEPT ![k] = registered]
                     /\ calls' = [calls EXCEPT ![k] = IF registered THEN 1 ELSE 0]
               /\ phase' = [phase EXCEPT ![k] = "copied"]
Walk(k) == /\ phase[k] = "copied"
           /\ calls' = [calls EXCEPT ![k] = IF complete[k] THEN @ ELSE @ + 1]
           /\ complete' = [complete EXCEPT ![k] = TRUE]
           /\ phase' = [phase EXCEPT ![k] = "done"]
           /\ UNCHANGED copies
Next == \E k \in Kinds: Localize(k) \/ Walk(k)
Spec == Init /\ [][Next]_vars
CompleteCopies == \A k \in Kinds: phase[k] # "pending" => copies[k] = Fields
Once == \A k \in Kinds: calls[k] <= 1
=============================================================================
