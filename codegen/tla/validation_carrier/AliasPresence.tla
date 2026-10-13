---------------------- MODULE AliasPresence ----------------------
EXTENDS Naturals
CONSTANT Mode
VARIABLES depth, phase, rootFlags, childNullable, originalNullable
vars == <<depth, phase, rootFlags, childNullable, originalNullable>>
Init == /\ depth \in 0..3
        /\ phase = "wrapped"
        /\ rootFlags = [i \in 0..depth |-> i = depth]
        /\ childNullable = TRUE
        /\ originalNullable = TRUE
Unwrap == /\ phase = "wrapped"
          /\ phase' = "concrete"
          /\ rootFlags' = [i \in 0..depth |->
                IF Mode = "shallow" /\ i > 1 THEN rootFlags[i] ELSE FALSE]
          /\ childNullable' = (Mode # "whole-graph")
          /\ UNCHANGED <<depth, originalNullable>>
Next == Unwrap
Spec == Init /\ [][Next]_vars
RootConcrete == phase = "concrete" => \A i \in 0..depth: ~rootFlags[i]
ChildPreserved == childNullable
SourcePreserved == originalNullable
==================================================================
