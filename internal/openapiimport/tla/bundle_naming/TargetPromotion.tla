-------------------------- MODULE TargetPromotion --------------------------
EXTENDS Naturals, FiniteSets
CONSTANT Policy
VARIABLES unseen, discovered, allocated, locations, phase
Targets == 1..4
\* 1 is an authored component; 2/3 are local nested targets; 4 is external.
vars == <<unseen,discovered,allocated,locations,phase>>
Init == /\ unseen = Targets /\ discovered = {} /\ allocated = {}
        /\ locations = [t \in Targets |-> 0] /\ phase = "discover"
Discover == /\ phase = "discover" /\ unseen # {}
            /\ \E t \in unseen:
                 /\ unseen' = unseen \ {t}
                 /\ discovered' = discovered \cup {t}
            /\ UNCHANGED <<allocated,locations,phase>>
Start == /\ phase = "discover" /\ unseen = {} /\ phase' = "allocate"
         /\ UNCHANGED <<unseen,discovered,allocated,locations>>
Allocate == /\ phase = "allocate" /\ allocated # discovered
            /\ LET t == CHOOSE n \in discovered \ allocated: \A m \in discovered \ allocated: n <= m
               IN /\ locations' = [locations EXCEPT ![t] =
                     IF t = 1 THEN 1 ELSE IF Policy = "entryShortcut" /\ t \in {2,3} THEN 0 ELSE t+1]
                  /\ allocated' = allocated \cup {t}
            /\ UNCHANGED <<unseen,discovered,phase>>
Finish == /\ phase = "allocate" /\ allocated = discovered /\ phase' = "done"
          /\ UNCHANGED <<unseen,discovered,allocated,locations>>
Next == Discover \/ Start \/ Allocate \/ Finish \/ (phase = "done" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
Consumable == phase = "done" => \A t \in Targets: locations[t] > 0
EntryStable == phase = "done" => locations[1] = 1
DeterministicClaims == phase = "done" => \A t \in {2,3,4}: locations[t] = t+1
=============================================================================
