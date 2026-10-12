------------------------------ MODULE GenPolicy ------------------------------
EXTENDS TLC
CONSTANTS RejectGen, ResolveAliases
VARIABLES inside, alias, createOnly, phase, unchanged
vars == <<inside, alias, createOnly, phase, unchanged>>
Init == /\ inside \in BOOLEAN /\ alias \in BOOLEAN /\ createOnly \in BOOLEAN
        /\ phase = "declared" /\ unchanged = TRUE
Admit == /\ phase = "declared"
         /\ phase' = IF RejectGen /\ (inside \/ (ResolveAliases /\ alias)) /\ createOnly THEN "rejected" ELSE "installed"
         /\ unchanged' = (phase' = "rejected")
         /\ UNCHANGED <<inside, alias, createOnly>>
Spec == Init /\ [][Admit]_vars
RejectContradiction == (inside \/ alias) /\ createOnly => unchanged
AdmitNormalOutputs == phase = "rejected" => (inside \/ alias) /\ createOnly
=============================================================================
