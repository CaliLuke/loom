------------------------- MODULE MigrationOrdering -------------------------
EXTENDS Naturals, TLC
CONSTANT Policy
VARIABLES phase, incompatible, migrated, rejected, admitted
vars == <<phase,incompatible,migrated,rejected,admitted>>
Init == /\ phase = 0 /\ incompatible \in BOOLEAN /\ migrated = FALSE
        /\ rejected = FALSE /\ admitted = FALSE
Acquire == /\ ~rejected /\ ~migrated
           /\ (IF Policy = "Before" THEN phase = 0 ELSE admitted)
           /\ migrated' = TRUE /\ phase' = phase + 1
           /\ UNCHANGED <<incompatible,rejected,admitted>>
Admit == /\ ~admitted /\ ~rejected
         /\ (IF Policy = "Before" THEN migrated ELSE phase = 0)
         /\ rejected' = incompatible /\ admitted' = ~incompatible
         /\ phase' = phase + 1 /\ UNCHANGED <<incompatible,migrated>>
Next == Acquire \/ Admit
Spec == Init /\ [][Next]_vars
RejectPreservesState == rejected => ~migrated
=============================================================================
