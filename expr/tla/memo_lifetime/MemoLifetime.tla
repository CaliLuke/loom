-------------------------- MODULE MemoLifetime --------------------------
EXTENDS Naturals, FiniteSets
CONSTANT GlobalOwner, EarlyRelease
VARIABLES held, slot, calls, state, choice, observed
vars == <<held, slot, calls, state, choice, observed>>
Owners == 1..3
Workers == 1..2
Keys == Owners \X (1..2)
None == <<0,0>>
Init == /\ held = Owners
        /\ slot = [k \in Keys |-> 0]
        /\ calls = [k \in Keys |-> 0]
        /\ state = [w \in Workers |-> "idle"]
        /\ choice = [w \in Workers |-> None]
        /\ observed = {}
Start(w,k) ==
 /\ state[w] = "idle" /\ k[1] \in held
 /\ choice' = [choice EXCEPT ![w] = k]
 /\ state' = [state EXCEPT ![w] = IF slot[k] = 0 THEN "creator" ELSE "reader"]
 /\ slot' = [slot EXCEPT ![k] = IF @ = 0 THEN 1 ELSE @]
 /\ calls' = [calls EXCEPT ![k] = IF slot[k] = 0 THEN @+1 ELSE @]
 /\ UNCHANGED <<held, observed>>
Publish(w) ==
 /\ state[w] = "creator"
 /\ slot' = [slot EXCEPT ![choice[w]] = 2]
 /\ state' = [state EXCEPT ![w] = "done"]
 /\ observed' = observed \cup {choice[w]}
 /\ UNCHANGED <<held, calls, choice>>
Read(w) ==
 /\ state[w] = "reader" /\ slot[choice[w]] = 2
 /\ state' = [state EXCEPT ![w] = "done"]
 /\ observed' = observed \cup {choice[w]}
 /\ UNCHANGED <<held, slot, calls, choice>>
Drop(o) == /\ o \in held /\ held' = held \ {o}
           /\ UNCHANGED <<slot, calls, state, choice, observed>>
Next == (\E w \in Workers, k \in Keys : Start(w,k))
        \/ (\E w \in Workers : Publish(w) \/ Read(w))
        \/ (\E o \in Owners : Drop(o))
Spec == Init /\ [][Next]_vars
Active == {choice[w][1] : w \in {x \in Workers : state[x] \in {"creator","reader"}}}
Used == {k[1] : k \in {x \in Keys : slot[x] # 0}}
Reachable == held \cup Active
Retained == IF GlobalOwner THEN Used ELSE IF EarlyRelease THEN Used \cap held ELSE Used \cap Reachable
NoUnusedRetention == Retained \subseteq Reachable
NoEarlyRelease == Active \subseteq Retained
Once == \A k \in Keys : calls[k] <= 1
Published == \A k \in observed : slot[k] = 2
=============================================================================
